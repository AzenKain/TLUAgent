package embedding

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"tluagent-web/pkg/config"
	"tluagent-web/pkg/netx"
)

const downloadHTTPTimeout = 30 * time.Minute

const (
	JobStateRunning = "running"
	JobStateDone    = "done"
	JobStateError   = "error"
)

const (
	DownloadKindModel   = "model"
	DownloadKindRuntime = "runtime"
)

var downloadHostPatterns = []string{
	"huggingface.co",
	"*.hf.co",
	"cdn-lfs*.huggingface.co",
	"github.com",
	"objects.githubusercontent.com",
	"release-assets*.githubusercontent.com",
}

// IsHostAllowed reports whether a download host matches the static allowlist.
func IsHostAllowed(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, pattern := range downloadHostPatterns {
		if matchHostPattern(pattern, host) {
			return true
		}
	}
	return false
}

func matchHostPattern(pattern, host string) bool {
	prefix, suffix, hasStar := strings.Cut(pattern, "*")
	if !hasStar {
		return host == pattern
	}
	if len(host) < len(prefix)+len(suffix) {
		return false
	}
	return strings.HasPrefix(host, prefix) && strings.HasSuffix(host, suffix)
}

// ChecksumRecord stores the integrity metadata of one downloaded file.
type ChecksumRecord struct {
	File      string `json:"file"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
	URL       string `json:"url"`
}

// ChecksumFile is the sidecar JSON holding checksums of downloaded files.
type ChecksumFile struct {
	Files []ChecksumRecord `json:"files"`
}

// SaveChecksums atomically writes the sidecar JSON next to the downloaded files.
func SaveChecksums(dir string, checksums *ChecksumFile) error {
	data, err := marshalJSON(checksums)
	if err != nil {
		return fmt.Errorf("marshal checksums: %w", err)
	}
	return atomicWriteFile(filepath.Join(dir, ChecksumsFileName), data, 0o644)
}

// LoadChecksums returns the sidecar checksums of a directory, or nil when absent.
func LoadChecksums(dir string) *ChecksumFile {
	data, err := readBoundedFile(filepath.Join(dir, ChecksumsFileName), MaxChecksumsSizeBytes)
	if err != nil {
		return nil
	}
	checksums := &ChecksumFile{}
	if err := unmarshalJSON(data, checksums); err != nil {
		return nil
	}
	return checksums
}

// FindChecksum returns the checksum record of one file inside a sidecar.
func (c *ChecksumFile) FindChecksum(fileName string) *ChecksumRecord {
	for i := range c.Files {
		if c.Files[i].File == fileName {
			return &c.Files[i]
		}
	}
	return nil
}

// DownloadJob tracks one background download task of the registry.
type DownloadJob struct {
	JobID           string `json:"job_id"`
	PresetID        string `json:"preset_id"`
	State           string `json:"state"`
	DownloadedBytes int64  `json:"downloaded_bytes"`
	TotalBytes      int64  `json:"total_bytes"`
	Error           string `json:"error,omitempty"`
}

// Downloader runs catalog downloads with an in-memory background job registry.
type Downloader struct {
	rootDir      string
	client       *http.Client
	maxFileBytes int64

	mu   sync.Mutex
	jobs map[string]*DownloadJob
}

// NewDownloader creates a downloader rooted at the ONNX data directory.
func NewDownloader(rootDir string) *Downloader {
	CleanupStaleTempFiles(rootDir, 1*time.Hour)
	maxMB := config.GetIntConfigWithDefault("EMBEDDING_MAX_DOWNLOAD_MB", 2048)
	return &Downloader{
		rootDir:      rootDir,
		client:       newDownloadHTTPClient(),
		maxFileBytes: int64(maxMB) << 20,
		jobs:         make(map[string]*DownloadJob),
	}
}

// Start launches a background download job for a catalog model or runtime preset.
func (d *Downloader) Start(presetID string) (*DownloadJob, error) {
	kind, model, runtimePreset, err := resolvePreset(presetID)
	if err != nil {
		return nil, err
	}
	d.mu.Lock()
	if len(d.jobs) >= 100 {
		d.evictOldJobsLocked()
	}
	for _, job := range d.jobs {
		if job.PresetID == presetID && job.State == JobStateRunning {
			d.mu.Unlock()
			return nil, fmt.Errorf("a download job for preset %q is already running", presetID)
		}
	}
	job := &DownloadJob{
		JobID:    uuid.NewString(),
		PresetID: presetID,
		State:    JobStateRunning,
	}
	if kind == DownloadKindModel {
		job.TotalBytes = sumFileSizes(model.FileURLs)
	} else {
		job.TotalBytes = runtimePreset.SizeBytes
	}
	d.jobs[job.JobID] = job
	d.mu.Unlock()

	if kind == DownloadKindModel {
		go d.runModelJob(job, model)
	} else {
		go d.runRuntimeJob(job, runtimePreset)
	}
	return d.jobSnapshot(job), nil
}

func (d *Downloader) evictOldJobsLocked() {
	for id, job := range d.jobs {
		if job.State != JobStateRunning {
			delete(d.jobs, id)
			if len(d.jobs) < 80 {
				break
			}
		}
	}
}

// Jobs returns a snapshot of all download jobs.
func (d *Downloader) Jobs() []DownloadJob {
	d.mu.Lock()
	defer d.mu.Unlock()
	jobs := make([]DownloadJob, 0, len(d.jobs))
	for _, job := range d.jobs {
		jobs = append(jobs, d.jobSnapshotLocked(job))
	}
	return jobs
}

func (d *Downloader) jobSnapshot(job *DownloadJob) *DownloadJob {
	d.mu.Lock()
	defer d.mu.Unlock()
	snapshot := d.jobSnapshotLocked(job)
	return &snapshot
}

func (d *Downloader) jobSnapshotLocked(job *DownloadJob) DownloadJob {
	snapshot := *job
	return snapshot
}

func (d *Downloader) failJob(job *DownloadJob, err error) {
	d.mu.Lock()
	job.State = JobStateError
	job.Error = err.Error()
	d.mu.Unlock()
}

func (d *Downloader) completeJob(job *DownloadJob) {
	d.mu.Lock()
	job.State = JobStateDone
	d.mu.Unlock()
}

func (d *Downloader) addProgress(job *DownloadJob, n int64) {
	d.mu.Lock()
	job.DownloadedBytes += n
	d.mu.Unlock()
}

func (d *Downloader) runModelJob(job *DownloadJob, preset CatalogModel) {
	ctx := context.Background()
	if err := d.downloadModel(ctx, job, preset); err != nil {
		d.failJob(job, err)
		return
	}
	d.completeJob(job)
}

func (d *Downloader) runRuntimeJob(job *DownloadJob, preset CatalogRuntime) {
	ctx := context.Background()
	if err := d.downloadRuntime(ctx, job, preset); err != nil {
		d.failJob(job, err)
		return
	}
	d.completeJob(job)
}

func (d *Downloader) downloadModel(ctx context.Context, job *DownloadJob, preset CatalogModel) error {
	destDir := filepath.Join(d.rootDir, preset.PresetID)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create model directory: %w", err)
	}
	manifest := preset.Manifest
	manifest.ModelID = preset.PresetID
	checksums := &ChecksumFile{}
	for _, file := range preset.FileURLs {
		record, err := d.downloadFile(ctx, job, file.URL, filepath.Join(destDir, file.DestFile), 0o644)
		if err != nil {
			return err
		}
		if file.SHA256 != "" && !strings.EqualFold(record.SHA256, file.SHA256) {
			return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", file.DestFile, file.SHA256, record.SHA256)
		}
		checksums.Files = append(checksums.Files, *record)
	}
	manifestData, err := marshalJSON(&manifest)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := atomicWriteFile(filepath.Join(destDir, ManifestFileName), manifestData, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	if err := SaveChecksums(destDir, checksums); err != nil {
		return fmt.Errorf("write checksums: %w", err)
	}
	return nil
}

func (d *Downloader) downloadRuntime(ctx context.Context, job *DownloadJob, preset CatalogRuntime) error {
	destDir := RuntimeDir(d.rootDir, preset.GOOS, preset.GOArch)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create runtime directory: %w", err)
	}
	libName, err := LibraryFileName(preset.GOOS)
	if err != nil {
		return err
	}
	tmpArchive, err := os.CreateTemp(destDir, ".archive-*")
	if err != nil {
		return fmt.Errorf("create temp archive: %w", err)
	}
	archivePath := tmpArchive.Name()
	defer os.Remove(archivePath)
	tmpArchive.Close()

	record, err := d.downloadFile(ctx, job, preset.URL, archivePath, 0o600)
	if err != nil {
		return err
	}
	if preset.ArchiveSHA256 != "" && !strings.EqualFold(record.SHA256, preset.ArchiveSHA256) {
		return fmt.Errorf("runtime archive checksum mismatch: expected %s, got %s", preset.ArchiveSHA256, record.SHA256)
	}
	switch preset.Archive {
	case "tgz":
		record, err = extractTGZ(archivePath, preset.InnerPath, destDir, libName, d.maxFileBytes)
	case "zip":
		record, err = extractZIP(archivePath, preset.InnerPath, destDir, libName, d.maxFileBytes)
	default:
		err = fmt.Errorf("unsupported archive format %q", preset.Archive)
	}
	if err != nil {
		return err
	}
	if preset.SHA256 != "" && !strings.EqualFold(record.SHA256, preset.SHA256) {
		return fmt.Errorf("runtime library checksum mismatch: expected %s, got %s", preset.SHA256, record.SHA256)
	}
	record.URL = preset.URL
	checksums := &ChecksumFile{Files: []ChecksumRecord{*record}}
	if err := SaveChecksums(destDir, checksums); err != nil {
		return fmt.Errorf("write checksums: %w", err)
	}
	return nil
}

func (d *Downloader) downloadFile(ctx context.Context, job *DownloadJob, rawURL, destPath string, mode os.FileMode) (*ChecksumRecord, error) {
	if err := validateDownloadURL(rawURL); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download url returned http status %d", resp.StatusCode)
	}
	if resp.ContentLength > d.maxFileBytes {
		return nil, fmt.Errorf("file exceeds max download size of %d MB", d.maxFileBytes>>20)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return nil, fmt.Errorf("create destination directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(destPath), ".download-*")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	hasher := sha256.New()
	writer := io.MultiWriter(tmp, hasher)
	var written int64
	buf := make([]byte, 64<<10)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if written+int64(n) > d.maxFileBytes {
				tmp.Close()
				return nil, fmt.Errorf("file exceeds max download size of %d MB", d.maxFileBytes>>20)
			}
			if _, writeErr := writer.Write(buf[:n]); writeErr != nil {
				tmp.Close()
				return nil, fmt.Errorf("write downloaded data: %w", writeErr)
			}
			written += int64(n)
			d.addProgress(job, int64(n))
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			tmp.Close()
			return nil, fmt.Errorf("read download stream: %w", readErr)
		}
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return nil, fmt.Errorf("chmod temp file: %w", err)
	}
	_ = os.Remove(destPath)
	if err := os.Rename(tmpName, destPath); err != nil {
		return nil, fmt.Errorf("move downloaded file into place: %w", err)
	}
	return &ChecksumRecord{
		File:      filepath.Base(destPath),
		SHA256:    hex.EncodeToString(hasher.Sum(nil)),
		SizeBytes: written,
		URL:       rawURL,
	}, nil
}

func validateDownloadURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse download url: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("download url must use https")
	}
	if parsed.Port() != "" && parsed.Port() != "443" {
		return fmt.Errorf("download url must use standard port 443")
	}
	if !IsHostAllowed(parsed.Hostname()) {
		return fmt.Errorf("download host %q is not in the allowlist", parsed.Hostname())
	}
	return nil
}

func newDownloadHTTPClient() *http.Client {
	client := netx.NewSafeHTTPClientWithConfig(downloadHTTPTimeout, false)
	baseCheck := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return fmt.Errorf("redirect scheme %q is not allowed, must be https", req.URL.Scheme)
		}
		if req.URL.Port() != "" && req.URL.Port() != "443" {
			return fmt.Errorf("redirect port %q is not allowed, must use standard port 443", req.URL.Port())
		}
		if !IsHostAllowed(req.URL.Hostname()) {
			return fmt.Errorf("redirect host %q is not in the allowlist", req.URL.Hostname())
		}
		if baseCheck != nil {
			return baseCheck(req, via)
		}
		return nil
	}
	return client
}

type budgetReader struct {
	r         io.Reader
	remaining int64
}

func (b *budgetReader) Read(p []byte) (int, error) {
	if b.remaining <= 0 {
		return 0, fmt.Errorf("decompressed archive stream exceeded maximum budget")
	}
	toRead := len(p)
	if int64(toRead) > b.remaining {
		toRead = int(b.remaining)
	}
	n, err := b.r.Read(p[:toRead])
	b.remaining -= int64(n)
	if b.remaining <= 0 && err == nil {
		return n, fmt.Errorf("decompressed archive stream exceeded maximum budget")
	}
	return n, err
}

func extractTGZ(archivePath, innerPath, destDir, destName string, maxBytes int64) (*ChecksumRecord, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("open gzip stream: %w", err)
	}
	defer gz.Close()
	budget := &budgetReader{r: gz, remaining: 4 * maxBytes}
	reader := tar.NewReader(budget)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("inner path %q not found in archive", innerPath)
		}
		if err != nil {
			return nil, fmt.Errorf("read tar entry: %w", err)
		}
		if header.Name != innerPath {
			continue
		}
		if header.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf("inner path %q is not a regular file", innerPath)
		}
		return streamToFile(reader, destDir, destName, maxBytes, 0o755)
	}
}

func extractZIP(archivePath, innerPath, destDir, destName string, maxBytes int64) (*ChecksumRecord, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open zip archive: %w", err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if file.Name != innerPath {
			continue
		}
		if file.FileInfo().IsDir() {
			return nil, fmt.Errorf("inner path %q is a directory", innerPath)
		}
		if file.UncompressedSize64 > uint64(maxBytes) {
			return nil, fmt.Errorf("file exceeds max download size of %d MB", maxBytes>>20)
		}
		src, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip entry: %w", err)
		}
		defer src.Close()
		return streamToFile(src, destDir, destName, maxBytes, 0o755)
	}
	return nil, fmt.Errorf("inner path %q not found in archive", innerPath)
}

func streamToFile(src io.Reader, destDir, destName string, maxBytes int64, mode os.FileMode) (*ChecksumRecord, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("create destination directory: %w", err)
	}
	destPath := filepath.Join(destDir, destName)
	tmp, err := os.CreateTemp(destDir, ".extract-*")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hasher), io.LimitReader(src, maxBytes+1))
	if err != nil {
		tmp.Close()
		return nil, fmt.Errorf("extract entry: %w", err)
	}
	if written > maxBytes {
		tmp.Close()
		return nil, fmt.Errorf("file exceeds max download size of %d MB", maxBytes>>20)
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return nil, fmt.Errorf("chmod extracted file: %w", err)
	}
	_ = os.Remove(destPath)
	if err := os.Rename(tmpName, destPath); err != nil {
		return nil, fmt.Errorf("move extracted file into place: %w", err)
	}
	return &ChecksumRecord{
		File:      destName,
		SHA256:    hex.EncodeToString(hasher.Sum(nil)),
		SizeBytes: written,
	}, nil
}

func resolvePreset(presetID string) (string, CatalogModel, CatalogRuntime, error) {
	catalog := BuiltinCatalog()
	for _, model := range catalog.Models {
		if model.PresetID == presetID {
			return DownloadKindModel, model, CatalogRuntime{}, nil
		}
	}
	for _, runtimePreset := range catalog.Runtimes {
		if runtimePreset.PresetID == presetID {
			return DownloadKindRuntime, CatalogModel{}, runtimePreset, nil
		}
	}
	return "", CatalogModel{}, CatalogRuntime{}, fmt.Errorf("unknown preset id %q", presetID)
}

func sumFileSizes(files []CatalogFileURL) int64 {
	var total int64
	for _, file := range files {
		total += file.SizeBytes
	}
	return total
}

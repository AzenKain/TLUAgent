package embedding

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsHostAllowed(t *testing.T) {
	allowed := []string{
		"huggingface.co",
		"cdn-lfs.huggingface.co",
		"cdn-lfs-us-1.huggingface.co",
		"cdn-lfs-eu-1.huggingface.co",
		"cas-bridge.xethub.hf.co",
		"transfer.xethub.hf.co",
		"github.com",
		"objects.githubusercontent.com",
		"release-assets.githubusercontent.com",
	}
	for _, host := range allowed {
		if !IsHostAllowed(host) {
			t.Fatalf("expected host %q to be allowed", host)
		}
	}
	rejected := []string{
		"evil.com",
		"huggingface.co.evil.com",
		"hf.co",
		"notgithub.com",
		"github.com.evil.com",
		"127.0.0.1",
		"localhost",
	}
	for _, host := range rejected {
		if IsHostAllowed(host) {
			t.Fatalf("expected host %q to be rejected", host)
		}
	}
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newFakeDownloader(t *testing.T, rootDir string, maxBytes int64, handler roundTripperFunc) *Downloader {
	t.Helper()
	downloader := NewDownloader(rootDir)
	downloader.maxFileBytes = maxBytes
	downloader.client = &http.Client{Transport: handler}
	return downloader
}

func serveBytes(status int, contentType string, body []byte) roundTripperFunc {
	return func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    status,
			Header:        http.Header{"Content-Type": []string{contentType}},
			Body:          io.NopCloser(bytes.NewReader(body)),
			ContentLength: int64(len(body)),
			Request:       req,
		}, nil
	}
}

func TestDownloadFile_StreamsAndHashes(t *testing.T) {
	root := t.TempDir()
	downloader := newFakeDownloader(t, root, 1<<20, serveBytes(http.StatusOK, "application/octet-stream", []byte("model-bytes")))
	job := &DownloadJob{JobID: "j1", State: JobStateRunning}
	dest := filepath.Join(root, "m", ModelFileName)
	record, err := downloader.downloadFile(context.Background(), job, "https://huggingface.co/org/repo/resolve/main/model.onnx", dest, 0o644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(data) != "model-bytes" {
		t.Fatalf("unexpected content: %q", data)
	}
	if record.SizeBytes != 11 || record.File != ModelFileName {
		t.Fatalf("unexpected record: %+v", record)
	}
	if len(record.SHA256) != 64 {
		t.Fatalf("unexpected sha256: %s", record.SHA256)
	}
	if job.DownloadedBytes != 11 {
		t.Fatalf("progress not tracked: %d", job.DownloadedBytes)
	}
}

func TestDownloadFile_SizeCapByBodyLength(t *testing.T) {
	root := t.TempDir()
	downloader := newFakeDownloader(t, root, 8, serveBytes(http.StatusOK, "application/octet-stream", []byte("0123456789abcdef")))
	job := &DownloadJob{JobID: "j2", State: JobStateRunning}
	_, err := downloader.downloadFile(context.Background(), job, "https://huggingface.co/model.onnx", filepath.Join(root, "model.onnx"), 0o644)
	if err == nil || !strings.Contains(err.Error(), "exceeds max download size") {
		t.Fatalf("expected size cap error, got: %v", err)
	}
}

func TestDownloadFile_SizeCapByContentLength(t *testing.T) {
	root := t.TempDir()
	handler := func(req *http.Request) (*http.Response, error) {
		resp := &http.Response{
			StatusCode:    http.StatusOK,
			Header:        http.Header{},
			Body:          io.NopCloser(bytes.NewReader(nil)),
			ContentLength: 1 << 30,
			Request:       req,
		}
		return resp, nil
	}
	downloader := newFakeDownloader(t, root, 1<<20, handler)
	job := &DownloadJob{JobID: "j3", State: JobStateRunning}
	_, err := downloader.downloadFile(context.Background(), job, "https://huggingface.co/model.onnx", filepath.Join(root, "model.onnx"), 0o644)
	if err == nil || !strings.Contains(err.Error(), "exceeds max download size") {
		t.Fatalf("expected content-length cap error, got: %v", err)
	}
}

func TestDownloadFile_RejectsNonAllowlistedHost(t *testing.T) {
	root := t.TempDir()
	downloader := newFakeDownloader(t, root, 1<<20, serveBytes(http.StatusOK, "text/plain", []byte("x")))
	job := &DownloadJob{JobID: "j4", State: JobStateRunning}
	_, err := downloader.downloadFile(context.Background(), job, "https://evil.com/model.onnx", filepath.Join(root, "model.onnx"), 0o644)
	if err == nil || !strings.Contains(err.Error(), "not in the allowlist") {
		t.Fatalf("expected allowlist rejection, got: %v", err)
	}
}

func TestDownloadFile_RejectsHTTPScheme(t *testing.T) {
	root := t.TempDir()
	downloader := newFakeDownloader(t, root, 1<<20, serveBytes(http.StatusOK, "text/plain", []byte("x")))
	job := &DownloadJob{JobID: "j5", State: JobStateRunning}
	_, err := downloader.downloadFile(context.Background(), job, "http://huggingface.co/model.onnx", filepath.Join(root, "model.onnx"), 0o644)
	if err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("expected https enforcement, got: %v", err)
	}
}

func TestDownloadFile_ErrorStatus(t *testing.T) {
	root := t.TempDir()
	downloader := newFakeDownloader(t, root, 1<<20, serveBytes(http.StatusNotFound, "text/plain", []byte("nope")))
	job := &DownloadJob{JobID: "j6", State: JobStateRunning}
	_, err := downloader.downloadFile(context.Background(), job, "https://huggingface.co/missing.onnx", filepath.Join(root, "model.onnx"), 0o644)
	if err == nil || !strings.Contains(err.Error(), "http status 404") {
		t.Fatalf("expected http status error, got: %v", err)
	}
}

func buildTestTGZ(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create tgz: %v", err)
	}
	defer file.Close()
	gz := gzip.NewWriter(file)
	writer := tar.NewWriter(gz)
	for name, content := range entries {
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}); err != nil {
			t.Fatalf("failed to write tar header: %v", err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write tar entry: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
}

func TestExtractTGZ_ExtractsOnlyInnerPath(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "ort.tgz")
	buildTestTGZ(t, archivePath, map[string]string{
		"onnxruntime-test-1.0/lib/libonnxruntime.so": "LIBDATA",
		"onnxruntime-test-1.0/README.md":             "readme",
	})
	destDir := filepath.Join(root, "runtime", "linux-amd64")
	record, err := extractTGZ(archivePath, "onnxruntime-test-1.0/lib/libonnxruntime.so", destDir, "libonnxruntime.so", 1<<20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(destDir, "libonnxruntime.so"))
	if err != nil || string(data) != "LIBDATA" {
		t.Fatalf("unexpected extracted content: %q (err %v)", data, err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "readme")); !os.IsNotExist(err) {
		t.Fatalf("entries outside inner_path must not be extracted")
	}
	if record.SHA256 == "" || record.SizeBytes != 7 {
		t.Fatalf("unexpected record: %+v", record)
	}
}

func TestExtractTGZ_InnerPathNotFound(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "ort.tgz")
	buildTestTGZ(t, archivePath, map[string]string{"other/file.txt": "x"})
	if _, err := extractTGZ(archivePath, "onnxruntime-test-1.0/lib/libonnxruntime.so", filepath.Join(root, "out"), "libonnxruntime.so", 1<<20); err == nil {
		t.Fatalf("expected error for missing inner path")
	}
}

func buildTestZIP(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create zip: %v", err)
	}
	defer file.Close()
	writer := zip.NewWriter(file)
	for name, content := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry: %v", err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write zip entry: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
}

func TestExtractZIP_ExtractsOnlyInnerPath(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "ort.zip")
	buildTestZIP(t, archivePath, map[string]string{
		"onnxruntime-test-1.0/lib/onnxruntime.dll": "DLLDATA",
		"onnxruntime-test-1.0/LICENSE":             "license",
	})
	destDir := filepath.Join(root, "runtime", "windows-amd64")
	record, err := extractZIP(archivePath, "onnxruntime-test-1.0/lib/onnxruntime.dll", destDir, "onnxruntime.dll", 1<<20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(destDir, "onnxruntime.dll"))
	if err != nil || string(data) != "DLLDATA" {
		t.Fatalf("unexpected extracted content: %q (err %v)", data, err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "LICENSE")); !os.IsNotExist(err) {
		t.Fatalf("entries outside inner_path must not be extracted")
	}
	if record.SizeBytes != 7 {
		t.Fatalf("unexpected record: %+v", record)
	}
}

func TestDownloadModel_EndToEnd(t *testing.T) {
	root := t.TempDir()
	catalog := BuiltinCatalog()
	preset := catalog.Models[0]
	files := make(map[string][]byte)
	for i, file := range preset.FileURLs {
		parsed, err := url.Parse(file.URL)
		if err != nil {
			t.Fatalf("failed to parse catalog url: %v", err)
		}
		var content []byte
		if file.DestFile == ModelFileName {
			content = []byte("fake-onnx-graph")
		} else {
			content = []byte(testVocabContent)
		}
		files[parsed.Path] = content
		hasher := sha256.New()
		hasher.Write(content)
		preset.FileURLs[i].SHA256 = hex.EncodeToString(hasher.Sum(nil))
	}
	handler := func(req *http.Request) (*http.Response, error) {
		body, ok := files[req.URL.Path]
		if !ok {
			return serveBytes(http.StatusNotFound, "text/plain", nil)(req)
		}
		return serveBytes(http.StatusOK, "application/octet-stream", body)(req)
	}
	downloader := newFakeDownloader(t, root, 1<<20, handler)
	job := &DownloadJob{JobID: "j7", PresetID: preset.PresetID, State: JobStateRunning}
	downloader.runModelJob(job, preset)
	if job.State != JobStateDone {
		t.Fatalf("expected done job, got %s (%s)", job.State, job.Error)
	}
	modelDir := filepath.Join(root, preset.PresetID)
	if _, err := os.Stat(filepath.Join(modelDir, ModelFileName)); err != nil {
		t.Fatalf("model file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(modelDir, VocabFileName)); err != nil {
		t.Fatalf("vocab file missing: %v", err)
	}
	manifest, err := LoadManifest(modelDir)
	if err != nil {
		t.Fatalf("downloaded manifest invalid: %v", err)
	}
	if manifest.ModelID != preset.PresetID {
		t.Fatalf("unexpected manifest model id: %s", manifest.ModelID)
	}
	checksums := LoadChecksums(modelDir)
	if checksums == nil || len(checksums.Files) != 2 {
		t.Fatalf("checksum sidecar missing or incomplete: %+v", checksums)
	}
	if record := checksums.FindChecksum(ModelFileName); record == nil || record.SHA256 == "" {
		t.Fatalf("model checksum missing: %+v", checksums)
	}
}

func TestDownloadModel_ChecksumMismatchRejection(t *testing.T) {
	root := t.TempDir()
	catalog := BuiltinCatalog()
	preset := catalog.Models[0]
	preset.FileURLs[0].SHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	handler := func(req *http.Request) (*http.Response, error) {
		return serveBytes(http.StatusOK, "application/octet-stream", []byte("tampered-content"))(req)
	}
	downloader := newFakeDownloader(t, root, 1<<20, handler)
	job := &DownloadJob{JobID: "j-mismatch", PresetID: preset.PresetID, State: JobStateRunning}
	downloader.runModelJob(job, preset)
	if job.State != JobStateError {
		t.Fatalf("expected job error on checksum mismatch, got %s", job.State)
	}
	if !strings.Contains(job.Error, "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got %s", job.Error)
	}
}

func TestDownloadRuntime_ChecksumMismatchRejection(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "archive.tgz")
	buildTestTGZ(t, archivePath, map[string]string{
		"onnxruntime-test-1.0/lib/libonnxruntime.so": "RUNTIMELIB",
	})
	archiveBytes, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("failed to read archive: %v", err)
	}
	handler := serveBytes(http.StatusOK, "application/gzip", archiveBytes)
	downloader := newFakeDownloader(t, root, 1<<20, handler)
	preset := CatalogRuntime{
		PresetID:      "onnxruntime-linux-amd64",
		GOOS:          "linux",
		GOArch:        "amd64",
		URL:           "https://github.com/microsoft/onnxruntime/releases/download/v1.30.0/onnxruntime-linux-x64-1.30.0.tgz",
		Archive:       "tgz",
		ArchiveSHA256: "0000000000000000000000000000000000000000000000000000000000000000",
		InnerPath:     "onnxruntime-test-1.0/lib/libonnxruntime.so",
	}
	job := &DownloadJob{JobID: "j-rt-mismatch", PresetID: preset.PresetID, State: JobStateRunning}
	downloader.runRuntimeJob(job, preset)
	if job.State != JobStateError {
		t.Fatalf("expected job error on archive checksum mismatch, got %s", job.State)
	}
	if !strings.Contains(job.Error, "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got %s", job.Error)
	}
}

func TestDownloadRuntime_EndToEnd(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "archive.tgz")
	buildTestTGZ(t, archivePath, map[string]string{
		"onnxruntime-test-1.0/lib/libonnxruntime.so": "RUNTIMELIB",
	})
	archiveBytes, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("failed to read archive: %v", err)
	}
	handler := serveBytes(http.StatusOK, "application/gzip", archiveBytes)
	downloader := newFakeDownloader(t, root, 1<<20, handler)
	preset := CatalogRuntime{
		PresetID:  "onnxruntime-linux-amd64",
		GOOS:      "linux",
		GOArch:    "amd64",
		URL:       "https://github.com/microsoft/onnxruntime/releases/download/v1.30.0/onnxruntime-linux-x64-1.30.0.tgz",
		Archive:   "tgz",
		InnerPath: "onnxruntime-test-1.0/lib/libonnxruntime.so",
	}
	job := &DownloadJob{JobID: "j8", PresetID: preset.PresetID, State: JobStateRunning}
	downloader.runRuntimeJob(job, preset)
	if job.State != JobStateDone {
		t.Fatalf("expected done job, got %s (%s)", job.State, job.Error)
	}
	libData, err := os.ReadFile(RuntimeLibraryPathOrSkip(t, root))
	if err != nil || string(libData) != "RUNTIMELIB" {
		t.Fatalf("unexpected runtime lib: %q (err %v)", libData, err)
	}
	checksums := LoadChecksums(RuntimeDir(root, "linux", "amd64"))
	if checksums == nil || len(checksums.Files) != 1 || checksums.Files[0].File != "libonnxruntime.so" {
		t.Fatalf("unexpected checksum sidecar: %+v", checksums)
	}
	if checksums.Files[0].URL == "" {
		t.Fatalf("runtime checksum must record the source url")
	}
}

func TestDownloader_StartUnknownPreset(t *testing.T) {
	root := t.TempDir()
	downloader := NewDownloader(root)
	if _, err := downloader.Start("does-not-exist"); err == nil {
		t.Fatalf("expected error for unknown preset")
	}
}

func TestDownloader_JobsSnapshot(t *testing.T) {
	root := t.TempDir()
	downloader := NewDownloader(root)
	if _, err := downloader.Start("no-such-preset"); err == nil {
		t.Fatalf("expected error for unknown preset")
	}
	if jobs := downloader.Jobs(); len(jobs) != 0 {
		t.Fatalf("no jobs must be registered on error")
	}
}

func RuntimeLibraryPathOrSkip(t *testing.T, root string) string {
	t.Helper()
	path, err := RuntimeLibraryPath(root, "linux", "amd64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return path
}

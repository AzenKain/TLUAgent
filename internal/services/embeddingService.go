package services

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/embedding"
	"tluagent-web/pkg/jsonx"
	"tluagent-web/pkg/llm"
)

const (
	SettingKeyEmbeddingProvider    = "EMBEDDING_PROVIDER"
	SettingKeyEmbeddingOnnxModelID = "EMBEDDING_ONNX_MODEL_ID"

	EmbeddingProviderAPI  = "api"
	EmbeddingProviderONNX = "onnx"

	OnnxModelStatusOK          = "ok"
	OnnxModelStatusInvalid     = "invalid"
	OnnxModelStatusUnvalidated = "unvalidated"
)

// EmbeddingService manages embedding provider settings, ONNX models, downloads, and validation.
type EmbeddingService interface {
	GetSettings(ctx context.Context) (response.EmbeddingSettingsResponse, error)
	UpdateSettings(ctx context.Context, req request.UpdateEmbeddingSettingsRequest) (response.EmbeddingSettingsResponse, error)
	ListOnnxModels(ctx context.Context) ([]response.OnnxModelResponse, error)
	GetCatalog() embedding.Catalog
	StartDownload(presetID string) (response.EmbeddingDownloadStartedResponse, error)
	ListDownloadJobs() []response.EmbeddingDownloadJobResponse
	ValidateModel(ctx context.Context, modelID string) (response.EmbeddingValidationResponse, error)
	ActivateModel(ctx context.Context, modelID string) (response.EmbeddingSettingsResponse, error)
	DeleteModel(ctx context.Context, modelID string) error
	SetLLMManager(manager *llm.Manager)
	SyncActiveEmbedder(ctx context.Context) error
}

type modelLockManager struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func newModelLockManager() *modelLockManager {
	return &modelLockManager{
		locks: make(map[string]*sync.Mutex),
	}
}

func (m *modelLockManager) tryLock(id string) (func(), bool) {
	m.mu.Lock()
	l, ok := m.locks[id]
	if !ok {
		l = &sync.Mutex{}
		m.locks[id] = l
	}
	m.mu.Unlock()
	if !l.TryLock() {
		return nil, false
	}
	return func() {
		l.Unlock()
	}, true
}

type embeddingService struct {
	settings   repositories.SettingsRepository
	downloader *embedding.Downloader
	rootDir    string
	modelLocks *modelLockManager
	llmManager *llm.Manager
}

// NewEmbeddingService creates an embedding service rooted at the ONNX data directory.
func NewEmbeddingService(settings repositories.SettingsRepository, rootDir string) EmbeddingService {
	return &embeddingService{
		settings:   settings,
		downloader: embedding.NewDownloader(rootDir),
		rootDir:    rootDir,
		modelLocks: newModelLockManager(),
	}
}

func (s *embeddingService) SetLLMManager(manager *llm.Manager) {
	s.llmManager = manager
}

func (s *embeddingService) SyncActiveEmbedder(ctx context.Context) error {
	if s.llmManager == nil {
		return nil
	}
	provider := s.getProviderSetting(ctx)
	if provider != EmbeddingProviderONNX {
		return nil
	}
	modelIDPtr := s.getModelIDSetting(ctx)
	if modelIDPtr == nil || *modelIDPtr == "" {
		return nil
	}
	modelID := *modelIDPtr
	modelDir := filepath.Join(s.rootDir, modelID)
	if _, err := os.Stat(modelDir); err != nil {
		return fmt.Errorf("onnx model directory does not exist: %w", err)
	}
	manifest, err := embedding.LoadManifest(modelDir)
	if err != nil {
		return fmt.Errorf("load onnx manifest: %w", err)
	}
	modelPath, err := embedding.FindModelFile(modelDir)
	if err != nil {
		return fmt.Errorf("find onnx model file: %w", err)
	}
	libPath, err := embedding.CurrentRuntimeLibraryPath(s.rootDir)
	if err != nil {
		return fmt.Errorf("resolve runtime library: %w", err)
	}
	embedder, err := embedding.NewOnnxEmbedder(modelDir, manifest, modelPath, libPath)
	if err != nil {
		return fmt.Errorf("create onnx embedder: %w", err)
	}
	s.llmManager.SetEmbedder(embedder)
	return nil
}

func (s *embeddingService) GetSettings(ctx context.Context) (response.EmbeddingSettingsResponse, error) {
	libPath, _ := embedding.CurrentRuntimeLibraryPath(s.rootDir)
	libPresent := false
	if libPath != "" {
		if _, err := os.Stat(libPath); err == nil {
			libPresent = true
		}
	}
	displayLibPath := libPath
	if rel, err := filepath.Rel(s.rootDir, libPath); err == nil && !strings.HasPrefix(rel, "..") {
		displayLibPath = filepath.ToSlash(rel)
	}
	return response.EmbeddingSettingsResponse{
		Provider:          s.getProviderSetting(ctx),
		ActiveOnnxModelID: s.getModelIDSetting(ctx),
		Runtime: response.EmbeddingRuntimeInfo{
			OS:         runtime.GOOS,
			Arch:       runtime.GOARCH,
			LibPresent: libPresent,
			LibPath:    displayLibPath,
		},
	}, nil
}

func (s *embeddingService) UpdateSettings(ctx context.Context, req request.UpdateEmbeddingSettingsRequest) (response.EmbeddingSettingsResponse, error) {
	provider := strings.TrimSpace(req.Provider)
	activeID := ""
	if req.ActiveOnnxModelID != nil {
		activeID = strings.TrimSpace(*req.ActiveOnnxModelID)
	}
	if activeID != "" {
		if err := embedding.SanitizeModelID(activeID); err != nil {
			return response.EmbeddingSettingsResponse{}, apperrors.New(apperrors.ErrBadRequest, err.Error())
		}
	}
	if provider == EmbeddingProviderONNX {
		target := activeID
		if target == "" {
			if current := s.getModelIDSetting(ctx); current != nil {
				target = *current
			}
		}
		if target == "" {
			return response.EmbeddingSettingsResponse{}, apperrors.New(apperrors.ErrBadRequest,
				"provider onnx requires active_onnx_model_id")
		}
		unlock, ok := s.modelLocks.tryLock(target)
		if !ok {
			return response.EmbeddingSettingsResponse{}, apperrors.New(apperrors.ErrConflict,
				"model is currently locked by another operation")
		}
		defer unlock()

		if _, err := embedding.VerifyValidationIntegrity(filepath.Join(s.rootDir, target)); err != nil {
			return response.EmbeddingSettingsResponse{}, apperrors.New(apperrors.ErrBadRequest,
				"the active model must be validated successfully before switching provider to onnx: "+err.Error())
		}
		if err := s.persistProvider(ctx, provider); err != nil {
			return response.EmbeddingSettingsResponse{}, err
		}
		if err := s.persistActiveModelID(ctx, target); err != nil {
			return response.EmbeddingSettingsResponse{}, err
		}
		_ = s.SyncActiveEmbedder(ctx)
		return s.GetSettings(ctx)
	}

	if err := s.persistProvider(ctx, provider); err != nil {
		return response.EmbeddingSettingsResponse{}, err
	}
	if activeID != "" {
		if err := s.persistActiveModelID(ctx, activeID); err != nil {
			return response.EmbeddingSettingsResponse{}, err
		}
	}
	return s.GetSettings(ctx)
}

func (s *embeddingService) ListOnnxModels(ctx context.Context) ([]response.OnnxModelResponse, error) {
	entries, err := os.ReadDir(s.rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []response.OnnxModelResponse{}, nil
		}
		return nil, err
	}
	activeID := s.getModelIDSetting(ctx)
	models := make([]response.OnnxModelResponse, 0)
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == embedding.RuntimeDirName || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		models = append(models, s.buildModelResponse(entry.Name(), activeID))
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ModelID < models[j].ModelID })
	return models, nil
}

func (s *embeddingService) GetCatalog() embedding.Catalog {
	return embedding.BuiltinCatalog()
}

func (s *embeddingService) StartDownload(presetID string) (response.EmbeddingDownloadStartedResponse, error) {
	if active := s.getModelIDSetting(context.Background()); active != nil && *active == presetID {
		return response.EmbeddingDownloadStartedResponse{}, apperrors.New(apperrors.ErrConflict,
			"cannot re-download the active onnx embedding model")
	}
	if !presetExists(presetID) {
		return response.EmbeddingDownloadStartedResponse{}, apperrors.New(apperrors.ErrBadRequest,
			"unknown preset id")
	}
	unlock, ok := s.modelLocks.tryLock(presetID)
	if !ok {
		return response.EmbeddingDownloadStartedResponse{}, apperrors.New(apperrors.ErrConflict,
			"model or preset is currently locked by another operation")
	}
	defer unlock()

	job, err := s.downloader.Start(presetID)
	if err != nil {
		return response.EmbeddingDownloadStartedResponse{}, apperrors.New(apperrors.ErrConflict, err.Error())
	}
	return response.EmbeddingDownloadStartedResponse{JobID: job.JobID}, nil
}

func (s *embeddingService) ListDownloadJobs() []response.EmbeddingDownloadJobResponse {
	jobs := s.downloader.Jobs()
	result := make([]response.EmbeddingDownloadJobResponse, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, response.EmbeddingDownloadJobResponse{
			JobID:           job.JobID,
			PresetID:        job.PresetID,
			State:           job.State,
			DownloadedBytes: job.DownloadedBytes,
			TotalBytes:      job.TotalBytes,
			Error:           job.Error,
		})
	}
	return result
}

func (s *embeddingService) ValidateModel(ctx context.Context, modelID string) (response.EmbeddingValidationResponse, error) {
	if err := embedding.SanitizeModelID(modelID); err != nil {
		return response.EmbeddingValidationResponse{}, apperrors.New(apperrors.ErrBadRequest, err.Error())
	}
	unlock, ok := s.modelLocks.tryLock(modelID)
	if !ok {
		return response.EmbeddingValidationResponse{}, apperrors.New(apperrors.ErrConflict,
			"model is currently locked by another operation")
	}
	defer unlock()

	result := embedding.ValidateModel(ctx, s.rootDir, modelID)
	return response.EmbeddingValidationResponse{
		OK:        result.OK,
		Dim:       result.Dim,
		LatencyMs: result.LatencyMs,
		Error:     result.Error,
	}, nil
}

func (s *embeddingService) ActivateModel(ctx context.Context, modelID string) (response.EmbeddingSettingsResponse, error) {
	if err := embedding.SanitizeModelID(modelID); err != nil {
		return response.EmbeddingSettingsResponse{}, apperrors.New(apperrors.ErrBadRequest, err.Error())
	}
	unlock, ok := s.modelLocks.tryLock(modelID)
	if !ok {
		return response.EmbeddingSettingsResponse{}, apperrors.New(apperrors.ErrConflict,
			"model is currently locked by another operation")
	}
	defer unlock()

	result := embedding.ValidateModel(ctx, s.rootDir, modelID)
	if !result.OK {
		return response.EmbeddingSettingsResponse{}, apperrors.New(apperrors.ErrBadRequest,
			"model validation failed: "+result.Error)
	}
	if err := s.persistProvider(ctx, EmbeddingProviderONNX); err != nil {
		return response.EmbeddingSettingsResponse{}, err
	}
	if err := s.persistActiveModelID(ctx, modelID); err != nil {
		return response.EmbeddingSettingsResponse{}, err
	}
	_ = s.SyncActiveEmbedder(ctx)
	return s.GetSettings(ctx)
}

func (s *embeddingService) DeleteModel(ctx context.Context, modelID string) error {
	if err := embedding.SanitizeModelID(modelID); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, err.Error())
	}
	if strings.EqualFold(modelID, embedding.RuntimeDirName) {
		return apperrors.New(apperrors.ErrBadRequest, "cannot delete reserved runtime directory")
	}
	if active := s.getModelIDSetting(ctx); active != nil && *active == modelID {
		return apperrors.New(apperrors.ErrConflict, "cannot delete the active onnx embedding model")
	}
	targetDir := filepath.Join(s.rootDir, modelID)
	info, err := os.Lstat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return apperrors.New(apperrors.ErrNotFound, "model not found")
		}
		return apperrors.New(apperrors.ErrInternalError, "failed to inspect model folder")
	}
	if !info.IsDir() {
		return apperrors.New(apperrors.ErrBadRequest, "target is not a directory")
	}
	unlock, ok := s.modelLocks.tryLock(modelID)
	if !ok {
		return apperrors.New(apperrors.ErrConflict, "model is currently locked by another operation")
	}
	defer unlock()

	if err := os.RemoveAll(targetDir); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "failed to remove model folder")
	}
	return nil
}

func (s *embeddingService) getProviderSetting(ctx context.Context) string {
	raw, err := s.settings.GetAppSetting(ctx, SettingKeyEmbeddingProvider)
	if err != nil || raw == "" {
		return EmbeddingProviderONNX
	}
	var provider string
	if err := jsonx.UnmarshalString(raw, &provider); err != nil || provider == "" {
		return EmbeddingProviderONNX
	}
	return provider
}

func (s *embeddingService) getModelIDSetting(ctx context.Context) *string {
	raw, err := s.settings.GetAppSetting(ctx, SettingKeyEmbeddingOnnxModelID)
	if err != nil || raw == "" {
		defaultID := "paraphrase-multilingual-minilm-l12-v2"
		return &defaultID
	}
	var modelID string
	if err := jsonx.UnmarshalString(raw, &modelID); err != nil || modelID == "" {
		defaultID := "paraphrase-multilingual-minilm-l12-v2"
		return &defaultID
	}
	return &modelID
}

func (s *embeddingService) persistProvider(ctx context.Context, provider string) error {
	encoded, err := jsonx.MarshalString(provider)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "failed to encode embedding provider setting")
	}
	if err := s.settings.UpsertAppSetting(ctx, SettingKeyEmbeddingProvider, encoded); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "failed to persist embedding provider setting")
	}
	return nil
}

func (s *embeddingService) persistActiveModelID(ctx context.Context, modelID string) error {
	encoded, err := jsonx.MarshalString(modelID)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "failed to encode active onnx model setting")
	}
	if err := s.settings.UpsertAppSetting(ctx, SettingKeyEmbeddingOnnxModelID, encoded); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "failed to persist active onnx model setting")
	}
	return nil
}

func (s *embeddingService) buildModelResponse(modelID string, activeID *string) response.OnnxModelResponse {
	modelDir := filepath.Join(s.rootDir, modelID)
	resp := response.OnnxModelResponse{
		ModelID:     modelID,
		DisplayName: modelID,
		Status:      OnnxModelStatusInvalid,
	}
	manifest, err := embedding.LoadManifest(modelDir)
	if err != nil {
		resp.Error = strPtr(sanitizePathInString(err.Error(), s.rootDir))
		return resp
	}
	resp.DisplayName = manifest.DisplayName
	resp.Dim = manifest.Dim
	record := embedding.LoadValidationRecord(modelDir)
	switch {
	case record == nil:
		resp.Status = OnnxModelStatusUnvalidated
	case record.OK:
		resp.Status = OnnxModelStatusOK
		resp.Error = nil
		resp.ValidatedAt = formatTimePtr(record.ValidatedAt)
	default:
		resp.Status = OnnxModelStatusInvalid
		resp.Error = strPtr(sanitizePathInString(record.Error, s.rootDir))
		resp.ValidatedAt = formatTimePtr(record.ValidatedAt)
	}
	resp.SizeBytes = directorySize(modelDir)
	resp.IsActive = activeID != nil && *activeID == modelID

	actualSHA256, checksumErr := modelChecksum(modelDir)
	resp.SHA256 = actualSHA256
	if checksumErr != nil && resp.Status == OnnxModelStatusOK {
		resp.Status = OnnxModelStatusInvalid
		resp.Error = strPtr(checksumErr.Error())
	}
	return resp
}

func presetExists(presetID string) bool {
	catalog := embedding.BuiltinCatalog()
	for _, model := range catalog.Models {
		if model.PresetID == presetID {
			return true
		}
	}
	for _, runtimePreset := range catalog.Runtimes {
		if runtimePreset.PresetID == presetID {
			return true
		}
	}
	return false
}

func directorySize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if info, infoErr := entry.Info(); infoErr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func modelChecksum(modelDir string) (*string, error) {
	modelFile, err := embedding.FindModelFile(modelDir)
	if err != nil {
		return nil, nil
	}
	actualSHA256, err := embedding.ComputeFileSHA256(modelFile)
	if err != nil {
		return nil, fmt.Errorf("failed to compute model checksum: %w", err)
	}
	checksums := embedding.LoadChecksums(modelDir)
	if checksums != nil {
		if record := checksums.FindChecksum(filepath.Base(modelFile)); record != nil {
			if !strings.EqualFold(actualSHA256, record.SHA256) {
				return strPtr(actualSHA256), fmt.Errorf("model file checksum mismatch with recorded checksums")
			}
		}
	}
	return strPtr(actualSHA256), nil
}

func sanitizePathInString(msg string, rootDir string) string {
	if rootDir != "" {
		msg = strings.ReplaceAll(msg, rootDir+string(filepath.Separator), "")
		msg = strings.ReplaceAll(msg, rootDir, "")
	}
	return msg
}

func formatTimePtr(t time.Time) *string {
	formatted := t.UTC().Format(time.RFC3339)
	return &formatted
}

func strPtr(value string) *string {
	return &value
}

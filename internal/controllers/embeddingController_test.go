package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/pkg/embedding"
	"tluagent-web/pkg/llm"
)

type fakeEmbeddingService struct {
	settings         response.EmbeddingSettingsResponse
	models           []response.OnnxModelResponse
	catalog          embedding.Catalog
	startedJobID     string
	jobs             []response.EmbeddingDownloadJobResponse
	validation       response.EmbeddingValidationResponse
	updateErr        error
	activateErr      error
	deleteErr        error
	validateErr      error
	downloadedPreset string
	deletedModelID   string
}

func (f *fakeEmbeddingService) GetSettings(ctx context.Context) (response.EmbeddingSettingsResponse, error) {
	return f.settings, nil
}

func (f *fakeEmbeddingService) UpdateSettings(ctx context.Context, req request.UpdateEmbeddingSettingsRequest) (response.EmbeddingSettingsResponse, error) {
	if f.updateErr != nil {
		return response.EmbeddingSettingsResponse{}, f.updateErr
	}
	f.settings.Provider = req.Provider
	return f.settings, nil
}

func (f *fakeEmbeddingService) ListOnnxModels(ctx context.Context) ([]response.OnnxModelResponse, error) {
	return f.models, nil
}

func (f *fakeEmbeddingService) GetCatalog() embedding.Catalog {
	return f.catalog
}

func (f *fakeEmbeddingService) StartDownload(presetID string) (response.EmbeddingDownloadStartedResponse, error) {
	f.downloadedPreset = presetID
	f.jobs = append(f.jobs, response.EmbeddingDownloadJobResponse{
		JobID: f.startedJobID, PresetID: presetID, State: "running",
	})
	return response.EmbeddingDownloadStartedResponse{JobID: f.startedJobID}, nil
}

func (f *fakeEmbeddingService) ListDownloadJobs() []response.EmbeddingDownloadJobResponse {
	return f.jobs
}

func (f *fakeEmbeddingService) ValidateModel(ctx context.Context, modelID string) (response.EmbeddingValidationResponse, error) {
	if f.validateErr != nil {
		return response.EmbeddingValidationResponse{}, f.validateErr
	}
	return f.validation, nil
}

func (f *fakeEmbeddingService) ActivateModel(ctx context.Context, modelID string) (response.EmbeddingSettingsResponse, error) {
	if f.activateErr != nil {
		return response.EmbeddingSettingsResponse{}, f.activateErr
	}
	f.settings.Provider = "onnx"
	return f.settings, nil
}

func (f *fakeEmbeddingService) DeleteModel(ctx context.Context, modelID string) error {
	f.deletedModelID = modelID
	return f.deleteErr
}

func (f *fakeEmbeddingService) SetLLMManager(_ *llm.Manager) {}

func (f *fakeEmbeddingService) SyncActiveEmbedder(_ context.Context) error { return nil }

func newFakeEmbeddingServer(t *testing.T) (*fakeEmbeddingService, *httptest.Server) {
	t.Helper()
	svc := &fakeEmbeddingService{
		settings: response.EmbeddingSettingsResponse{
			Provider: "api",
			Runtime: response.EmbeddingRuntimeInfo{
				OS: "linux", Arch: "amd64", LibPresent: false, LibPath: "/data/onnx/runtime/linux-amd64/libonnxruntime.so",
			},
		},
		models: []response.OnnxModelResponse{
			{
				ModelID: "all-minilm-l6-v2", DisplayName: "all-MiniLM-L6-v2", Dim: 384,
				SizeBytes: 90636722, Status: "ok", IsActive: false,
			},
		},
		startedJobID: "job-123",
		validation:   response.EmbeddingValidationResponse{OK: true, Dim: 384, LatencyMs: 12.5},
		catalog:      embedding.BuiltinCatalog(),
	}
	ctrl := controllers.NewEmbeddingController(svc)
	mux := http.NewServeMux()
	handler := func(h http.HandlerFunc) http.Handler { return h }
	mux.Handle("GET /api/admin/embedding/settings", handler(ctrl.GetSettings))
	mux.Handle("PUT /api/admin/embedding/settings", handler(ctrl.UpdateSettings))
	mux.Handle("GET /api/admin/embedding/onnx/models", handler(ctrl.ListModels))
	mux.Handle("GET /api/admin/embedding/onnx/catalog", handler(ctrl.GetCatalog))
	mux.Handle("POST /api/admin/embedding/onnx/download", handler(ctrl.StartDownload))
	mux.Handle("GET /api/admin/embedding/onnx/download/status", handler(ctrl.DownloadStatus))
	mux.Handle("POST /api/admin/embedding/onnx/models/{modelId}/validate", handler(ctrl.ValidateModel))
	mux.Handle("POST /api/admin/embedding/onnx/models/{modelId}/activate", handler(ctrl.ActivateModel))
	mux.Handle("DELETE /api/admin/embedding/onnx/models/{modelId}", handler(ctrl.DeleteModel))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return svc, server
}

type envelope struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func TestEmbeddingController_GetSettings(t *testing.T) {
	_, server := newFakeEmbeddingServer(t)
	resp, err := http.Get(server.URL + "/api/admin/embedding/settings")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body envelope
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !body.Status {
		t.Fatalf("expected status true")
	}
	var settings response.EmbeddingSettingsResponse
	if err := json.Unmarshal(body.Data, &settings); err != nil {
		t.Fatalf("failed to decode settings: %v", err)
	}
	if settings.Provider != "api" || settings.Runtime.OS != "linux" || settings.Runtime.LibPath == "" {
		t.Fatalf("unexpected settings payload: %+v", settings)
	}
}

func TestEmbeddingController_UpdateSettings(t *testing.T) {
	_, server := newFakeEmbeddingServer(t)
	modelID := "all-minilm-l6-v2"
	payload, _ := json.Marshal(request.UpdateEmbeddingSettingsRequest{Provider: "onnx", ActiveOnnxModelID: &modelID})
	req, _ := http.NewRequest(http.MethodPut, server.URL+"/api/admin/embedding/settings", bytes.NewReader(payload))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body envelope
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Message == "" {
		t.Fatalf("expected a success message")
	}
}

func TestEmbeddingController_UpdateSettings_InvalidBody(t *testing.T) {
	_, server := newFakeEmbeddingServer(t)
	payload, _ := json.Marshal(map[string]string{"provider": "quantum"})
	req, _ := http.NewRequest(http.MethodPut, server.URL+"/api/admin/embedding/settings", bytes.NewReader(payload))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestEmbeddingController_ListModels(t *testing.T) {
	_, server := newFakeEmbeddingServer(t)
	resp, err := http.Get(server.URL + "/api/admin/embedding/onnx/models")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	var body envelope
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	var models []response.OnnxModelResponse
	if err := json.Unmarshal(body.Data, &models); err != nil {
		t.Fatalf("failed to decode models: %v", err)
	}
	if len(models) != 1 || models[0].ModelID != "all-minilm-l6-v2" || models[0].Dim != 384 {
		t.Fatalf("unexpected models payload: %+v", models)
	}
}

func TestEmbeddingController_CatalogAndDownload(t *testing.T) {
	svc, server := newFakeEmbeddingServer(t)
	resp, err := http.Get(server.URL + "/api/admin/embedding/onnx/catalog")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	var body envelope
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	var catalog embedding.Catalog
	if err := json.Unmarshal(body.Data, &catalog); err != nil {
		t.Fatalf("failed to decode catalog: %v", err)
	}
	if len(catalog.Models) == 0 || len(catalog.Runtimes) != 4 {
		t.Fatalf("unexpected catalog payload: %+v", catalog)
	}
	if catalog.Models[0].Manifest.ModelID != "" {
		t.Fatalf("manifest internals must not leak into the catalog payload")
	}

	downloadResp, err := http.Post(server.URL+"/api/admin/embedding/onnx/download", "application/json",
		bytes.NewReader([]byte(`{"preset_id":"all-minilm-l6-v2"}`)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer downloadResp.Body.Close()
	var downloadBody envelope
	if err := json.NewDecoder(downloadResp.Body).Decode(&downloadBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !downloadBody.Status || svc.downloadedPreset != "all-minilm-l6-v2" {
		t.Fatalf("download start failed: %+v", downloadBody)
	}
	var started response.EmbeddingDownloadStartedResponse
	if err := json.Unmarshal(downloadBody.Data, &started); err != nil || started.JobID != "job-123" {
		t.Fatalf("unexpected job payload: %+v", started)
	}

	statusResp, err := http.Get(server.URL + "/api/admin/embedding/onnx/download/status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer statusResp.Body.Close()
	var statusBody envelope
	if err := json.NewDecoder(statusResp.Body).Decode(&statusBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	var status response.EmbeddingDownloadStatusResponse
	if err := json.Unmarshal(statusBody.Data, &status); err != nil {
		t.Fatalf("failed to decode status: %v", err)
	}
	if len(status.Jobs) != 1 || status.Jobs[0].JobID != "job-123" {
		t.Fatalf("unexpected jobs payload: %+v", status)
	}
}

func TestEmbeddingController_ValidateActivateDelete(t *testing.T) {
	svc, server := newFakeEmbeddingServer(t)

	validateResp, err := http.Post(server.URL+"/api/admin/embedding/onnx/models/all-minilm-l6-v2/validate", "application/json", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer validateResp.Body.Close()
	var validateBody envelope
	if err := json.NewDecoder(validateResp.Body).Decode(&validateBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	var validation response.EmbeddingValidationResponse
	if err := json.Unmarshal(validateBody.Data, &validation); err != nil {
		t.Fatalf("failed to decode validation: %v", err)
	}
	if !validation.OK || validation.Dim != 384 || validation.LatencyMs != 12.5 {
		t.Fatalf("unexpected validation payload: %+v", validation)
	}

	activateResp, err := http.Post(server.URL+"/api/admin/embedding/onnx/models/all-minilm-l6-v2/activate", "application/json", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer activateResp.Body.Close()
	if activateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on activate, got %d", activateResp.StatusCode)
	}

	delReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/admin/embedding/onnx/models/all-minilm-l6-v2", nil)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d", delResp.StatusCode)
	}
	if svc.deletedModelID != "all-minilm-l6-v2" {
		t.Fatalf("delete handler must forward the model id: %q", svc.deletedModelID)
	}
}

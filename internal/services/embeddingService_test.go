package services_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
	"tluagent-web/pkg/embedding"
	"tluagent-web/pkg/jsonx"
)

func newTestEmbeddingService(t *testing.T) (services.EmbeddingService, string) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}
	settingsRepo := repositories.NewSettingsRepository(db, cache.NewRamCache())
	rootDir := t.TempDir()
	return services.NewEmbeddingService(settingsRepo, rootDir), rootDir
}

func writeTestModel(t *testing.T, rootDir string, modelID string, manifest embedding.Manifest, validateOK bool) {
	t.Helper()
	modelDir := filepath.Join(rootDir, modelID)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatalf("failed to create model dir: %v", err)
	}
	manifest.ModelID = modelID
	data, err := jsonx.Marshal(&manifest)
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, embedding.ManifestFileName), data, 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, embedding.VocabFileName), []byte("[UNK]\n[CLS]\n[SEP]\n"), 0o644); err != nil {
		t.Fatalf("failed to write vocab: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, embedding.ModelFileName), []byte("onnx"), 0o644); err != nil {
		t.Fatalf("failed to write model: %v", err)
	}
	if validateOK {
		if err := embedding.SaveValidationRecord(modelDir, true, ""); err != nil {
			t.Fatalf("failed to save validation record: %v", err)
		}
	}
}

func validServiceManifest() embedding.Manifest {
	return embedding.Manifest{
		SchemaVersion: 1,
		DisplayName:   "Test Model",
		Dim:           384,
		MaxSeqTokens:  128,
		Pooling:       "mean",
		Normalize:     true,
		Inputs:        []string{"input_ids", "attention_mask"},
		Outputs:       []string{"last_hidden_state"},
		Tokenizer:     "wordpiece",
		Lowercase:     true,
		StripAccents:  false,
	}
}

func TestEmbeddingService_DefaultSettings(t *testing.T) {
	svc, _ := newTestEmbeddingService(t)
	settings, err := svc.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.Provider != "onnx" {
		t.Fatalf("expected default provider onnx, got %s", settings.Provider)
	}
	if settings.ActiveOnnxModelID == nil || *settings.ActiveOnnxModelID != "paraphrase-multilingual-minilm-l12-v2" {
		t.Fatalf("expected default model id paraphrase-multilingual-minilm-l12-v2, got %v", settings.ActiveOnnxModelID)
	}
	if settings.Runtime.OS == "" || settings.Runtime.Arch == "" {
		t.Fatalf("runtime info must be populated: %+v", settings.Runtime)
	}
	if settings.Runtime.LibPresent {
		t.Fatalf("runtime lib must be absent in a fresh test root")
	}
	if settings.Runtime.LibPath == "" {
		t.Fatalf("lib path must always be populated")
	}
}

func TestEmbeddingService_ListModels_Statuses(t *testing.T) {
	svc, root := newTestEmbeddingService(t)
	ctx := context.Background()

	if models, err := svc.ListOnnxModels(ctx); err != nil || len(models) != 0 {
		t.Fatalf("expected empty model list, got %v (err %v)", models, err)
	}

	validID := "valid-model"
	writeTestModel(t, root, validID, validServiceManifest(), false)
	brokenID := "broken-model"
	modelDir := filepath.Join(root, brokenID)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, "notes.txt"), []byte("no manifest here"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	models, err := svc.ListOnnxModels(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	byID := map[string]ModelShape{}
	for _, m := range models {
		byID[m.ModelID] = ModelShape{Status: m.Status, Error: m.Error, DisplayName: m.DisplayName}
	}
	if byID[validID].Status != "unvalidated" || byID[validID].Error != nil {
		t.Fatalf("expected unvalidated model, got %+v", byID[validID])
	}
	if byID[brokenID].Status != "invalid" || byID[brokenID].Error == nil {
		t.Fatalf("expected invalid model with error, got %+v", byID[brokenID])
	}
	if byID[brokenID].DisplayName != brokenID {
		t.Fatalf("invalid folders must fall back to the folder name: %+v", byID[brokenID])
	}
}

func TestEmbeddingService_UpdateSettings_RequiresValidatedModel(t *testing.T) {
	svc, root := newTestEmbeddingService(t)
	ctx := context.Background()

	unvalidated := "unvalidated-model"
	writeTestModel(t, root, unvalidated, validServiceManifest(), false)
	_, err := svc.UpdateSettings(ctx, request.UpdateEmbeddingSettingsRequest{
		Provider:          "onnx",
		ActiveOnnxModelID: &unvalidated,
	})
	if err == nil {
		t.Fatalf("expected error when switching to onnx with an unvalidated model")
	}

	validated := "validated-model"
	writeTestModel(t, root, validated, validServiceManifest(), true)
	settings, err := svc.UpdateSettings(ctx, request.UpdateEmbeddingSettingsRequest{
		Provider:          "onnx",
		ActiveOnnxModelID: &validated,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.Provider != "onnx" || settings.ActiveOnnxModelID == nil || *settings.ActiveOnnxModelID != validated {
		t.Fatalf("unexpected settings: %+v", settings)
	}

	reloaded, err := svc.GetSettings(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reloaded.Provider != "onnx" {
		t.Fatalf("provider setting must persist, got %s", reloaded.Provider)
	}
}

func TestEmbeddingService_ActivateModel_ValidationGate(t *testing.T) {
	svc, root := newTestEmbeddingService(t)
	ctx := context.Background()

	writeTestModel(t, root, "model-a", validServiceManifest(), false)
	if _, err := svc.ActivateModel(ctx, "model-a"); err == nil {
		t.Fatalf("activation must fail when the runtime library is missing")
	}
	if _, err := svc.ActivateModel(ctx, "../escape"); err == nil {
		t.Fatalf("activation must reject unsafe model ids")
	}
	settings, err := svc.GetSettings(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.ActiveOnnxModelID != nil && *settings.ActiveOnnxModelID == "model-a" {
		t.Fatalf("failed activation must not persist model-a as the active model")
	}
}

func TestEmbeddingService_DeleteModel_GuardsActiveModel(t *testing.T) {
	svc, root := newTestEmbeddingService(t)
	ctx := context.Background()

	target := "doomed-model"
	writeTestModel(t, root, target, validServiceManifest(), true)
	if _, err := svc.UpdateSettings(ctx, request.UpdateEmbeddingSettingsRequest{
		Provider:          "onnx",
		ActiveOnnxModelID: &target,
	}); err != nil {
		t.Fatalf("failed to activate model: %v", err)
	}
	if err := svc.DeleteModel(ctx, target); err == nil {
		t.Fatalf("deleting the active model must be refused")
	}
	if _, err := os.Stat(filepath.Join(root, target)); err != nil {
		t.Fatalf("active model folder must survive deletion attempt: %v", err)
	}

	other := "other-model"
	writeTestModel(t, root, other, validServiceManifest(), true)
	if err := svc.DeleteModel(ctx, other); err != nil {
		t.Fatalf("unexpected error deleting inactive model: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, other)); !os.IsNotExist(err) {
		t.Fatalf("inactive model folder must be removed")
	}

	if err := svc.DeleteModel(ctx, "../escape"); err == nil {
		t.Fatalf("deletion must reject unsafe model ids")
	}
}

func TestEmbeddingService_StartDownload_UnknownPreset(t *testing.T) {
	svc, _ := newTestEmbeddingService(t)
	if _, err := svc.StartDownload("no-such-preset"); err == nil {
		t.Fatalf("expected error for unknown preset")
	}
}

func TestEmbeddingService_Catalog(t *testing.T) {
	svc, _ := newTestEmbeddingService(t)
	catalog := svc.GetCatalog()
	if len(catalog.Models) == 0 || len(catalog.Runtimes) != 4 {
		t.Fatalf("unexpected catalog: %d models, %d runtimes", len(catalog.Models), len(catalog.Runtimes))
	}
}

type ModelShape struct {
	Status      string
	Error       *string
	DisplayName string
}

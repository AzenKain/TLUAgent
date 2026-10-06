package embedding

import (
	"os"
	"path/filepath"
	"testing"
)

func validTestManifest() Manifest {
	return Manifest{
		SchemaVersion: 1,
		ModelID:       "test-model",
		DisplayName:   "Test Model",
		Dim:           384,
		MaxSeqTokens:  256,
		Pooling:       PoolingMean,
		Normalize:     true,
		QueryPrefix:   "",
		Inputs:        []string{InputIDsName, AttentionMaskName, TokenTypeIDsName},
		Outputs:       []string{"last_hidden_state"},
		Tokenizer:     TokenizerWordPiece,
		Lowercase:     true,
		StripAccents:  false,
	}
}

func TestManifest_LoadValid(t *testing.T) {
	dir := t.TempDir()
	writeManifestFile(t, dir, validTestManifest())
	manifest, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("expected valid manifest, got error: %v", err)
	}
	if manifest.ModelID != "test-model" || manifest.Dim != 384 {
		t.Fatalf("unexpected manifest content: %+v", manifest)
	}
}

func TestManifest_ParseErrors(t *testing.T) {
	cases := map[string]func(*Manifest){
		"bad schema version":  func(m *Manifest) { m.SchemaVersion = 2 },
		"empty model id":      func(m *Manifest) { m.ModelID = "" },
		"path in model id":    func(m *Manifest) { m.ModelID = "a/b" },
		"empty display name":  func(m *Manifest) { m.DisplayName = "" },
		"zero dim":            func(m *Manifest) { m.Dim = 0 },
		"small max seq":       func(m *Manifest) { m.MaxSeqTokens = 2 },
		"bad pooling":         func(m *Manifest) { m.Pooling = "sum" },
		"normalize false":     func(m *Manifest) { m.Normalize = false },
		"bad tokenizer":       func(m *Manifest) { m.Tokenizer = "sentencepiece" },
		"empty inputs":        func(m *Manifest) { m.Inputs = nil },
		"missing input_ids":   func(m *Manifest) { m.Inputs = []string{AttentionMaskName} },
		"empty outputs":       func(m *Manifest) { m.Outputs = nil },
		"extra unknown input": func(m *Manifest) { m.Inputs = []string{InputIDsName, AttentionMaskName, "logits"} },
	}
	for name, mutate := range cases {
		manifest := validTestManifest()
		mutate(&manifest)
		if err := manifest.Validate(); err == nil {
			t.Fatalf("%s: expected validation error", name)
		}
	}
}

func TestManifest_MissingFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadManifest(dir); err == nil {
		t.Fatalf("expected error when manifest.json is missing")
	}
}

func TestManifest_CorruptJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if _, err := LoadManifest(dir); err == nil {
		t.Fatalf("expected error for corrupt manifest json")
	}
}

func writeManifestFile(t *testing.T, dir string, manifest Manifest) {
	t.Helper()
	data, err := marshalJSON(&manifest)
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), data, 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
}

func TestFindModelFile_PrefersModelOnnx(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a_model.onnx", ModelFileName} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("failed to write model file: %v", err)
		}
	}
	found, err := FindModelFile(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(found) != ModelFileName {
		t.Fatalf("expected model.onnx to be preferred, got %s", found)
	}
}

func TestFindModelFile_FallsBackToAnyOnnx(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "custom_graph.onnx"), []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write model file: %v", err)
	}
	found, err := FindModelFile(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(found) != "custom_graph.onnx" {
		t.Fatalf("unexpected model file: %s", found)
	}
}

func TestFindModelFile_NoneFound(t *testing.T) {
	dir := t.TempDir()
	if _, err := FindModelFile(dir); err == nil {
		t.Fatalf("expected error when no onnx file exists")
	}
}

func TestValidationRecord_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	if LoadValidationRecord(dir) != nil {
		t.Fatalf("expected nil record before saving")
	}
	if err := SaveValidationRecord(dir, false, "dim mismatch"); err != nil {
		t.Fatalf("failed to save record: %v", err)
	}
	record := LoadValidationRecord(dir)
	if record == nil || record.OK || record.Error != "dim mismatch" {
		t.Fatalf("unexpected record: %+v", record)
	}
	if record.ValidatedAt.IsZero() {
		t.Fatalf("validated_at must be set")
	}
}

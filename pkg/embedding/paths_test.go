package embedding

import (
	"strings"
	"testing"
)

func TestLibraryFileName_PerGOOS(t *testing.T) {
	cases := map[string]string{
		"windows": "onnxruntime.dll",
		"linux":   "libonnxruntime.so",
		"darwin":  "libonnxruntime.dylib",
	}
	for goos, expected := range cases {
		name, err := LibraryFileName(goos)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", goos, err)
		}
		if name != expected {
			t.Fatalf("%s: expected %s, got %s", goos, expected, name)
		}
	}
	if _, err := LibraryFileName("freebsd"); err == nil {
		t.Fatalf("expected error for unsupported GOOS")
	}
}

func TestRuntimeLibraryPath_PerPlatform(t *testing.T) {
	path, err := RuntimeLibraryPath("/data/onnx", "windows", "amd64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/data/onnx/runtime/windows-amd64/onnxruntime.dll" {
		t.Fatalf("unexpected path: %s", path)
	}
	path, err = RuntimeLibraryPath("/data/onnx", "linux", "arm64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/data/onnx/runtime/linux-arm64/libonnxruntime.so" {
		t.Fatalf("unexpected path: %s", path)
	}
	path, err = RuntimeLibraryPath("/data/onnx", "darwin", "arm64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/data/onnx/runtime/darwin-arm64/libonnxruntime.dylib" {
		t.Fatalf("unexpected path: %s", path)
	}
}

func TestSanitizeModelID(t *testing.T) {
	valid := []string{"all-minilm-l6-v2", "model_1", "My.Model"}
	for _, id := range valid {
		if err := SanitizeModelID(id); err != nil {
			t.Fatalf("expected %q to be valid, got %v", id, err)
		}
	}
	invalid := []string{"", ".", "..", "a/b", "a\\b", "../escape", "model\x00id", "model.", "model ", " runtime ", "runtime", "RUNTIME", "NUL", "COM1", strings.Repeat("a", 129)}
	for _, id := range invalid {
		if err := SanitizeModelID(id); err == nil {
			t.Fatalf("expected %q to be rejected", id)
		}
	}
}

func TestBuiltinCatalog_CoversExactRuntimeMatrix(t *testing.T) {
	catalog := BuiltinCatalog()
	expected := map[string]bool{
		"windows-amd64": false,
		"linux-amd64":   false,
		"linux-arm64":   false,
		"darwin-arm64":  false,
	}
	for _, runtimePreset := range catalog.Runtimes {
		key := runtimePreset.GOOS + "-" + runtimePreset.GOArch
		if _, ok := expected[key]; !ok {
			t.Fatalf("unexpected runtime platform in catalog: %s", key)
		}
		expected[key] = true
	}
	for key, present := range expected {
		if !present {
			t.Fatalf("missing runtime platform in catalog: %s", key)
		}
	}
	if len(catalog.Models) == 0 {
		t.Fatalf("catalog must contain model presets")
	}
	for _, model := range catalog.Models {
		if err := model.Manifest.Validate(); err != nil {
			t.Fatalf("catalog manifest for %s is invalid: %v", model.PresetID, err)
		}
		if model.Manifest.ModelID != model.PresetID {
			t.Fatalf("catalog manifest model_id must match preset_id for %s", model.PresetID)
		}
		var total int64
		for _, file := range model.FileURLs {
			total += file.SizeBytes
		}
		if total != model.ApproxSizeBytes {
			t.Fatalf("approx size mismatch for %s", model.PresetID)
		}
	}
}

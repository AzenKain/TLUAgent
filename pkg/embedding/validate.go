package embedding

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
)

const validationProbeText = "onnx embedding validation probe 42"

// ValidationResult carries the outcome of an end-to-end model validation run.
type ValidationResult struct {
	OK        bool    `json:"ok"`
	Dim       int     `json:"dim"`
	LatencyMs float64 `json:"latency_ms"`
	Error     string  `json:"error,omitempty"`
}

// ValidateModel runs the full manifest, file, runtime, and inference check for a model folder and persists the outcome.
func ValidateModel(ctx context.Context, rootDir, modelID string) *ValidationResult {
	result := &ValidationResult{}
	modelDir, fail := resolveModelDir(rootDir, modelID)
	if fail != nil {
		return finishValidation(rootDir, modelDir, result, fail)
	}

	manifest, err := LoadManifest(modelDir)
	if err != nil {
		return finishValidation(rootDir, modelDir, result, err)
	}
	modelPath, err := FindModelFile(modelDir)
	if err != nil {
		return finishValidation(rootDir, modelDir, result, err)
	}
	tokenizerFile := VocabFileName
	if manifest.Tokenizer == TokenizerUnigram {
		tokenizerFile = TokenizerFileName
	}
	tokenizerPath := filepath.Join(modelDir, tokenizerFile)
	if err := checkNotSymlink(tokenizerPath); err != nil {
		return finishValidation(rootDir, modelDir, result, err)
	}
	if _, err := os.Stat(tokenizerPath); err != nil {
		return finishValidation(rootDir, modelDir, result, fmt.Errorf("tokenizer file %s is missing in model folder", tokenizerFile))
	}
	libPath, err := CurrentRuntimeLibraryPath(rootDir)
	if err != nil {
		return finishValidation(rootDir, modelDir, result, err)
	}
	if _, err := os.Stat(libPath); err != nil {
		return finishValidation(rootDir, modelDir, result, fmt.Errorf(
			"onnxruntime library for %s/%s is missing, download it from the catalog",
			runtime.GOOS, runtime.GOARCH))
	}
	if checksums := LoadChecksums(modelDir); checksums != nil {
		if rec := checksums.FindChecksum(filepath.Base(modelPath)); rec != nil {
			actual, err := ComputeFileSHA256(modelPath)
			if err != nil {
				return finishValidation(rootDir, modelDir, result, err)
			}
			if !strings.EqualFold(actual, rec.SHA256) {
				return finishValidation(rootDir, modelDir, result, fmt.Errorf("model file checksum mismatch: expected %s, got %s", rec.SHA256, actual))
			}
		}
	}
	if err := InitRuntimeEnvironment(libPath); err != nil {
		return finishValidation(rootDir, modelDir, result, fmt.Errorf("initialize onnxruntime: %w", err))
	}
	if err := checkManifestAgainstModel(modelPath, manifest); err != nil {
		return finishValidation(rootDir, modelDir, result, err)
	}

	embedder, err := NewOnnxEmbedder(modelDir, manifest, modelPath, libPath)
	if err != nil {
		return finishValidation(rootDir, modelDir, result, err)
	}
	defer embedder.Close()

	vectors, err := embedder.Embed(ctx, []string{validationProbeText})
	if err != nil {
		return finishValidation(rootDir, modelDir, result, err)
	}
	result.Dim = len(vectors[0])
	if result.Dim != manifest.Dim {
		return finishValidation(rootDir, modelDir, result, fmt.Errorf(
			"embedding dim %d does not match manifest dim %d", result.Dim, manifest.Dim))
	}
	selfCosine := CosineSimilarity(vectors[0], vectors[0])
	if selfCosine < 0.999 {
		return finishValidation(rootDir, modelDir, result, fmt.Errorf(
			"self cosine similarity %.6f is below 0.999", selfCosine))
	}

	start := time.Now()
	if _, err := embedder.Embed(ctx, []string{validationProbeText}); err != nil {
		return finishValidation(rootDir, modelDir, result, err)
	}
	result.LatencyMs = float64(time.Since(start).Microseconds()) / 1000.0
	result.OK = true
	if err := SaveValidationRecord(modelDir, true, ""); err != nil {
		result.OK = false
		result.Error = fmt.Sprintf("persist validation record: %v", err)
	}
	return result
}

func resolveModelDir(rootDir, modelID string) (string, error) {
	if err := SanitizeModelID(modelID); err != nil {
		return "", err
	}
	modelDir := filepath.Join(rootDir, modelID)
	if err := checkNotSymlink(modelDir); err != nil {
		return "", err
	}
	return modelDir, nil
}

func finishValidation(rootDir, modelDir string, result *ValidationResult, err error) *ValidationResult {
	if err != nil {
		result.OK = false
		result.Error = sanitizePathInError(err, rootDir)
	}
	if modelDir != "" {
		_ = SaveValidationRecord(modelDir, result.OK, result.Error)
	}
	return result
}

func sanitizePathInError(err error, rootDir string) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if rootDir != "" {
		msg = strings.ReplaceAll(msg, rootDir+string(filepath.Separator), "")
		msg = strings.ReplaceAll(msg, rootDir, "")
	}
	return msg
}

func checkManifestAgainstModel(modelPath string, manifest *Manifest) error {
	modelInputs, modelOutputs, err := ortGetIOInfo(modelPath)
	if err != nil {
		return fmt.Errorf("read model input/output metadata: %w", err)
	}
	for _, name := range manifest.Inputs {
		if !slices.ContainsFunc(modelInputs, func(info ortIOInfo) bool { return info.Name == name }) {
			return fmt.Errorf("manifest input %q does not exist in the model", name)
		}
	}
	for _, info := range modelInputs {
		if !slices.Contains(manifest.Inputs, info.Name) {
			return fmt.Errorf("model input %q is missing from the manifest inputs", info.Name)
		}
	}
	for _, name := range manifest.Outputs {
		if !slices.ContainsFunc(modelOutputs, func(info ortIOInfo) bool { return info.Name == name }) {
			return fmt.Errorf("manifest output %q does not exist in the model", name)
		}
	}
	for _, info := range modelOutputs {
		if !slices.Contains(manifest.Outputs, info.Name) {
			return fmt.Errorf("model output %q is missing from the manifest outputs", info.Name)
		}
	}
	return nil
}

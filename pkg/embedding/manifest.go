package embedding

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"tluagent-web/pkg/jsonx"
)

const (
	ManifestFileName   = "manifest.json"
	VocabFileName      = "vocab.txt"
	ModelFileName      = "model.onnx"
	ChecksumsFileName  = "checksums.json"
	ValidationFileName = "validation.json"

	PoolingMean        = "mean"
	PoolingCLS         = "cls"
	TokenizerWordPiece = "wordpiece"

	InputIDsName      = "input_ids"
	AttentionMaskName = "attention_mask"
	TokenTypeIDsName  = "token_type_ids"

	ManifestSchemaVersion = 1
)

// Manifest describes a local ONNX embedding model folder and its inference contract.
type Manifest struct {
	SchemaVersion int      `json:"schema_version"`
	ModelID       string   `json:"model_id"`
	DisplayName   string   `json:"display_name"`
	Dim           int      `json:"dim"`
	MaxSeqTokens  int      `json:"max_seq_tokens"`
	Pooling       string   `json:"pooling"`
	Normalize     bool     `json:"normalize"`
	QueryPrefix   string   `json:"query_prefix"`
	Inputs        []string `json:"inputs"`
	Outputs       []string `json:"outputs"`
	Tokenizer     string   `json:"tokenizer"`
	Lowercase     bool     `json:"lowercase"`
	StripAccents  bool     `json:"strip_accents"`
}

// Validate checks manifest invariants required for validation and inference.
func (m *Manifest) Validate() error {
	if m.SchemaVersion != ManifestSchemaVersion {
		return fmt.Errorf("unsupported manifest schema_version %d", m.SchemaVersion)
	}
	if strings.TrimSpace(m.ModelID) == "" {
		return fmt.Errorf("manifest model_id is required")
	}
	if strings.ContainsAny(m.ModelID, "/\\") {
		return fmt.Errorf("manifest model_id must not contain path separators")
	}
	if strings.TrimSpace(m.DisplayName) == "" {
		return fmt.Errorf("manifest display_name is required")
	}
	if m.Dim <= 0 {
		return fmt.Errorf("manifest dim must be positive")
	}
	if m.MaxSeqTokens <= 2 {
		return fmt.Errorf("manifest max_seq_tokens must be greater than 2")
	}
	if m.Pooling != PoolingMean && m.Pooling != PoolingCLS {
		return fmt.Errorf("manifest pooling must be %q or %q", PoolingMean, PoolingCLS)
	}
	if !m.Normalize {
		return fmt.Errorf("manifest normalize must be true")
	}
	if m.Tokenizer != TokenizerWordPiece && m.Tokenizer != TokenizerUnigram {
		return fmt.Errorf("manifest tokenizer %q is not supported, only %q or %q", m.Tokenizer, TokenizerWordPiece, TokenizerUnigram)
	}
	if len(m.Inputs) == 0 {
		return fmt.Errorf("manifest inputs must not be empty")
	}
	if !slices.Contains(m.Inputs, InputIDsName) || !slices.Contains(m.Inputs, AttentionMaskName) {
		return fmt.Errorf("manifest inputs must include %q and %q", InputIDsName, AttentionMaskName)
	}
	for _, name := range m.Inputs {
		if name != InputIDsName && name != AttentionMaskName && name != TokenTypeIDsName {
			return fmt.Errorf("manifest input %q is not supported", name)
		}
	}
	if len(m.Outputs) == 0 {
		return fmt.Errorf("manifest outputs must not be empty")
	}
	return nil
}

// LoadManifest reads and validates manifest.json from a model directory.
func LoadManifest(modelDir string) (*Manifest, error) {
	if err := checkNotSymlink(modelDir); err != nil {
		return nil, fmt.Errorf("invalid model directory: %w", err)
	}
	data, err := readBoundedFile(filepath.Join(modelDir, ManifestFileName), MaxManifestSizeBytes)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	m := &Manifest{}
	if err := jsonx.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return m, nil
}

// FindModelFile locates the ONNX graph file inside a model folder, preferring model.onnx.
func FindModelFile(modelDir string) (string, error) {
	if err := checkNotSymlink(modelDir); err != nil {
		return "", fmt.Errorf("invalid model directory: %w", err)
	}
	preferred := filepath.Join(modelDir, ModelFileName)
	if info, err := os.Lstat(preferred); err == nil && !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		return preferred, nil
	}
	entries, err := os.ReadDir(modelDir)
	if err != nil {
		return "", fmt.Errorf("read model directory: %w", err)
	}
	var candidates []string
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".onnx") {
			candidates = append(candidates, entry.Name())
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no .onnx model file found in model folder")
	}
	slices.Sort(candidates)
	return filepath.Join(modelDir, candidates[0]), nil
}

// ValidationRecord stores the persisted outcome of the last model validation run.
type ValidationRecord struct {
	OK          bool      `json:"ok"`
	Error       string    `json:"error,omitempty"`
	ValidatedAt time.Time `json:"validated_at"`
	ModelSHA256 string    `json:"model_sha256,omitempty"`
}

// SaveValidationRecord atomically persists a validation outcome into the model folder.
func SaveValidationRecord(modelDir string, ok bool, validationError string) error {
	var modelSHA256 string
	if ok {
		if modelPath, err := FindModelFile(modelDir); err == nil {
			modelSHA256, _ = ComputeFileSHA256(modelPath)
		}
	}
	record := ValidationRecord{
		OK:          ok,
		Error:       validationError,
		ValidatedAt: time.Now().UTC(),
		ModelSHA256: modelSHA256,
	}
	data, err := jsonx.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal validation record: %w", err)
	}
	return atomicWriteFile(filepath.Join(modelDir, ValidationFileName), data, 0o644)
}

// LoadValidationRecord returns the persisted validation outcome, or nil when absent or unreadable.
func LoadValidationRecord(modelDir string) *ValidationRecord {
	data, err := readBoundedFile(filepath.Join(modelDir, ValidationFileName), MaxValidationSizeBytes)
	if err != nil {
		return nil
	}
	record := &ValidationRecord{}
	if err := jsonx.Unmarshal(data, record); err != nil {
		return nil
	}
	return record
}

// VerifyValidationIntegrity ensures that the validation record is genuine, unexpired, and matches disk hashes.
func VerifyValidationIntegrity(modelDir string) (*ValidationRecord, error) {
	if err := checkNotSymlink(modelDir); err != nil {
		return nil, fmt.Errorf("invalid model directory: %w", err)
	}
	record := LoadValidationRecord(modelDir)
	if record == nil || !record.OK {
		return nil, fmt.Errorf("the model has not been successfully validated")
	}
	if record.ModelSHA256 == "" {
		return nil, fmt.Errorf("validation record is missing cryptographic model checksum")
	}
	modelPath, err := FindModelFile(modelDir)
	if err != nil {
		return nil, err
	}
	modelInfo, err := os.Lstat(modelPath)
	if err != nil {
		return nil, err
	}
	if modelInfo.ModTime().After(record.ValidatedAt) {
		return nil, fmt.Errorf("model file was modified after validation")
	}
	manifestPath := filepath.Join(modelDir, ManifestFileName)
	manifestInfo, err := os.Lstat(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("manifest file is missing: %w", err)
	}
	if manifestInfo.ModTime().After(record.ValidatedAt) {
		return nil, fmt.Errorf("manifest file was modified after validation")
	}
	manifest, err := LoadManifest(modelDir)
	if err != nil {
		return nil, err
	}
	tokenizerFile := VocabFileName
	if manifest.Tokenizer == TokenizerUnigram {
		tokenizerFile = TokenizerFileName
	}
	tokPath := filepath.Join(modelDir, tokenizerFile)
	tokInfo, err := os.Lstat(tokPath)
	if err != nil {
		return nil, fmt.Errorf("tokenizer file is missing: %w", err)
	}
	if tokInfo.ModTime().After(record.ValidatedAt) {
		return nil, fmt.Errorf("tokenizer file was modified after validation")
	}
	actualSHA256, err := ComputeFileSHA256(modelPath)
	if err != nil {
		return nil, fmt.Errorf("compute model checksum: %w", err)
	}
	if !strings.EqualFold(actualSHA256, record.ModelSHA256) {
		return nil, fmt.Errorf("model file checksum mismatch with validation record")
	}
	checksums := LoadChecksums(modelDir)
	if checksums != nil {
		if rec := checksums.FindChecksum(filepath.Base(modelPath)); rec != nil {
			if !strings.EqualFold(rec.SHA256, actualSHA256) {
				return nil, fmt.Errorf("model checksum does not match recorded catalog checksum")
			}
		}
	}
	return record, nil
}

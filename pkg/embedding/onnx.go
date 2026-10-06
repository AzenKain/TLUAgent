package embedding

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	runtimeMu  sync.Mutex
	runtimeErr error
	runtimeNew sync.Once
)

// InitRuntimeEnvironment loads the onnxruntime shared library exactly once per process after verifying integrity.
func InitRuntimeEnvironment(libPath string) error {
	if err := requireCgoRuntime(); err != nil {
		return err
	}
	if err := checkNotSymlink(libPath); err != nil {
		return fmt.Errorf("invalid runtime library: %w", err)
	}
	if checksums := LoadChecksums(filepath.Dir(libPath)); checksums != nil {
		if rec := checksums.FindChecksum(filepath.Base(libPath)); rec != nil {
			actual, err := ComputeFileSHA256(libPath)
			if err != nil {
				return fmt.Errorf("verify runtime library: %w", err)
			}
			if !strings.EqualFold(actual, rec.SHA256) {
				return fmt.Errorf("runtime library integrity check failed: expected %s, got %s", rec.SHA256, actual)
			}
		}
	}
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	if runtimeErr != nil {
		return runtimeErr
	}
	runtimeNew.Do(func() {
		ortSetLibraryPath(libPath)
		runtimeErr = ortInitializeEnvironment()
	})
	return runtimeErr
}

// TextTokenizer defines the interface for tokenizing input text to tensor input IDs.
type TextTokenizer interface {
	TokenIDs(text string, maxSeqTokens int) []int64
}

// OnnxEmbedder is a local CPU embedder implementing the llm.Embedder contract.
type OnnxEmbedder struct {
	modelID   string
	manifest  *Manifest
	modelPath string
	libPath   string
	tokenizer TextTokenizer

	mu      sync.Mutex
	session ortSession
	inited  bool
	initErr error
}

// NewOnnxEmbedder loads the vocabulary eagerly and defers ORT session creation to first use.
func NewOnnxEmbedder(modelDir string, manifest *Manifest, modelPath, libPath string) (*OnnxEmbedder, error) {
	var tokenizer TextTokenizer
	var err error

	if manifest.Tokenizer == TokenizerUnigram {
		tokPath := filepath.Join(modelDir, TokenizerFileName)
		if _, statErr := os.Stat(tokPath); statErr != nil {
			tokPath = filepath.Join(modelDir, VocabFileName)
		}
		tokenizer, err = LoadUnigramTokenizer(tokPath)
		if err != nil {
			return nil, err
		}
	} else {
		tokenizer, err = LoadWordPieceTokenizer(filepath.Join(modelDir, VocabFileName), manifest.Lowercase, manifest.StripAccents)
		if err != nil {
			return nil, err
		}
	}

	return &OnnxEmbedder{
		modelID:   manifest.ModelID,
		manifest:  manifest,
		modelPath: modelPath,
		libPath:   libPath,
		tokenizer: tokenizer,
	}, nil
}

// Model returns the model identifier of this embedder.
func (e *OnnxEmbedder) Model() string {
	return e.modelID
}

// Embed generates one normalized vector per input text.
func (e *OnnxEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, errors.New("input texts list is empty")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.ensureSession(); err != nil {
		return nil, err
	}
	vectors := make([][]float32, len(texts))
	for i, text := range texts {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("embedding cancelled: %w", err)
		}
		vec, err := e.runLocked(text)
		if err != nil {
			return nil, err
		}
		vectors[i] = vec
	}
	return vectors, nil
}

// EmbedSingle generates a single embedding vector for one text string.
func (e *OnnxEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	vectors, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vectors[0], nil
}

// Close destroys the ORT session; the global environment stays initialized.
func (e *OnnxEmbedder) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.session == nil {
		return nil
	}
	err := e.session.Destroy()
	e.session = nil
	e.inited = false
	return err
}

func (e *OnnxEmbedder) ensureSession() error {
	if e.inited {
		return e.initErr
	}
	e.initErr = e.createSession()
	e.inited = true
	return e.initErr
}

func (e *OnnxEmbedder) createSession() error {
	if err := InitRuntimeEnvironment(e.libPath); err != nil {
		return fmt.Errorf("initialize onnxruntime: %w", err)
	}
	session, err := ortNewDynamicSession(e.modelPath, e.manifest.Inputs, e.manifest.Outputs)
	if err != nil {
		return fmt.Errorf("create onnx session: %w", err)
	}
	e.session = session
	return nil
}

func (e *OnnxEmbedder) runLocked(text string) ([]float32, error) {
	ids := e.tokenizer.TokenIDs(text, e.manifest.MaxSeqTokens)
	if len(ids) == 0 {
		return nil, errors.New("tokenized text produced no input ids")
	}
	seqLen := len(ids)
	dims := []int64{1, int64(seqLen)}

	inputs := make([]ortValue, 0, len(e.manifest.Inputs))
	defer func() {
		for _, value := range inputs {
			_ = value.Destroy()
		}
	}()
	for _, name := range e.manifest.Inputs {
		value, err := e.buildInputTensor(name, dims, ids)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, value)
	}

	outputs := make([]ortValue, len(e.manifest.Outputs))
	defer func() {
		for _, value := range outputs {
			if value != nil {
				_ = value.Destroy()
			}
		}
	}()
	if err := e.session.Run(inputs, outputs); err != nil {
		return nil, fmt.Errorf("onnx run failed: %w", err)
	}
	data, shape, ok := outputs[0].floatData()
	if !ok {
		return nil, fmt.Errorf("unexpected output tensor type for %q", e.manifest.Outputs[0])
	}
	return e.extractVector(data, shape, seqLen)
}

func (e *OnnxEmbedder) buildInputTensor(name string, dims []int64, ids []int64) (ortValue, error) {
	switch name {
	case InputIDsName:
		return ortNewInt64Tensor(dims, ids)
	case AttentionMaskName:
		mask := make([]int64, len(ids))
		for i := range mask {
			mask[i] = 1
		}
		return ortNewInt64Tensor(dims, mask)
	case TokenTypeIDsName:
		return ortNewInt64Tensor(dims, make([]int64, len(ids)))
	default:
		return nil, fmt.Errorf("unsupported model input %q", name)
	}
}

func (e *OnnxEmbedder) extractVector(data []float32, shape []int64, seqLen int) ([]float32, error) {
	var vector []float32
	switch len(shape) {
	case 3:
		dim := int(shape[2])
		if dim != e.manifest.Dim {
			return nil, fmt.Errorf("model output dim %d does not match manifest dim %d", dim, e.manifest.Dim)
		}
		if int(shape[1]) != seqLen {
			return nil, fmt.Errorf("model output sequence length %d does not match input length %d", shape[1], seqLen)
		}
		vector = e.poolOutput(data, seqLen, dim)
	case 2:
		dim := int(shape[1])
		if dim != e.manifest.Dim {
			return nil, fmt.Errorf("model output dim %d does not match manifest dim %d", dim, e.manifest.Dim)
		}
		vector = make([]float32, dim)
		copy(vector, data[:dim])
	default:
		return nil, fmt.Errorf("unsupported output tensor rank %d", len(shape))
	}
	if e.manifest.Normalize {
		L2Normalize(vector)
	}
	return vector, nil
}

func (e *OnnxEmbedder) poolOutput(hidden []float32, seqLen, dim int) []float32 {
	mask := make([]int64, seqLen)
	for i := range mask {
		mask[i] = 1
	}
	if e.manifest.Pooling == PoolingCLS {
		return CLSPool(hidden, seqLen, dim)
	}
	return MeanPool(hidden, seqLen, dim, mask)
}

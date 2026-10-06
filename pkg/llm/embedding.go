package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Embedder generates vector embeddings for textual input.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	EmbedSingle(ctx context.Context, text string) ([]float32, error)
	Model() string
}

// OpenAIEmbedder connects to any OpenAI-compatible embedding API (OpenRouter, OpenAI, Ollama).
type OpenAIEmbedder struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewOpenAIEmbedder creates an embedder instance.
func NewOpenAIEmbedder(baseURL, apiKey, model string, allowPrivate ...bool) *OpenAIEmbedder {
	allowPriv := false
	if len(allowPrivate) > 0 && allowPrivate[0] {
		allowPriv = true
	}
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	endpoint := baseURL
	if !strings.HasSuffix(endpoint, "/embeddings") {
		if strings.HasSuffix(endpoint, "/v1") {
			endpoint += "/embeddings"
		} else {
			endpoint += "/v1/embeddings"
		}
	}

	if model == "" {
		model = "qwen/qwen3-embedding-8b"
	}

	return &OpenAIEmbedder{
		baseURL:    endpoint,
		apiKey:     apiKey,
		model:      model,
		httpClient: NewSafeRetryClient(60*time.Second, 3, allowPriv),
	}
}

func (e *OpenAIEmbedder) Model() string {
	return e.model
}

type openAIEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openAIEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

// Embed generates vector embeddings for a list of texts.
func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, errors.New("input texts list is empty")
	}

	reqBody := openAIEmbeddingRequest{
		Model: e.model,
		Input: texts,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("embedding HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedding response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API returned HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var rawResp openAIEmbeddingResponse
	if err := json.Unmarshal(bodyBytes, &rawResp); err != nil {
		return nil, fmt.Errorf("failed to parse embedding JSON response: %w", err)
	}

	if rawResp.Error != nil && rawResp.Error.Message != "" {
		return nil, fmt.Errorf("embedding API error: %s", rawResp.Error.Message)
	}

	if len(rawResp.Data) == 0 {
		return nil, errors.New("embedding API returned no data vectors")
	}

	result := make([][]float32, len(texts))
	for _, item := range rawResp.Data {
		if item.Index >= 0 && item.Index < len(result) {
			result[item.Index] = item.Embedding
		}
	}

	return result, nil
}

// EmbedSingle generates a single embedding vector for a single text string.
func (e *OpenAIEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	vecs, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, errors.New("empty embedding result")
	}
	return vecs[0], nil
}

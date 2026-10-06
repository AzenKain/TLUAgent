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

// RerankResult represents a ranked document with relevance score.
type RerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
	Document       string  `json:"document,omitempty"`
}

// Reranker re-scores and re-ranks candidate documents against a query.
type Reranker interface {
	Rerank(ctx context.Context, query string, documents []string, topN *int) ([]RerankResult, error)
	Model() string
}

// OpenRouterReranker connects to OpenRouter /api/v1/rerank (e.g. cohere/rerank-4-pro).
type OpenRouterReranker struct {
	endpoint   string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewOpenRouterReranker creates an OpenRouter reranker instance.
func NewOpenRouterReranker(baseURL, apiKey, model string, allowPrivate ...bool) *OpenRouterReranker {
	allowPriv := false
	if len(allowPrivate) > 0 && allowPrivate[0] {
		allowPriv = true
	}
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	endpoint := baseURL
	if !strings.HasSuffix(endpoint, "/rerank") {
		if strings.HasSuffix(endpoint, "/v1") {
			endpoint += "/rerank"
		} else {
			endpoint += "/v1/rerank"
		}
	}

	if model == "" {
		model = "cohere/rerank-4-pro"
	}

	return &OpenRouterReranker{
		endpoint:   endpoint,
		apiKey:     apiKey,
		model:      model,
		httpClient: NewSafeRetryClient(45*time.Second, 3, allowPriv),
	}
}

func (r *OpenRouterReranker) Model() string {
	return r.model
}

type openRouterRerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      *int     `json:"top_n,omitempty"`
}

type openRouterRerankResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
		Document       any     `json:"document,omitempty"`
	} `json:"results"`
	Usage struct {
		SearchUnits int     `json:"search_units"`
		Cost        float64 `json:"cost"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

// Rerank re-scores and sorts documents by semantic relevance to the query.
func (r *OpenRouterReranker) Rerank(ctx context.Context, query string, documents []string, topN *int) ([]RerankResult, error) {
	if len(documents) == 0 {
		return nil, errors.New("documents list is empty")
	}
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("query is empty")
	}

	reqBody := openRouterRerankRequest{
		Model:     r.model,
		Query:     query,
		Documents: documents,
		TopN:      topN,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rerank request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create rerank HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if r.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+r.apiKey)
	}

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("rerank HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read rerank response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rerank API returned HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var rawResp openRouterRerankResponse
	if err := json.Unmarshal(bodyBytes, &rawResp); err != nil {
		return nil, fmt.Errorf("failed to parse rerank JSON response: %w", err)
	}

	if rawResp.Error != nil && rawResp.Error.Message != "" {
		return nil, fmt.Errorf("rerank API error: %s", rawResp.Error.Message)
	}

	results := make([]RerankResult, 0, len(rawResp.Results))
	for _, item := range rawResp.Results {
		docText := ""
		if str, ok := item.Document.(string); ok {
			docText = str
		} else if m, ok := item.Document.(map[string]any); ok {
			if t, ok := m["text"].(string); ok {
				docText = t
			}
		}
		if docText == "" && item.Index >= 0 && item.Index < len(documents) {
			docText = documents[item.Index]
		}

		results = append(results, RerankResult{
			Index:          item.Index,
			RelevanceScore: item.RelevanceScore,
			Document:       docText,
		})
	}

	return results, nil
}

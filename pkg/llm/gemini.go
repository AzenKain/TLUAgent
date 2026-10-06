package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var keyParamPattern = regexp.MustCompile(`(?i)([?&](?:key|api_key|apikey|access_token|token)=)[^&"\s]+`)

func redactKeyParams(message string) string {
	return keyParamPattern.ReplaceAllString(message, "${1}REDACTED")
}

// GeminiConfig holds configuration for Google Gemini API.
type GeminiConfig struct {
	APIKey               string
	DefaultModel         string
	BaseURL              string
	HTTPClient           *http.Client
	AllowPrivateNetworks bool
}

// GeminiProvider connects to Google Generative AI (Gemini) API.
type GeminiProvider struct {
	config     GeminiConfig
	httpClient *http.Client
	baseURL    string
}

// NewGeminiProvider creates a Google Gemini provider instance.
func NewGeminiProvider(cfg GeminiConfig) *GeminiProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "gemini-2.5-flash"
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = NewSafeRetryClient(120*time.Second, 3, cfg.AllowPrivateNetworks)
	}

	return &GeminiProvider{
		config: GeminiConfig{
			APIKey:       cfg.APIKey,
			DefaultModel: defaultModel,
			BaseURL:      baseURL,
			HTTPClient:   httpClient,
		},
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

func (p *GeminiProvider) Name() string {
	return "google-gemini"
}

func (p *GeminiProvider) DefaultModel() string {
	return p.config.DefaultModel
}

// Chat performs a non-streaming generateContent call to Gemini.
func (p *GeminiProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.config.DefaultModel
	}
	model = cleanGeminiModel(model)

	endpoint := fmt.Sprintf("%s/models/%s:generateContent", p.baseURL, model)

	reqBody, err := p.buildRequestBody(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build Gemini request body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		httpReq.Header.Set("x-goog-api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Gemini HTTP request failed: %s", redactKeyParams(err.Error()))
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gemini response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gemini API returned HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var rawResp geminiGenerateContentResponse
	if err := json.Unmarshal(respBytes, &rawResp); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini JSON response: %w", err)
	}

	if rawResp.Error != nil && rawResp.Error.Message != "" {
		return nil, fmt.Errorf("Gemini API error: %s", rawResp.Error.Message)
	}

	if len(rawResp.Candidates) == 0 {
		return nil, errors.New("Gemini API returned no candidates")
	}

	candidate := rawResp.Candidates[0]
	var textBuilder strings.Builder
	var reasoningBuilder strings.Builder
	var toolCalls []ToolCall

	for _, part := range candidate.Content.Parts {
		if part.Thought {
			reasoningBuilder.WriteString(part.Text)
		} else if part.Text != "" {
			textBuilder.WriteString(part.Text)
		}

		if part.FunctionCall != nil {
			argsBytes, _ := json.Marshal(part.FunctionCall.Args)
			toolCalls = append(toolCalls, ToolCall{
				ID:   fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, time.Now().UnixNano()),
				Type: "function",
				Function: FunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsBytes),
				},
			})
		}
	}

	finishReason := mapGeminiFinishReason(candidate.FinishReason, len(toolCalls) > 0)

	var usage TokenUsage
	if rawResp.UsageMetadata != nil {
		usage = TokenUsage{
			PromptTokens:     rawResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: rawResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      rawResp.UsageMetadata.TotalTokenCount,
			ReasoningTokens:  rawResp.UsageMetadata.ThoughtsTokenCount,
		}
	}

	return &ChatResponse{
		ID:    rawResp.ResponseID,
		Model: rawResp.ModelVersion,
		Message: Message{
			Role:             RoleAssistant,
			Content:          textBuilder.String(),
			ReasoningContent: reasoningBuilder.String(),
			ToolCalls:        toolCalls,
		},
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}

// ChatStream initiates a streaming Server-Sent Events (SSE) connection to Gemini.
func (p *GeminiProvider) ChatStream(ctx context.Context, req *ChatRequest) (StreamReader, error) {
	model := req.Model
	if model == "" {
		model = p.config.DefaultModel
	}
	model = cleanGeminiModel(model)

	endpoint := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", p.baseURL, model)

	reqBody, err := p.buildRequestBody(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build Gemini stream request body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP stream request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if p.config.APIKey != "" {
		httpReq.Header.Set("x-goog-api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Gemini HTTP stream request failed: %s", redactKeyParams(err.Error()))
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("Gemini API stream returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	return &geminiStreamReader{
		body:   resp.Body,
		reader: bufio.NewReader(resp.Body),
	}, nil
}

func (p *GeminiProvider) buildRequestBody(req *ChatRequest) ([]byte, error) {
	var systemParts []geminiPart
	if req.SystemPrompt != "" {
		systemParts = append(systemParts, geminiPart{Text: req.SystemPrompt})
	}

	contents := make([]geminiContent, 0, len(req.Messages))

	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			systemParts = append(systemParts, geminiPart{Text: msg.Content})
			continue
		}

		role := "user"
		if msg.Role == RoleAssistant {
			role = "model"
		}

		var parts []geminiPart

		if len(msg.Parts) > 0 {
			for _, p := range msg.Parts {
				switch p.Type {
				case ContentPartText:
					if p.Text != "" {
						parts = append(parts, geminiPart{Text: p.Text})
					}
				case ContentPartImage:
					if p.Image != nil && p.Image.Data != "" {
						mime := p.Image.MimeType
						if mime == "" {
							mime = "image/png"
						}
						parts = append(parts, geminiPart{
							InlineData: &geminiBlob{
								MimeType: mime,
								Data:     p.Image.Data,
							},
						})
					}
				}
			}
		} else if msg.Content != "" {
			parts = append(parts, geminiPart{Text: msg.Content})
		}

		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				var argsMap map[string]any
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &argsMap)
				if argsMap == nil {
					argsMap = make(map[string]any)
				}
				parts = append(parts, geminiPart{
					FunctionCall: &geminiFunctionCall{
						Name: tc.Function.Name,
						Args: argsMap,
					},
				})
			}
		}

		if msg.Role == RoleTool {
			role = "user"
			var respMap map[string]any
			if err := json.Unmarshal([]byte(msg.Content), &respMap); err != nil {
				respMap = map[string]any{"response": msg.Content}
			}
			parts = append(parts, geminiPart{
				FunctionResponse: &geminiFunctionResponse{
					Name:     msg.Name,
					Response: respMap,
				},
			})
		}

		if len(parts) > 0 {
			contents = append(contents, geminiContent{
				Role:  role,
				Parts: parts,
			})
		}
	}

	payload := map[string]any{
		"contents": contents,
	}

	if len(systemParts) > 0 {
		payload["systemInstruction"] = geminiContent{
			Parts: systemParts,
		}
	}

	genConfig := make(map[string]any)
	if req.Temperature != nil {
		genConfig["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		genConfig["topP"] = *req.TopP
	}
	if req.MaxTokens != nil {
		genConfig["maxOutputTokens"] = *req.MaxTokens
	}
	if len(req.Stop) > 0 {
		genConfig["stopSequences"] = req.Stop
	}
	if req.JSONMode {
		genConfig["responseMimeType"] = "application/json"
	}
	if req.ThinkingDisabled || (req.ThinkingEnabled != nil && !*req.ThinkingEnabled) {
		genConfig["thinkingConfig"] = map[string]any{"thinkingBudget": 0}
	} else if req.ThinkingBudget != nil && *req.ThinkingBudget > 0 {
		genConfig["thinkingConfig"] = map[string]any{"thinkingBudget": *req.ThinkingBudget}
	} else if req.ThinkingEnabled != nil && *req.ThinkingEnabled {
		genConfig["thinkingConfig"] = map[string]any{"thinkingBudget": -1}
	}
	if len(genConfig) > 0 {
		payload["generationConfig"] = genConfig
	}

	if len(req.Tools) > 0 {
		declarations := make([]geminiFunctionDeclaration, 0, len(req.Tools))
		for _, t := range req.Tools {
			declarations = append(declarations, geminiFunctionDeclaration{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			})
		}
		payload["tools"] = []map[string]any{
			{"functionDeclarations": declarations},
		}
	}

	return json.Marshal(payload)
}

func cleanGeminiModel(m string) string {
	m = strings.TrimPrefix(m, "models/")
	return m
}

func mapGeminiFinishReason(reason string, hasToolCalls bool) string {
	if hasToolCalls {
		return "tool_calls"
	}
	switch strings.ToUpper(reason) {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY", "RECITATION":
		return "content_filter"
	default:
		return "stop"
	}
}

type geminiStreamReader struct {
	body   io.ReadCloser
	reader *bufio.Reader
	closed bool
}

func (r *geminiStreamReader) Recv() (*StreamChunk, error) {
	if r.closed {
		return nil, ErrStreamClosed
	}

	for {
		lineBytes, err := r.reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) && len(lineBytes) == 0 {
				return nil, io.EOF
			}
			if len(lineBytes) == 0 {
				return nil, err
			}
		}

		line := strings.TrimSpace(string(lineBytes))
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil, io.EOF
		}

		var rawChunk geminiGenerateContentResponse
		if err := json.Unmarshal([]byte(data), &rawChunk); err != nil {
			continue
		}

		var textBuilder strings.Builder
		var reasoningBuilder strings.Builder
		var toolCallChunks []ToolCallChunk
		var finishReason string

		if len(rawChunk.Candidates) > 0 {
			candidate := rawChunk.Candidates[0]
			for _, part := range candidate.Content.Parts {
				if part.Thought {
					reasoningBuilder.WriteString(part.Text)
				} else if part.Text != "" {
					textBuilder.WriteString(part.Text)
				}

				if part.FunctionCall != nil {
					argsBytes, _ := json.Marshal(part.FunctionCall.Args)
					toolCallChunks = append(toolCallChunks, ToolCallChunk{
						Index: len(toolCallChunks),
						ID:    fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, time.Now().UnixNano()),
						Type:  "function",
						Function: FunctionCallChunk{
							Name:      part.FunctionCall.Name,
							Arguments: string(argsBytes),
						},
					})
				}
			}

			finishReason = mapGeminiFinishReason(candidate.FinishReason, len(toolCallChunks) > 0)
		}

		var usage *TokenUsage
		if rawChunk.UsageMetadata != nil {
			usage = &TokenUsage{
				PromptTokens:     rawChunk.UsageMetadata.PromptTokenCount,
				CompletionTokens: rawChunk.UsageMetadata.CandidatesTokenCount,
				TotalTokens:      rawChunk.UsageMetadata.TotalTokenCount,
				ReasoningTokens:  rawChunk.UsageMetadata.ThoughtsTokenCount,
			}
		}

		return &StreamChunk{
			ID:    rawChunk.ResponseID,
			Model: rawChunk.ModelVersion,
			Delta: MessageDelta{
				Role:             RoleAssistant,
				Content:          textBuilder.String(),
				ReasoningContent: reasoningBuilder.String(),
				ToolCalls:        toolCallChunks,
			},
			FinishReason: finishReason,
			Usage:        usage,
		}, nil
	}
}

func (r *geminiStreamReader) Close() error {
	r.closed = true
	if r.body != nil {
		return r.body.Close()
	}
	return nil
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	Thought          bool                    `json:"thought,omitempty"`
	InlineData       *geminiBlob             `json:"inlineData,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiBlob struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type geminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiFunctionDeclaration struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type geminiGenerateContentResponse struct {
	Candidates []struct {
		Content struct {
			Role  string       `json:"role"`
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
		Index        int    `json:"index"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
		ThoughtsTokenCount   int `json:"thoughtsTokenCount"`
	} `json:"usageMetadata"`
	ModelVersion string `json:"modelVersion"`
	ResponseID   string `json:"responseId"`
	Error        *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

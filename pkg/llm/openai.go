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
	"strings"
	"time"
)

// OpenAIConfig holds configuration for OpenAI-compatible providers.
type OpenAIConfig struct {
	BaseURL              string
	APIKey               string
	DefaultModel         string
	HTTPClient           *http.Client
	CustomHeaders        map[string]string
	AllowPrivateNetworks bool
}

// OpenAIProvider connects to OpenAI or any OpenAI-compatible API (Vilao, DeepSeek, Ollama, OpenRouter).
type OpenAIProvider struct {
	config     OpenAIConfig
	httpClient *http.Client
	endpoint   string
}

// NewOpenAIProvider creates an OpenAI compatible provider instance.
func NewOpenAIProvider(cfg OpenAIConfig) *OpenAIProvider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	endpoint := baseURL
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		if strings.HasSuffix(endpoint, "/v1") {
			endpoint += "/chat/completions"
		} else {
			endpoint += "/v1/chat/completions"
		}
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = NewSafeRetryClient(120*time.Second, 3, cfg.AllowPrivateNetworks)
	}

	return &OpenAIProvider{
		config:     cfg,
		httpClient: httpClient,
		endpoint:   endpoint,
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai-compatible"
}

func (p *OpenAIProvider) DefaultModel() string {
	return p.config.DefaultModel
}

// Chat performs a non-streaming chat completion request.
func (p *OpenAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	reqBody, err := p.buildRequestBody(req, false)
	if err != nil {
		return nil, fmt.Errorf("failed to build OpenAI request body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	p.applyHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var rawResp openAIChatCompletionResponse
	if err := json.Unmarshal(respBytes, &rawResp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI JSON response: %w", err)
	}

	if rawResp.Error != nil && rawResp.Error.Message != "" {
		return nil, fmt.Errorf("OpenAI API error: %s", rawResp.Error.Message)
	}

	if len(rawResp.Choices) == 0 {
		return nil, errors.New("OpenAI API returned no choices")
	}

	choice := rawResp.Choices[0]
	toolCalls := make([]ToolCall, 0, len(choice.Message.ToolCalls))
	for _, tc := range choice.Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	reasoning := choice.Message.ReasoningContent
	if reasoning == "" && choice.Message.Reasoning != "" {
		reasoning = choice.Message.Reasoning
	}

	return &ChatResponse{
		ID:    rawResp.ID,
		Model: rawResp.Model,
		Message: Message{
			Role:             Role(choice.Message.Role),
			Content:          choice.Message.Content,
			ReasoningContent: reasoning,
			ToolCalls:        toolCalls,
		},
		FinishReason: choice.FinishReason,
		Usage: TokenUsage{
			PromptTokens:     rawResp.Usage.PromptTokens,
			CompletionTokens: rawResp.Usage.CompletionTokens,
			TotalTokens:      rawResp.Usage.TotalTokens,
			ReasoningTokens:  rawResp.Usage.CompletionTokensDetails.ReasoningTokens,
		},
	}, nil
}

// ChatStream initiates a streaming Server-Sent Events (SSE) connection.
func (p *OpenAIProvider) ChatStream(ctx context.Context, req *ChatRequest) (StreamReader, error) {
	reqBody, err := p.buildRequestBody(req, true)
	if err != nil {
		return nil, fmt.Errorf("failed to build OpenAI stream request body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP stream request: %w", err)
	}

	p.applyHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP stream request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("OpenAI API stream returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	return &openAIStreamReader{
		body:   resp.Body,
		reader: bufio.NewReader(resp.Body),
	}, nil
}

func (p *OpenAIProvider) applyHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}
	for k, v := range p.config.CustomHeaders {
		req.Header.Set(k, v)
	}
}

func (p *OpenAIProvider) buildRequestBody(req *ChatRequest, stream bool) ([]byte, error) {
	model := req.Model
	if model == "" {
		model = p.config.DefaultModel
	}

	messages := make([]openAIMessage, 0, len(req.Messages)+1)
	if req.SystemPrompt != "" {
		messages = append(messages, openAIMessage{
			Role:    string(RoleSystem),
			Content: req.SystemPrompt,
		})
	}

	for _, msg := range req.Messages {
		oMsg := openAIMessage{
			Role:       string(msg.Role),
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
		}

		if len(msg.Parts) > 0 {
			parts := make([]any, 0, len(msg.Parts))
			for _, part := range msg.Parts {
				switch part.Type {
				case ContentPartText:
					parts = append(parts, map[string]any{"type": "text", "text": part.Text})
				case ContentPartImage:
					if part.Image != nil {
						url := part.Image.URL
						if url == "" && part.Image.Data != "" {
							mime := part.Image.MimeType
							if mime == "" {
								mime = "image/png"
							}
							url = fmt.Sprintf("data:%s;base64,%s", mime, part.Image.Data)
						}
						parts = append(parts, map[string]any{
							"type": "image_url",
							"image_url": map[string]any{
								"url": url,
							},
						})
					}
				}
			}
			oMsg.Content = parts
		} else {
			oMsg.Content = msg.Content
		}

		if len(msg.ToolCalls) > 0 {
			tcs := make([]openAIToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				tcs = append(tcs, openAIToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: openAIFunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
			oMsg.ToolCalls = tcs
		}

		messages = append(messages, oMsg)
	}

	payload := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   stream,
	}

	if req.Temperature != nil {
		payload["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		payload["top_p"] = *req.TopP
	}
	if req.MaxTokens != nil {
		payload["max_tokens"] = *req.MaxTokens
	}
	if len(req.Stop) > 0 {
		payload["stop"] = req.Stop
	}
	if req.JSONMode {
		payload["response_format"] = map[string]string{"type": "json_object"}
	}
	if len(req.Tools) > 0 {
		payload["tools"] = req.Tools
		if req.ToolChoice != nil {
			payload["tool_choice"] = req.ToolChoice
		}
	}
	if req.ReasoningEffort != "" {
		payload["reasoning_effort"] = req.ReasoningEffort
	}
	if req.ThinkingDisabled || (req.ThinkingEnabled != nil && !*req.ThinkingEnabled) {
		payload["reasoning_effort"] = "none"
		payload["reasoning"] = map[string]any{"effort": "none"}
	} else if req.ReasoningEffort != "" || (req.ThinkingBudget != nil && *req.ThinkingBudget > 0) || (req.ThinkingEnabled != nil && *req.ThinkingEnabled) {
		reasoning := make(map[string]any)
		if req.ReasoningEffort != "" {
			reasoning["effort"] = req.ReasoningEffort
		}
		if req.ThinkingBudget != nil && *req.ThinkingBudget > 0 {
			reasoning["max_tokens"] = *req.ThinkingBudget
		}
		if len(reasoning) > 0 {
			payload["reasoning"] = reasoning
		}
	}
	if stream {
		payload["stream_options"] = map[string]any{"include_usage": true}
	}

	return json.Marshal(payload)
}

type openAIStreamReader struct {
	body   io.ReadCloser
	reader *bufio.Reader
	closed bool
}

func (r *openAIStreamReader) Recv() (*StreamChunk, error) {
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

		var rawChunk openAIChatStreamChunk
		if err := json.Unmarshal([]byte(data), &rawChunk); err != nil {
			continue
		}

		var delta MessageDelta
		var finishReason string

		if len(rawChunk.Choices) > 0 {
			choice := rawChunk.Choices[0]
			finishReason = choice.FinishReason
			delta.Role = Role(choice.Delta.Role)
			delta.Content = choice.Delta.Content

			reasoning := choice.Delta.ReasoningContent
			if reasoning == "" && choice.Delta.Reasoning != "" {
				reasoning = choice.Delta.Reasoning
			}
			if reasoning == "" && choice.Delta.Thought != "" {
				reasoning = choice.Delta.Thought
			}
			delta.ReasoningContent = reasoning

			if len(choice.Delta.ToolCalls) > 0 {
				tcChunks := make([]ToolCallChunk, 0, len(choice.Delta.ToolCalls))
				for _, tc := range choice.Delta.ToolCalls {
					tcChunks = append(tcChunks, ToolCallChunk{
						Index: tc.Index,
						ID:    tc.ID,
						Type:  tc.Type,
						Function: FunctionCallChunk{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					})
				}
				delta.ToolCalls = tcChunks
			}
		}

		var usage *TokenUsage
		if rawChunk.Usage != nil {
			usage = &TokenUsage{
				PromptTokens:     rawChunk.Usage.PromptTokens,
				CompletionTokens: rawChunk.Usage.CompletionTokens,
				TotalTokens:      rawChunk.Usage.TotalTokens,
				ReasoningTokens:  rawChunk.Usage.CompletionTokensDetails.ReasoningTokens,
			}
		}

		return &StreamChunk{
			ID:           rawChunk.ID,
			Model:        rawChunk.Model,
			Delta:        delta,
			FinishReason: finishReason,
			Usage:        usage,
		}, nil
	}
}

func (r *openAIStreamReader) Close() error {
	r.closed = true
	if r.body != nil {
		return r.body.Close()
	}
	return nil
}

type openAIMessage struct {
	Role       string            `json:"role"`
	Content    any               `json:"content"`
	Name       string            `json:"name,omitempty"`
	ToolCalls  []openAIToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIChatCompletionResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role             string           `json:"role"`
			Content          string           `json:"content"`
			ReasoningContent string           `json:"reasoning_content,omitempty"`
			Reasoning        string           `json:"reasoning,omitempty"`
			ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens            int `json:"prompt_tokens"`
		CompletionTokens        int `json:"completion_tokens"`
		TotalTokens             int `json:"total_tokens"`
		CompletionTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"completion_tokens_details"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type openAIChatStreamChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role             string `json:"role,omitempty"`
			Content          string `json:"content,omitempty"`
			ReasoningContent string `json:"reasoning_content,omitempty"`
			Reasoning        string `json:"reasoning,omitempty"`
			Thought          string `json:"thought,omitempty"`
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id,omitempty"`
				Type     string `json:"type,omitempty"`
				Function struct {
					Name      string `json:"name,omitempty"`
					Arguments string `json:"arguments,omitempty"`
				} `json:"function,omitempty"`
			} `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens            int `json:"prompt_tokens"`
		CompletionTokens        int `json:"completion_tokens"`
		TotalTokens             int `json:"total_tokens"`
		CompletionTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"completion_tokens_details"`
	} `json:"usage,omitempty"`
}

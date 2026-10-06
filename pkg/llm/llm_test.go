package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"tluagent-web/pkg/config"
	"tluagent-web/pkg/netx"
)

func init() {
	netx.AllowPrivateIPsInTest = true
}

func TestOpenAIProvider_MockNonStream(t *testing.T) {
	mockResponse := openAIChatCompletionResponse{
		ID:    "chatcmpl-test-123",
		Model: "mock-model",
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role             string           `json:"role"`
				Content          string           `json:"content"`
				ReasoningContent string           `json:"reasoning_content,omitempty"`
				Reasoning        string           `json:"reasoning,omitempty"`
				ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: struct {
					Role             string           `json:"role"`
					Content          string           `json:"content"`
					ReasoningContent string           `json:"reasoning_content,omitempty"`
					Reasoning        string           `json:"reasoning,omitempty"`
					ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
				}{
					Role:             "assistant",
					Content:          "Hello! How can I help you?",
					ReasoningContent: "The user greeted, a friendly reply is expected.",
				},
				FinishReason: "stop",
			},
		},
	}
	mockResponse.Usage.PromptTokens = 10
	mockResponse.Usage.CompletionTokens = 20
	mockResponse.Usage.TotalTokens = 30
	mockResponse.Usage.CompletionTokensDetails.ReasoningTokens = 12

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-secret-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var reqMap map[string]any
		_ = json.NewDecoder(r.Body).Decode(&reqMap)
		if reqMap["stream"] == true {
			http.Error(w, "stream expected false", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(OpenAIConfig{
		BaseURL:      server.URL,
		APIKey:       "test-secret-token",
		DefaultModel: "mock-model",
	})

	resp, err := provider.Chat(context.Background(), &ChatRequest{
		Messages: []Message{
			SystemMessage("You are an academic assistant."),
			UserMessage("Hello!"),
		},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Message.Content != "Hello! How can I help you?" {
		t.Fatalf("Unexpected content: %s", resp.Message.Content)
	}
	if resp.Message.ReasoningContent != "The user greeted, a friendly reply is expected." {
		t.Fatalf("Unexpected reasoning content: %s", resp.Message.ReasoningContent)
	}
	if resp.Usage.ReasoningTokens != 12 {
		t.Fatalf("Unexpected reasoning tokens: %d", resp.Usage.ReasoningTokens)
	}
}

func TestOpenAIProvider_MockStream(t *testing.T) {
	ssePayload := `data: {"id":"c-1","model":"mock","choices":[{"index":0,"delta":{"role":"assistant","reasoning_content":"Thinking..."}}]}

data: {"id":"c-1","model":"mock","choices":[{"index":0,"delta":{"content":"Hello"}}]}

data: {"id":"c-1","model":"mock","choices":[{"index":0,"delta":{"content":" world!"},"finish_reason":"stop"}]}

data: [DONE]

`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(ssePayload))
	}))
	defer server.Close()

	provider := NewOpenAIProvider(OpenAIConfig{
		BaseURL:      server.URL,
		APIKey:       "test-token",
		DefaultModel: "mock-model",
	})

	stream, err := provider.ChatStream(context.Background(), &ChatRequest{
		Messages: []Message{UserMessage("Hi")},
	})
	if err != nil {
		t.Fatalf("ChatStream failed: %v", err)
	}

	var chunks []string
	var reasoningChunks []string
	resp, err := ConsumeStream(stream, func(chunk *StreamChunk) error {
		if chunk.Delta.Content != "" {
			chunks = append(chunks, chunk.Delta.Content)
		}
		if chunk.Delta.ReasoningContent != "" {
			reasoningChunks = append(reasoningChunks, chunk.Delta.ReasoningContent)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ConsumeStream failed: %v", err)
	}

	if resp.Message.Content != "Hello world!" {
		t.Fatalf("Expected 'Hello world!', got '%s'", resp.Message.Content)
	}
	if resp.Message.ReasoningContent != "Thinking..." {
		t.Fatalf("Expected 'Thinking...', got '%s'", resp.Message.ReasoningContent)
	}
	if len(chunks) != 2 {
		t.Fatalf("Expected 2 text chunks, got %d", len(chunks))
	}
}

func TestGeminiProvider_MockNonStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "gemini-test-key" {
			http.Error(w, "invalid key", http.StatusUnauthorized)
			return
		}

		mockResp := geminiGenerateContentResponse{
			ResponseID:   "gem-resp-1",
			ModelVersion: "gemini-2.5-flash",
			Candidates: []struct {
				Content struct {
					Role  string       `json:"role"`
					Parts []geminiPart `json:"parts"`
				} `json:"content"`
				FinishReason string `json:"finishReason"`
				Index        int    `json:"index"`
			}{
				{
					Content: struct {
						Role  string       `json:"role"`
						Parts []geminiPart `json:"parts"`
					}{
						Role: "model",
						Parts: []geminiPart{
							{Text: "I am thinking...", Thought: true},
							{Text: "The answer is 42."},
						},
					},
					FinishReason: "STOP",
					Index:        0,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	provider := NewGeminiProvider(GeminiConfig{
		BaseURL:      server.URL,
		APIKey:       "gemini-test-key",
		DefaultModel: "gemini-2.5-flash",
	})

	resp, err := provider.Chat(context.Background(), &ChatRequest{
		Messages: []Message{
			UserMessage("What is the meaning of life?"),
		},
	})
	if err != nil {
		t.Fatalf("Gemini Chat failed: %v", err)
	}

	if resp.Message.Content != "The answer is 42." {
		t.Fatalf("Unexpected content: %s", resp.Message.Content)
	}
	if resp.Message.ReasoningContent != "I am thinking..." {
		t.Fatalf("Unexpected reasoning: %s", resp.Message.ReasoningContent)
	}
}

func TestManager_MultiProviderAndFallback(t *testing.T) {
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer failServer.Close()

	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAIChatCompletionResponse{
			ID: "ok-1",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role             string           `json:"role"`
					Content          string           `json:"content"`
					ReasoningContent string           `json:"reasoning_content,omitempty"`
					Reasoning        string           `json:"reasoning,omitempty"`
					ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Role             string           `json:"role"`
						Content          string           `json:"content"`
						ReasoningContent string           `json:"reasoning_content,omitempty"`
						Reasoning        string           `json:"reasoning,omitempty"`
						ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
					}{
						Role:    "assistant",
						Content: "Fallback success!",
					},
					FinishReason: "stop",
				},
			},
		})
	}))
	defer successServer.Close()

	pFail := NewOpenAIProvider(OpenAIConfig{
		BaseURL: failServer.URL,
		APIKey:  "fail",
	})
	pSuccess := NewOpenAIProvider(OpenAIConfig{
		BaseURL: successServer.URL,
		APIKey:  "success",
	})

	manager := NewManager()
	manager.Register("primary", pFail)
	manager.Register("backup", pSuccess)

	resp, err := manager.ChatWithFallback(context.Background(), &ChatRequest{
		Messages: []Message{UserMessage("Test fallback")},
	}, "primary", "backup")

	if err != nil {
		t.Fatalf("Expected fallback to succeed, got error: %v", err)
	}
	if resp.Message.Content != "Fallback success!" {
		t.Fatalf("Expected 'Fallback success!', got: %s", resp.Message.Content)
	}
}

func TestOpenAIProvider_ToolCalling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAIChatCompletionResponse{
			ID: "call-1",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role             string           `json:"role"`
					Content          string           `json:"content"`
					ReasoningContent string           `json:"reasoning_content,omitempty"`
					Reasoning        string           `json:"reasoning,omitempty"`
					ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Role             string           `json:"role"`
						Content          string           `json:"content"`
						ReasoningContent string           `json:"reasoning_content,omitempty"`
						Reasoning        string           `json:"reasoning,omitempty"`
						ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
					}{
						Role: "assistant",
						ToolCalls: []openAIToolCall{
							{
								ID:   "call_tlu_123",
								Type: "function",
								Function: openAIFunctionCall{
									Name:      "lookup_regulation",
									Arguments: `{"article": 14, "keyword": "credit improvement"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
		})
	}))
	defer server.Close()

	provider := NewOpenAIProvider(OpenAIConfig{
		BaseURL:      server.URL,
		APIKey:       "token",
		DefaultModel: "test-model",
	})

	resp, err := provider.Chat(context.Background(), &ChatRequest{
		Messages: []Message{UserMessage("Credit improvement regulations?")},
		Tools: []ToolDefinition{
			{
				Type: "function",
				Function: FunctionDefinition{
					Name:        "lookup_regulation",
					Description: "Look up TLU training regulations",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"article": map[string]any{"type": "integer"},
							"keyword": map[string]any{"type": "string"},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ToolCalling Chat failed: %v", err)
	}

	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(resp.Message.ToolCalls))
	}
	tc := resp.Message.ToolCalls[0]
	if tc.ID != "call_tlu_123" || tc.Function.Name != "lookup_regulation" {
		t.Fatalf("Unexpected tool call data: %+v", tc)
	}
}

func TestOpenAIProvider_StreamingToolCallingAssembly(t *testing.T) {
	c1, _ := json.Marshal(map[string]any{
		"id": "c-stream-1",
		"choices": []any{
			map[string]any{
				"index": 0,
				"delta": map[string]any{
					"role": "assistant",
					"tool_calls": []any{
						map[string]any{
							"index": 0,
							"id":    "call_abc",
							"type":  "function",
							"function": map[string]any{
								"name":      "search_curriculum",
								"arguments": "",
							},
						},
					},
				},
			},
		},
	})
	c2, _ := json.Marshal(map[string]any{
		"id": "c-stream-1",
		"choices": []any{
			map[string]any{
				"index": 0,
				"delta": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"index": 0,
							"function": map[string]any{
								"arguments": `{"major":`,
							},
						},
					},
				},
			},
		},
	})
	c3, _ := json.Marshal(map[string]any{
		"id": "c-stream-1",
		"choices": []any{
			map[string]any{
				"index": 0,
				"delta": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"index": 0,
							"function": map[string]any{
								"arguments": `"CNTT"}`,
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "data: %s\n\ndata: %s\n\ndata: %s\n\ndata: [DONE]\n\n", c1, c2, c3)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(OpenAIConfig{
		BaseURL: server.URL,
		APIKey:  "token",
	})

	stream, err := provider.ChatStream(context.Background(), &ChatRequest{
		Messages: []Message{UserMessage("IT department courses")},
	})
	if err != nil {
		t.Fatalf("ChatStream error: %v", err)
	}

	resp, err := ConsumeStream(stream, nil)
	if err != nil {
		t.Fatalf("ConsumeStream error: %v", err)
	}

	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("Expected 1 assembled tool call, got %d", len(resp.Message.ToolCalls))
	}
	tc := resp.Message.ToolCalls[0]
	if tc.ID != "call_abc" || tc.Function.Name != "search_curriculum" {
		t.Fatalf("Unexpected tool call: %+v", tc)
	}
	if tc.Function.Arguments != `{"major":"CNTT"}` {
		t.Fatalf("Expected arguments '{\"major\":\"CNTT\"}', got '%s'", tc.Function.Arguments)
	}
}

// TestLive_EnvAPIs tests live API connectivity if credentials exist in .env
func TestLive_EnvAPIs(t *testing.T) {
	_ = config.LoadEnv()

	manager, err := NewManagerFromEnv()
	if err != nil {
		t.Skipf("Skipping live test: %v", err)
		return
	}

	if p, err := manager.Get("vilao"); err == nil || func() bool { p, err = manager.Get("openai"); return err == nil }() {
		t.Run("Live_OpenAI_Vilao_Streaming", func(t *testing.T) {
			token := os.Getenv("TOKEN")
			if token == "" {
				token = os.Getenv("OPENAI_API_KEY")
			}
			if token == "" {
				t.Skip("No OpenAI/Vilao token configured")
			}

			subCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			stream, err := p.ChatStream(subCtx, &ChatRequest{
				Messages: []Message{
					UserMessage("Hello, quickly count from 1 to 3."),
				},
			})
			if err != nil {
				if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") || strings.Contains(err.Error(), "quota") || strings.Contains(err.Error(), "deadline exceeded") {
					t.Skipf("Skipping live test due to API quota or timeout: %v", err)
				}
				t.Fatalf("Live OpenAI/Vilao stream error: %v", err)
			}

			var chunks []string
			resp, err := ConsumeStream(stream, func(c *StreamChunk) error {
				if c.Delta.Content != "" {
					chunks = append(chunks, c.Delta.Content)
				}
				return nil
			})
			if err != nil {
				if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") || strings.Contains(err.Error(), "quota") || strings.Contains(err.Error(), "deadline exceeded") {
					t.Skipf("Skipping live test due to API quota or timeout: %v", err)
				}
				t.Fatalf("Live OpenAI/Vilao consume error: %v", err)
			}

			if strings.TrimSpace(resp.Message.Content) == "" {
				t.Fatalf("Expected non-empty live response")
			}
			t.Logf("Live OpenAI/Vilao streamed %d chunks. Output: %s", len(chunks), resp.Message.Content)
		})
	}

	if p, err := manager.Get("gemini"); err == nil {
		t.Run("Live_Google_Gemini_Streaming", func(t *testing.T) {
			geminiKey := os.Getenv("GOOGLE_AI_API_KEY")
			if geminiKey == "" {
				t.Skip("No Google Gemini API key configured")
			}

			subCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			stream, err := p.ChatStream(subCtx, &ChatRequest{
				Messages: []Message{
					UserMessage("Hello, reply with a single word: OK."),
				},
			})
			if err != nil {
				if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") || strings.Contains(err.Error(), "quota") || strings.Contains(err.Error(), "deadline exceeded") {
					t.Skipf("Skipping live test due to API quota or timeout: %v", err)
				}
				t.Fatalf("Live Gemini stream error: %v", err)
			}

			var chunks []string
			resp, err := ConsumeStream(stream, func(c *StreamChunk) error {
				if c.Delta.Content != "" {
					chunks = append(chunks, c.Delta.Content)
				}
				return nil
			})
			if err != nil {
				if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") || strings.Contains(err.Error(), "quota") || strings.Contains(err.Error(), "deadline exceeded") {
					t.Skipf("Skipping live test due to API quota or timeout: %v", err)
				}
				t.Fatalf("Live Gemini consume error: %v", err)
			}

			if strings.TrimSpace(resp.Message.Content) == "" {
				t.Fatalf("Expected non-empty live Gemini response")
			}
			t.Logf("Live Gemini streamed %d chunks. Output: %s", len(chunks), resp.Message.Content)
		})
	}

	if manager.Embedder() != nil {
		t.Run("Live_OpenRouter_Embeddings_Qwen3", func(t *testing.T) {
			subCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			texts := []string{
				"Thang Long University",
				"Credit-based training regulations",
			}

			vecs, err := manager.Embedder().Embed(subCtx, texts)
			if err != nil {
				if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "deadline exceeded") || strings.Contains(err.Error(), "rate-limited") || strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
					t.Skipf("Skipping live test due to upstream provider rate limit / timeout: %v", err)
				}
				t.Fatalf("Live OpenRouter Embed error: %v", err)
			}

			if len(vecs) != 2 {
				t.Fatalf("Expected 2 embedding vectors, got %d", len(vecs))
			}
			if len(vecs[0]) == 0 {
				t.Fatalf("Vector 0 is empty")
			}

			t.Logf("Successfully generated embeddings using %s! Dimensions: %d", manager.Embedder().Model(), len(vecs[0]))
		})
	}

	if manager.Reranker() != nil {
		t.Run("Live_OpenRouter_Reranker_Cohere", func(t *testing.T) {
			subCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			query := "Regulations on credit improvement and course re-registration"
			docs := []string{
				"Students with grade D or D+ may re-register the course to improve their grade point average.",
				"Procedure to reissue a student card when lost or damaged at the student affairs office.",
				"Thang Long University training regulations require students to complete all courses in the curriculum.",
			}

			results, err := manager.Reranker().Rerank(subCtx, query, docs, nil)
			if err != nil {
				if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "deadline exceeded") || strings.Contains(err.Error(), "rate-limited") || strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
					t.Skipf("Skipping live test due to upstream provider rate limit / timeout: %v", err)
				}
				t.Fatalf("Live OpenRouter Rerank error: %v", err)
			}

			if len(results) == 0 {
				t.Fatalf("Expected non-empty rerank results")
			}

			t.Logf("Rerank Results from %s (Query: '%s'):", manager.Reranker().Model(), query)
			for i, res := range results {
				t.Logf("  [%d] Score: %.4f | Doc: %s", i+1, res.RelevanceScore, res.Document)
			}

			if results[0].Index != 0 {
				t.Fatalf("Expected top ranked document to be index 0, got %d", results[0].Index)
			}
		})
	}
}

func TestOpenAIEmbedder_Mock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAIEmbeddingResponse{
			Object: "list",
			Model:  "mock-embed",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{Object: "embedding", Embedding: []float32{0.1, 0.2, 0.3}, Index: 0},
				{Object: "embedding", Embedding: []float32{0.4, 0.5, 0.6}, Index: 1},
			},
		})
	}))
	defer server.Close()

	embedder := NewOpenAIEmbedder(server.URL, "test-key", "mock-embed")
	vecs, err := embedder.Embed(context.Background(), []string{"hello", "world"})
	if err != nil {
		t.Fatalf("Mock Embed failed: %v", err)
	}

	if len(vecs) != 2 || len(vecs[0]) != 3 {
		t.Fatalf("Unexpected mock embeddings: %+v", vecs)
	}
}

func TestOpenRouterReranker_Mock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openRouterRerankResponse{
			ID:    "rerank-mock-1",
			Model: "mock-rerank",
			Results: []struct {
				Index          int     `json:"index"`
				RelevanceScore float64 `json:"relevance_score"`
				Document       any     `json:"document,omitempty"`
			}{
				{Index: 1, RelevanceScore: 0.95, Document: "doc 2"},
				{Index: 0, RelevanceScore: 0.30, Document: "doc 1"},
			},
		})
	}))
	defer server.Close()

	reranker := NewOpenRouterReranker(server.URL, "test-key", "mock-rerank")
	results, err := reranker.Rerank(context.Background(), "test query", []string{"doc 1", "doc 2"}, nil)
	if err != nil {
		t.Fatalf("Mock Rerank failed: %v", err)
	}

	if len(results) != 2 || results[0].Index != 1 || results[0].RelevanceScore != 0.95 {
		t.Fatalf("Unexpected mock rerank results: %+v", results)
	}
}

func TestVisionCapabilityDetection(t *testing.T) {
	reg := NewCapabilityRegistry("")
	reg.RegisterCapability("inclusionai/ling-3.1-flash", ModelCapability{SupportsVision: false, MaxImages: 0})
	reg.RegisterCapability("google/gemini-2.5-flash", ModelCapability{SupportsVision: true, MaxImages: 5})

	lingCap := reg.DetectCapabilities(context.Background(), "inclusionai/ling-3.1-flash")
	if lingCap.SupportsVision {
		t.Fatalf("Expected inclusionai/ling-3.1-flash to NOT support vision")
	}

	geminiCap := reg.DetectCapabilities(context.Background(), "google/gemini-2.5-flash")
	if !geminiCap.SupportsVision || geminiCap.MaxImages != 5 {
		t.Fatalf("Expected google/gemini-2.5-flash to support vision with max 5 images")
	}

	unregisteredCap := reg.DetectCapabilities(context.Background(), "unknown-model")
	if unregisteredCap.SupportsVision {
		t.Fatalf("Expected unknown model to default to no vision")
	}
}

func TestAdaptiveVisionRequest(t *testing.T) {
	reg := NewCapabilityRegistry("")
	ctx := context.Background()

	msgWithImage := UserMessageWithImageURL("Look at this image", "https://example.com/test.png")
	req := &ChatRequest{
		Model:    "inclusionai/ling-3.1-flash",
		Messages: []Message{msgWithImage},
	}

	if !HasImages(req.Messages) {
		t.Fatalf("Expected HasImages to be true")
	}

	degraded, err := AdaptiveChatRequest(ctx, reg, req, "inclusionai/ling-3.1-flash", VisionStrategyDegrade, "")
	if err != nil {
		t.Fatalf("Degrade failed: %v", err)
	}
	if HasImages(degraded.Messages) {
		t.Fatalf("Degraded request should not contain images")
	}
	if !strings.Contains(degraded.Messages[0].Content, "omitted") {
		t.Fatalf("Degraded request should contain text placeholder, got: %s", degraded.Messages[0].Content)
	}

	fallback, err := AdaptiveChatRequest(ctx, reg, req, "inclusionai/ling-3.1-flash", VisionStrategyFallback, "google/gemini-2.5-flash")
	if err != nil {
		t.Fatalf("Fallback failed: %v", err)
	}
	if fallback.Model != "google/gemini-2.5-flash" {
		t.Fatalf("Expected fallback model google/gemini-2.5-flash, got: %s", fallback.Model)
	}

	_, err = AdaptiveChatRequest(ctx, reg, req, "inclusionai/ling-3.1-flash", VisionStrategyReject, "")
	if err == nil {
		t.Fatalf("Expected error when rejecting non-vision model")
	}

	pureTextReq := &ChatRequest{
		Model:    "inclusionai/ling-3.1-flash",
		Messages: []Message{UserMessage("Hello")},
	}
	adapted, err := AdaptiveChatRequest(ctx, reg, pureTextReq, "inclusionai/ling-3.1-flash", VisionStrategyReject, "")
	if err != nil {
		t.Fatalf("Pure text should never error, got: %v", err)
	}
	if adapted.Model != "inclusionai/ling-3.1-flash" {
		t.Fatalf("Pure text should retain original model")
	}
}

func TestLive_OpenRouter_LingFlash_TextOnly(t *testing.T) {
	_ = config.LoadEnv()
	openRouterKey := os.Getenv("OPEN_ROUTER_API")
	if openRouterKey == "" {
		t.Skip("No OPEN_ROUTER_API configured")
	}

	provider := NewOpenAIProvider(OpenAIConfig{
		BaseURL:      "https://openrouter.ai/api/v1",
		APIKey:       openRouterKey,
		DefaultModel: "inclusionai/ling-3.1-flash",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := provider.Chat(ctx, &ChatRequest{
		Messages: []Message{
			UserMessage("Short question: what is the capital of Vietnam?"),
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate-limited") {
			t.Skipf("Skipping due to upstream provider rate limit: %v", err)
		}
		t.Fatalf("Ling 3.1 Flash text chat failed: %v", err)
	}

	if !strings.Contains(resp.Message.Content, "Hanoi") {
		t.Fatalf("Expected response to mention Hanoi, got: %s", resp.Message.Content)
	}
	t.Logf("Ling 3.1 Flash answered successfully: %s", strings.TrimSpace(resp.Message.Content))
}

// TestLive_OpenRouter_Embedding_Qwen tests live embedding generation via OpenRouter.
func TestLive_OpenRouter_Embedding_Qwen(t *testing.T) {
	_ = config.LoadEnv()
	openRouterKey := os.Getenv("OPEN_ROUTER_API")
	if openRouterKey == "" {
		t.Skip("No OPEN_ROUTER_API configured")
	}

	embedder := NewOpenAIEmbedder("https://openrouter.ai/api/v1", openRouterKey, "qwen/qwen3-embedding-8b")
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	texts := []string{
		"Credit-based training regulations at Thang Long University.",
		"Rules for awarding study encouragement scholarships to students.",
	}

	embeddings, err := embedder.Embed(ctx, texts)
	if err != nil {
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "deadline exceeded") || strings.Contains(err.Error(), "rate-limited") || strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
			t.Skipf("Skipping live test due to upstream provider rate limit / timeout: %v", err)
		}
		t.Fatalf("OpenRouter Qwen3 embedding failed: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Fatalf("Expected %d vectors, got %d", len(texts), len(embeddings))
	}

	for i, vec := range embeddings {
		if len(vec) == 0 {
			t.Fatalf("Embedding vector %d is empty", i)
		}
		t.Logf("Vector %d dimensions: %d, sample: [%.4f, %.4f, %.4f...]", i, len(vec), vec[0], vec[1], vec[2])
	}
}

// TestLive_OpenRouter_Rerank_Cohere tests live document reranking via OpenRouter.
func TestLive_OpenRouter_Rerank_Cohere(t *testing.T) {
	_ = config.LoadEnv()
	openRouterKey := os.Getenv("OPEN_ROUTER_API")
	if openRouterKey == "" {
		t.Skip("No OPEN_ROUTER_API configured")
	}

	reranker := NewOpenRouterReranker("https://openrouter.ai/api/v1", openRouterKey, "cohere/rerank-4-pro")
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	query := "How many credits must a student register for in a main semester?"
	docs := []string{
		"In a main semester students must register for at least 14 credits and at most 25 credits per the regulations.",
		"Tuition for each program is computed as registered credits multiplied by the per-credit fee.",
		"Students may apply for leave or suspend their studies if they meet the health-related conditions.",
	}

	topN := 2
	results, err := reranker.Rerank(ctx, query, docs, &topN)
	if err != nil {
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "deadline exceeded") || strings.Contains(err.Error(), "rate-limited") || strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
			t.Skipf("Skipping live test due to upstream provider rate limit / timeout: %v", err)
		}
		t.Fatalf("OpenRouter Cohere rerank failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("Expected rerank results, got empty slice")
	}

	t.Logf("Rerank results for query '%s':", query)
	for _, r := range results {
		t.Logf("Rank: idx=%d score=%.4f doc='%s'", r.Index, r.RelevanceScore, docs[r.Index])
	}

	if results[0].Index != 0 {
		t.Fatalf("Expected doc 0 to be the top ranked result, got index %d with score %.4f", results[0].Index, results[0].RelevanceScore)
	}
}

func TestSSRFBlocking(t *testing.T) {
	netx.AllowPrivateIPsInTest = false
	defer func() {
		netx.AllowPrivateIPsInTest = true
	}()

	provider := NewOpenAIProvider(OpenAIConfig{
		BaseURL:      "http://127.0.0.1:12345/v1",
		APIKey:       "dummy",
		DefaultModel: "test-model",
	})

	_, err := provider.Chat(context.Background(), &ChatRequest{
		Messages: []Message{UserMessage("ping")},
	})
	if err == nil || !strings.Contains(err.Error(), "private/restricted IP") {
		t.Fatalf("expected SSRF block error, got: %v", err)
	}
}

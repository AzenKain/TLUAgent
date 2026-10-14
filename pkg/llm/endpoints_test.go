package llm

import (
	"context"
	"testing"
)

// TestEndpoints_ResolveOpenAIChatEndpoint verifies OpenAI chat endpoint resolution rules.
func TestEndpoints_ResolveOpenAIChatEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "empty base URL uses default",
			baseURL:  "",
			expected: DefaultOpenAIBaseURL + EndpointChatCompletions,
		},
		{
			name:     "base URL with v1 suffix",
			baseURL:  "https://custom-api.com/v1",
			expected: "https://custom-api.com/v1/chat/completions",
		},
		{
			name:     "base URL with trailing slash",
			baseURL:  "https://custom-api.com/v1/",
			expected: "https://custom-api.com/v1/chat/completions",
		},
		{
			name:     "base URL without v1 suffix",
			baseURL:  "https://custom-api.com",
			expected: "https://custom-api.com/v1/chat/completions",
		},
		{
			name:     "base URL already includes full chat completions path",
			baseURL:  "https://custom-api.com/v1/chat/completions",
			expected: "https://custom-api.com/v1/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveOpenAIChatEndpoint(tt.baseURL)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

// TestEndpoints_ResolveOpenAIEmbeddingsEndpoint verifies embedding endpoint resolution.
func TestEndpoints_ResolveOpenAIEmbeddingsEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "empty base URL uses default OpenRouter",
			baseURL:  "",
			expected: DefaultOpenRouterBaseURL + EndpointEmbeddings,
		},
		{
			name:     "base URL with v1 suffix",
			baseURL:  "https://custom-api.com/v1",
			expected: "https://custom-api.com/v1/embeddings",
		},
		{
			name:     "base URL without v1 suffix",
			baseURL:  "https://custom-api.com",
			expected: "https://custom-api.com/v1/embeddings",
		},
		{
			name:     "base URL already includes full embeddings path",
			baseURL:  "https://custom-api.com/v1/embeddings",
			expected: "https://custom-api.com/v1/embeddings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveOpenAIEmbeddingsEndpoint(tt.baseURL)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

// TestEndpoints_ResolveOpenRouterRerankEndpoint verifies rerank endpoint resolution.
func TestEndpoints_ResolveOpenRouterRerankEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "empty base URL uses default OpenRouter",
			baseURL:  "",
			expected: DefaultOpenRouterBaseURL + EndpointRerank,
		},
		{
			name:     "base URL with v1 suffix",
			baseURL:  "https://custom-api.com/v1",
			expected: "https://custom-api.com/v1/rerank",
		},
		{
			name:     "base URL without v1 suffix",
			baseURL:  "https://custom-api.com",
			expected: "https://custom-api.com/v1/rerank",
		},
		{
			name:     "base URL already includes rerank path",
			baseURL:  "https://custom-api.com/v1/rerank",
			expected: "https://custom-api.com/v1/rerank",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveOpenRouterRerankEndpoint(tt.baseURL)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

// TestEndpoints_GeminiEndpoints verifies Gemini generate and stream endpoint resolution.
func TestEndpoints_GeminiEndpoints(t *testing.T) {
	t.Run("CleanGeminiModel", func(t *testing.T) {
		if CleanGeminiModel("models/gemini-2.5-flash") != "gemini-2.5-flash" {
			t.Fatalf("failed to strip models/ prefix")
		}
		if CleanGeminiModel("gemini-2.5-flash") != "gemini-2.5-flash" {
			t.Fatalf("modified clean model string unexpectedly")
		}
	})

	t.Run("ResolveGeminiGenerateContentEndpoint", func(t *testing.T) {
		got := ResolveGeminiGenerateContentEndpoint("", "gemini-2.5-flash")
		expected := DefaultGeminiBaseURL + "/models/gemini-2.5-flash:generateContent"
		if got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}

		gotCustom := ResolveGeminiGenerateContentEndpoint("https://custom.googleapis.com", "models/my-model")
		expectedCustom := "https://custom.googleapis.com/models/my-model:generateContent"
		if gotCustom != expectedCustom {
			t.Fatalf("expected %q, got %q", expectedCustom, gotCustom)
		}
	})

	t.Run("ResolveGeminiStreamGenerateContentEndpoint", func(t *testing.T) {
		got := ResolveGeminiStreamGenerateContentEndpoint("", "gemini-2.5-flash")
		expected := DefaultGeminiBaseURL + "/models/gemini-2.5-flash:streamGenerateContent?alt=sse"
		if got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})
}

// TestManager_NewManagerWithConfig tests initialization of Manager from configuration.
func TestManager_NewManagerWithConfig(t *testing.T) {
	mockProv := NewOpenAIProvider(OpenAIConfig{
		BaseURL:      "http://127.0.0.1:9999/v1",
		APIKey:       "test-key",
		DefaultModel: "test-model",
	})

	reg := NewCapabilityRegistry()
	reg.RegisterCapability("test-model", ModelCapability{SupportsVision: true})

	mgr := NewManagerWithConfig(ManagerConfig{
		DefaultProvider: "primary",
		Providers: map[string]Provider{
			"primary": mockProv,
		},
		Registry:            reg,
		VisionStrategy:      VisionStrategyReject,
		VisionFallbackModel: "fallback-model",
	})

	if mgr.VisionStrategy() != VisionStrategyReject {
		t.Fatalf("expected vision strategy %v, got %v", VisionStrategyReject, mgr.VisionStrategy())
	}
	if mgr.VisionFallbackModel() != "fallback-model" {
		t.Fatalf("expected fallback model 'fallback-model', got %q", mgr.VisionFallbackModel())
	}
	if !mgr.Registry().SupportsVision(context.Background(), "test-model") {
		t.Fatalf("expected test-model to support vision in registry")
	}

	p, err := mgr.Default()
	if err != nil || p == nil {
		t.Fatalf("expected default provider to be found: %v", err)
	}
	if p.Name() != ProviderOpenAICompatible {
		t.Fatalf("expected provider name %q, got %q", ProviderOpenAICompatible, p.Name())
	}
}

package llm

import (
	"fmt"
	"strings"
)

const (
	DefaultOpenAIBaseURL     = "https://api.openai.com/v1"
	DefaultGeminiBaseURL     = "https://generativelanguage.googleapis.com/v1beta"
	DefaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"

	DefaultGeminiModel = "gemini-2.5-flash"

	ProviderOpenAICompatible = "openai-compatible"
	ProviderGoogleGemini     = "google-gemini"

	EndpointV1                          = "/v1"
	EndpointChatCompletions             = "/chat/completions"
	EndpointV1ChatCompletions           = "/v1/chat/completions"
	EndpointEmbeddings                  = "/embeddings"
	EndpointV1Embeddings                = "/v1/embeddings"
	EndpointRerank                      = "/rerank"
	EndpointV1Rerank                    = "/v1/rerank"
	EndpointGeminiGenerateContent       = "/models/%s:generateContent"
	EndpointGeminiStreamGenerateContent = "/models/%s:streamGenerateContent?alt=sse"

	HeaderContentType   = "Content-Type"
	HeaderAccept        = "Accept"
	HeaderAuthorization = "Authorization"
	HeaderGoogAPIKey    = "x-goog-api-key"

	MIMEApplicationJSON = "application/json"
	MIMETextEventStream = "text/event-stream"

	AuthBearerPrefix = "Bearer "
)

// CleanGeminiModel strips the models/ prefix if present.
func CleanGeminiModel(model string) string {
	return strings.TrimPrefix(model, "models/")
}

// ResolveOpenAIChatEndpoint returns the full endpoint URL for OpenAI chat completions.
func ResolveOpenAIChatEndpoint(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = DefaultOpenAIBaseURL
	}
	if strings.HasSuffix(baseURL, EndpointChatCompletions) {
		return baseURL
	}
	if strings.HasSuffix(baseURL, EndpointV1) {
		return baseURL + EndpointChatCompletions
	}
	return baseURL + EndpointV1ChatCompletions
}

// ResolveOpenAIEmbeddingsEndpoint returns the full endpoint URL for OpenAI compatible embeddings.
func ResolveOpenAIEmbeddingsEndpoint(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = DefaultOpenRouterBaseURL
	}
	if strings.HasSuffix(baseURL, EndpointEmbeddings) {
		return baseURL
	}
	if strings.HasSuffix(baseURL, EndpointV1) {
		return baseURL + EndpointEmbeddings
	}
	return baseURL + EndpointV1Embeddings
}

// ResolveOpenRouterRerankEndpoint returns the full endpoint URL for OpenRouter reranking.
func ResolveOpenRouterRerankEndpoint(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = DefaultOpenRouterBaseURL
	}
	if strings.HasSuffix(baseURL, EndpointRerank) {
		return baseURL
	}
	if strings.HasSuffix(baseURL, EndpointV1) {
		return baseURL + EndpointRerank
	}
	return baseURL + EndpointV1Rerank
}

// ResolveGeminiGenerateContentEndpoint returns the generateContent endpoint for Gemini.
func ResolveGeminiGenerateContentEndpoint(baseURL, model string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = DefaultGeminiBaseURL
	}
	return fmt.Sprintf("%s"+EndpointGeminiGenerateContent, baseURL, CleanGeminiModel(model))
}

// ResolveGeminiStreamGenerateContentEndpoint returns the streamGenerateContent endpoint for Gemini.
func ResolveGeminiStreamGenerateContentEndpoint(baseURL, model string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = DefaultGeminiBaseURL
	}
	return fmt.Sprintf("%s"+EndpointGeminiStreamGenerateContent, baseURL, CleanGeminiModel(model))
}

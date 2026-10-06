package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedactKeyParams(t *testing.T) {
	input := `Post "https://example.com/v1/models/m:generateContent?key=super-secret-value": dial tcp 1.2.3.4:443: connect: connection refused`
	got := redactKeyParams(input)
	if strings.Contains(got, "super-secret-value") {
		t.Fatalf("Key value must be redacted, got %q", got)
	}
	if !strings.Contains(got, "key=REDACTED") {
		t.Fatalf("Expected redacted key parameter, got %q", got)
	}

	other := redactKeyParams("prefix&api_key=abc123&x=1 suffix")
	if strings.Contains(other, "abc123") {
		t.Fatalf("api_key must be redacted, got %q", other)
	}

	unrelated := redactKeyParams("plain error without parameters")
	if unrelated != "plain error without parameters" {
		t.Fatalf("Unrelated messages must pass through unchanged, got %q", unrelated)
	}
}

func TestGeminiEndpoints_DoNotContainKeyParam(t *testing.T) {
	var capturedURLs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURLs = append(capturedURLs, r.URL.String())
		http.Error(w, `{"error":{"code":401,"message":"invalid"}}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	provider := NewGeminiProvider(GeminiConfig{
		BaseURL: server.URL,
		APIKey:  "gemini-test-key",
	})

	_, _ = provider.Chat(context.Background(), &ChatRequest{Model: "gemini-test"})
	_, _ = provider.ChatStream(context.Background(), &ChatRequest{Model: "gemini-test"})

	if len(capturedURLs) != 2 {
		t.Fatalf("Expected two captured requests, got %d", len(capturedURLs))
	}
	for _, url := range capturedURLs {
		if strings.Contains(url, "key=") {
			t.Fatalf("Endpoint URL must not contain the API key: %s", url)
		}
	}
}

func TestGeminiRequestFailure_ErrorDoesNotContainKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	provider := NewGeminiProvider(GeminiConfig{
		BaseURL: server.URL,
		APIKey:  "gemini-test-key",
	})

	_, err := provider.Chat(context.Background(), &ChatRequest{Model: "gemini-test"})
	if err == nil {
		t.Fatalf("Expected request against closed server to fail")
	}
	if strings.Contains(err.Error(), "gemini-test-key") {
		t.Fatalf("Error message must not contain the API key: %v", err)
	}
}

package llm_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/validator"
)

type redteamRoundTripper func(*http.Request) (*http.Response, error)

func (f redteamRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func redteamCredential(t *testing.T) string {
	t.Helper()
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		t.Fatal("cannot generate runtime credential")
	}
	return hex.EncodeToString(value)
}

// This passing test reproduces redirect leakage with production redirect handling and an in-memory network.
func TestRedTeamObservedWeaknessGeminiCrossOriginRedirect(t *testing.T) {
	for _, stream := range []bool{false, true} {
		name := "Chat"
		if stream {
			name = "ChatStream"
		}
		t.Run(name, func(t *testing.T) {
			credential := redteamCredential(t)
			client := llm.NewCustomRetryClient(time.Second, 0, time.Millisecond, time.Millisecond, false)
			defer client.CloseIdleConnections()
			var originBody string
			var forwardedCredential, forwardedBody, downgraded bool
			var requests int
			client.Transport.(*llm.RetryableRoundTripper).Transport = redteamRoundTripper(func(req *http.Request) (*http.Response, error) {
				requests++
				body, err := io.ReadAll(io.LimitReader(req.Body, 64<<10))
				if err != nil {
					t.Fatal("cannot read bounded mock request")
				}
				if req.URL.Host == "redteam-origin.invalid" {
					originBody = string(body)
					return &http.Response{
						StatusCode: http.StatusTemporaryRedirect,
						Header: http.Header{
							"Location": []string{"http://redteam-destination.invalid/capture"},
						},
						Body:    io.NopCloser(strings.NewReader("")),
						Request: req,
					}, nil
				}
				if req.URL.Host != "redteam-destination.invalid" {
					t.Fatal("unexpected in-memory destination")
				}
				forwardedCredential = req.Header.Get("x-goog-api-key") == credential
				forwardedBody = req.Method == http.MethodPost && string(body) == originBody
				downgraded = req.URL.Scheme == "http"
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("request rejected")),
					Request:    req,
				}, nil
			})
			provider := llm.NewGeminiProvider(llm.GeminiConfig{
				BaseURL:      "https://redteam-origin.invalid/v1beta",
				APIKey:       credential,
				DefaultModel: "redteam-model",
				HTTPClient:   client,
			})
			req := &llm.ChatRequest{Messages: []llm.Message{llm.UserMessage("bounded local audit input")}}
			var err error
			if stream {
				_, err = provider.ChatStream(context.Background(), req)
			} else {
				_, err = provider.Chat(context.Background(), req)
			}
			if err == nil {
				t.Fatal("cross-origin redirect should be rejected with error")
			}
			if requests > 1 || forwardedCredential || forwardedBody || downgraded {
				t.Fatal("cross-origin credential or POST-body forwarding detected")
			}
		})
	}
}

type redteamProviderRepository struct {
	repositories.LLMRepository
	captured sqlc.CreateLLMProviderParams
}

func (r *redteamProviderRepository) CreateProvider(_ context.Context, p sqlc.CreateLLMProviderParams) (sqlc.LlmProvider, error) {
	r.captured = p
	return sqlc.LlmProvider{
		ID:                   p.ID,
		Name:                 p.Name,
		ProviderType:         p.ProviderType,
		BaseUrl:              p.BaseUrl,
		ApiKeyCiphertext:     p.ApiKeyCiphertext,
		CustomHeadersJson:    p.CustomHeadersJson,
		AllowPrivateNetworks: p.AllowPrivateNetworks,
	}, nil
}

// Tests encryption at rest and masking of custom header credentials.
func TestRedTeamObservedWeaknessCustomHeaderCredentialExposure(t *testing.T) {
	t.Setenv("LLM_ENCRYPTION_KEY", redteamCredential(t))
	t.Setenv("DATA_DIR", t.TempDir())
	apiCredential := redteamCredential(t)
	headerCredential := redteamCredential(t)
	headers, err := json.Marshal(map[string]string{"X-Api-Key": headerCredential})
	if err != nil {
		t.Fatal("cannot encode custom headers")
	}
	payload, err := json.Marshal(request.CreateLLMProviderRequest{
		ID:                "redteam-provider",
		Name:              "Redteam Provider",
		ProviderType:      "custom",
		BaseURL:           "https://redteam-provider.invalid/v1",
		APIKey:            apiCredential,
		CustomHeadersJSON: string(headers),
	})
	if err != nil {
		t.Fatal("cannot encode provider request")
	}
	var dto request.CreateLLMProviderRequest
	req := httptest.NewRequest(http.MethodPost, "/api/admin/llm/providers", strings.NewReader(string(payload)))
	if errs := validator.ValidateBodyDto(req, &dto); len(errs) != 0 {
		t.Fatal("runtime custom credential did not pass actual DTO validation")
	}
	repo := &redteamProviderRepository{}
	service := services.NewLLMConfigService(repo, nil, nil)
	result, err := service.CreateProvider(context.Background(), dto)
	if err != nil {
		t.Fatal("provider creation failed")
	}
	if repo.captured.ApiKeyCiphertext == apiCredential {
		t.Fatal("main API credential was not encrypted")
	}
	decrypted, err := crypto.DecryptAESGCM(repo.captured.ApiKeyCiphertext, crypto.GetLLMEncryptionKey())
	if err != nil || decrypted != apiCredential {
		t.Fatal("main API credential encryption control failed")
	}
	if strings.Contains(repo.captured.CustomHeadersJson, headerCredential) {
		t.Fatal("custom header credential was stored plaintext in database parameters")
	}
	decHeaders, err := crypto.DecryptAESGCM(repo.captured.CustomHeadersJson, crypto.GetLLMEncryptionKey())
	if err != nil || !strings.Contains(decHeaders, headerCredential) {
		t.Fatal("custom header ciphertext failed to decrypt")
	}
	if strings.Contains(result.CustomHeadersJSON, headerCredential) {
		t.Fatal("custom header credential was not masked in API response")
	}
	if result.MaskedAPIKey == apiCredential {
		t.Fatal("main API credential masking control failed")
	}
}

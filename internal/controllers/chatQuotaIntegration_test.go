package controllers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/routes"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/database"
	"tluagent-web/pkg/llm"
)

func setupChatTestServer(t *testing.T) (*http.ServeMux, *sql.DB) {
	t.Helper()

	t.Setenv("CHAT_GUEST_DAILY_QUOTA", "2")
	t.Setenv("CHAT_USER_DAILY_QUOTA", "6")

	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	ramCache := cache.NewTheineCache(10 << 20)
	txManager := database.NewTxManager(db)
	roleRepo := repositories.NewRoleRepository(db, ramCache)
	userRepo := repositories.NewUserRepository(db, ramCache)
	settingsRepo := repositories.NewSettingsRepository(db, ramCache)
	llmRepo := repositories.NewLLMRepository(db, ramCache)
	ruleRepo := repositories.NewMemoryRuleRepository()
	chatRepo := repositories.NewChatRepository(db, ramCache)

	permCache := services.NewPermissionCache(roleRepo)
	if err := permCache.Reload(context.Background()); err != nil {
		t.Fatalf("Failed to reload permCache: %v", err)
	}

	authSvc := services.NewAuthService(userRepo, roleRepo, settingsRepo, txManager)
	advisorySvc := services.NewAdvisoryService(ruleRepo, llm.NewManager(), userRepo, llmRepo)
	chatSessionSvc := services.NewChatSessionService(chatRepo, userRepo)
	chatQuotaSvc := services.NewChatQuotaService(ramCache)

	authCtrl := controllers.NewAuthController(authSvc)
	chatCtrl := controllers.NewChatController(advisorySvc, chatSessionSvc, nil)
	adminChatCtrl := controllers.NewAdminChatController(chatSessionSvc)
	healthCtrl := controllers.NewHealthController()

	mux := http.NewServeMux()
	routes.RegisterAuthRoutes(mux, authCtrl, userRepo)
	routes.RegisterAPIRoutes(mux, healthCtrl, chatCtrl, adminChatCtrl, userRepo, permCache, chatQuotaSvc)

	return mux, db
}

func postChat(t *testing.T, mux *http.ServeMux, query string, token string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"query": query})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestChat_GuestAllowedAndQuotaEnforced(t *testing.T) {
	mux, db := setupChatTestServer(t)
	defer db.Close()

	first := postChat(t, mux, "hello advisor", "")
	if first.Code != http.StatusOK {
		t.Fatalf("Guest chat should be allowed (guest role has chat.ask), got %d, body: %s", first.Code, first.Body.String())
	}

	if rec := postChat(t, mux, "second question", ""); rec.Code != http.StatusOK {
		t.Fatalf("Second guest chat should be allowed, got %d", rec.Code)
	}

	if rec := postChat(t, mux, "third question", ""); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("Third guest chat must hit the daily quota with 429, got %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestChat_AuthenticatedQuotaAndValidation(t *testing.T) {
	mux, db := setupChatTestServer(t)
	defer db.Close()

	adminPass, _ := crypto.GenerateRandomHex(16)
	setupBody, _ := json.Marshal(map[string]string{
		"email":     "chatchat@thanglong.edu.vn",
		"password":  adminPass,
		"full_name": "Chat Admin",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewReader(setupBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Setup failed: %d, body: %s", rec.Code, rec.Body.String())
	}
	adminToken, _ := authCookies(t, rec)

	oversized := make([]byte, 4001)
	for i := range oversized {
		oversized[i] = 'a'
	}
	rec = postChat(t, mux, string(oversized), adminToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Query exceeding 4000 characters must be rejected with 400, got %d", rec.Code)
	}

	tooManyImages, _ := json.Marshal(map[string]any{
		"query":  "what is in these images?",
		"images": []string{"https://example.com/a.png", "https://example.com/b.png", "https://example.com/c.png", "https://example.com/d.png", "https://example.com/e.png", "https://example.com/f.png"},
	})
	req = httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(tooManyImages))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("More than 5 images must be rejected with 400, got %d", rec.Code)
	}

	for i := 0; i < 4; i++ {
		if rec := postChat(t, mux, "question "+string(rune('a'+i)), adminToken); rec.Code != http.StatusOK {
			t.Fatalf("Authenticated chat %d should be allowed, got %d", i+1, rec.Code)
		}
	}
	if rec := postChat(t, mux, "one too many", adminToken); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("Authenticated chat beyond quota must return 429, got %d", rec.Code)
	}
}

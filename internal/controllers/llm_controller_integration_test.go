package controllers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/routes"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/database"
	"tluagent-web/pkg/llm"
)

func setupLLMTestServer(t *testing.T) (*http.ServeMux, *sql.DB, string) {
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

	permCache := services.NewPermissionCache(roleRepo)
	if err := permCache.Reload(context.Background()); err != nil {
		t.Fatalf("Failed to reload permCache: %v", err)
	}

	manager := llm.NewManager()
	llmConfigSvc := services.NewLLMConfigService(llmRepo, manager)
	authSvc := services.NewAuthService(userRepo, roleRepo, settingsRepo, txManager)

	authCtrl := controllers.NewAuthController(authSvc)
	llmCtrl := controllers.NewLLMController(llmConfigSvc)

	mux := http.NewServeMux()
	routes.RegisterAuthRoutes(mux, authCtrl, userRepo)
	routes.RegisterLLMRoutes(mux, llmCtrl, userRepo, permCache)

	dynPass, _ := crypto.GenerateRandomHex(16)
	setupDto := request.SetupDto{
		Email:       "admin@tlu.edu.vn",
		Password:    dynPass,
		FullName:    "System Administrator",
		StudentCode: "ADMIN001",
	}
	body, _ := json.Marshal(setupDto)
	req := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var adminToken string
	for _, c := range w.Result().Cookies() {
		if c.Name == "access_token" {
			adminToken = c.Value
		}
	}
	if adminToken == "" {
		t.Fatalf("Expected access token cookie from setup response")
	}

	return mux, db, adminToken
}

func TestLLMController_Endpoints(t *testing.T) {
	mux, db, adminToken := setupLLMTestServer(t)
	defer db.Close()

	t.Run("Public_GetActiveChatModels_EmptyInitial", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/chat/models", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got: %d", w.Code)
		}
	})

	t.Run("Admin_CreateProvider_UnauthorizedWithoutToken", func(t *testing.T) {
		isTrue := true
		testKey, err := crypto.GenerateRandomHex(16)
		if err != nil {
			t.Fatalf("Failed to generate test key: %v", err)
		}
		dto := request.CreateLLMProviderRequest{
			ID:           "test-openrouter",
			Name:         "OpenRouter",
			ProviderType: "openrouter",
			BaseURL:      "https://openrouter.ai/api/v1",
			APIKey:       testKey,
			IsActive:     &isTrue,
			IsDefault:    &isTrue,
		}
		body, _ := json.Marshal(dto)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/llm/providers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("Expected 401 Unauthorized, got: %d", w.Code)
		}
	})

	t.Run("Admin_CreateProvider_SuccessWithAdminToken", func(t *testing.T) {
		isTrue := true
		rawKey := os.Getenv("OPEN_ROUTER_API")
		if rawKey == "" {
			generated, err := crypto.GenerateRandomHex(24)
			if err != nil {
				t.Fatalf("Failed to generate test key: %v", err)
			}
			rawKey = generated
		}
		dto := request.CreateLLMProviderRequest{
			ID:           "test-openrouter",
			Name:         "OpenRouter",
			ProviderType: "openrouter",
			BaseURL:      "https://openrouter.ai/api/v1",
			APIKey:       rawKey,
			IsActive:     &isTrue,
			IsDefault:    &isTrue,
		}
		body, _ := json.Marshal(dto)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/llm/providers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201 Created, got: %d, body: %s", w.Code, w.Body.String())
		}

		var resp response.CommonResponse
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		dataBytes, _ := json.Marshal(resp.Data)
		var prov response.LLMProviderResponse
		_ = json.Unmarshal(dataBytes, &prov)

		if prov.MaskedAPIKey == rawKey {
			t.Fatalf("API key must be masked in response")
		}
	})

	t.Run("Admin_CreateModels_AndVerifyClientVisionFlags", func(t *testing.T) {
		isTrue := true
		isFalse := false

		lingDto := request.CreateLLMModelRequest{
			ID:             "ling-flash",
			ProviderID:     "test-openrouter",
			Name:           "Ling 3.1 Flash (Text Only)",
			ModelKey:       "inclusionai/ling-3.1-flash",
			ModelType:      "chat",
			IsActive:       &isTrue,
			IsDefault:      &isTrue,
			OrderIndex:     1,
			ContextLength:  32768,
			VisionMode:     "manual",
			SupportsVision: &isFalse,
			MaxImages:      0,
		}
		lingBody, _ := json.Marshal(lingDto)
		lingReq := httptest.NewRequest(http.MethodPost, "/api/admin/llm/models", bytes.NewReader(lingBody))
		lingReq.Header.Set("Content-Type", "application/json")
		lingReq.Header.Set("Authorization", "Bearer "+adminToken)
		lingW := httptest.NewRecorder()
		mux.ServeHTTP(lingW, lingReq)

		if lingW.Code != http.StatusCreated {
			t.Fatalf("Failed to create ling model: %d, body: %s", lingW.Code, lingW.Body.String())
		}

		geminiDto := request.CreateLLMModelRequest{
			ID:             "gemini-flash",
			ProviderID:     "test-openrouter",
			Name:           "Gemini 2.5 Flash (Vision)",
			ModelKey:       "google/gemini-2.5-flash",
			ModelType:      "chat",
			IsActive:       &isTrue,
			IsDefault:      &isFalse,
			OrderIndex:     2,
			ContextLength:  1048576,
			VisionMode:     "manual",
			SupportsVision: &isTrue,
			MaxImages:      5,
		}
		gemBody, _ := json.Marshal(geminiDto)
		gemReq := httptest.NewRequest(http.MethodPost, "/api/admin/llm/models", bytes.NewReader(gemBody))
		gemReq.Header.Set("Content-Type", "application/json")
		gemReq.Header.Set("Authorization", "Bearer "+adminToken)
		gemW := httptest.NewRecorder()
		mux.ServeHTTP(gemW, gemReq)

		if gemW.Code != http.StatusCreated {
			t.Fatalf("Failed to create gemini model: %d, body: %s", gemW.Code, gemW.Body.String())
		}

		clientReq := httptest.NewRequest(http.MethodGet, "/api/chat/models", nil)
		clientW := httptest.NewRecorder()
		mux.ServeHTTP(clientW, clientReq)

		if clientW.Code != http.StatusOK {
			t.Fatalf("Expected 200 for /api/chat/models, got: %d", clientW.Code)
		}

		var clientResp response.CommonResponse
		_ = json.Unmarshal(clientW.Body.Bytes(), &clientResp)
		dataBytes, _ := json.Marshal(clientResp.Data)
		var publicModels []map[string]any
		_ = json.Unmarshal(dataBytes, &publicModels)

		if len(publicModels) != 2 {
			t.Fatalf("Expected 2 chat models in anonymous client endpoint, got: %d", len(publicModels))
		}
		for _, m := range publicModels {
			for _, required := range []string{"id", "name", "context_length", "supports_vision"} {
				if _, ok := m[required]; !ok {
					t.Fatalf("Anonymous model payload missing required field %s: %v", required, m)
				}
			}
			for _, forbidden := range []string{"model_key", "provider_name", "max_images", "is_default"} {
				if _, ok := m[forbidden]; ok {
					t.Fatalf("Anonymous model payload must not expose %s: %v", forbidden, m)
				}
			}
			if m["id"] == "ling-flash" && m["supports_vision"] != false {
				t.Fatalf("Ling Flash must have supports_vision=false for anonymous callers, got: %v", m["supports_vision"])
			}
			if m["id"] == "gemini-flash" && m["supports_vision"] != true {
				t.Fatalf("Gemini Flash must have supports_vision=true for anonymous callers, got: %v", m["supports_vision"])
			}
		}

		authClientReq := httptest.NewRequest(http.MethodGet, "/api/chat/models", nil)
		authClientReq.Header.Set("Authorization", "Bearer "+adminToken)
		authClientW := httptest.NewRecorder()
		mux.ServeHTTP(authClientW, authClientReq)

		if authClientW.Code != http.StatusOK {
			t.Fatalf("Expected 200 for authenticated /api/chat/models, got: %d", authClientW.Code)
		}
		var authClientResp response.CommonResponse
		_ = json.Unmarshal(authClientW.Body.Bytes(), &authClientResp)
		authDataBytes, _ := json.Marshal(authClientResp.Data)
		var chatModels []response.ChatModelClientDTO
		_ = json.Unmarshal(authDataBytes, &chatModels)

		if len(chatModels) != 2 {
			t.Fatalf("Expected 2 chat models in authenticated client endpoint, got: %d", len(chatModels))
		}

		for _, m := range chatModels {
			if m.ID == "ling-flash" {
				if m.SupportsVision || m.MaxImages != 0 {
					t.Fatalf("Ling Flash must have supports_vision=false and max_images=0, got: vision=%v, max=%d", m.SupportsVision, m.MaxImages)
				}
			}
			if m.ID == "gemini-flash" {
				if !m.SupportsVision || m.MaxImages != 5 {
					t.Fatalf("Gemini Flash must have supports_vision=true and max_images=5, got: vision=%v, max=%d", m.SupportsVision, m.MaxImages)
				}
			}
		}
	})

	t.Run("LLM Chain Management Lifecycle", func(t *testing.T) {
		isTrue := true
		chainReqDto := request.CreateLLMChainRequest{
			ID:               "chat-fallback-chain",
			ChainType:        "chat",
			Name:             "Main Chat Fallback",
			Description:      "Sequential fallback for chat models",
			IsActive:         &isTrue,
			FailureThreshold: 3,
			CooldownSeconds:  300,
			RetryCount:       3,
		}
		chainBody, _ := json.Marshal(chainReqDto)
		chainReq := httptest.NewRequest(http.MethodPost, "/api/admin/llm/chains", bytes.NewReader(chainBody))
		chainReq.Header.Set("Content-Type", "application/json")
		chainReq.Header.Set("Authorization", "Bearer "+adminToken)
		chainW := httptest.NewRecorder()
		mux.ServeHTTP(chainW, chainReq)

		if chainW.Code != http.StatusCreated {
			t.Fatalf("Failed to create chain: %d, body: %s", chainW.Code, chainW.Body.String())
		}

		addNodeDto := request.AddChainNodeRequest{
			ModelID:  "ling-flash",
			Priority: 1,
			IsActive: &isTrue,
		}
		nodeBody, _ := json.Marshal(addNodeDto)
		nodeReq := httptest.NewRequest(http.MethodPost, "/api/admin/llm/chains/chat-fallback-chain/nodes", bytes.NewReader(nodeBody))
		nodeReq.Header.Set("Content-Type", "application/json")
		nodeReq.Header.Set("Authorization", "Bearer "+adminToken)
		nodeW := httptest.NewRecorder()
		mux.ServeHTTP(nodeW, nodeReq)

		if nodeW.Code != http.StatusCreated {
			t.Fatalf("Failed to add node to chain: %d, body: %s", nodeW.Code, nodeW.Body.String())
		}

		var addNodeResp response.CommonResponse
		_ = json.Unmarshal(nodeW.Body.Bytes(), &addNodeResp)
		nodeDataBytes, _ := json.Marshal(addNodeResp.Data)
		var createdNode response.LLMChainNodeResponse
		_ = json.Unmarshal(nodeDataBytes, &createdNode)

		if createdNode.HealthState != "closed" || createdNode.Priority != 1 {
			t.Fatalf("Expected health_state closed and priority 1, got: %+v", createdNode)
		}

		listReq := httptest.NewRequest(http.MethodGet, "/api/admin/llm/chains", nil)
		listReq.Header.Set("Authorization", "Bearer "+adminToken)
		listW := httptest.NewRecorder()
		mux.ServeHTTP(listW, listReq)

		if listW.Code != http.StatusOK {
			t.Fatalf("Failed to list chains: %d", listW.Code)
		}

		resetReq := httptest.NewRequest(http.MethodPost, "/api/admin/llm/chains/chat-fallback-chain/nodes/"+createdNode.ID+"/reset", nil)
		resetReq.Header.Set("Authorization", "Bearer "+adminToken)
		resetW := httptest.NewRecorder()
		mux.ServeHTTP(resetW, resetReq)

		if resetW.Code != http.StatusOK {
			t.Fatalf("Failed to reset node breaker: %d, body: %s", resetW.Code, resetW.Body.String())
		}

		delNodeReq := httptest.NewRequest(http.MethodDelete, "/api/admin/llm/chains/chat-fallback-chain/nodes/"+createdNode.ID, nil)
		delNodeReq.Header.Set("Authorization", "Bearer "+adminToken)
		delNodeW := httptest.NewRecorder()
		mux.ServeHTTP(delNodeW, delNodeReq)

		if delNodeW.Code != http.StatusOK {
			t.Fatalf("Failed to remove node from chain: %d, body: %s", delNodeW.Code, delNodeW.Body.String())
		}

		delChainReq := httptest.NewRequest(http.MethodDelete, "/api/admin/llm/chains/chat-fallback-chain", nil)
		delChainReq.Header.Set("Authorization", "Bearer "+adminToken)
		delChainW := httptest.NewRecorder()
		mux.ServeHTTP(delChainW, delChainReq)

		if delChainW.Code != http.StatusOK {
			t.Fatalf("Failed to delete chain: %d, body: %s", delChainW.Code, delChainW.Body.String())
		}
	})
}

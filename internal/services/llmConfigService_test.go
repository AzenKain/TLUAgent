package services_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/database"
	"tluagent-web/pkg/llm"
)

func TestLLMConfigService_ProviderAndModelLifecycle(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	ramCache := cache.NewTheineCache(10 << 20)
	repo := repositories.NewLLMRepository(db, ramCache)
	manager := llm.NewManager()
	svc := services.NewLLMConfigService(repo, manager)
	ctx := context.Background()

	rawKey := os.Getenv("OPEN_ROUTER_API")
	if rawKey == "" {
		generated, err := crypto.GenerateRandomHex(24)
		if err != nil {
			t.Fatalf("Failed to generate test key: %v", err)
		}
		rawKey = generated
	}

	isTrue := true
	provResp, err := svc.CreateProvider(ctx, request.CreateLLMProviderRequest{
		ID:           "test-openrouter",
		Name:         "OpenRouter Provider",
		ProviderType: "openrouter",
		BaseURL:      "https://openrouter.ai/api/v1",
		APIKey:       rawKey,
		IsActive:     &isTrue,
		IsDefault:    &isTrue,
	})
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}

	if provResp.MaskedAPIKey == rawKey {
		t.Fatalf("API key must be masked in response")
	}
	if provResp.TimeoutSeconds != 60 || provResp.MaxRetries != 3 {
		t.Fatalf("Expected default timeout 60 and max_retries 3, got timeout %d, retries %d", provResp.TimeoutSeconds, provResp.MaxRetries)
	}

	isFalse := false
	lingModel, err := svc.CreateModel(ctx, request.CreateLLMModelRequest{
		ID:             "ling-flash",
		ProviderID:     "test-openrouter",
		Name:           "Ling 3.1 Flash",
		ModelKey:       "inclusionai/ling-3.1-flash",
		ModelType:      "chat",
		IsActive:       &isTrue,
		IsDefault:      &isTrue,
		VisionMode:     "manual",
		SupportsVision: &isFalse,
		MaxImages:      0,
	})
	if err != nil {
		t.Fatalf("CreateModel failed: %v", err)
	}
	if lingModel.SupportsVision || lingModel.MaxImages != 0 {
		t.Fatalf("Expected text-only model")
	}

	geminiModel, err := svc.CreateModel(ctx, request.CreateLLMModelRequest{
		ID:             "gemini-flash",
		ProviderID:     "test-openrouter",
		Name:           "Gemini 2.5 Flash",
		ModelKey:       "google/gemini-2.5-flash",
		ModelType:      "chat",
		IsActive:       &isTrue,
		IsDefault:      &isFalse,
		VisionMode:     "manual",
		SupportsVision: &isTrue,
		MaxImages:      5,
	})
	if err != nil {
		t.Fatalf("Create vision model failed: %v", err)
	}
	if !geminiModel.SupportsVision || geminiModel.MaxImages != 5 {
		t.Fatalf("Expected vision model with max 5 images")
	}

	clientModels, err := svc.GetActiveChatModels(ctx)
	if err != nil || len(clientModels) != 2 {
		t.Fatalf("Expected 2 client chat models, got %d, err: %v", len(clientModels), err)
	}

	var foundTextOnly, foundVision bool
	for _, m := range clientModels {
		if m.ModelKey == "inclusionai/ling-3.1-flash" {
			foundTextOnly = true
			if m.SupportsVision || m.MaxImages != 0 {
				t.Fatalf("Client DTO for ling-flash should have supports_vision=false and max_images=0")
			}
		}
		if m.ModelKey == "google/gemini-2.5-flash" {
			foundVision = true
			if !m.SupportsVision || m.MaxImages != 5 {
				t.Fatalf("Client DTO for gemini-flash should have supports_vision=true and max_images=5")
			}
		}
	}

	if !foundTextOnly || !foundVision {
		t.Fatalf("Missing expected models in client list")
	}

	cap := manager.Registry().DetectCapabilities(ctx, "inclusionai/ling-3.1-flash")
	if cap.SupportsVision {
		t.Fatalf("Manager registry should have synced supports_vision=false for ling-flash")
	}

	capGemini := manager.Registry().DetectCapabilities(ctx, "google/gemini-2.5-flash")
	if !capGemini.SupportsVision || capGemini.MaxImages != 5 {
		t.Fatalf("Manager registry should have synced supports_vision=true and max_images=5 for gemini-flash")
	}
}

func TestLLMConfigService_ChainsAndEmbeddingHomogeneity(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	ramCache := cache.NewTheineCache(10 << 20)
	repo := repositories.NewLLMRepository(db, ramCache)
	manager := llm.NewManager()
	svc := services.NewLLMConfigService(repo, manager)
	ctx := context.Background()

	testKey1, err := crypto.GenerateRandomHex(24)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}
	testKey2, err := crypto.GenerateRandomHex(24)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	isTrue := true
	isFalse := false

	prov1, err := svc.CreateProvider(ctx, request.CreateLLMProviderRequest{
		ID:           "prov-openai",
		Name:         "OpenAI Primary",
		ProviderType: "openai",
		BaseURL:      "https://api.openai.com/v1",
		APIKey:       testKey1,
		IsActive:     &isTrue,
	})
	if err != nil {
		t.Fatalf("CreateProvider prov-openai failed: %v", err)
	}

	prov2, err := svc.CreateProvider(ctx, request.CreateLLMProviderRequest{
		ID:           "prov-azure",
		Name:         "Azure Secondary",
		ProviderType: "openai",
		BaseURL:      "https://azure-openai.azure.com/v1",
		APIKey:       testKey2,
		IsActive:     &isTrue,
	})
	if err != nil {
		t.Fatalf("CreateProvider prov-azure failed: %v", err)
	}

	embModel1, err := svc.CreateModel(ctx, request.CreateLLMModelRequest{
		ID:             "emb-openai-small",
		ProviderID:     prov1.ID,
		Name:           "Text Embedding 3 Small (OpenAI)",
		ModelKey:       "text-embedding-3-small",
		ModelType:      "embedding",
		IsActive:       &isTrue,
		VisionMode:     "manual",
		SupportsVision: &isFalse,
	})
	if err != nil {
		t.Fatalf("CreateModel emb-openai-small failed: %v", err)
	}

	embModel2, err := svc.CreateModel(ctx, request.CreateLLMModelRequest{
		ID:             "emb-azure-small",
		ProviderID:     prov2.ID,
		Name:           "Text Embedding 3 Small (Azure)",
		ModelKey:       "text-embedding-3-small",
		ModelType:      "embedding",
		IsActive:       &isTrue,
		VisionMode:     "manual",
		SupportsVision: &isFalse,
	})
	if err != nil {
		t.Fatalf("CreateModel emb-azure-small failed: %v", err)
	}

	diffEmbModel, err := svc.CreateModel(ctx, request.CreateLLMModelRequest{
		ID:             "emb-qwen-diff",
		ProviderID:     prov1.ID,
		Name:           "Qwen Embedding",
		ModelKey:       "qwen/qwen-2.5-embedding",
		ModelType:      "embedding",
		IsActive:       &isTrue,
		VisionMode:     "manual",
		SupportsVision: &isFalse,
	})
	if err != nil {
		t.Fatalf("CreateModel emb-qwen-diff failed: %v", err)
	}

	embChain, err := svc.CreateChain(ctx, request.CreateLLMChainRequest{
		ID:               "emb-chain-prod",
		ChainType:        "embedding",
		Name:             "Production Embedding Fallback",
		Description:      "Multi-provider fallback for text-embedding-3-small",
		IsActive:         &isTrue,
		FailureThreshold: 3,
		CooldownSeconds:  120,
		RetryCount:       3,
	})
	if err != nil {
		t.Fatalf("CreateChain failed: %v", err)
	}

	node1, err := svc.AddChainNode(ctx, embChain.ID, request.AddChainNodeRequest{
		ModelID:  embModel1.ID,
		Priority: 1,
		IsActive: &isTrue,
	})
	if err != nil {
		t.Fatalf("AddChainNode node1 failed: %v", err)
	}
	if node1.HealthState != "closed" || node1.Priority != 1 {
		t.Fatalf("Unexpected node1 state: %+v", node1)
	}

	_, err = svc.AddChainNode(ctx, embChain.ID, request.AddChainNodeRequest{
		ModelID:  diffEmbModel.ID,
		Priority: 2,
		IsActive: &isTrue,
	})
	if err == nil {
		t.Fatalf("Expected error when adding mismatched embedding model to embedding chain, but got nil")
	}

	node2, err := svc.AddChainNode(ctx, embChain.ID, request.AddChainNodeRequest{
		ModelID:  embModel2.ID,
		Priority: 2,
		IsActive: &isTrue,
	})
	if err != nil {
		t.Fatalf("AddChainNode node2 with matching model key should succeed, got: %v", err)
	}
	if node2.Priority != 2 {
		t.Fatalf("Expected node2 priority 2, got %d", node2.Priority)
	}

	chainDetail, err := svc.GetChain(ctx, embChain.ID)
	if err != nil {
		t.Fatalf("GetChain failed: %v", err)
	}
	if len(chainDetail.Nodes) != 2 {
		t.Fatalf("Expected 2 nodes in chain, got %d", len(chainDetail.Nodes))
	}

	if err := svc.ResetNodeCircuitBreaker(ctx, embChain.ID, node1.ID); err != nil {
		t.Fatalf("ResetNodeCircuitBreaker failed: %v", err)
	}
}

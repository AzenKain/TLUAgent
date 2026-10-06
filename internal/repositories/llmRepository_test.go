package repositories_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/database"
)

func TestLLMRepository_CRUDAndCache(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	ramCache := cache.NewTheineCache(10 << 20)
	repo := repositories.NewLLMRepository(db, ramCache)
	ctx := context.Background()

	mockCipher, err := crypto.GenerateRandomHex(32)
	if err != nil {
		t.Fatalf("Failed to generate mock cipher: %v", err)
	}

	createdProv, err := repo.CreateProvider(ctx, sqlc.CreateLLMProviderParams{
		ID:                   "openrouter",
		Name:                 "OpenRouter",
		ProviderType:         "openai",
		BaseUrl:              "https://openrouter.ai/api/v1",
		ApiKeyCiphertext:     mockCipher,
		IsActive:             1,
		IsDefault:            1,
		CustomHeadersJson:    "{}",
		TimeoutSeconds:       60,
		MaxRetries:           3,
		RetryInitialWaitMs:   500,
		RetryMaxWaitMs:       5000,
		AllowPrivateNetworks: 0,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	if createdProv.ID != "openrouter" {
		t.Fatalf("Expected openrouter, got: %s", createdProv.ID)
	}

	provs, err := repo.ListProviders(ctx)
	if err != nil || len(provs) != 1 {
		t.Fatalf("Expected 1 provider, got %d, err: %v", len(provs), err)
	}

	createdModel, err := repo.CreateModel(ctx, sqlc.CreateLLMModelParams{
		ID:             "ling-flash",
		ProviderID:     "openrouter",
		Name:           "Ling 3.1 Flash",
		ModelKey:       "inclusionai/ling-3.1-flash",
		ModelType:      "chat",
		IsActive:       1,
		IsDefault:      1,
		OrderIndex:     1,
		ContextLength:  32768,
		VisionMode:     "manual",
		SupportsVision: 0,
		MaxImages:      0,
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	if createdModel.SupportsVision != 0 || createdModel.MaxImages != 0 {
		t.Fatalf("Expected text-only model with 0 max images")
	}

	visionModel, err := repo.CreateModel(ctx, sqlc.CreateLLMModelParams{
		ID:             "gemini-flash",
		ProviderID:     "openrouter",
		Name:           "Gemini 2.5 Flash",
		ModelKey:       "google/gemini-2.5-flash",
		ModelType:      "chat",
		IsActive:       1,
		IsDefault:      0,
		OrderIndex:     2,
		ContextLength:  1048576,
		VisionMode:     "auto",
		SupportsVision: 1,
		MaxImages:      5,
	})
	if err != nil {
		t.Fatalf("Failed to create vision model: %v", err)
	}
	if visionModel.SupportsVision != 1 || visionModel.MaxImages != 5 {
		t.Fatalf("Expected vision model with max 5 images")
	}

	chatModels, err := repo.ListActiveChatModelsWithProvider(ctx)
	if err != nil || len(chatModels) != 2 {
		t.Fatalf("Expected 2 active chat models, got %d, err: %v", len(chatModels), err)
	}

	modelWithProv, err := repo.GetModelWithProvider(ctx, "ling-flash")
	if err != nil || modelWithProv.ProviderName != "OpenRouter" {
		t.Fatalf("Failed to get model with provider: %v", err)
	}

	updated, err := repo.UpdateModelVisionCapability(ctx, sqlc.UpdateModelVisionCapabilityParams{
		ID:             "ling-flash",
		VisionMode:     "manual",
		SupportsVision: 1,
		MaxImages:      2,
	})
	if err != nil || updated.SupportsVision != 1 || updated.MaxImages != 2 {
		t.Fatalf("Failed to update vision capability: %v", err)
	}
}

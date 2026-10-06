package repositories_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func TestSettingsRepository_CacheAndInvalidation(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	ramCache := cache.NewTheineCache(10 << 20)
	repo := repositories.NewSettingsRepository(db, ramCache)
	ctx := context.Background()

	_, err = repo.GetSetupState(ctx, "root_admin_id")
	if err == nil {
		t.Fatalf("Expected error for non-existent setup state key")
	}

	if err := repo.UpsertSetupState(ctx, "root_admin_id", "admin-123"); err != nil {
		t.Fatalf("Failed to upsert setup state: %v", err)
	}

	val1, err := repo.GetSetupState(ctx, "root_admin_id")
	if err != nil || val1 != "admin-123" {
		t.Fatalf("Expected admin-123, got: %s (err: %v)", val1, err)
	}

	var cachedState string
	cacheKey := cache.BuildKey("setup_state", "key", "root_admin_id")
	if err := ramCache.Get(ctx, cacheKey, &cachedState); err != nil || cachedState != "admin-123" {
		t.Fatalf("Setup state was not cached: %s (err: %v)", cachedState, err)
	}

	if err := repo.UpsertSetupState(ctx, "root_admin_id", "admin-456"); err != nil {
		t.Fatalf("Failed to update setup state: %v", err)
	}

	if err := ramCache.Get(ctx, cacheKey, &cachedState); err == nil {
		t.Fatalf("Expected cache to be invalidated after upsert, but found: %s", cachedState)
	}

	val2, err := repo.GetSetupState(ctx, "root_admin_id")
	if err != nil || val2 != "admin-456" {
		t.Fatalf("Expected admin-456, got: %s (err: %v)", val2, err)
	}

	settingVal := `{"ai_provider":"gemini","model":"gemini-1.5-flash"}`
	if err := repo.UpsertAppSetting(ctx, "ai_config", settingVal); err != nil {
		t.Fatalf("Failed to upsert app setting: %v", err)
	}

	setting1, err := repo.GetAppSetting(ctx, "ai_config")
	if err != nil || setting1 != settingVal {
		t.Fatalf("Expected %s, got: %s (err: %v)", settingVal, setting1, err)
	}

	var cachedSetting string
	settingCacheKey := cache.BuildKey("settings", "key", "ai_config")
	if err := ramCache.Get(ctx, settingCacheKey, &cachedSetting); err != nil || cachedSetting != settingVal {
		t.Fatalf("App setting was not cached: %s (err: %v)", cachedSetting, err)
	}

	newSettingVal := `{"ai_provider":"openai","model":"gpt-4o"}`
	if err := repo.UpsertAppSetting(ctx, "ai_config", newSettingVal); err != nil {
		t.Fatalf("Failed to update app setting: %v", err)
	}

	if err := ramCache.Get(ctx, settingCacheKey, &cachedSetting); err == nil {
		t.Fatalf("Expected cache to be invalidated after upsert, but found: %s", cachedSetting)
	}

	setting2, err := repo.GetAppSetting(ctx, "ai_config")
	if err != nil || setting2 != newSettingVal {
		t.Fatalf("Expected %s, got: %s (err: %v)", newSettingVal, setting2, err)
	}
}

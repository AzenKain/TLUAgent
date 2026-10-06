package services_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
	"tluagent-web/pkg/memory"
)

func TestAgentService_Flow(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	ramCache := cache.NewTheineCache(10 << 20)
	repo := repositories.NewAgentRepository(db, ramCache)
	svc := services.NewAgentService(repo)
	ctx := context.Background()

	if err := svc.SeedDefaultsIfEmpty(ctx); err != nil {
		t.Fatalf("Failed to seed defaults: %v", err)
	}

	prompts, err := svc.GetPrompts(ctx)
	if err != nil || len(prompts) < 2 {
		t.Fatalf("Expected at least 2 seeded prompts, got: %d (err: %v)", len(prompts), err)
	}

	skills, err := svc.GetSkills(ctx)
	if err != nil || len(skills) < 5 {
		t.Fatalf("Expected at least 5 seeded skills, got: %d (err: %v)", len(skills), err)
	}

	updatedPrompt, err := svc.UpdatePrompt(ctx, "soul", "Custom Title", "Custom Content", "admin-1")
	if err != nil || updatedPrompt.Content != "Custom Content" {
		t.Fatalf("Failed to update prompt: %+v (err: %v)", updatedPrompt, err)
	}

	resetPrompt, err := svc.ResetPrompt(ctx, "soul", "admin-1")
	if err != nil || resetPrompt.Content == "Custom Content" {
		t.Fatalf("Expected reset prompt to restore original content, got: %s", resetPrompt.Content)
	}

	if err := svc.ToggleSkill(ctx, "tuition_calculation", false, "admin-1"); err != nil {
		t.Fatalf("Failed to toggle skill: %v", err)
	}

	gotSkill, err := svc.GetSkill(ctx, "tuition_calculation")
	if err != nil || gotSkill.IsEnabled {
		t.Fatalf("Expected skill to be disabled, got: %+v", gotSkill)
	}

	profile := &memory.StudentAcademicProfile{
		UserID:    "u1",
		StudentID: "A36001",
		FullName:  "Test Student",
	}
	ragItems := []*models.RAGSearchResultItem{
		{
			DocTitle: "Doc 1",
			Content:  "Content 1",
		},
	}

	promptStr, sources, err := svc.BuildSystemPrompt(ctx, profile, ragItems)
	if err != nil {
		t.Fatalf("Failed to build system prompt: %v", err)
	}
	if !strings.Contains(promptStr, "TLUAgent") {
		t.Fatalf("Expected prompt to contain soul information")
	}
	if !strings.Contains(promptStr, "A36001") {
		t.Fatalf("Expected prompt to contain student profile")
	}
	if !strings.Contains(promptStr, "Doc 1") {
		t.Fatalf("Expected prompt to contain RAG document")
	}
	if len(sources) != 1 || sources[0] != "Doc 1" {
		t.Fatalf("Expected Doc 1 in sources, got: %+v", sources)
	}

	preview, err := svc.PreviewSystemPrompt(ctx, "K36", true)
	if err != nil || !strings.Contains(preview, "K36") {
		t.Fatalf("Failed to generate preview: %v", err)
	}

	settingsRepo := repositories.NewSettingsRepository(db, ramCache)
	svc.SetSettingsRepository(settingsRepo)

	defaultCfg, err := svc.GetCompactorSettings(ctx)
	if err != nil || defaultCfg.MaxContextTokens != 250000 {
		t.Fatalf("Expected default max context tokens 250000, got %v", defaultCfg)
	}

	updatedCfg, err := svc.UpdateCompactorSettings(ctx, &request.UpdateCompactorSettingsRequest{
		MaxContextTokens:      128000,
		CompactThresholdRatio: 0.75,
		KeepRecentTurns:       8,
	})
	if err != nil || updatedCfg.MaxContextTokens != 128000 || updatedCfg.KeepRecentTurns != 8 {
		t.Fatalf("Failed to update compactor settings: %+v", updatedCfg)
	}

	refetchedCfg, err := svc.GetCompactorSettings(ctx)
	if err != nil || refetchedCfg.MaxContextTokens != 128000 {
		t.Fatalf("Expected refetched tokens 128000, got %+v", refetchedCfg)
	}
}


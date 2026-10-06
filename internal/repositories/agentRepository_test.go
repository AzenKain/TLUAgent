package repositories_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func TestAgentRepository_PromptsAndSkills(t *testing.T) {
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
	ctx := context.Background()

	prompt, err := repo.GetPrompt(ctx, "soul")
	if err != nil {
		t.Fatalf("Unexpected error querying missing prompt: %v", err)
	}
	if prompt != nil {
		t.Fatalf("Expected nil prompt, got: %+v", prompt)
	}

	p1 := &models.AgentPrompt{
		ID:       "soul",
		Title:    "TLU Advisor Soul",
		Content:  "You are TLU academic advisor.",
		IsActive: true,
	}
	if err := repo.UpsertPrompt(ctx, p1); err != nil {
		t.Fatalf("Failed to upsert prompt: %v", err)
	}

	gotP1, err := repo.GetPrompt(ctx, "soul")
	if err != nil || gotP1 == nil || gotP1.Content != p1.Content {
		t.Fatalf("Failed to get cached prompt: %+v (err: %v)", gotP1, err)
	}

	prompts, err := repo.ListPrompts(ctx)
	if err != nil || len(prompts) != 1 {
		t.Fatalf("Expected 1 prompt, got %d (err: %v)", len(prompts), err)
	}

	s1 := &models.AgentSkill{
		ID:          "calc_tuition",
		Name:        "Tuition Calculation",
		Description: "Calculate tuition per credit",
		Content:     "Rules for tuition calculation",
		IsEnabled:   true,
		Priority:    10,
	}
	if err := repo.UpsertSkill(ctx, s1); err != nil {
		t.Fatalf("Failed to upsert skill: %v", err)
	}

	gotS1, err := repo.GetSkill(ctx, "calc_tuition")
	if err != nil || gotS1 == nil || gotS1.Name != s1.Name {
		t.Fatalf("Failed to get skill: %+v (err: %v)", gotS1, err)
	}

	skills, err := repo.ListEnabledSkills(ctx)
	if err != nil || len(skills) != 1 {
		t.Fatalf("Expected 1 enabled skill, got %d (err: %v)", len(skills), err)
	}

	if err := repo.UpdateSkillEnabled(ctx, "calc_tuition", false, "admin-1"); err != nil {
		t.Fatalf("Failed to toggle skill: %v", err)
	}

	skills, err = repo.ListEnabledSkills(ctx)
	if err != nil || len(skills) != 0 {
		t.Fatalf("Expected 0 enabled skills after toggle, got %d", len(skills))
	}

	if err := repo.DeleteSkill(ctx, "calc_tuition"); err != nil {
		t.Fatalf("Failed to delete skill: %v", err)
	}

	deletedS1, err := repo.GetSkill(ctx, "calc_tuition")
	if err != nil || deletedS1 != nil {
		t.Fatalf("Expected nil after deletion, got: %+v", deletedS1)
	}
}

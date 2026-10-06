package services_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func TestMemoryService_Flow(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	ramCache := cache.NewTheineCache(10 << 20)
	userRepo := repositories.NewUserRepository(db, ramCache)
	memRepo := repositories.NewMemoryRepository(db, ramCache)
	svc := services.NewMemoryService(memRepo, userRepo)
	ctx := context.Background()

	testUserID := "student-uuid-456"
	_, err = userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           testUserID,
		Email:        "student456@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Tran Van B", Valid: true},
		StudentCode:  sql.NullString{String: "B45678", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	emptyContext, err := svc.BuildLongTermMemoryPromptContext(ctx, "unknown-user-999")
	if err != nil || emptyContext != "" {
		t.Fatalf("Expected empty prompt context for unknown user, got: %s", emptyContext)
	}

	initialContext, err := svc.BuildLongTermMemoryPromptContext(ctx, testUserID)
	if err != nil || !strings.Contains(initialContext, "NORMAL") {
		t.Fatalf("Expected auto-initialized profile context for known user, got: %s", initialContext)
	}

	profile := &models.StudentProfile{
		UserID:           testUserID,
		Major:            "Information Technology",
		Cohort:           "K36",
		AcademicStanding: "Good",
		CompletedCredits: 110,
		CumulativeGPA:    3.42,
		TargetGPA:        3.60,
		AdvisorNotes:     "Focusing on graduation thesis in Spring semester",
		UpdatedAt:        time.Now(),
	}

	savedProfile, err := svc.UpsertStudentProfile(ctx, profile)
	if err != nil || savedProfile == nil {
		t.Fatalf("Failed to upsert student profile: %v", err)
	}

	fetchedProfile, err := svc.GetStudentProfile(ctx, testUserID)
	if err != nil || fetchedProfile.Cohort != "K36" {
		t.Fatalf("Failed to retrieve profile: %+v", fetchedProfile)
	}

	mem := &models.UserMemoryItem{
		UserID:      testUserID,
		Category:    "ACADEMIC_GOAL",
		MemoryKey:   "graduation_timeline",
		MemoryValue: "Expects to defend graduation thesis in May 2026",
		Confidence:  0.95,
	}

	savedMem, err := svc.SaveUserMemory(ctx, mem)
	if err != nil || savedMem.ID == "" {
		t.Fatalf("Failed to save user memory: %v", err)
	}

	memories, err := svc.ListUserMemories(ctx, testUserID)
	if err != nil || len(memories) != 1 {
		t.Fatalf("Expected 1 memory item, got %d (err: %v)", len(memories), err)
	}

	promptContext, err := svc.BuildLongTermMemoryPromptContext(ctx, testUserID)
	if err != nil {
		t.Fatalf("Failed to build prompt context: %v", err)
	}
	if !strings.Contains(promptContext, "K36") || !strings.Contains(promptContext, "Information Technology") {
		t.Fatalf("Prompt context missing profile data: %s", promptContext)
	}
	if !strings.Contains(promptContext, "graduation_timeline") {
		t.Fatalf("Prompt context missing memory facts: %s", promptContext)
	}

	if err := svc.DeleteUserMemory(ctx, savedMem.ID, testUserID); err != nil {
		t.Fatalf("Failed to delete user memory: %v", err)
	}

	memoriesAfterDel, err := svc.ListUserMemories(ctx, testUserID)
	if err != nil || len(memoriesAfterDel) != 0 {
		t.Fatalf("Expected 0 memories after deletion, got: %d", len(memoriesAfterDel))
	}
}

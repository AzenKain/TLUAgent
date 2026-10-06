package repositories_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func TestMemoryRepository_ProfileAndMemories(t *testing.T) {
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
	repo := repositories.NewMemoryRepository(db, ramCache)
	ctx := context.Background()

	testUserID := "user-test-123"
	_, err = userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           testUserID,
		Email:        "svtest@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Nguyen Van A", Valid: true},
		StudentCode:  sql.NullString{String: "A12345", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	profile, err := repo.GetStudentProfile(ctx, testUserID)
	if err != nil {
		t.Fatalf("Unexpected error querying missing profile: %v", err)
	}
	if profile != nil {
		t.Fatalf("Expected nil profile for new user, got: %+v", profile)
	}

	newProfile := &models.StudentProfile{
		UserID:           testUserID,
		Major:            "Information Technology",
		Cohort:           "K63",
		AcademicStanding: "Good",
		CompletedCredits: 98,
		CumulativeGPA:    3.65,
		TargetGPA:        3.80,
		AdvisorNotes:     "Prepares for graduation project next semester",
		UpdatedAt:        time.Now(),
	}

	savedProfile, err := repo.UpsertStudentProfile(ctx, newProfile)
	if err != nil {
		t.Fatalf("Failed to upsert student profile: %v", err)
	}
	if savedProfile == nil || savedProfile.Major != "Information Technology" {
		t.Fatalf("Upsert returned unexpected profile: %+v", savedProfile)
	}

	gotProfile, err := repo.GetStudentProfile(ctx, testUserID)
	if err != nil || gotProfile == nil {
		t.Fatalf("Failed to fetch cached profile: %+v (err: %v)", gotProfile, err)
	}
	if gotProfile.Cohort != "K63" || gotProfile.CumulativeGPA != 3.65 {
		t.Fatalf("Profile data mismatch: %+v", gotProfile)
	}

	mem1 := &models.UserMemoryItem{
		ID:          "mem-1",
		UserID:      testUserID,
		Category:    "academic_goal",
		MemoryKey:   "target_graduate_term",
		MemoryValue: "Target to graduate by Summer 2025 with distinction",
		Confidence:  0.95,
		Source:      "chat",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mem2 := &models.UserMemoryItem{
		ID:          "mem-2",
		UserID:      testUserID,
		Category:    "elective_preference",
		MemoryKey:   "elective_interest",
		MemoryValue: "Interested in Cloud Computing and Machine Learning courses",
		Confidence:  0.88,
		Source:      "chat",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if _, err := repo.UpsertUserMemory(ctx, mem1); err != nil {
		t.Fatalf("Failed to upsert memory 1: %v", err)
	}
	if _, err := repo.UpsertUserMemory(ctx, mem2); err != nil {
		t.Fatalf("Failed to upsert memory 2: %v", err)
	}

	memories, err := repo.ListUserMemories(ctx, testUserID)
	if err != nil {
		t.Fatalf("Failed to list user memories: %v", err)
	}
	if len(memories) != 2 {
		t.Fatalf("Expected 2 memories, got %d", len(memories))
	}

	count, err := repo.CountUserMemories(ctx, testUserID)
	if err != nil {
		t.Fatalf("Failed to count memories: %v", err)
	}
	if count != 2 {
		t.Fatalf("Expected memory count 2, got %d", count)
	}

	cachedMem1, err := repo.GetUserMemoryByID(ctx, "mem-1")
	if err != nil || cachedMem1 == nil {
		t.Fatalf("Failed to get memory by id: %+v (err: %v)", cachedMem1, err)
	}
	if cachedMem1.MemoryKey != "target_graduate_term" {
		t.Fatalf("MemoryKey mismatch: %+v", cachedMem1)
	}

	if err := repo.DeleteUserMemory(ctx, "mem-1", testUserID); err != nil {
		t.Fatalf("Failed to delete memory: %v", err)
	}

	memoriesAfterDelete, err := repo.ListUserMemories(ctx, testUserID)
	if err != nil {
		t.Fatalf("Failed to list memories after delete: %v", err)
	}
	if len(memoriesAfterDelete) != 1 {
		t.Fatalf("Expected 1 memory after delete, got %d", len(memoriesAfterDelete))
	}

	deletedMem, err := repo.GetUserMemoryByID(ctx, "mem-1")
	if err != nil {
		t.Fatalf("Unexpected error fetching deleted memory: %v", err)
	}
	if deletedMem != nil {
		t.Fatalf("Expected nil for deleted memory, got: %+v", deletedMem)
	}
}

package services_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

// TestTranscriptService verifies transcript summary calculation, diff analysis, and batch persistence.
func TestTranscriptService(t *testing.T) {
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
	transcriptRepo := repositories.NewTranscriptRepository(db, ramCache)
	memoryRepo := repositories.NewMemoryRepository(db, ramCache)
	transcriptSvc := services.NewTranscriptService(transcriptRepo, memoryRepo, nil)
	ctx := context.Background()

	testUser, err := userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           "test-transcript-user",
		Email:        "student_trans@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Đỗ Duy Khánh", Valid: true},
		StudentCode:  sql.NullString{String: "A44519", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	userID := testUser.ID

	initialBatch := []*models.StudentCourseGradeRecord{
		{
			CourseCode:  "CS100",
			CourseName:  "Tin đại cương",
			Credits:     2.0,
			Score10:     9.8,
			Score4:      4.0,
			GradeLetter: "A+",
			IsPassed:    true,
			TermTaken:   "HK01-2022",
		},
		{
			CourseCode:  "VL101",
			CourseName:  "Tiếng Việt thực hành",
			Credits:     2.0,
			Score10:     0.0,
			Score4:      0.0,
			GradeLetter: "F",
			IsPassed:    false,
			TermTaken:   "HK01-2022",
		},
		{
			CourseCode:  "MA101",
			CourseName:  "Logic toán",
			Credits:     3.0,
			Score10:     9.5,
			Score4:      4.0,
			GradeLetter: "A+",
			IsPassed:    true,
			TermTaken:   "HK01-2022",
		},
	}

	summary, err := transcriptSvc.BatchSaveTranscript(ctx, userID, initialBatch)
	if err != nil {
		t.Fatalf("Initial BatchSaveTranscript failed: %v", err)
	}
	if summary.PassedCoursesCount != 2 || summary.FailedCoursesCount != 1 {
		t.Fatalf("Expected 2 passed, 1 failed, got %d passed, %d failed", summary.PassedCoursesCount, summary.FailedCoursesCount)
	}
	if summary.TotalCreditsCompleted != 5.0 {
		t.Fatalf("Expected 5.0 completed credits, got %f", summary.TotalCreditsCompleted)
	}

	secondBatch := []*models.StudentCourseGradeRecord{
		{
			CourseCode:  "VL101",
			CourseName:  "Tiếng Việt thực hành",
			Credits:     2.0,
			Score10:     7.7,
			Score4:      3.0,
			GradeLetter: "B",
			IsPassed:    true,
			TermTaken:   "HK03-2023",
		},
		{
			CourseCode:  "CS122",
			CourseName:  "Lập trình hướng đối tượng",
			Credits:     3.0,
			Score10:     8.6,
			Score4:      4.0,
			GradeLetter: "A",
			IsPassed:    true,
			TermTaken:   "HK03-2023",
		},
	}

	diff, err := transcriptSvc.AnalyzeTranscriptDiff(ctx, userID, secondBatch)
	if err != nil {
		t.Fatalf("AnalyzeTranscriptDiff failed: %v", err)
	}
	if !diff.HasChanges {
		t.Fatalf("Expected diff to have changes")
	}
	if diff.UpgradedCoursesCount != 1 {
		t.Fatalf("Expected 1 upgraded course (VL101 F -> B), got %d", diff.UpgradedCoursesCount)
	}
	if diff.NewCoursesCount != 1 {
		t.Fatalf("Expected 1 new course (CS122), got %d", diff.NewCoursesCount)
	}

	updatedSummary, err := transcriptSvc.BatchSaveTranscript(ctx, userID, secondBatch)
	if err != nil {
		t.Fatalf("Second BatchSaveTranscript failed: %v", err)
	}
	if updatedSummary.FailedCoursesCount != 0 {
		t.Fatalf("Expected 0 failed courses after upgrade, got %d", updatedSummary.FailedCoursesCount)
	}
	if updatedSummary.TotalCreditsCompleted != 10.0 {
		t.Fatalf("Expected 10.0 completed credits (2+2+3+3), got %f", updatedSummary.TotalCreditsCompleted)
	}
}

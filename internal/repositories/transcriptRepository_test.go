package repositories_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

// TestTranscriptRepository verifies Cache IDs pattern, batch transactional upsert, and cache invalidation.
func TestTranscriptRepository(t *testing.T) {
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
	repo := repositories.NewTranscriptRepository(db, ramCache)
	ctx := context.Background()

	testUser, err := userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           "test-student-1",
		Email:        "student1@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Đỗ Duy Khánh", Valid: true},
		StudentCode:  sql.NullString{String: "A44519", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	testUserID := testUser.ID

	t.Run("BatchUpsertAndCacheIDsPattern", func(t *testing.T) {
		params := []sqlc.UpsertTranscriptGradeParams{
			{
				ID:          "tg_1",
				UserID:      testUserID,
				CourseCode:  "CS100",
				CourseName:  "Tin đại cương",
				Credits:     2.0,
				Score10:     9.8,
				Score4:      4.0,
				GradeLetter: "A+",
				IsPassed:    1,
				IsRetake:    0,
				TermTaken:   "HK01-2022",
			},
			{
				ID:          "tg_2",
				UserID:      testUserID,
				CourseCode:  "VL101",
				CourseName:  "Tiếng Việt thực hành",
				Credits:     2.0,
				Score10:     0.0,
				Score4:      0.0,
				GradeLetter: "F",
				IsPassed:    0,
				IsRetake:    0,
				TermTaken:   "HK01-2022",
			},
		}

		results, err := repo.BatchUpsert(ctx, testUserID, params)
		if err != nil {
			t.Fatalf("BatchUpsert failed: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("Expected 2 inserted items, got %d", len(results))
		}

		list, err := repo.ListByUserID(ctx, testUserID)
		if err != nil {
			t.Fatalf("ListByUserID failed: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("Expected 2 records from Cache IDs resolution, got %d", len(list))
		}

		cachedList, err := repo.ListByUserID(ctx, testUserID)
		if err != nil {
			t.Fatalf("Second ListByUserID failed: %v", err)
		}
		if len(cachedList) != 2 {
			t.Fatalf("Expected 2 cached records, got %d", len(cachedList))
		}

		updateParams := []sqlc.UpsertTranscriptGradeParams{
			{
				ID:          "tg_2",
				UserID:      testUserID,
				CourseCode:  "VL101",
				CourseName:  "Tiếng Việt thực hành",
				Credits:     2.0,
				Score10:     7.7,
				Score4:      3.0,
				GradeLetter: "B",
				IsPassed:    1,
				IsRetake:    1,
				TermTaken:   "HK03-2023",
			},
		}

		updatedResults, err := repo.BatchUpsert(ctx, testUserID, updateParams)
		if err != nil {
			t.Fatalf("BatchUpsert update failed: %v", err)
		}
		if len(updatedResults) != 1 {
			t.Fatalf("Expected 1 updated record, got %d", len(updatedResults))
		}

		refreshed, err := repo.GetByUserAndCourse(ctx, testUserID, "VL101")
		if err != nil {
			t.Fatalf("GetByUserAndCourse failed: %v", err)
		}
		if refreshed.Score10 != 7.7 || refreshed.GradeLetter != "B" || !refreshed.IsPassed {
			t.Fatalf("Expected upgraded grade B (7.7), got %s (%f)", refreshed.GradeLetter, refreshed.Score10)
		}

		cnt, err := repo.CountByUserID(ctx, testUserID)
		if err != nil || cnt != 2 {
			t.Fatalf("Expected count 2, got %d (err: %v)", cnt, err)
		}
	})
}

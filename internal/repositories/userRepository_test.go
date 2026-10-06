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

func TestUserRepository_CacheAndSingleflight(t *testing.T) {
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
	ctx := context.Background()

	u1, err := userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           "user-alpha-1",
		Email:        "alpha1@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Alpha One", Valid: true},
		StudentCode:  sql.NullString{String: "A00001", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create user 1: %v", err)
	}

	u2, err := userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           "user-beta-2",
		Email:        "beta2@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Beta Two", Valid: true},
		StudentCode:  sql.NullString{String: "B00002", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create user 2: %v", err)
	}

	gotByID, err := userRepo.GetByID(ctx, u1.ID)
	if err != nil || gotByID == nil || gotByID.Email != u1.Email {
		t.Fatalf("GetByID failed: %+v (err: %v)", gotByID, err)
	}

	gotWithoutDel, err := userRepo.GetByIDWithoutDeleted(ctx, u1.ID)
	if err != nil || gotWithoutDel == nil || gotWithoutDel.ID != u1.ID {
		t.Fatalf("GetByIDWithoutDeleted failed: %+v (err: %v)", gotWithoutDel, err)
	}

	byIDs, err := userRepo.GetByIDs(ctx, []string{u1.ID, u2.ID})
	if err != nil || len(byIDs) != 2 {
		t.Fatalf("GetByIDs failed: %d items (err: %v)", len(byIDs), err)
	}

	count, err := userRepo.Count(ctx, sqlc.CountUsersParams{})
	if err != nil || count != 2 {
		t.Fatalf("Count failed: got %d, expected 2 (err: %v)", count, err)
	}

	userRepo.InvalidateUserCache(ctx, u1.ID, u1.Email)

	gotAfterInvalidate, err := userRepo.GetByID(ctx, u1.ID)
	if err != nil || gotAfterInvalidate == nil {
		t.Fatalf("GetByID after cache invalidation failed: %+v (err: %v)", gotAfterInvalidate, err)
	}
}

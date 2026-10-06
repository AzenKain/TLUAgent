package repositories_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func TestNotificationRepository_CRUD(t *testing.T) {
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
	repo := repositories.NewNotificationRepository(db, ramCache)
	ctx := context.Background()

	userID := "user-notif-001"
	_, err = userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           userID,
		Email:        "sv_notif@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Nguyen Thi B", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	notif := &models.UserNotification{
		ID:      "notif-001",
		UserID:  userID,
		Title:   "Co van hoc tap da tra loi",
		Content: "Thac mac cua ban ve chuan dau ra da duoc giai dap.",
		Type:    "INQUIRY_ANSWERED",
	}

	created, err := repo.CreateNotification(ctx, notif)
	if err != nil {
		t.Fatalf("Failed to create notification: %v", err)
	}
	if created.IsRead {
		t.Fatalf("Expected new notification to be unread")
	}

	unread, err := repo.CountUnreadNotifications(ctx, userID)
	if err != nil || unread != 1 {
		t.Fatalf("Expected 1 unread notification, got %d", unread)
	}

	list, err := repo.ListNotificationsByUserID(ctx, userID, 10, 0)
	if err != nil || len(list) != 1 {
		t.Fatalf("Expected 1 notification in list, got %d", len(list))
	}

	if err := repo.MarkNotificationAsRead(ctx, "notif-001", userID); err != nil {
		t.Fatalf("Failed to mark notification as read: %v", err)
	}

	unreadAfter, err := repo.CountUnreadNotifications(ctx, userID)
	if err != nil || unreadAfter != 0 {
		t.Fatalf("Expected 0 unread notifications after mark, got %d", unreadAfter)
	}
}

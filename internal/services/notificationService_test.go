package services_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func TestNotificationService_Flow(t *testing.T) {
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
	notifRepo := repositories.NewNotificationRepository(db, ramCache)
	svc := services.NewNotificationService(notifRepo)
	ctx := context.Background()

	userID := "user-notif-svc-1"
	_, err = userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           userID,
		Email:        "sv_notif_svc@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Nguyen Thi C", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	_, err = svc.CreateNotification(ctx, &models.UserNotification{
		ID:      "notif-1",
		UserID:  userID,
		Title:   "Co van da tra loi",
		Content: "Cau tra loi cua co van",
		Type:    "INQUIRY_ANSWERED",
	})
	if err != nil {
		t.Fatalf("Failed to create notification: %v", err)
	}

	unread, err := svc.GetUnreadCount(ctx, userID)
	if err != nil || unread != 1 {
		t.Fatalf("Expected 1 unread notification, got %d", unread)
	}

	items, unreadInList, err := svc.ListUserNotifications(ctx, userID, &request.ListNotificationsDto{
		PaginationDto: request.PaginationDto{
			Page:  1,
			Limit: 10,
		},
	})
	if err != nil || len(items) != 1 || unreadInList != 1 {
		t.Fatalf("Expected 1 item, unread=1, got len=%d unread=%d", len(items), unreadInList)
	}

	if err := svc.MarkAsRead(ctx, "notif-1", userID); err != nil {
		t.Fatalf("Failed to mark as read: %v", err)
	}

	unreadAfter, err := svc.GetUnreadCount(ctx, userID)
	if err != nil || unreadAfter != 0 {
		t.Fatalf("Expected 0 unread, got %d", unreadAfter)
	}
}

package services

import (
	"context"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
)

// NotificationService provides student notification queries and read tracking.
type NotificationService interface {
	CreateNotification(ctx context.Context, notif *models.UserNotification) (*response.NotificationResponse, error)
	ListUserNotifications(ctx context.Context, userID string, dto *request.ListNotificationsDto) ([]*response.NotificationResponse, int, error)
	MarkAsRead(ctx context.Context, id, userID string) error
	MarkAllAsRead(ctx context.Context, userID string) error
	GetUnreadCount(ctx context.Context, userID string) (int, error)
}

type notificationService struct {
	repo repositories.NotificationRepository
}

// NewNotificationService creates an instance of NotificationService.
func NewNotificationService(repo repositories.NotificationRepository) NotificationService {
	return &notificationService{repo: repo}
}

func (s *notificationService) CreateNotification(ctx context.Context, notif *models.UserNotification) (*response.NotificationResponse, error) {
	created, err := s.repo.CreateNotification(ctx, notif)
	if err != nil {
		return nil, err
	}
	return created.ToResponse(), nil
}

func (s *notificationService) ListUserNotifications(ctx context.Context, userID string, dto *request.ListNotificationsDto) ([]*response.NotificationResponse, int, error) {
	limit := 20
	offset := 0
	if dto != nil {
		limit = dto.GetLimit(20)
		offset = dto.GetOffset(20)
	}

	items, err := s.repo.ListNotificationsByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	unread, _ := s.repo.CountUnreadNotifications(ctx, userID)
	return models.NotificationsToResponse(items), unread, nil
}

func (s *notificationService) MarkAsRead(ctx context.Context, id, userID string) error {
	return s.repo.MarkNotificationAsRead(ctx, id, userID)
}

func (s *notificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllNotificationsAsRead(ctx, userID)
}

func (s *notificationService) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.CountUnreadNotifications(ctx, userID)
}

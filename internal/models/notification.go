package models

import (
	"time"

	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
)

type UserNotification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	InquiryID string    `json:"inquiry_id,omitempty"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

func FromSQLCNotification(row *sqlc.UserNotification) *UserNotification {
	if row == nil {
		return nil
	}
	var inqID string
	if row.InquiryID.Valid {
		inqID = row.InquiryID.String
	}
	return &UserNotification{
		ID:        row.ID,
		UserID:    row.UserID,
		InquiryID: inqID,
		Title:     row.Title,
		Content:   row.Content,
		Type:      row.Type,
		IsRead:    row.IsRead == 1,
		CreatedAt: row.CreatedAt,
	}
}

// ToResponse converts a UserNotification domain model to a response DTO.
func (n *UserNotification) ToResponse() *response.NotificationResponse {
	if n == nil {
		return nil
	}
	return &response.NotificationResponse{
		ID:        n.ID,
		UserID:    n.UserID,
		InquiryID: n.InquiryID,
		Title:     n.Title,
		Content:   n.Content,
		Type:      n.Type,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt,
	}
}

// NotificationsToResponse converts a batch of UserNotification to response DTOs.
func NotificationsToResponse(items []*UserNotification) []*response.NotificationResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.NotificationResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

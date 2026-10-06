package models

import (
	"time"

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

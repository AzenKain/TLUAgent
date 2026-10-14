package response

import (
	"time"
)

type InquiryResponse struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	ConversationID    string     `json:"conversation_id,omitempty"`
	StudentName       string     `json:"student_name"`
	StudentCode       string     `json:"student_code"`
	StudentClass      string     `json:"student_class"`
	Question          string     `json:"question"`
	Context           string     `json:"context,omitempty"`
	Status            string     `json:"status"`
	TeacherID         string               `json:"teacher_id,omitempty"`
	TeacherName       string               `json:"teacher_name,omitempty"`
	TeacherReply      string               `json:"teacher_reply,omitempty"`
	AnsweredAt        *time.Time           `json:"answered_at,omitempty"`
	KnowledgeChunkID  string               `json:"knowledge_chunk_id,omitempty"`
	SupersededByDocID string               `json:"superseded_by_doc_id,omitempty"`
	IsExpired         bool                 `json:"is_expired"`
	ExpiredReason     string               `json:"expired_reason,omitempty"`
	ExpiredAt         *time.Time           `json:"expired_at,omitempty"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
}

type NotificationResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	InquiryID string    `json:"inquiry_id,omitempty"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

type UnreadCountResponse struct {
	UnreadCount int `json:"unread_count"`
}

// InquiryListResponse is a paginated envelope of student or advisor inquiries.
type InquiryListResponse struct {
	Items []*InquiryResponse `json:"items"`
	Total int                `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
}

// NotificationListResponse is a paginated envelope of student notifications with an unread counter.
type NotificationListResponse struct {
	Items       []*NotificationResponse `json:"items"`
	UnreadCount int                     `json:"unread_count"`
	Page        int                     `json:"page"`
	Limit       int                     `json:"limit"`
}

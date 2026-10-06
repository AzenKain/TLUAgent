package models

import (
	"database/sql"
	"time"

	"tluagent-web/internal/gen/sqlc"
)

type InquiryStatus string

const (
	InquiryStatusPending  InquiryStatus = "PENDING"
	InquiryStatusAnswered InquiryStatus = "ANSWERED"
	InquiryStatusExpired  InquiryStatus = "EXPIRED"
	InquiryStatusRejected InquiryStatus = "REJECTED"
)

type AdvisoryInquiry struct {
	ID                string        `json:"id"`
	UserID            string        `json:"user_id"`
	ConversationID    string        `json:"conversation_id,omitempty"`
	StudentName       string        `json:"student_name"`
	StudentCode       string        `json:"student_code"`
	StudentClass      string        `json:"student_class"`
	Question          string        `json:"question"`
	Context           string        `json:"context,omitempty"`
	Status            InquiryStatus `json:"status"`
	TeacherID         string        `json:"teacher_id,omitempty"`
	TeacherName       string        `json:"teacher_name,omitempty"`
	TeacherReply      string        `json:"teacher_reply,omitempty"`
	AnsweredAt        *time.Time    `json:"answered_at,omitempty"`
	KnowledgeChunkID  string        `json:"knowledge_chunk_id,omitempty"`
	SupersededByDocID string        `json:"superseded_by_doc_id,omitempty"`
	IsExpired         bool          `json:"is_expired"`
	ExpiredReason     string        `json:"expired_reason,omitempty"`
	ExpiredAt         *time.Time    `json:"expired_at,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

func FromSQLCInquiry(row *sqlc.AdvisoryInquiry) *AdvisoryInquiry {
	if row == nil {
		return nil
	}
	var convID string
	if row.ConversationID.Valid {
		convID = row.ConversationID.String
	}
	var teacherID string
	if row.TeacherID.Valid {
		teacherID = row.TeacherID.String
	}
	var chunkID string
	if row.KnowledgeChunkID.Valid {
		chunkID = row.KnowledgeChunkID.String
	}
	var superDocID string
	if row.SupersededByDocID.Valid {
		superDocID = row.SupersededByDocID.String
	}
	var answeredAt *time.Time
	if row.AnsweredAt.Valid {
		t := row.AnsweredAt.Time
		answeredAt = &t
	}
	var expiredAt *time.Time
	if row.ExpiredAt.Valid {
		t := row.ExpiredAt.Time
		expiredAt = &t
	}

	return &AdvisoryInquiry{
		ID:                row.ID,
		UserID:            row.UserID,
		ConversationID:    convID,
		StudentName:       row.StudentName,
		StudentCode:       row.StudentCode,
		StudentClass:      row.StudentClass,
		Question:          row.Question,
		Context:           row.Context,
		Status:            InquiryStatus(row.Status),
		TeacherID:         teacherID,
		TeacherName:       row.TeacherName,
		TeacherReply:      row.TeacherReply,
		AnsweredAt:        answeredAt,
		KnowledgeChunkID:  chunkID,
		SupersededByDocID: superDocID,
		IsExpired:         row.IsExpired == 1,
		ExpiredReason:     row.ExpiredReason,
		ExpiredAt:         expiredAt,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func NullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func NullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

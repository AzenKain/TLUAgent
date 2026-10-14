package models

import (
	"tluagent-web/internal/dtos/response"
)

// ChatConversationEntity represents a user chat session stored in the database.
type ChatConversationEntity struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Title     string `json:"title"`
	ModelID   string `json:"model_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ToResponse converts a ChatConversationEntity domain model to response DTO.
func (c *ChatConversationEntity) ToResponse() *response.ChatSessionResponse {
	if c == nil {
		return nil
	}
	return &response.ChatSessionResponse{
		ID:        c.ID,
		UserID:    c.UserID,
		Title:     c.Title,
		ModelID:   c.ModelID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// ChatConversationsToResponse converts a batch of ChatConversationEntity to response DTOs.
func ChatConversationsToResponse(items []*ChatConversationEntity) []*response.ChatSessionResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.ChatSessionResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

// ChatMessageEntity represents an individual message in a chat conversation.
type ChatMessageEntity struct {
	ID             string   `json:"id"`
	ConversationID string   `json:"conversation_id"`
	Sender         string   `json:"sender"`
	Content        string   `json:"content"`
	Sources        []string `json:"sources"`
	Images         []string `json:"images"`
	Feedback       string   `json:"feedback,omitempty"`
	CanEscalate    bool     `json:"can_escalate"`
	CreatedAt      string   `json:"created_at"`
}

// ToResponse converts a ChatMessageEntity domain model to response DTO.
func (m *ChatMessageEntity) ToResponse() *response.ChatMessageResponse {
	if m == nil {
		return nil
	}
	return &response.ChatMessageResponse{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		Sender:         m.Sender,
		Content:        m.Content,
		Sources:        m.Sources,
		Images:         m.Images,
		Feedback:       m.Feedback,
		CanEscalate:    m.CanEscalate,
		CreatedAt:      m.CreatedAt,
	}
}

// ChatMessagesToResponse converts a batch of ChatMessageEntity to response DTOs.
func ChatMessagesToResponse(items []*ChatMessageEntity) []*response.ChatMessageResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.ChatMessageResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

// AdminConversationSummary represents aggregated conversation metadata for administration.
type AdminConversationSummary struct {
	ID                string `json:"id"`
	UserID            string `json:"user_id"`
	UserEmail         string `json:"user_email"`
	UserFullName      string `json:"user_name"`
	StudentCode       string `json:"student_code"`
	Title             string `json:"title"`
	ModelID           string `json:"model_id"`
	MessageCount      int    `json:"total_messages"`
	FeedbackUpCount   int    `json:"thumbs_up"`
	FeedbackDownCount int    `json:"thumbs_down"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// ToResponse converts an AdminConversationSummary domain model to response DTO.
func (s *AdminConversationSummary) ToResponse() *response.AdminConversationSummaryResponse {
	if s == nil {
		return nil
	}
	return &response.AdminConversationSummaryResponse{
		ID:                s.ID,
		UserID:            s.UserID,
		UserEmail:         s.UserEmail,
		UserFullName:      s.UserFullName,
		StudentCode:       s.StudentCode,
		Title:             s.Title,
		ModelID:           s.ModelID,
		MessageCount:      s.MessageCount,
		FeedbackUpCount:   s.FeedbackUpCount,
		FeedbackDownCount: s.FeedbackDownCount,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}

// AdminConversationSummariesToResponse converts a batch of AdminConversationSummary to response DTOs.
func AdminConversationSummariesToResponse(items []*AdminConversationSummary) []*response.AdminConversationSummaryResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.AdminConversationSummaryResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

// ConversationDetailResponse bundles conversation metadata with its message trajectory.
type ConversationDetailResponse struct {
	Conversation *ChatConversationEntity `json:"conversation"`
	User         *UserEntity             `json:"user,omitempty"`
	Messages     []*ChatMessageEntity    `json:"messages"`
}

// ToResponse converts a ConversationDetailResponse domain model to response DTO.
func (d *ConversationDetailResponse) ToResponse() *response.ConversationDetailResponse {
	if d == nil {
		return nil
	}
	return &response.ConversationDetailResponse{
		Conversation: d.Conversation.ToResponse(),
		User:         conversationUserToResponse(d.User),
		Messages:     ChatMessagesToResponse(d.Messages),
	}
}

func conversationUserToResponse(u *UserEntity) *response.ConversationUserResponse {
	if u == nil {
		return nil
	}
	var roles []*response.RoleSimpleResponse
	if u.Roles != nil {
		roles = RolesToResponse(u.Roles)
	}
	return &response.ConversationUserResponse{
		ID:           u.ID,
		Email:        u.Email,
		FullName:     u.FullName,
		StudentCode:  u.StudentCode,
		AvatarUrl:    u.AvatarUrl,
		AuthProvider: u.AuthProvider,
		TokenVersion: u.TokenVersion,
		Roles:        roles,
		IsDeleted:    u.IsDeleted,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

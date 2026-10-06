package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
)

// ChatSessionService orchestrates user conversation persistence and administrator moderation.
type ChatSessionService interface {
	ListUserSessions(ctx context.Context, userID string, dto *request.ListUserSessionsDto) ([]*models.ChatConversationEntity, int, error)
	GetSessionDetails(ctx context.Context, sessionID, userID string) (*models.ConversationDetailResponse, error)
	CreateSession(ctx context.Context, userID, title, modelID string) (*models.ChatConversationEntity, error)
	UpdateSessionTitle(ctx context.Context, sessionID, userID, title string) error
	DeleteUserSession(ctx context.Context, sessionID, userID string) error
	RecordExchange(ctx context.Context, dto *request.RecordChatExchangeDto) (string, error)
	UpdateFeedback(ctx context.Context, messageID, userID, feedback string) error

	ListAdminConversations(ctx context.Context, dto *request.ListAdminConversationsDto) ([]*models.AdminConversationSummary, int, error)
	GetAdminConversationDetail(ctx context.Context, sessionID string) (*models.ConversationDetailResponse, error)
	DeleteAdminConversation(ctx context.Context, sessionID string) error
	GetAdminStats(ctx context.Context) (map[string]any, error)
}

type chatSessionService struct {
	chatRepo repositories.ChatRepository
	userRepo repositories.UserRepository
}

// NewChatSessionService instantiates a chat session service.
func NewChatSessionService(chatRepo repositories.ChatRepository, userRepo repositories.UserRepository) ChatSessionService {
	return &chatSessionService{
		chatRepo: chatRepo,
		userRepo: userRepo,
	}
}

func mapChatRepoError(err error, notFoundMessage, failureMessage string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return apperrors.New(apperrors.ErrNotFound, notFoundMessage)
	}
	return apperrors.New(apperrors.ErrInternalError, failureMessage)
}

func (s *chatSessionService) ListUserSessions(ctx context.Context, userID string, dto *request.ListUserSessionsDto) ([]*models.ChatConversationEntity, int, error) {
	limit := 20
	offset := 0
	if dto != nil {
		limit = dto.GetLimit(20)
		offset = dto.GetOffset(20)
	}
	convs, total, err := s.chatRepo.ListConversationsByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.New(apperrors.ErrInternalError, "Failed to list chat sessions")
	}
	return convs, total, nil
}

func (s *chatSessionService) GetSessionDetails(ctx context.Context, sessionID, userID string) (*models.ConversationDetailResponse, error) {
	conv, err := s.chatRepo.GetConversationByID(ctx, sessionID)
	if err != nil {
		return nil, mapChatRepoError(err, "Session not found", "Failed to load session")
	}
	if conv.UserID != userID {
		return nil, apperrors.New(apperrors.ErrForbidden, "You do not have access to this session")
	}

	messages, err := s.chatRepo.ListMessagesByConversationID(ctx, sessionID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load session messages")
	}

	return &models.ConversationDetailResponse{
		Conversation: conv,
		Messages:     messages,
	}, nil
}

func (s *chatSessionService) CreateSession(ctx context.Context, userID, title, modelID string) (*models.ChatConversationEntity, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "New Chat"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	conv := &models.ChatConversationEntity{
		ID:        uuid.NewString(),
		UserID:    userID,
		Title:     title,
		ModelID:   modelID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.chatRepo.CreateConversation(ctx, conv); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to create session")
	}
	return conv, nil
}

func (s *chatSessionService) UpdateSessionTitle(ctx context.Context, sessionID, userID, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return apperrors.New(apperrors.ErrBadRequest, "Title cannot be empty")
	}
	if err := s.chatRepo.UpdateConversationTitle(ctx, sessionID, userID, title); err != nil {
		return mapChatRepoError(err, "Session not found", "Failed to update session title")
	}
	return nil
}

func (s *chatSessionService) DeleteUserSession(ctx context.Context, sessionID, userID string) error {
	if err := s.chatRepo.DeleteConversation(ctx, sessionID, userID); err != nil {
		return mapChatRepoError(err, "Session not found", "Failed to delete session")
	}
	return nil
}

func (s *chatSessionService) RecordExchange(ctx context.Context, dto *request.RecordChatExchangeDto) (string, error) {
	if dto == nil || dto.UserID == "" || dto.UserID == "0" {
		return "", nil
	}

	var conv *models.ChatConversationEntity
	var err error

	sessionID := dto.SessionID
	if sessionID != "" {
		conv, err = s.chatRepo.GetConversationByID(ctx, sessionID)
		if err == nil && conv != nil && conv.UserID != dto.UserID {
			conv = nil
			sessionID = ""
		}
	}

	if conv == nil || errors.Is(err, sql.ErrNoRows) || sessionID == "" {
		title := strings.TrimSpace(dto.UserQuery)
		runes := []rune(title)
		if len(runes) > 40 {
			title = string(runes[:40]) + "..."
		}
		if title == "" {
			title = "New Chat"
		}

		conv, err = s.CreateSession(ctx, dto.UserID, title, dto.ModelID)
		if err != nil {
			return "", err
		}
		sessionID = conv.ID
	}

	now := time.Now().UTC().Format(time.RFC3339)
	userMsg := &models.ChatMessageEntity{
		ID:             uuid.NewString(),
		ConversationID: sessionID,
		Sender:         "user",
		Content:        dto.UserQuery,
		Images:         dto.UserImages,
		CreatedAt:      now,
	}
	if err := s.chatRepo.CreateMessage(ctx, userMsg); err != nil {
		return sessionID, apperrors.New(apperrors.ErrInternalError, "Failed to record chat message")
	}

	botMsg := &models.ChatMessageEntity{
		ID:             uuid.NewString(),
		ConversationID: sessionID,
		Sender:         "assistant",
		Content:        dto.BotReply,
		Sources:        dto.BotSources,
		CreatedAt:      now,
	}
	if err := s.chatRepo.CreateMessage(ctx, botMsg); err != nil {
		return sessionID, apperrors.New(apperrors.ErrInternalError, "Failed to record chat reply")
	}

	_ = s.chatRepo.TouchConversation(ctx, sessionID)
	return sessionID, nil
}

func (s *chatSessionService) UpdateFeedback(ctx context.Context, messageID, userID, feedback string) error {
	if userID == "" || userID == "0" {
		return apperrors.New(apperrors.ErrUnauthorized, "Unauthorized")
	}
	feedback = strings.ToLower(strings.TrimSpace(feedback))
	if feedback == "clear" {
		feedback = ""
	}
	if err := s.chatRepo.UpdateMessageFeedback(ctx, messageID, userID, feedback); err != nil {
		return mapChatRepoError(err, "Message not found", "Failed to update feedback")
	}
	return nil
}

func (s *chatSessionService) ListAdminConversations(ctx context.Context, dto *request.ListAdminConversationsDto) ([]*models.AdminConversationSummary, int, error) {
	limit := 20
	offset := 0
	search := ""
	feedback := ""
	modelID := ""
	if dto != nil {
		limit = dto.GetLimit(20)
		offset = dto.GetOffset(20)
		search = dto.Q
		feedback = dto.Feedback
		modelID = dto.ModelID
	}
	summaries, total, err := s.chatRepo.ListAdminConversations(ctx, search, feedback, modelID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.New(apperrors.ErrInternalError, "Failed to list conversations")
	}
	return summaries, total, nil
}

func (s *chatSessionService) GetAdminConversationDetail(ctx context.Context, sessionID string) (*models.ConversationDetailResponse, error) {
	conv, err := s.chatRepo.GetConversationByID(ctx, sessionID)
	if err != nil {
		return nil, mapChatRepoError(err, "Conversation not found", "Failed to load conversation")
	}

	messages, err := s.chatRepo.ListMessagesByConversationID(ctx, sessionID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load conversation messages")
	}

	var userEntity *models.UserEntity
	if s.userRepo != nil {
		userEntity, _ = s.userRepo.GetByID(ctx, conv.UserID)
	}

	return &models.ConversationDetailResponse{
		Conversation: conv,
		User:         userEntity,
		Messages:     messages,
	}, nil
}

func (s *chatSessionService) DeleteAdminConversation(ctx context.Context, sessionID string) error {
	if err := s.chatRepo.DeleteConversationAdmin(ctx, sessionID); err != nil {
		return mapChatRepoError(err, "Conversation not found", "Failed to delete conversation")
	}
	return nil
}

func (s *chatSessionService) GetAdminStats(ctx context.Context) (map[string]any, error) {
	totalChats, totalMessages, thumbsUp, thumbsDown, err := s.chatRepo.GetAdminConversationStats(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load chat statistics")
	}

	satisfactionRate := 0.0
	totalRatings := thumbsUp + thumbsDown
	if totalRatings > 0 {
		satisfactionRate = float64(thumbsUp) / float64(totalRatings) * 100
	}

	return map[string]any{
		"total_conversations": totalChats,
		"total_messages":      totalMessages,
		"feedback_up":         thumbsUp,
		"feedback_down":       thumbsDown,
		"satisfaction_rate":   satisfactionRate,
	}, nil
}

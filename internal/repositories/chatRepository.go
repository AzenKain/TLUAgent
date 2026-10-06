package repositories

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/convert"
	"tluagent-web/pkg/jsonx"
)

// ChatRepository defines database operations for conversations, messages, and admin analytics.
type ChatRepository interface {
	CreateConversation(ctx context.Context, conv *models.ChatConversationEntity) error
	GetConversationByID(ctx context.Context, id string) (*models.ChatConversationEntity, error)
	GetConversationsByIDs(ctx context.Context, ids []string) ([]*models.ChatConversationEntity, error)
	ListConversationsByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.ChatConversationEntity, int, error)
	UpdateConversationTitle(ctx context.Context, id, userID, title string) error
	TouchConversation(ctx context.Context, id string) error
	DeleteConversation(ctx context.Context, id, userID string) error
	DeleteConversationAdmin(ctx context.Context, id string) error

	CreateMessage(ctx context.Context, msg *models.ChatMessageEntity) error
	ListMessagesByConversationID(ctx context.Context, convID string) ([]*models.ChatMessageEntity, error)
	UpdateMessageFeedback(ctx context.Context, messageID, userID, feedback string) error

	ListAdminConversations(ctx context.Context, search, feedback, modelID string, limit, offset int) ([]*models.AdminConversationSummary, int, error)
	GetAdminConversationStats(ctx context.Context) (totalChats int, totalMessages int, thumbsUp int, thumbsDown int, err error)

	InvalidateConversationCache(ctx context.Context, id, userID string)
	WithTx(tx *sql.Tx) ChatRepository
}

type chatRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

// NewChatRepository creates a chat repository instance backed by sqlc and RAM cache.
func NewChatRepository(db sqlc.DBTX, c cache.Cache) ChatRepository {
	return &chatRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *chatRepository) WithTx(tx *sql.Tx) ChatRepository {
	return &chatRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sfg:  &singleflight.Group{},
	}
}

func (r *chatRepository) CreateConversation(ctx context.Context, conv *models.ChatConversationEntity) error {
	now := time.Now().UTC()
	var modelID sql.NullString
	if conv.ModelID != "" {
		modelID = sql.NullString{String: conv.ModelID, Valid: true}
	}

	created, err := r.q.CreateConversation(ctx, sqlc.CreateConversationParams{
		ID:        conv.ID,
		UserID:    conv.UserID,
		Title:     conv.Title,
		ModelID:   modelID,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return err
	}

	conv.CreatedAt = created.CreatedAt.Format(time.RFC3339)
	conv.UpdatedAt = created.UpdatedAt.Format(time.RFC3339)
	r.InvalidateConversationCache(ctx, conv.ID, conv.UserID)
	return nil
}

func (r *chatRepository) GetConversationByID(ctx context.Context, id string) (*models.ChatConversationEntity, error) {
	key := cache.BuildKey("chat_conv", "id", id)
	if r.c != nil && !r.inTx {
		var conv models.ChatConversationEntity
		if err := r.c.Get(ctx, key, &conv); err == nil {
			return &conv, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetConversationByID(ctx, id)
		if err != nil {
			return nil, err
		}
		convPtr := &models.ChatConversationEntity{
			ID:        row.ID,
			UserID:    row.UserID,
			Title:     row.Title,
			ModelID:   convert.NullStringToString(row.ModelID),
			CreatedAt: row.CreatedAt.Format(time.RFC3339),
			UpdatedAt: row.UpdatedAt.Format(time.RFC3339),
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, convPtr, constants.NormalCacheDuration)
		}
		return convPtr, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.ChatConversationEntity), nil
}

func (r *chatRepository) GetConversationsByIDs(ctx context.Context, ids []string) ([]*models.ChatConversationEntity, error) {
	if len(ids) == 0 {
		return []*models.ChatConversationEntity{}, nil
	}

	resultMap := make(map[string]*models.ChatConversationEntity, len(ids))
	var missingIDs []string

	for _, id := range ids {
		key := cache.BuildKey("chat_conv", "id", id)
		if r.c != nil && !r.inTx {
			var conv models.ChatConversationEntity
			if err := r.c.Get(ctx, key, &conv); err == nil {
				resultMap[id] = &conv
				continue
			}
		}
		missingIDs = append(missingIDs, id)
	}

	if len(missingIDs) > 0 {
		rows, err := r.q.GetConversationsByIDs(ctx, missingIDs)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			convPtr := &models.ChatConversationEntity{
				ID:        row.ID,
				UserID:    row.UserID,
				Title:     row.Title,
				ModelID:   convert.NullStringToString(row.ModelID),
				CreatedAt: row.CreatedAt.Format(time.RFC3339),
				UpdatedAt: row.UpdatedAt.Format(time.RFC3339),
			}
			resultMap[convPtr.ID] = convPtr
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, cache.BuildKey("chat_conv", "id", convPtr.ID), convPtr, constants.NormalCacheDuration)
			}
		}
	}

	out := make([]*models.ChatConversationEntity, 0, len(ids))
	for _, id := range ids {
		if conv, ok := resultMap[id]; ok {
			out = append(out, conv)
		}
	}
	return out, nil
}

func (r *chatRepository) ListConversationsByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.ChatConversationEntity, int, error) {
	total, err := r.q.CountConversationsByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	ids, err := r.q.ListConversationIDsByUserID(ctx, sqlc.ListConversationIDsByUserIDParams{
		UserID: userID,
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	convs, err := r.GetConversationsByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	return convs, int(total), nil
}

func (r *chatRepository) UpdateConversationTitle(ctx context.Context, id, userID, title string) error {
	now := time.Now().UTC()
	rows, err := r.q.UpdateConversationTitle(ctx, sqlc.UpdateConversationTitleParams{
		Title:     title,
		UpdatedAt: now,
		ID:        id,
		UserID:    userID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	r.InvalidateConversationCache(ctx, id, userID)
	return nil
}

func (r *chatRepository) TouchConversation(ctx context.Context, id string) error {
	now := time.Now().UTC()
	err := r.q.TouchConversation(ctx, sqlc.TouchConversationParams{
		UpdatedAt: now,
		ID:        id,
	})
	if err == nil {
		r.InvalidateConversationCache(ctx, id, "")
	}
	return err
}

func (r *chatRepository) DeleteConversation(ctx context.Context, id, userID string) error {
	rows, err := r.q.DeleteConversation(ctx, sqlc.DeleteConversationParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	r.InvalidateConversationCache(ctx, id, userID)
	return nil
}

func (r *chatRepository) DeleteConversationAdmin(ctx context.Context, id string) error {
	rows, err := r.q.DeleteConversationAdmin(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	r.InvalidateConversationCache(ctx, id, "")
	return nil
}

func (r *chatRepository) CreateMessage(ctx context.Context, msg *models.ChatMessageEntity) error {
	sourcesJSON, err := jsonx.Marshal(msg.Sources)
	if err != nil {
		sourcesJSON = []byte("[]")
	}
	imagesJSON, err := jsonx.Marshal(msg.Images)
	if err != nil {
		imagesJSON = []byte("[]")
	}

	now := time.Now().UTC()
	var fb sql.NullString
	if msg.Feedback != "" {
		fb = sql.NullString{String: msg.Feedback, Valid: true}
	}

	created, err := r.q.CreateMessage(ctx, sqlc.CreateMessageParams{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		Sender:         msg.Sender,
		Content:        msg.Content,
		SourcesJson:    string(sourcesJSON),
		ImagesJson:     string(imagesJSON),
		Feedback:       fb,
		CreatedAt:      now,
	})
	if err != nil {
		return err
	}

	msg.CreatedAt = created.CreatedAt.Format(time.RFC3339)
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("chat_msgs", "conv", msg.ConversationID), cache.BuildKey("chat_admin", "stats"))
	}
	return nil
}

func (r *chatRepository) ListMessagesByConversationID(ctx context.Context, convID string) ([]*models.ChatMessageEntity, error) {
	key := cache.BuildKey("chat_msgs", "conv", convID)
	if r.c != nil && !r.inTx {
		var msgs []*models.ChatMessageEntity
		if err := r.c.Get(ctx, key, &msgs); err == nil {
			return msgs, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.ListMessagesByConversationID(ctx, convID)
		if err != nil {
			return nil, err
		}
		msgs := make([]*models.ChatMessageEntity, 0, len(rows))
		for _, row := range rows {
			m := &models.ChatMessageEntity{
				ID:             row.ID,
				ConversationID: row.ConversationID,
				Sender:         row.Sender,
				Content:        row.Content,
				Feedback:       convert.NullStringToString(row.Feedback),
				CreatedAt:      row.CreatedAt.Format(time.RFC3339),
			}
			_ = jsonx.Unmarshal([]byte(row.SourcesJson), &m.Sources)
			_ = jsonx.Unmarshal([]byte(row.ImagesJson), &m.Images)
			msgs = append(msgs, m)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, msgs, constants.NormalCacheDuration)
		}
		return msgs, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.ChatMessageEntity), nil
}

func (r *chatRepository) UpdateMessageFeedback(ctx context.Context, messageID, userID, feedback string) error {
	if userID == "" {
		return errors.New("user ID is required to update message feedback")
	}

	var fb sql.NullString
	if feedback != "" {
		fb = sql.NullString{String: feedback, Valid: true}
	}

	rows, err := r.q.UpdateMessageFeedback(ctx, sqlc.UpdateMessageFeedbackParams{
		Feedback: fb,
		ID:       messageID,
		UserID:   userID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	if r.c != nil {
		_ = r.c.DelByPattern(ctx, "chat_msgs*")
		_ = r.c.Del(ctx, cache.BuildKey("chat_admin", "stats"))
	}
	return nil
}

func escapeLikePattern(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '%', '_', '\\':
			b.WriteRune('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (r *chatRepository) ListAdminConversations(ctx context.Context, search, feedback, modelID string, limit, offset int) ([]*models.AdminConversationSummary, int, error) {
	var searchPattern any = nil
	if strings.TrimSpace(search) != "" {
		searchPattern = "%" + escapeLikePattern(strings.TrimSpace(search)) + "%"
	}
	var modelIDArg any = nil
	if strings.TrimSpace(modelID) != "" {
		modelIDArg = strings.TrimSpace(modelID)
	}

	var total int64
	var summaries []*models.AdminConversationSummary

	switch feedback {
	case "up":
		cnt, err := r.q.CountAdminConversationsFeedbackUp(ctx, sqlc.CountAdminConversationsFeedbackUpParams{
			SearchPattern: searchPattern,
			ModelID:       modelIDArg,
		})
		if err != nil {
			return nil, 0, err
		}
		total = cnt

		rows, err := r.q.ListAdminConversationsFeedbackUp(ctx, sqlc.ListAdminConversationsFeedbackUpParams{
			SearchPattern: searchPattern,
			ModelID:       modelIDArg,
			Limit:         int64(limit),
			Offset:        int64(offset),
		})
		if err != nil {
			return nil, 0, err
		}
		for _, row := range rows {
			summaries = append(summaries, &models.AdminConversationSummary{
				ID:                row.ID,
				UserID:            row.UserID,
				UserEmail:         row.UserEmail,
				UserFullName:      row.UserFullName,
				StudentCode:       row.StudentCode,
				Title:             row.Title,
				ModelID:           row.ModelID,
				MessageCount:      int(row.MessageCount),
				FeedbackUpCount:   int(row.FeedbackUpCount),
				FeedbackDownCount: int(row.FeedbackDownCount),
				CreatedAt:         row.CreatedAt.Format(time.RFC3339),
				UpdatedAt:         row.UpdatedAt.Format(time.RFC3339),
			})
		}
	case "down":
		cnt, err := r.q.CountAdminConversationsFeedbackDown(ctx, sqlc.CountAdminConversationsFeedbackDownParams{
			SearchPattern: searchPattern,
			ModelID:       modelIDArg,
		})
		if err != nil {
			return nil, 0, err
		}
		total = cnt

		rows, err := r.q.ListAdminConversationsFeedbackDown(ctx, sqlc.ListAdminConversationsFeedbackDownParams{
			SearchPattern: searchPattern,
			ModelID:       modelIDArg,
			Limit:         int64(limit),
			Offset:        int64(offset),
		})
		if err != nil {
			return nil, 0, err
		}
		for _, row := range rows {
			summaries = append(summaries, &models.AdminConversationSummary{
				ID:                row.ID,
				UserID:            row.UserID,
				UserEmail:         row.UserEmail,
				UserFullName:      row.UserFullName,
				StudentCode:       row.StudentCode,
				Title:             row.Title,
				ModelID:           row.ModelID,
				MessageCount:      int(row.MessageCount),
				FeedbackUpCount:   int(row.FeedbackUpCount),
				FeedbackDownCount: int(row.FeedbackDownCount),
				CreatedAt:         row.CreatedAt.Format(time.RFC3339),
				UpdatedAt:         row.UpdatedAt.Format(time.RFC3339),
			})
		}
	default:
		cnt, err := r.q.CountAdminConversationsNoFeedback(ctx, sqlc.CountAdminConversationsNoFeedbackParams{
			SearchPattern: searchPattern,
			ModelID:       modelIDArg,
		})
		if err != nil {
			return nil, 0, err
		}
		total = cnt

		rows, err := r.q.ListAdminConversationsNoFeedback(ctx, sqlc.ListAdminConversationsNoFeedbackParams{
			SearchPattern: searchPattern,
			ModelID:       modelIDArg,
			Limit:         int64(limit),
			Offset:        int64(offset),
		})
		if err != nil {
			return nil, 0, err
		}
		for _, row := range rows {
			summaries = append(summaries, &models.AdminConversationSummary{
				ID:                row.ID,
				UserID:            row.UserID,
				UserEmail:         row.UserEmail,
				UserFullName:      row.UserFullName,
				StudentCode:       row.StudentCode,
				Title:             row.Title,
				ModelID:           row.ModelID,
				MessageCount:      int(row.MessageCount),
				FeedbackUpCount:   int(row.FeedbackUpCount),
				FeedbackDownCount: int(row.FeedbackDownCount),
				CreatedAt:         row.CreatedAt.Format(time.RFC3339),
				UpdatedAt:         row.UpdatedAt.Format(time.RFC3339),
			})
		}
	}

	return summaries, int(total), nil
}

type adminStatsData struct {
	TotalChats    int `json:"total_chats"`
	TotalMessages int `json:"total_messages"`
	ThumbsUp      int `json:"thumbs_up"`
	ThumbsDown    int `json:"thumbs_down"`
}

func (r *chatRepository) GetAdminConversationStats(ctx context.Context) (totalChats int, totalMessages int, thumbsUp int, thumbsDown int, err error) {
	key := cache.BuildKey("chat_admin", "stats")
	if r.c != nil && !r.inTx {
		var stats adminStatsData
		if err := r.c.Get(ctx, key, &stats); err == nil {
			return stats.TotalChats, stats.TotalMessages, stats.ThumbsUp, stats.ThumbsDown, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		cTotal, err := r.q.GetAdminTotalChats(ctx)
		if err != nil {
			return nil, err
		}
		mStats, err := r.q.GetAdminMessageStats(ctx)
		if err != nil {
			return nil, err
		}
		stats := adminStatsData{
			TotalChats:    int(cTotal),
			TotalMessages: int(mStats.TotalMessages),
			ThumbsUp:      int(mStats.ThumbsUp),
			ThumbsDown:    int(mStats.ThumbsDown),
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, stats, constants.ListCacheDuration)
		}
		return stats, nil
	})
	if err != nil {
		return 0, 0, 0, 0, err
	}
	s := v.(adminStatsData)
	return s.TotalChats, s.TotalMessages, s.ThumbsUp, s.ThumbsDown, nil
}

func (r *chatRepository) InvalidateConversationCache(ctx context.Context, id, userID string) {
	if r.c == nil {
		return
	}
	_ = r.c.Del(ctx, cache.BuildKey("chat_admin", "stats"))
	if id != "" {
		_ = r.c.Del(ctx, cache.BuildKey("chat_conv", "id", id))
		_ = r.c.Del(ctx, cache.BuildKey("chat_msgs", "conv", id))
	}
	if userID != "" {
		_ = r.c.DelByPattern(ctx, cache.BuildKey("chat_convs", "user", userID)+"*")
	}
}

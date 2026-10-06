package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
)

// NotificationRepository manages user notifications and read receipts.
type NotificationRepository interface {
	CreateNotification(ctx context.Context, notif *models.UserNotification) (*models.UserNotification, error)
	GetNotificationByID(ctx context.Context, id string) (*models.UserNotification, error)
	GetNotificationsByIDs(ctx context.Context, ids []string) ([]*models.UserNotification, error)
	ListNotificationsByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.UserNotification, error)
	CountUnreadNotifications(ctx context.Context, userID string) (int, error)
	MarkNotificationAsRead(ctx context.Context, id, userID string) error
	MarkAllNotificationsAsRead(ctx context.Context, userID string) error
	InvalidateNotificationCache(ctx context.Context, userID string)
	WithTx(tx *sql.Tx) NotificationRepository
}

type notificationRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

// NewNotificationRepository creates an instance of NotificationRepository.
func NewNotificationRepository(db sqlc.DBTX, c cache.Cache) NotificationRepository {
	return &notificationRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *notificationRepository) WithTx(tx *sql.Tx) NotificationRepository {
	return &notificationRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sfg:  &singleflight.Group{},
	}
}

func (r *notificationRepository) InvalidateNotificationCache(ctx context.Context, userID string) {
	if r.c == nil {
		return
	}
	_ = r.c.Del(ctx,
		cache.BuildKey("notifications", "user", userID),
		cache.BuildKey("notifications", "unread", userID),
	)
}

func (r *notificationRepository) CreateNotification(ctx context.Context, notif *models.UserNotification) (*models.UserNotification, error) {
	now := time.Now()
	row, err := r.q.CreateNotification(ctx, sqlc.CreateNotificationParams{
		ID:        notif.ID,
		UserID:    notif.UserID,
		InquiryID: models.NullString(notif.InquiryID),
		Title:     notif.Title,
		Content:   notif.Content,
		Type:      notif.Type,
		CreatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	r.InvalidateNotificationCache(ctx, notif.UserID)
	return models.FromSQLCNotification(&row), nil
}

func (r *notificationRepository) GetNotificationByID(ctx context.Context, id string) (*models.UserNotification, error) {
	cacheKey := cache.BuildKey("notification", "id", id)
	if r.c != nil && !r.inTx {
		var cached models.UserNotification
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		row, err := r.q.GetNotificationByID(ctx, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, err
		}
		item := models.FromSQLCNotification(&row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, item, constants.NormalCacheDuration)
		}
		return item, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*models.UserNotification), nil
}

func (r *notificationRepository) GetNotificationsByIDs(ctx context.Context, ids []string) ([]*models.UserNotification, error) {
	if len(ids) == 0 {
		return []*models.UserNotification{}, nil
	}

	results := make([]*models.UserNotification, len(ids))
	missingIDs := make([]string, 0, len(ids))
	idToIndex := make(map[string]int, len(ids))

	for i, id := range ids {
		idToIndex[id] = i
		cacheKey := cache.BuildKey("notification", "id", id)
		var cached models.UserNotification
		if r.c != nil && !r.inTx && r.c.Get(ctx, cacheKey, &cached) == nil {
			results[i] = &cached
		} else {
			missingIDs = append(missingIDs, id)
		}
	}

	if len(missingIDs) == 0 {
		return results, nil
	}

	sfgKey := fmt.Sprintf("notifications:get_by_ids:%v", missingIDs)
	fetchedData, err, _ := r.sfg.Do(sfgKey, func() (any, error) {
		rows, err := r.q.GetNotificationsByIDs(ctx, missingIDs)
		if err != nil {
			return nil, err
		}
		return rows, nil
	})
	if err != nil {
		return nil, err
	}

	rows := fetchedData.([]sqlc.UserNotification)
	for _, row := range rows {
		item := models.FromSQLCNotification(&row)
		idx, ok := idToIndex[item.ID]
		if ok {
			results[idx] = item
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cache.BuildKey("notification", "id", item.ID), item, constants.NormalCacheDuration)
		}
	}

	finalResults := make([]*models.UserNotification, 0, len(results))
	for _, res := range results {
		if res != nil {
			finalResults = append(finalResults, res)
		}
	}

	return finalResults, nil
}

func (r *notificationRepository) ListNotificationsByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.UserNotification, error) {
	ids, err := r.q.ListNotificationIDsByUserID(ctx, sqlc.ListNotificationIDsByUserIDParams{
		UserID: userID,
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, err
	}
	return r.GetNotificationsByIDs(ctx, ids)
}

func (r *notificationRepository) CountUnreadNotifications(ctx context.Context, userID string) (int, error) {
	cacheKey := cache.BuildKey("notifications", "unread", userID)
	if r.c != nil && !r.inTx {
		var count int
		if err := r.c.Get(ctx, cacheKey, &count); err == nil {
			return count, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		count, err := r.q.CountUnreadNotifications(ctx, userID)
		if err != nil {
			return 0, err
		}
		c := int(count)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, c, constants.NormalCacheDuration)
		}
		return c, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int), nil
}

func (r *notificationRepository) MarkNotificationAsRead(ctx context.Context, id, userID string) error {
	if err := r.q.MarkNotificationAsRead(ctx, sqlc.MarkNotificationAsReadParams{
		ID:     id,
		UserID: userID,
	}); err != nil {
		return err
	}
	r.InvalidateNotificationCache(ctx, userID)
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("notification", "id", id))
	}
	return nil
}

func (r *notificationRepository) MarkAllNotificationsAsRead(ctx context.Context, userID string) error {
	if err := r.q.MarkAllNotificationsAsRead(ctx, userID); err != nil {
		return err
	}
	r.InvalidateNotificationCache(ctx, userID)
	return nil
}

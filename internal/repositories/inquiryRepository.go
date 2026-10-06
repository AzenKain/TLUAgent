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

// InquiryRepository manages student inquiries and lecturer advice lifecycle.
type InquiryRepository interface {
	CreateInquiry(ctx context.Context, inq *models.AdvisoryInquiry) (*models.AdvisoryInquiry, error)
	GetInquiryByID(ctx context.Context, id string) (*models.AdvisoryInquiry, error)
	GetInquiriesByIDs(ctx context.Context, ids []string) ([]*models.AdvisoryInquiry, error)
	ListInquiriesByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.AdvisoryInquiry, int, error)
	ListInquiries(ctx context.Context, status, search string, limit, offset int) ([]*models.AdvisoryInquiry, int, error)
	AnswerInquiry(ctx context.Context, id, teacherID, teacherName, teacherReply string, answeredAt time.Time, chunkID string) (*models.AdvisoryInquiry, error)
	ExpireInquiry(ctx context.Context, id, supersededByDocID, reason string, expiredAt time.Time) (*models.AdvisoryInquiry, error)
	ListActiveAnsweredInquiries(ctx context.Context) ([]*models.AdvisoryInquiry, error)
	InvalidateInquiryCache(ctx context.Context, id, userID string)
	WithTx(tx *sql.Tx) InquiryRepository
}

type inquiryRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

// NewInquiryRepository creates a new instance of InquiryRepository.
func NewInquiryRepository(db sqlc.DBTX, c cache.Cache) InquiryRepository {
	return &inquiryRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *inquiryRepository) WithTx(tx *sql.Tx) InquiryRepository {
	return &inquiryRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sfg:  &singleflight.Group{},
	}
}

func (r *inquiryRepository) InvalidateInquiryCache(ctx context.Context, id, userID string) {
	if r.c == nil {
		return
	}
	keys := []string{
		cache.BuildKey("inquiry", "id", id),
		cache.BuildKey("inquiries", "user", userID),
		cache.BuildKey("inquiries", "active_answered"),
		cache.BuildKey("inquiries", "list"),
	}
	_ = r.c.Del(ctx, keys...)
}

func (r *inquiryRepository) CreateInquiry(ctx context.Context, inq *models.AdvisoryInquiry) (*models.AdvisoryInquiry, error) {
	now := time.Now()
	row, err := r.q.CreateInquiry(ctx, sqlc.CreateInquiryParams{
		ID:             inq.ID,
		UserID:         inq.UserID,
		ConversationID: models.NullString(inq.ConversationID),
		StudentName:    inq.StudentName,
		StudentCode:    inq.StudentCode,
		StudentClass:   inq.StudentClass,
		Question:       inq.Question,
		Context:        inq.Context,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		return nil, err
	}
	r.InvalidateInquiryCache(ctx, inq.ID, inq.UserID)
	return models.FromSQLCInquiry(&row), nil
}

func (r *inquiryRepository) GetInquiryByID(ctx context.Context, id string) (*models.AdvisoryInquiry, error) {
	cacheKey := cache.BuildKey("inquiry", "id", id)
	if r.c != nil && !r.inTx {
		var cached models.AdvisoryInquiry
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		row, err := r.q.GetInquiryByID(ctx, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, err
		}
		item := models.FromSQLCInquiry(&row)
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
	return v.(*models.AdvisoryInquiry), nil
}

func (r *inquiryRepository) GetInquiriesByIDs(ctx context.Context, ids []string) ([]*models.AdvisoryInquiry, error) {
	if len(ids) == 0 {
		return []*models.AdvisoryInquiry{}, nil
	}

	results := make([]*models.AdvisoryInquiry, len(ids))
	missingIDs := make([]string, 0, len(ids))
	idToIndex := make(map[string]int, len(ids))

	for i, id := range ids {
		idToIndex[id] = i
		cacheKey := cache.BuildKey("inquiry", "id", id)
		var cached models.AdvisoryInquiry
		if r.c != nil && !r.inTx && r.c.Get(ctx, cacheKey, &cached) == nil {
			results[i] = &cached
		} else {
			missingIDs = append(missingIDs, id)
		}
	}

	if len(missingIDs) == 0 {
		return results, nil
	}

	sfgKey := fmt.Sprintf("inquiries:get_by_ids:%v", missingIDs)
	fetchedData, err, _ := r.sfg.Do(sfgKey, func() (any, error) {
		rows, err := r.q.GetInquiriesByIDs(ctx, missingIDs)
		if err != nil {
			return nil, err
		}
		return rows, nil
	})
	if err != nil {
		return nil, err
	}

	rows := fetchedData.([]sqlc.AdvisoryInquiry)
	for _, row := range rows {
		item := models.FromSQLCInquiry(&row)
		idx, ok := idToIndex[item.ID]
		if ok {
			results[idx] = item
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cache.BuildKey("inquiry", "id", item.ID), item, constants.NormalCacheDuration)
		}
	}

	finalResults := make([]*models.AdvisoryInquiry, 0, len(results))
	for _, res := range results {
		if res != nil {
			finalResults = append(finalResults, res)
		}
	}

	return finalResults, nil
}

func (r *inquiryRepository) ListInquiriesByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.AdvisoryInquiry, int, error) {
	totalKey := cache.BuildKey("inquiries", "user_count", userID)
	var total int
	if r.c != nil && !r.inTx {
		_ = r.c.Get(ctx, totalKey, &total)
	}
	if total == 0 {
		v, err, _ := r.sfg.Do(totalKey, func() (any, error) {
			count, err := r.q.CountInquiriesByUserID(ctx, userID)
			if err != nil {
				return 0, err
			}
			return int(count), nil
		})
		if err != nil {
			return nil, 0, err
		}
		total = v.(int)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, totalKey, total, constants.NormalCacheDuration)
		}
	}

	ids, err := r.q.ListInquiryIDsByUserID(ctx, sqlc.ListInquiryIDsByUserIDParams{
		UserID: userID,
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	items, err := r.GetInquiriesByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *inquiryRepository) ListInquiries(ctx context.Context, status, search string, limit, offset int) ([]*models.AdvisoryInquiry, int, error) {
	countParams := sqlc.CountInquiriesParams{
		Status: models.NullString(status),
		Search: models.NullString(search),
	}
	total, err := r.q.CountInquiries(ctx, countParams)
	if err != nil {
		return nil, 0, err
	}

	ids, err := r.q.ListInquiryIDs(ctx, sqlc.ListInquiryIDsParams{
		Status: models.NullString(status),
		Search: models.NullString(search),
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	items, err := r.GetInquiriesByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	return items, int(total), nil
}

func (r *inquiryRepository) AnswerInquiry(ctx context.Context, id, teacherID, teacherName, teacherReply string, answeredAt time.Time, chunkID string) (*models.AdvisoryInquiry, error) {
	now := time.Now()
	row, err := r.q.AnswerInquiry(ctx, sqlc.AnswerInquiryParams{
		TeacherID:        models.NullString(teacherID),
		TeacherName:      teacherName,
		TeacherReply:     teacherReply,
		AnsweredAt:       sql.NullTime{Time: answeredAt, Valid: true},
		KnowledgeChunkID: models.NullString(chunkID),
		UpdatedAt:        now,
		ID:               id,
	})
	if err != nil {
		return nil, err
	}
	item := models.FromSQLCInquiry(&row)
	r.InvalidateInquiryCache(ctx, id, item.UserID)
	return item, nil
}

func (r *inquiryRepository) ExpireInquiry(ctx context.Context, id, supersededByDocID, reason string, expiredAt time.Time) (*models.AdvisoryInquiry, error) {
	now := time.Now()
	row, err := r.q.ExpireInquiry(ctx, sqlc.ExpireInquiryParams{
		SupersededByDocID: models.NullString(supersededByDocID),
		ExpiredReason:     reason,
		ExpiredAt:         sql.NullTime{Time: expiredAt, Valid: true},
		UpdatedAt:         now,
		ID:                id,
	})
	if err != nil {
		return nil, err
	}
	item := models.FromSQLCInquiry(&row)
	r.InvalidateInquiryCache(ctx, id, item.UserID)
	return item, nil
}

func (r *inquiryRepository) ListActiveAnsweredInquiries(ctx context.Context) ([]*models.AdvisoryInquiry, error) {
	cacheKey := cache.BuildKey("inquiries", "active_answered")
	if r.c != nil && !r.inTx {
		var cached []*models.AdvisoryInquiry
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		rows, err := r.q.ListActiveAnsweredInquiries(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]*models.AdvisoryInquiry, 0, len(rows))
		for _, row := range rows {
			items = append(items, models.FromSQLCInquiry(&row))
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, items, constants.NormalCacheDuration)
		}
		return items, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.AdvisoryInquiry), nil
}

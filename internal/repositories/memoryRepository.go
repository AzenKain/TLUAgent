package repositories

import (
	"context"
	"database/sql"
	"errors"

	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
)

// MemoryRepository manages persistent academic profile and student memory storage.
type MemoryRepository interface {
	GetStudentProfile(ctx context.Context, userID string) (*models.StudentProfile, error)
	UpsertStudentProfile(ctx context.Context, profile *models.StudentProfile) (*models.StudentProfile, error)

	GetUserMemoryByID(ctx context.Context, id string) (*models.UserMemoryItem, error)
	GetUserMemoriesByIDs(ctx context.Context, ids []string) ([]*models.UserMemoryItem, error)
	ListUserMemories(ctx context.Context, userID string) ([]*models.UserMemoryItem, error)
	UpsertUserMemory(ctx context.Context, memory *models.UserMemoryItem) (*models.UserMemoryItem, error)
	DeleteUserMemory(ctx context.Context, id, userID string) error
	CountUserMemories(ctx context.Context, userID string) (int, error)
	DeleteStudentProfile(ctx context.Context, userID string) error
	ClearAllUserMemories(ctx context.Context, userID string) error

	InvalidateProfileCache(ctx context.Context, userID string)
	InvalidateMemoryCache(ctx context.Context, id, userID string)
	WithTx(tx *sql.Tx) MemoryRepository
}

type memoryRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

// NewMemoryRepository creates a new instance of MemoryRepository.
func NewMemoryRepository(db sqlc.DBTX, c cache.Cache) MemoryRepository {
	return &memoryRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *memoryRepository) WithTx(tx *sql.Tx) MemoryRepository {
	return &memoryRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sfg:  &singleflight.Group{},
	}
}

func (r *memoryRepository) GetStudentProfile(ctx context.Context, userID string) (*models.StudentProfile, error) {
	key := cache.BuildKey("student_profile", "user_id", userID)
	if r.c != nil && !r.inTx {
		var cached models.StudentProfile
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetStudentProfileByUserID(ctx, userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, err
		}
		profile := (&models.StudentProfile{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, profile, constants.NormalCacheDuration)
		}
		return profile, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*models.StudentProfile), nil
}

func (r *memoryRepository) UpsertStudentProfile(ctx context.Context, profile *models.StudentProfile) (*models.StudentProfile, error) {
	params := profile.ToSqlcUpsertParams()
	row, err := r.q.UpsertStudentProfile(ctx, params)
	if err != nil {
		return nil, err
	}
	r.InvalidateProfileCache(ctx, profile.UserID)
	return (&models.StudentProfile{}).FromSqlc(row), nil
}

func (r *memoryRepository) GetUserMemoryByID(ctx context.Context, id string) (*models.UserMemoryItem, error) {
	key := cache.BuildKey("user_memory", "id", id)
	if r.c != nil && !r.inTx {
		var cached models.UserMemoryItem
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetUserMemoryByID(ctx, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, err
		}
		item := (&models.UserMemoryItem{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, item, constants.NormalCacheDuration)
		}
		return item, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*models.UserMemoryItem), nil
}

func (r *memoryRepository) GetUserMemoriesByIDs(ctx context.Context, ids []string) ([]*models.UserMemoryItem, error) {
	if len(ids) == 0 {
		return []*models.UserMemoryItem{}, nil
	}

	resultMap := make(map[string]*models.UserMemoryItem, len(ids))
	var missingIDs []string

	for _, id := range ids {
		key := cache.BuildKey("user_memory", "id", id)
		if r.c != nil && !r.inTx {
			var cached models.UserMemoryItem
			if err := r.c.Get(ctx, key, &cached); err == nil {
				resultMap[id] = &cached
				continue
			}
		}
		missingIDs = append(missingIDs, id)
	}

	if len(missingIDs) > 0 {
		rows, err := r.q.GetUserMemoriesByIDs(ctx, missingIDs)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			item := (&models.UserMemoryItem{}).FromSqlc(row)
			resultMap[item.ID] = item
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, cache.BuildKey("user_memory", "id", item.ID), item, constants.NormalCacheDuration)
			}
		}
	}

	out := make([]*models.UserMemoryItem, 0, len(ids))
	for _, id := range ids {
		if item, ok := resultMap[id]; ok {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *memoryRepository) ListUserMemories(ctx context.Context, userID string) ([]*models.UserMemoryItem, error) {
	key := cache.BuildKey("user_memories_ids", "user_id", userID)
	var ids []string
	if r.c != nil && !r.inTx {
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return r.GetUserMemoriesByIDs(ctx, ids)
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		foundIDs, err := r.q.ListUserMemoryIDsByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, foundIDs, constants.NormalCacheDuration)
		}
		return foundIDs, nil
	})
	if err != nil {
		return nil, err
	}

	ids = v.([]string)
	return r.GetUserMemoriesByIDs(ctx, ids)
}

func (r *memoryRepository) UpsertUserMemory(ctx context.Context, memory *models.UserMemoryItem) (*models.UserMemoryItem, error) {
	params := memory.ToSqlcUpsertParams()
	row, err := r.q.UpsertUserMemory(ctx, params)
	if err != nil {
		return nil, err
	}
	r.InvalidateMemoryCache(ctx, memory.ID, memory.UserID)
	return (&models.UserMemoryItem{}).FromSqlc(row), nil
}

func (r *memoryRepository) DeleteUserMemory(ctx context.Context, id, userID string) error {
	rows, err := r.q.DeleteUserMemory(ctx, sqlc.DeleteUserMemoryParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	r.InvalidateMemoryCache(ctx, id, userID)
	return nil
}

func (r *memoryRepository) CountUserMemories(ctx context.Context, userID string) (int, error) {
	key := cache.BuildKey("user_memories_count", "user_id", userID)
	if r.c != nil && !r.inTx {
		var cnt int
		if err := r.c.Get(ctx, key, &cnt); err == nil {
			return cnt, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		count, err := r.q.CountUserMemories(ctx, userID)
		if err != nil {
			return 0, err
		}
		intCount := int(count)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, intCount, constants.NormalCacheDuration)
		}
		return intCount, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int), nil
}

func (r *memoryRepository) DeleteStudentProfile(ctx context.Context, userID string) error {
	_, err := r.q.DeleteStudentProfile(ctx, userID)
	if err != nil {
		return err
	}
	r.InvalidateProfileCache(ctx, userID)
	return nil
}

func (r *memoryRepository) ClearAllUserMemories(ctx context.Context, userID string) error {
	_, err := r.q.ClearAllUserMemories(ctx, userID)
	if err != nil {
		return err
	}
	r.InvalidateMemoryCache(ctx, "", userID)
	return nil
}

func (r *memoryRepository) InvalidateProfileCache(ctx context.Context, userID string) {
	if r.c == nil {
		return
	}
	_ = r.c.Del(ctx, cache.BuildKey("student_profile", "user_id", userID))
}

func (r *memoryRepository) InvalidateMemoryCache(ctx context.Context, id, userID string) {
	if r.c == nil {
		return
	}
	if id != "" {
		_ = r.c.Del(ctx, cache.BuildKey("user_memory", "id", id))
	}
	if userID != "" {
		_ = r.c.Del(ctx,
			cache.BuildKey("user_memories_ids", "user_id", userID),
			cache.BuildKey("user_memories_count", "user_id", userID),
		)
	}
}

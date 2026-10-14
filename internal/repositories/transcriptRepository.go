package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
)

// TranscriptRepository defines persistence operations for student academic course grades.
type TranscriptRepository interface {
	ListByUserID(ctx context.Context, userID string) ([]*models.StudentCourseGradeRecord, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.StudentCourseGradeRecord, error)
	GetByID(ctx context.Context, id string) (*models.StudentCourseGradeRecord, error)
	GetByUserAndCourse(ctx context.Context, userID, courseCode string) (*models.StudentCourseGradeRecord, error)
	UpsertGrade(ctx context.Context, params sqlc.UpsertTranscriptGradeParams) (*models.StudentCourseGradeRecord, error)
	BatchUpsert(ctx context.Context, userID string, params []sqlc.UpsertTranscriptGradeParams) ([]*models.StudentCourseGradeRecord, error)
	DeleteGrade(ctx context.Context, id, userID string) error
	DeleteByUserID(ctx context.Context, userID string) error
	CountByUserID(ctx context.Context, userID string) (int64, error)
	InvalidateUserCache(ctx context.Context, userID string)
	WithTx(tx *sql.Tx) TranscriptRepository
}

type transcriptRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	db   *sql.DB
	inTx bool
	sfg  *singleflight.Group
}

// NewTranscriptRepository initializes a new transcript repository instance.
func NewTranscriptRepository(db sqlc.DBTX, c cache.Cache) TranscriptRepository {
	var sqlDB *sql.DB
	if concreteDB, ok := db.(*sql.DB); ok {
		sqlDB = concreteDB
	}
	return &transcriptRepository{
		q:   sqlc.New(db),
		c:   c,
		db:  sqlDB,
		sfg: &singleflight.Group{},
	}
}

func (r *transcriptRepository) WithTx(tx *sql.Tx) TranscriptRepository {
	return &transcriptRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		db:   r.db,
		inTx: true,
		sfg:  &singleflight.Group{},
	}
}

func (r *transcriptRepository) InvalidateUserCache(ctx context.Context, userID string) {
	if r.c == nil {
		return
	}
	_ = r.c.Del(ctx, cache.BuildKey("trans", "user_ids", userID))
	_ = r.c.Del(ctx, cache.BuildKey("trans", "user_count", userID))
}

func (r *transcriptRepository) InvalidateItemCache(ctx context.Context, id, userID, courseCode string) {
	if r.c == nil {
		return
	}
	_ = r.c.Del(ctx, cache.BuildKey("trans", "item", id))
	_ = r.c.Del(ctx, cache.BuildKey("trans", "user_course", userID, courseCode))
	r.InvalidateUserCache(ctx, userID)
}

func (r *transcriptRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.StudentCourseGradeRecord, error) {
	if len(ids) == 0 {
		return []*models.StudentCourseGradeRecord{}, nil
	}

	resultMap := make(map[string]*models.StudentCourseGradeRecord, len(ids))
	var missingIDs []string

	for _, id := range ids {
		key := cache.BuildKey("trans", "item", id)
		if r.c != nil && !r.inTx {
			var record models.StudentCourseGradeRecord
			if err := r.c.Get(ctx, key, &record); err == nil {
				resultMap[id] = &record
				continue
			}
		}
		missingIDs = append(missingIDs, id)
	}

	if len(missingIDs) > 0 {
		sfgKey := "trans:get_by_ids:" + strings.Join(missingIDs, ",")
		v, err, _ := r.sfg.Do(sfgKey, func() (any, error) {
			rows, err := r.q.GetTranscriptGradesByIDs(ctx, missingIDs)
			if err != nil {
				return nil, err
			}
			fetched := make([]*models.StudentCourseGradeRecord, 0, len(rows))
			for _, row := range rows {
				record := (&models.StudentCourseGradeRecord{}).FromSqlc(row)
				if r.c != nil && !r.inTx {
					_ = r.c.Set(ctx, cache.BuildKey("trans", "item", record.ID), record, constants.NormalCacheDuration)
				}
				fetched = append(fetched, record)
			}
			return fetched, nil
		})
		if err != nil {
			return nil, err
		}
		for _, record := range v.([]*models.StudentCourseGradeRecord) {
			resultMap[record.ID] = record
		}
	}

	out := make([]*models.StudentCourseGradeRecord, 0, len(ids))
	for _, id := range ids {
		if record, ok := resultMap[id]; ok {
			out = append(out, record)
		}
	}
	return out, nil
}

func (r *transcriptRepository) ListByUserID(ctx context.Context, userID string) ([]*models.StudentCourseGradeRecord, error) {
	key := cache.BuildKey("trans", "user_ids", userID)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cachedIDs []string
			if err := r.c.Get(ctx, key, &cachedIDs); err == nil && len(cachedIDs) > 0 {
				return cachedIDs, nil
			}
		}

		ids, err := r.q.ListTranscriptGradeIDsByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}

		if r.c != nil && !r.inTx && len(ids) > 0 {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}

	ids := val.([]string)
	return r.GetByIDs(ctx, ids)
}

func (r *transcriptRepository) GetByID(ctx context.Context, id string) (*models.StudentCourseGradeRecord, error) {
	key := cache.BuildKey("trans", "item", id)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached models.StudentCourseGradeRecord
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return &cached, nil
			}
		}

		row, err := r.q.GetTranscriptGradeByID(ctx, id)
		if err != nil {
			return nil, err
		}

		record := (&models.StudentCourseGradeRecord{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, record, constants.NormalCacheDuration)
		}
		return record, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.StudentCourseGradeRecord), nil
}

func (r *transcriptRepository) GetByUserAndCourse(ctx context.Context, userID, courseCode string) (*models.StudentCourseGradeRecord, error) {
	key := cache.BuildKey("trans", "user_course", userID, courseCode)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached models.StudentCourseGradeRecord
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return &cached, nil
			}
		}

		row, err := r.q.GetTranscriptGradeByUserAndCourse(ctx, sqlc.GetTranscriptGradeByUserAndCourseParams{
			UserID:     userID,
			CourseCode: courseCode,
		})
		if err != nil {
			return nil, err
		}

		record := (&models.StudentCourseGradeRecord{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, record, constants.NormalCacheDuration)
		}
		return record, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.StudentCourseGradeRecord), nil
}

func (r *transcriptRepository) UpsertGrade(ctx context.Context, params sqlc.UpsertTranscriptGradeParams) (*models.StudentCourseGradeRecord, error) {
	row, err := r.q.UpsertTranscriptGrade(ctx, params)
	if err != nil {
		return nil, err
	}
	record := (&models.StudentCourseGradeRecord{}).FromSqlc(row)
	r.InvalidateItemCache(ctx, record.ID, record.UserID, record.CourseCode)
	return record, nil
}

func (r *transcriptRepository) BatchUpsert(ctx context.Context, userID string, params []sqlc.UpsertTranscriptGradeParams) ([]*models.StudentCourseGradeRecord, error) {
	if len(params) == 0 {
		return []*models.StudentCourseGradeRecord{}, nil
	}

	if r.inTx || r.db == nil {
		var results []*models.StudentCourseGradeRecord
		for _, p := range params {
			row, err := r.q.UpsertTranscriptGrade(ctx, p)
			if err != nil {
				return nil, err
			}
			rec := (&models.StudentCourseGradeRecord{}).FromSqlc(row)
			results = append(results, rec)
			r.InvalidateItemCache(ctx, rec.ID, rec.UserID, rec.CourseCode)
		}
		r.InvalidateUserCache(ctx, userID)
		return results, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	qtx := r.q.WithTx(tx)
	results := make([]*models.StudentCourseGradeRecord, 0, len(params))
	for _, p := range params {
		row, err := qtx.UpsertTranscriptGrade(ctx, p)
		if err != nil {
			return nil, fmt.Errorf("upsert transcript grade %s: %w", p.CourseCode, err)
		}
		rec := (&models.StudentCourseGradeRecord{}).FromSqlc(row)
		results = append(results, rec)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit batch transaction: %w", err)
	}

	for _, rec := range results {
		r.InvalidateItemCache(ctx, rec.ID, rec.UserID, rec.CourseCode)
	}
	r.InvalidateUserCache(ctx, userID)
	return results, nil
}

func (r *transcriptRepository) DeleteGrade(ctx context.Context, id, userID string) error {
	rec, _ := r.GetByID(ctx, id)
	if err := r.q.DeleteTranscriptGrade(ctx, id); err != nil {
		return err
	}
	courseCode := ""
	if rec != nil {
		courseCode = rec.CourseCode
	}
	r.InvalidateItemCache(ctx, id, userID, courseCode)
	return nil
}

func (r *transcriptRepository) DeleteByUserID(ctx context.Context, userID string) error {
	if err := r.q.DeleteTranscriptByUserID(ctx, userID); err != nil {
		return err
	}
	r.InvalidateUserCache(ctx, userID)
	return nil
}

func (r *transcriptRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	key := cache.BuildKey("trans", "user_count", userID)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached int64
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}

		cnt, err := r.q.CountTranscriptGradesByUserID(ctx, userID)
		if err != nil {
			return int64(0), err
		}

		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, cnt, constants.NormalCacheDuration)
		}
		return cnt, nil
	})
	if err != nil {
		return 0, err
	}
	return val.(int64), nil
}

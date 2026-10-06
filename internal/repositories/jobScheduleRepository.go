package repositories

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/jsonx"
)

type JobScheduleRepository interface {
	Get(ctx context.Context, id string) (*models.JobScheduleEntity, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.JobScheduleEntity, error)
	List(ctx context.Context) ([]*models.JobScheduleEntity, error)
	ListDue(ctx context.Context, now time.Time) ([]*models.JobScheduleEntity, error)
	Create(ctx context.Context, params sqlc.CreateJobScheduleParams) (*models.JobScheduleEntity, error)
	Update(ctx context.Context, params sqlc.UpdateJobScheduleParams) (*models.JobScheduleEntity, error)
	Delete(ctx context.Context, id string) error
	Claim(ctx context.Context, id, jobID string, now, nextRun time.Time) (bool, error)
	ReleaseClaim(ctx context.Context, id, jobID string, retryAt time.Time) error
	InvalidateScheduleCache(ctx context.Context, id string)
	WithTx(tx *sql.Tx) JobScheduleRepository
}

type jobScheduleRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

func NewJobScheduleRepository(db sqlc.DBTX, c cache.Cache) JobScheduleRepository {
	return &jobScheduleRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *jobScheduleRepository) WithTx(tx *sql.Tx) JobScheduleRepository {
	return &jobScheduleRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sfg:  &singleflight.Group{},
	}
}

func (r *jobScheduleRepository) InvalidateScheduleCache(ctx context.Context, id string) {
	if r.c == nil {
		return
	}
	_ = r.c.Del(ctx, cache.BuildKey("job_schedule", "id", id))
	_ = r.c.Del(ctx, cache.BuildKey("job_schedules", "all"))
}

func (r *jobScheduleRepository) Get(ctx context.Context, id string) (*models.JobScheduleEntity, error) {
	key := cache.BuildKey("job_schedule", "id", id)
	if r.c != nil && !r.inTx {
		var s models.JobScheduleEntity
		if err := r.c.Get(ctx, key, &s); err == nil {
			return &s, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetJobScheduleByID(ctx, id)
		if err != nil {
			return nil, err
		}
		entity := (&models.JobScheduleEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, entity, constants.NormalCacheDuration)
		}
		return entity, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.JobScheduleEntity), nil
}

func (r *jobScheduleRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.JobScheduleEntity, error) {
	if len(ids) == 0 {
		return []*models.JobScheduleEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("job_schedule", "id", id)
	}

	schedules := make([]*models.JobScheduleEntity, 0, len(ids))
	missingIds := []string{}
	missingKeys := []string{}

	if r.c != nil && !r.inTx {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var s models.JobScheduleEntity
				if err := jsonx.Unmarshal(bytes, &s); err == nil {
					schedules = append(schedules, &s)
					continue
				}
			}
			missingIds = append(missingIds, ids[i])
			missingKeys = append(missingKeys, keys[i])
		}
	} else {
		missingIds = ids
		missingKeys = keys
	}

	if len(missingIds) > 0 {
		sfgKey := "get_job_schedules_by_ids:" + strings.Join(missingIds, ",")
		v, err, _ := r.sfg.Do(sfgKey, func() (any, error) {
			rows, err := r.q.GetJobSchedulesByIDs(ctx, missingIds)
			if err != nil {
				return nil, err
			}
			dbSchedules := make([]*models.JobScheduleEntity, len(rows))
			toCache := make(map[string]any, len(rows))
			for i, row := range rows {
				entity := (&models.JobScheduleEntity{}).FromSqlc(row)
				dbSchedules[i] = entity
				toCache[cache.BuildKey("job_schedule", "id", row.ID)] = entity
			}
			if r.c != nil && !r.inTx && len(toCache) > 0 {
				_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
			}
			return dbSchedules, nil
		})
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, v.([]*models.JobScheduleEntity)...)
	}

	idMap := make(map[string]*models.JobScheduleEntity, len(schedules))
	for _, s := range schedules {
		idMap[s.ID] = s
	}
	ordered := make([]*models.JobScheduleEntity, 0, len(ids))
	for _, id := range ids {
		if s, ok := idMap[id]; ok {
			ordered = append(ordered, s)
		}
	}
	return ordered, nil
}

func (r *jobScheduleRepository) List(ctx context.Context) ([]*models.JobScheduleEntity, error) {
	ids, err := r.q.ListJobScheduleIDs(ctx)
	if err != nil {
		return nil, err
	}
	return r.GetByIDs(ctx, ids)
}

func (r *jobScheduleRepository) ListDue(ctx context.Context, now time.Time) ([]*models.JobScheduleEntity, error) {
	ids, err := r.q.ListDueJobScheduleIDs(ctx, now)
	if err != nil {
		return nil, err
	}
	return r.GetByIDs(ctx, ids)
}

func (r *jobScheduleRepository) Create(ctx context.Context, params sqlc.CreateJobScheduleParams) (*models.JobScheduleEntity, error) {
	row, err := r.q.CreateJobSchedule(ctx, params)
	if err != nil {
		return nil, err
	}
	entity := (&models.JobScheduleEntity{}).FromSqlc(row)
	r.InvalidateScheduleCache(ctx, entity.ID)
	return entity, nil
}

func (r *jobScheduleRepository) Update(ctx context.Context, params sqlc.UpdateJobScheduleParams) (*models.JobScheduleEntity, error) {
	row, err := r.q.UpdateJobSchedule(ctx, params)
	if err != nil {
		return nil, err
	}
	entity := (&models.JobScheduleEntity{}).FromSqlc(row)
	r.InvalidateScheduleCache(ctx, entity.ID)
	return entity, nil
}

func (r *jobScheduleRepository) Delete(ctx context.Context, id string) error {
	err := r.q.DeleteJobSchedule(ctx, id)
	if err == nil {
		r.InvalidateScheduleCache(ctx, id)
	}
	return err
}

func (r *jobScheduleRepository) Claim(ctx context.Context, id, jobID string, now, nextRun time.Time) (bool, error) {
	rows, err := r.q.ClaimJobSchedule(ctx, sqlc.ClaimJobScheduleParams{
		Now:       sql.NullTime{Time: now, Valid: true},
		JobID:     sql.NullString{String: jobID, Valid: true},
		NextRunAt: nextRun,
		ID:        id,
	})
	if err != nil {
		return false, err
	}
	if rows > 0 {
		r.InvalidateScheduleCache(ctx, id)
		return true, nil
	}
	return false, nil
}

func (r *jobScheduleRepository) ReleaseClaim(ctx context.Context, id, jobID string, retryAt time.Time) error {
	_, err := r.q.ReleaseJobScheduleClaim(ctx, sqlc.ReleaseJobScheduleClaimParams{
		ID:       id,
		JobID:    sql.NullString{String: jobID, Valid: true},
		RetryAt:  retryAt,
	})
	if err == nil {
		r.InvalidateScheduleCache(ctx, id)
	}
	return err
}

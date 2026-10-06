package repositories

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/convert"
	"tluagent-web/pkg/jsonx"
)

type JobRepository interface {
	GetJob(ctx context.Context, id string) (*models.JobEntity, error)
	GetJobsByIDs(ctx context.Context, ids []string) ([]*models.JobEntity, error)
	CreateJob(ctx context.Context, id, jobType, status, payload string) (*models.JobEntity, error)
	UpdateJobStatus(ctx context.Context, id, status, errorMsg string) (*models.JobEntity, error)
	UpdateJobProgress(ctx context.Context, id string, progress, total int64) (*models.JobEntity, error)
	ListJobs(ctx context.Context, status, jobType string, limit, offset int64) ([]*models.JobEntity, int64, error)
	MarkRunningJobsInterrupted(ctx context.Context) error
	PruneFinishedJobs(ctx context.Context, keep int64) error
	InvalidateJobCache(ctx context.Context, id string)
	WithTx(tx *sql.Tx) JobRepository
}

type jobRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

func NewJobRepository(db sqlc.DBTX, c cache.Cache) JobRepository {
	return &jobRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *jobRepository) WithTx(tx *sql.Tx) JobRepository {
	return &jobRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sfg:  &singleflight.Group{},
	}
}

func (r *jobRepository) InvalidateJobCache(ctx context.Context, id string) {
	if r.c == nil {
		return
	}
	_ = r.c.Del(ctx, cache.BuildKey("job", "id", id))
}

func (r *jobRepository) GetJob(ctx context.Context, id string) (*models.JobEntity, error) {
	key := cache.BuildKey("job", "id", id)
	if r.c != nil && !r.inTx {
		var job models.JobEntity
		if err := r.c.Get(ctx, key, &job); err == nil {
			return &job, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetJobByID(ctx, id)
		if err != nil {
			return nil, err
		}
		job := (&models.JobEntity{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, job, constants.NormalCacheDuration)
		}
		return job, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.JobEntity), nil
}

func (r *jobRepository) GetJobsByIDs(ctx context.Context, ids []string) ([]*models.JobEntity, error) {
	if len(ids) == 0 {
		return []*models.JobEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("job", "id", id)
	}

	jobs := make([]*models.JobEntity, 0, len(ids))
	missingIds := []string{}
	missingKeys := []string{}

	if r.c != nil && !r.inTx {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var job models.JobEntity
				if err := jsonx.Unmarshal(bytes, &job); err == nil {
					jobs = append(jobs, &job)
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
		sfgKey := "get_jobs_by_ids:" + strings.Join(missingIds, ",")
		v, err, _ := r.sfg.Do(sfgKey, func() (any, error) {
			rows, err := r.q.GetJobsByIDs(ctx, missingIds)
			if err != nil {
				return nil, err
			}
			dbJobs := make([]*models.JobEntity, len(rows))
			toCache := make(map[string]any, len(rows))
			for i, row := range rows {
				job := (&models.JobEntity{}).FromSqlc(row)
				dbJobs[i] = job
				toCache[cache.BuildKey("job", "id", row.ID)] = job
			}
			if r.c != nil && !r.inTx && len(toCache) > 0 {
				_ = r.c.MSet(ctx, toCache, constants.NormalCacheDuration)
			}
			return dbJobs, nil
		})
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, v.([]*models.JobEntity)...)
	}

	idMap := make(map[string]*models.JobEntity, len(jobs))
	for _, job := range jobs {
		idMap[job.ID] = job
	}
	orderedJobs := make([]*models.JobEntity, 0, len(ids))
	for _, id := range ids {
		if job, ok := idMap[id]; ok {
			orderedJobs = append(orderedJobs, job)
		}
	}
	return orderedJobs, nil
}

func (r *jobRepository) CreateJob(ctx context.Context, id, jobType, status, payload string) (*models.JobEntity, error) {
	row, err := r.q.CreateJob(ctx, sqlc.CreateJobParams{
		ID:          id,
		Type:        jobType,
		Status:      status,
		Progress:    0,
		Total:       0,
		ErrorMsg:    sql.NullString{},
		PayloadJson: convert.StringToNullString(payload),
	})
	if err != nil {
		return nil, err
	}
	job := (&models.JobEntity{}).FromSqlc(row)
	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, cache.BuildKey("job", "id", id), job, constants.NormalCacheDuration)
	}
	return job, nil
}

func (r *jobRepository) UpdateJobStatus(ctx context.Context, id, status, errorMsg string) (*models.JobEntity, error) {
	row, err := r.q.UpdateJobStatus(ctx, sqlc.UpdateJobStatusParams{
		Status:   status,
		ErrorMsg: convert.StringToNullString(errorMsg),
		ID:       id,
	})
	if err != nil {
		return nil, err
	}
	r.InvalidateJobCache(ctx, id)
	return (&models.JobEntity{}).FromSqlc(row), nil
}

func (r *jobRepository) UpdateJobProgress(ctx context.Context, id string, progress, total int64) (*models.JobEntity, error) {
	row, err := r.q.UpdateJobProgress(ctx, sqlc.UpdateJobProgressParams{
		Progress: progress,
		Total:    total,
		ID:       id,
	})
	if err != nil {
		return nil, err
	}
	r.InvalidateJobCache(ctx, id)
	return (&models.JobEntity{}).FromSqlc(row), nil
}

func (r *jobRepository) ListJobs(ctx context.Context, status, jobType string, limit, offset int64) ([]*models.JobEntity, int64, error) {
	countKey := cache.BuildKey("job_count", status, jobType)
	var total int64
	if r.c != nil && !r.inTx {
		_ = r.c.Get(ctx, countKey, &total)
	}

	if total == 0 {
		v, err, _ := r.sfg.Do(countKey, func() (any, error) {
			c, err := r.q.CountJobs(ctx, sqlc.CountJobsParams{
				Status: status,
				Type:   jobType,
			})
			if err != nil {
				return int64(0), err
			}
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, countKey, c, constants.ShortCacheDuration)
			}
			return c, nil
		})
		if err != nil {
			return nil, 0, err
		}
		total = v.(int64)
	}

	if total == 0 {
		return []*models.JobEntity{}, 0, nil
	}

	ids, err := r.q.ListFilteredJobIDs(ctx, sqlc.ListFilteredJobIDsParams{
		Status: status,
		Type:   jobType,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	jobs, err := r.GetJobsByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
	})

	return jobs, total, nil
}

func (r *jobRepository) MarkRunningJobsInterrupted(ctx context.Context) error {
	return r.q.MarkRunningJobsInterrupted(ctx)
}

func (r *jobRepository) PruneFinishedJobs(ctx context.Context, keep int64) error {
	_, err := r.q.PruneFinishedJobs(ctx, keep)
	return err
}

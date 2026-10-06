package services

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
)

type JobScheduleService interface {
	List(ctx context.Context) ([]*response.JobScheduleResponse, error)
	Create(ctx context.Context, dto *request.UpsertJobScheduleRequest) (*response.JobScheduleResponse, error)
	Update(ctx context.Context, id string, dto *request.UpsertJobScheduleRequest) (*response.JobScheduleResponse, error)
	Delete(ctx context.Context, id string) error
	RunNow(ctx context.Context, id string) (*response.JobResponse, error)
	SeedDefaultSchedulesIfEmpty(ctx context.Context) error
	Start()
	Stop()
}

type jobScheduleService struct {
	repo      repositories.JobScheduleRepository
	jobs      *jobService
	stop      chan struct{}
	stopOnce  sync.Once
	wake      chan struct{}
	started   bool
	startedMu sync.Mutex
}

func NewJobScheduleService(repo repositories.JobScheduleRepository, jobs *jobService) *jobScheduleService {
	return &jobScheduleService{
		repo: repo,
		jobs: jobs,
		stop: make(chan struct{}),
		wake: make(chan struct{}, 1),
	}
}

func (s *jobScheduleService) List(ctx context.Context) ([]*response.JobScheduleResponse, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return models.JobSchedulesToResponse(rows), nil
}

func (s *jobScheduleService) validate(dto *request.UpsertJobScheduleRequest) error {
	if dto == nil || strings.TrimSpace(dto.Name) == "" {
		return apperrors.New(apperrors.ErrBadRequest, "schedule name is required")
	}
	if dto.TaskType == "" && dto.JobType != "" {
		dto.TaskType = dto.JobType
	}
	dto.TaskType = s.jobs.canonicalTaskType(dto.TaskType)
	if _, ok := s.jobs.tasks[dto.TaskType]; !ok {
		return apperrors.New(apperrors.ErrBadRequest, "unsupported task type")
	}
	if dto.IntervalMinutes <= 0 && dto.IntervalSec > 0 {
		dto.IntervalMinutes = (dto.IntervalSec + 59) / 60
	}
	if dto.IntervalMinutes < 1 || dto.IntervalMinutes > 525600 {
		return apperrors.New(apperrors.ErrBadRequest, "interval must be between 1 and 525600 minutes")
	}
	if dto.IsActive != nil {
		dto.Enabled = *dto.IsActive
	}
	return nil
}

func schedulePayload(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}

func (s *jobScheduleService) Create(ctx context.Context, dto *request.UpsertJobScheduleRequest) (*response.JobScheduleResponse, error) {
	if err := s.validate(dto); err != nil {
		return nil, err
	}
	now := time.Now()
	enabled := int64(0)
	if dto.Enabled {
		enabled = 1
	}
	entity, err := s.repo.Create(ctx, sqlc.CreateJobScheduleParams{
		ID:              "sched-" + uuid.Must(uuid.NewV7()).String()[:8],
		Name:            strings.TrimSpace(dto.Name),
		TaskType:        dto.TaskType,
		PayloadJson:     schedulePayload(dto.PayloadJSON),
		IntervalMinutes: dto.IntervalMinutes,
		Enabled:         enabled,
		NextRunAt:       now.Add(time.Duration(dto.IntervalMinutes) * time.Minute),
	})
	if err != nil {
		return nil, err
	}
	s.signal()
	return entity.ToResponse(), nil
}

func (s *jobScheduleService) Update(ctx context.Context, id string, dto *request.UpsertJobScheduleRequest) (*response.JobScheduleResponse, error) {
	if err := s.validate(dto); err != nil {
		return nil, err
	}
	enabled := int64(0)
	if dto.Enabled {
		enabled = 1
	}
	entity, err := s.repo.Update(ctx, sqlc.UpdateJobScheduleParams{
		Name:            strings.TrimSpace(dto.Name),
		TaskType:        dto.TaskType,
		PayloadJson:     schedulePayload(dto.PayloadJSON),
		IntervalMinutes: dto.IntervalMinutes,
		Enabled:         enabled,
		NextRunAt:       time.Now().Add(time.Duration(dto.IntervalMinutes) * time.Minute),
		ID:              id,
	})
	if err != nil {
		return nil, err
	}
	s.signal()
	return entity.ToResponse(), nil
}

func (s *jobScheduleService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *jobScheduleService) RunNow(ctx context.Context, id string) (*response.JobResponse, error) {
	schedule, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	payload := ""
	if schedule.PayloadJSON != nil {
		payload = *schedule.PayloadJSON
	}
	return s.jobs.Trigger(ctx, schedule.TaskType, payload)
}

func (s *jobScheduleService) SeedDefaultSchedulesIfEmpty(ctx context.Context) error {
	existing, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}

	defaults := []struct {
		ID       string
		Name     string
		TaskType string
		Interval int64
		Payload  string
	}{
		{
			ID:       "sched-crawl-notices",
			Name:     "Tự động cào Thông báo mới (TLU Notices Crawler)",
			TaskType: "crawl_notices",
			Interval: 120,
			Payload:  `{"limit": 20}`,
		},
		{
			ID:       "sched-crawl-news",
			Name:     "Tự động cào Tin tức học vụ (TLU News Crawler)",
			TaskType: "crawl_news",
			Interval: 240,
			Payload:  `{"limit": 10}`,
		},
		{
			ID:       "sched-crawl-static",
			Name:     "Cập nhật các trang tĩnh & Sổ tay sinh viên (Static Pages Crawler)",
			TaskType: "crawl_static_pages",
			Interval: 1440,
			Payload:  `{}`,
		},
		{
			ID:       "sched-pipeline-clean",
			Name:     "Làm sạch văn bản HTML & PDF sang Markdown (Document Clean Pipeline)",
			TaskType: "pipeline_clean_docs",
			Interval: 60,
			Payload:  `{}`,
		},
		{
			ID:       "sched-vectorize",
			Name:     "Tạo Vector Embedding & Nạp Đồ Thị Tri Thức (Knowledge Vectorizer)",
			TaskType: "vectorize_knowledge",
			Interval: 120,
			Payload:  `{}`,
		},
		{
			ID:       "sched-supersede-check",
			Name:     "Rà soát xung đột bãi bỏ quy chế (Multi-Agent Supersession Conflict Detector)",
			TaskType: "supersede_check",
			Interval: 180,
			Payload:  `{}`,
		},
	}

	now := time.Now()
	for _, def := range defaults {
		_, err := s.repo.Create(ctx, sqlc.CreateJobScheduleParams{
			ID:              def.ID,
			Name:            def.Name,
			TaskType:        def.TaskType,
			PayloadJson:     schedulePayload(def.Payload),
			IntervalMinutes: def.Interval,
			Enabled:         1,
			NextRunAt:       now.Add(time.Duration(def.Interval) * time.Minute),
		})
		if err != nil {
			log.Warn().Err(err).Str("id", def.ID).Msg("failed to seed default schedule")
		}
	}
	s.signal()
	return nil
}

func (s *jobScheduleService) Start() {
	s.startedMu.Lock()
	if s.started {
		s.startedMu.Unlock()
		return
	}
	s.started = true
	s.startedMu.Unlock()
	go s.loop()
}

func (s *jobScheduleService) Stop() {
	s.stopOnce.Do(func() { close(s.stop) })
}

func (s *jobScheduleService) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *jobScheduleService) loop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.runDue()
		case <-s.wake:
			s.runDue()
		}
	}
}

func (s *jobScheduleService) runDue() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	now := time.Now()
	due, err := s.repo.ListDue(ctx, now)
	if err != nil {
		log.Error().Err(err).Msg("failed to list due job schedules")
		return
	}
	for _, schedule := range due {
		jobID := "job-" + uuid.Must(uuid.NewV7()).String()
		nextRun := now.Add(time.Duration(schedule.IntervalMinutes) * time.Minute)
		claimed, err := s.repo.Claim(ctx, schedule.ID, jobID, now, nextRun)
		if err != nil || !claimed {
			continue
		}
		payload := ""
		if schedule.PayloadJSON != nil {
			payload = *schedule.PayloadJSON
		}
		if _, err := s.jobs.triggerWithID(ctx, jobID, schedule.TaskType, payload); err != nil {
			_ = s.repo.ReleaseClaim(ctx, schedule.ID, jobID, time.Now().Add(time.Minute))
			log.Error().Err(err).Str("schedule_id", schedule.ID).Msg("failed to enqueue scheduled job")
		}
	}
}

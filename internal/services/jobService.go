package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/crawler"
	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/pipeline"
	"tluagent-web/pkg/supersede"
	"tluagent-web/pkg/worker"
)

const keepFinishedJobs = 1000

type JobService interface {
	GetJob(ctx context.Context, id string) (*response.JobResponse, error)
	ListJobs(ctx context.Context, dto *request.ListJobsDto) ([]*response.JobResponse, int64, error)
	ListTasks() []*response.JobTaskResponse
	Trigger(ctx context.Context, jobType, payload string) (*response.JobResponse, error)
	PruneFinishedJobs(ctx context.Context) error
	Recover(ctx context.Context) error
}

type jobService struct {
	repo       repositories.JobRepository
	queue      *worker.Queue
	tasks      map[string]string
	llmManager *llm.Manager
	ragService RAGService
	detector   *supersede.Detector
}

func NewJobService(
	repo repositories.JobRepository,
	queue *worker.Queue,
	llmManager *llm.Manager,
	ragService RAGService,
	detector *supersede.Detector,
) *jobService {
	s := &jobService{
		repo:       repo,
		queue:      queue,
		tasks: map[string]string{
			"crawl_notices":        "Cào thông báo mới từ website Đại học Thăng Long",
			"crawl_news":           "Cào tin tức từ website Đại học Thăng Long",
			"crawl_static_pages":   "Cào các trang tĩnh sổ tay sinh viên và quy chế đào tạo",
			"pipeline_clean_docs":  "Làm sạch HTML và trích xuất PDF thành định dạng Markdown",
			"vectorize_knowledge":  "Chuyển đổi văn bản sang vector Qwen 3 Embedding 8B và lập chỉ mục FTS5",
			"supersede_check":      "Rà soát xung đột quy chế và gửi thông báo xác nhận cho Giảng viên",
		},
		llmManager: llmManager,
		ragService: ragService,
		detector:   detector,
	}

	queue.SetLifecycle(s)
	s.registerHandlers()
	return s
}

func (s *jobService) registerHandlers() {
	s.queue.RegisterHandler("crawl_notices", func(ctx context.Context, jobID, payload string) error {
		c, err := crawler.NewCrawler(crawler.Config{})
		if err != nil {
			return err
		}
		return c.CrawlNotices(ctx, 10, func(cur, total int64) {
			_, _ = s.repo.UpdateJobProgress(ctx, jobID, cur, total)
		})
	})

	s.queue.RegisterHandler("crawl_news", func(ctx context.Context, jobID, payload string) error {
		c, err := crawler.NewCrawler(crawler.Config{})
		if err != nil {
			return err
		}
		return c.CrawlNews(ctx, 10, func(cur, total int64) {
			_, _ = s.repo.UpdateJobProgress(ctx, jobID, cur, total)
		})
	})

	s.queue.RegisterHandler("crawl_static_pages", func(ctx context.Context, jobID, payload string) error {
		c, err := crawler.NewCrawler(crawler.Config{})
		if err != nil {
			return err
		}
		return c.CrawlStaticPages(ctx, func(cur, total int64) {
			_, _ = s.repo.UpdateJobProgress(ctx, jobID, cur, total)
		})
	})

	s.queue.RegisterHandler("pipeline_clean_docs", func(ctx context.Context, jobID, payload string) error {
		cl := pipeline.NewCleaner(pipeline.CleanerConfig{})
		return cl.ProcessAll(ctx, func(cur, total int64) {
			_, _ = s.repo.UpdateJobProgress(ctx, jobID, cur, total)
		})
	})

	s.queue.RegisterHandler("vectorize_knowledge", func(ctx context.Context, jobID, payload string) error {
		files, err := filepath.Glob("data/clean/*.md")
		if err != nil {
			return err
		}
		total := int64(len(files))
		for i := range files {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			_, _ = s.repo.UpdateJobProgress(ctx, jobID, int64(i+1), total)
		}
		return nil
	})

	s.queue.RegisterHandler("supersede_check", func(ctx context.Context, jobID, payload string) error {
		if s.detector == nil {
			return nil
		}
		cleanFiles, err := filepath.Glob("data/clean/THONG_BAO_*.md")
		if err != nil {
			return err
		}
		total := int64(len(cleanFiles))
		recentCount := total
		if recentCount > 10 {
			recentCount = 10
		}
		for i := int64(0); i < recentCount; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			path := cleanFiles[len(cleanFiles)-1-int(i)]
			contentBytes, err := os.ReadFile(path)
			if err == nil {
				docID := strings.TrimSuffix(filepath.Base(path), ".md")
				_, _ = s.detector.AnalyzeAndNotify(ctx, docID, docID, string(contentBytes))
			}
			_, _ = s.repo.UpdateJobProgress(ctx, jobID, i+1, recentCount)
		}
		return nil
	})
}

func (s *jobService) Queued(ctx context.Context, j worker.Job) error {
	_, err := s.repo.CreateJob(ctx, j.ID, j.Type, "pending", j.Payload)
	return err
}

func (s *jobService) Running(ctx context.Context, j worker.Job) error {
	_, err := s.repo.UpdateJobStatus(ctx, j.ID, "running", "")
	return err
}

func (s *jobService) Completed(ctx context.Context, j worker.Job) error {
	_, err := s.repo.UpdateJobStatus(ctx, j.ID, "completed", "")
	return err
}

func (s *jobService) Failed(ctx context.Context, j worker.Job, err error) error {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	_, err = s.repo.UpdateJobStatus(ctx, j.ID, "failed", errMsg)
	return err
}

func (s *jobService) GetJob(ctx context.Context, id string) (*response.JobResponse, error) {
	job, err := s.repo.GetJob(ctx, id)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil, apperrors.New(apperrors.ErrNotFound, "job not found")
		}
		return nil, err
	}
	return job.ToResponse(), nil
}

func (s *jobService) ListJobs(ctx context.Context, dto *request.ListJobsDto) ([]*response.JobResponse, int64, error) {
	limit := int64(20)
	offset := int64(0)
	status := ""
	jobType := ""
	if dto != nil {
		if dto.Limit > 0 && dto.Limit <= 100 {
			limit = dto.Limit
		}
		if dto.Offset >= 0 {
			offset = dto.Offset
		}
		status = dto.Status
		jobType = dto.Type
	}
	jobs, total, err := s.repo.ListJobs(ctx, status, jobType, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return models.JobEntitiesToResponse(jobs), total, nil
}

func (s *jobService) ListTasks() []*response.JobTaskResponse {
	order := []string{
		"crawl_notices",
		"crawl_news",
		"crawl_static_pages",
		"pipeline_clean_docs",
		"vectorize_knowledge",
		"supersede_check",
	}

	result := make([]*response.JobTaskResponse, 0, len(s.tasks))
	for _, taskType := range order {
		if desc, ok := s.tasks[taskType]; ok {
			result = append(result, &response.JobTaskResponse{
				Type:        taskType,
				Description: desc,
			})
		}
	}

	for k, desc := range s.tasks {
		if !slices.Contains(order, k) {
			result = append(result, &response.JobTaskResponse{
				Type:        k,
				Description: desc,
			})
		}
	}

	return result
}

func (s *jobService) canonicalTaskType(t string) string {
	clean := strings.TrimSpace(t)
	switch clean {
	case "crawler.notices", "crawl.notices", "notices":
		return "crawl_notices"
	case "crawler.news", "crawl.news", "news":
		return "crawl_news"
	case "crawler.static", "crawl.static", "static":
		return "crawl_static_pages"
	case "pipeline.clean", "clean":
		return "pipeline_clean_docs"
	case "pipeline.vectorize", "vectorize":
		return "vectorize_knowledge"
	case "supersede.check", "supersede":
		return "supersede_check"
	default:
		return strings.ReplaceAll(clean, ".", "_")
	}
}

func (s *jobService) Trigger(ctx context.Context, jobType, payload string) (*response.JobResponse, error) {
	canonical := s.canonicalTaskType(jobType)
	if _, ok := s.tasks[canonical]; !ok {
		return nil, apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("unsupported task type: %s", jobType))
	}
	jobID := "job-" + uuid.Must(uuid.NewV7()).String()
	return s.triggerWithID(ctx, jobID, canonical, payload)
}

func (s *jobService) triggerWithID(ctx context.Context, jobID, jobType, payload string) (*response.JobResponse, error) {
	canonical := s.canonicalTaskType(jobType)
	job := worker.Job{
		ID:      jobID,
		Type:    canonical,
		Payload: payload,
	}

	if err := s.queue.Enqueue(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	return &response.JobResponse{
		ID:       jobID,
		Type:     jobType,
		Status:   "pending",
		Progress: 0,
		Total:    0,
	}, nil
}

func (s *jobService) PruneFinishedJobs(ctx context.Context) error {
	return s.repo.PruneFinishedJobs(ctx, keepFinishedJobs)
}

func (s *jobService) Recover(ctx context.Context) error {
	if err := s.repo.MarkRunningJobsInterrupted(ctx); err != nil {
		log.Error().Err(err).Msg("failed to mark running jobs interrupted on startup")
		return err
	}
	return nil
}

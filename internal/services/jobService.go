package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"tluagent-web/internal/agent"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/convert"
	"tluagent-web/pkg/crawler"
	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/worker"
)

const keepFinishedJobs = 1000

var TLUStaticRoutes = []string{
	"/gioi-thieu",
	"/gioi-thieu/su-mang-muc-tieu",
	"/gioi-thieu/co-so-vat-chat",
	"/phong-ban/co-cau-to-chuc",
	"/ba-cong-khai",
	"/gioi-thieu/hoi-dong-giao-su-co-so",
	"/sites/default/files/2023-08/GCN-KDCL-CSGD-TR.-DH-THANG-LONG.pdf",
	"/khoa-toan-tin-hoc",
	"/khoa-kinh-te-quan-li",
	"/khoa-khoa-hoc-suc-khoe",
	"/khoa-ngoai-ngu",
	"/khoa-khoa-hoc-xa-hoi-va-nhan-van",
	"/khoa-du-lich",
	"/khoa-truyen-thong-da-phuong-tien",
	"/khoa-am-nhac-ung-dung",
	"/tuyen-sinh",
	"/tuyen-sinh/tuyen-sinh-dai-hoc-chinh-quy",
	"/tuyen-sinh/tuyen-sinh-sau-dai-hoc",
	"/tuyen-sinh/tuyen-sinh-dao-tao-thac-si-quoc-te",
	"/doi-song-thang-long",
	"/thong-tin-tai-lieu",
	"/lien-he",
	"/so-tay-sinh-vien",
	"/quy-che-dao-tao",
	"/chuong-trinh-dao-tao",
}

var TLUIgnoredDriveIDs = []string{
	"1Tqsw8oUCCFbUQ7BRnlrTAy8AXeDxKIBb",
}

func resolveTLUPriority(category, title string) int {
	clean := strings.ToLower(convert.StripAccents(title))
	if strings.Contains(clean, "hoc phi") {
		return 90
	}
	if strings.Contains(clean, "quy che") || strings.Contains(clean, "chuan dau ra") {
		return 85
	}
	if strings.Contains(clean, "nghi hoc") || strings.Contains(clean, "lich thi") {
		return 75
	}
	return 50
}

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
	detector   *Detector
}

func NewJobService(
	repo repositories.JobRepository,
	queue *worker.Queue,
	llmManager *llm.Manager,
	ragService RAGService,
	detector *Detector,
) *jobService {
	s := &jobService{
		repo:  repo,
		queue: queue,
		tasks: map[string]string{
			constants.JobTypeCleanCorpus:      "Wipe RAG knowledge base and conflict audit logs",
			constants.JobTypeCrawlNotices:     "End-to-end in-memory crawl, extraction, embedding, and supersession of announcements",
			constants.JobTypeCrawlNews:        "End-to-end in-memory crawl, extraction, and embedding of news",
			constants.JobTypeCrawlStaticPages: "End-to-end in-memory crawl of student handbook and regulation static pages",
			constants.JobTypeSupersedeCheck:   "Agent-driven regulation conflict, amendment, and supersession evaluation",
		},
		llmManager: llmManager,
		ragService: ragService,
		detector:   detector,
	}

	queue.SetLifecycle(s)
	s.registerHandlers()
	return s
}

type crawlPayload struct {
	MaxPages    int   `json:"max_pages"`
	Limit       int `json:"limit"`
	Pages       int `json:"pages"`
	MaxTimeDays int `json:"max_time_days"`
	MaxTime     int `json:"max_time"`
	Days        int `json:"days"`
}

func parseCrawlPayload(payload string) (int, int) {
	payload = strings.TrimSpace(payload)
	if payload == "" || payload == "{}" {
		return 0, 0
	}

	var p crawlPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return 0, 0
	}

	maxPages := 0
	if p.MaxPages > 0 {
		maxPages = p.MaxPages
	} else if p.Limit > 0 {
		maxPages = p.Limit
	} else if p.Pages > 0 {
		maxPages = p.Pages
	}

	maxTimeDays := 0
	if p.MaxTimeDays > 0 {
		maxTimeDays = p.MaxTimeDays
	} else if p.MaxTime > 0 {
		maxTimeDays = p.MaxTime
	} else if p.Days > 0 {
		maxTimeDays = p.Days
	}

	return maxPages, maxTimeDays
}

func (s *jobService) registerHandlers() {
	s.queue.RegisterHandler(constants.JobTypeCleanCorpus, func(ctx context.Context, jobID, payload string) error {
		if s.ragService != nil {
			if err := s.ragService.WipeCorpus(ctx); err != nil {
				return err
			}
		}
		_, _ = s.repo.UpdateJobProgress(ctx, jobID, 1, 1)
		return nil
	})

	s.queue.RegisterHandler(constants.JobTypeCrawlNotices, func(ctx context.Context, jobID, payload string) error {
		maxPages, maxTimeDays := parseCrawlPayload(payload)

		c, err := crawler.NewCrawler(crawler.Config{
			BaseURL:         constants.DefaultUniversityBaseURL,
			DelayMs:         60,
			Concurrency:     3,
			MaxRetries:      5,
			IgnoredDriveIDs: TLUIgnoredDriveIDs,
			MaxTimeDays:     maxTimeDays,
			VisionOCR:        s.visionOCRHandler(),
		})
		if err != nil {
			return err
		}

		var docFilter func(id, rawURL, title string) bool
		if s.ragService != nil {
			docFilter, _ = s.ragService.BuildDocumentFilter(ctx)
		}

		skipChecker := func(ctx context.Context, id, rawURL, title string) bool {
			if docFilter != nil && docFilter(id, rawURL, title) {
				return true
			}
			if s.ragService == nil {
				return false
			}
			return s.ragService.DocumentExistsByURL(ctx, rawURL)
		}

		handler := func(ctx context.Context, doc *crawler.CrawledDocument) error {
			return s.processCrawledDocumentInMemory(ctx, doc)
		}

		return c.CrawlPagedCategory(ctx, constants.SlugNotices, constants.DocTypeNotice, maxPages, skipChecker, handler, func(cur, total int64) {
			_, _ = s.repo.UpdateJobProgress(ctx, jobID, cur, total)
		})
	})

	s.queue.RegisterHandler(constants.JobTypeCrawlNews, func(ctx context.Context, jobID, payload string) error {
		maxPages, maxTimeDays := parseCrawlPayload(payload)

		c, err := crawler.NewCrawler(crawler.Config{
			BaseURL:         constants.DefaultUniversityBaseURL,
			DelayMs:         60,
			Concurrency:     3,
			MaxRetries:      5,
			IgnoredDriveIDs: TLUIgnoredDriveIDs,
			MaxTimeDays:     maxTimeDays,
			VisionOCR:        s.visionOCRHandler(),
		})
		if err != nil {
			return err
		}

		var docFilter func(id, rawURL, title string) bool
		if s.ragService != nil {
			docFilter, _ = s.ragService.BuildDocumentFilter(ctx)
		}

		skipChecker := func(ctx context.Context, id, rawURL, title string) bool {
			if docFilter != nil && docFilter(id, rawURL, title) {
				return true
			}
			if s.ragService == nil {
				return false
			}
			return s.ragService.DocumentExistsByURL(ctx, rawURL)
		}

		handler := func(ctx context.Context, doc *crawler.CrawledDocument) error {
			return s.processCrawledDocumentInMemory(ctx, doc)
		}

		return c.CrawlPagedCategory(ctx, constants.SlugNews, constants.DocTypeNews, maxPages, skipChecker, handler, func(cur, total int64) {
			_, _ = s.repo.UpdateJobProgress(ctx, jobID, cur, total)
		})
	})

	s.queue.RegisterHandler(constants.JobTypeCrawlStaticPages, func(ctx context.Context, jobID, payload string) error {
		c, err := crawler.NewCrawler(crawler.Config{
			BaseURL:         constants.DefaultUniversityBaseURL,
			DelayMs:         60,
			Concurrency:     3,
			MaxRetries:      5,
			IgnoredDriveIDs: TLUIgnoredDriveIDs,
			VisionOCR:        s.visionOCRHandler(),
		})
		if err != nil {
			return err
		}

		var docFilter func(id, rawURL, title string) bool
		if s.ragService != nil {
			docFilter, _ = s.ragService.BuildDocumentFilter(ctx)
		}

		skipChecker := func(ctx context.Context, id, rawURL, title string) bool {
			if docFilter != nil && docFilter(id, rawURL, title) {
				return true
			}
			if s.ragService == nil {
				return false
			}
			return s.ragService.DocumentExistsByURL(ctx, rawURL)
		}

		handler := func(ctx context.Context, doc *crawler.CrawledDocument) error {
			return s.processCrawledDocumentInMemory(ctx, doc)
		}

		return c.CrawlRoutes(ctx, TLUStaticRoutes, constants.DocTypeStatic, skipChecker, handler, func(cur, total int64) {
			_, _ = s.repo.UpdateJobProgress(ctx, jobID, cur, total)
		})
	})


	s.queue.RegisterHandler(constants.JobTypeSupersedeCheck, func(ctx context.Context, jobID, payload string) error {
		if s.detector == nil || s.detector.Engine() == nil {
			return nil
		}
		return s.detector.Engine().AuditCorpusChronological(ctx, func(cur, total int64) {
			_, _ = s.repo.UpdateJobProgress(ctx, jobID, cur, total)
		})
	})
}

func (s *jobService) visionOCRHandler() crawler.VisionOCRHandler {
	if s.llmManager == nil {
		return nil
	}
	return func(ctx context.Context, images [][]byte) (string, error) {
		provider, err := s.llmManager.Default()
		if err != nil {
			return "", err
		}
		parts := make([]llm.ContentPart, 0, len(images)+1)
		parts = append(parts, llm.ContentPart{
			Type: llm.ContentPartText,
			Text: agent.PromptDocumentOCRScan(),
		})
		for _, img := range images {
			parts = append(parts, llm.ContentPart{
				Type: llm.ContentPartImage,
				Image: &llm.ImageSource{
					Data:     base64.StdEncoding.EncodeToString(img),
					MimeType: "image/png",
				},
			})
		}
		resp, err := provider.Chat(ctx, &llm.ChatRequest{
			Model: provider.DefaultModel(),
			Messages: []llm.Message{
				{
					Role:  llm.RoleUser,
					Parts: parts,
				},
			},
		})
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(resp.Message.Content), nil
	}
}

func (s *jobService) processCrawledDocumentInMemory(ctx context.Context, item *crawler.CrawledDocument) error {
	if item == nil || strings.TrimSpace(item.Content) == "" {
		return nil
	}
	cleanTitle := strings.ToLower(strings.TrimSpace(item.Title))
	if cleanTitle == "" || convert.StripAccents(cleanTitle) == "tim kiem" || cleanTitle == "search" {
		return nil
	}

	if s.ragService != nil {
		if s.ragService.DocumentExistsByURL(ctx, item.URL) {
			return nil
		}
		if existing, _ := s.ragService.GetDocumentDetail(ctx, item.ID); existing != nil {
			return nil
		}
	}

	docType, domain, cohort, academicYear := extractCrawledDocumentMetadata(item.ID, item.Title, item.Category, item.PublishDate)

	doc := &models.Document{
		ID:               item.ID,
		DocCode:          item.ID,
		Title:            item.Title,
		Type:             docType,
		Domain:           domain,
		ApplicableCohort: cohort,
		AcademicYear:     academicYear,
		Status:           constants.DocStatusActive,
		PriorityLevel:    resolveTLUPriority(item.Category, item.Title),
		PublishDate:      item.PublishDate,
		FilePath:         item.URL,
		CharCount:        len(item.Content),
	}

	chunkTexts := splitIntoChunks(item.Content, 1200, 150)
	var embeddings [][]float32
	if s.llmManager != nil && len(chunkTexts) > 0 {
		batchSize := 32
		for i := 0; i < len(chunkTexts); i += batchSize {
			end := i + batchSize
			if end > len(chunkTexts) {
				end = len(chunkTexts)
			}
			batch, err := s.llmManager.ExecuteEmbedding(ctx, chunkTexts[i:end])
			if err != nil {
				break
			}
			embeddings = append(embeddings, batch...)
		}
	}

	var chunks []*models.DocumentChunk
	for idx, text := range chunkTexts {
		var emb []float32
		if idx < len(embeddings) {
			emb = embeddings[idx]
		}
		chunks = append(chunks, &models.DocumentChunk{
			ID:               fmt.Sprintf(constants.PrefixChunk+"%s_%d", doc.ID, idx),
			DocumentID:       doc.ID,
			ChunkIndex:       idx,
			ChunkType:        constants.ChunkTypeContent,
			Content:          text,
			Status:           constants.DocStatusActive,
			Domain:           domain,
			ApplicableCohort: cohort,
			AcademicYear:     academicYear,
			PriorityLevel:    doc.PriorityLevel,
			Embedding:        emb,
		})
	}

	docNode := &models.KGNode{
		ID:       constants.PrefixDocNode + doc.ID,
		NodeType: constants.KGNodeTypeDocument,
		Name:     doc.Title,
		Code:     doc.ID,
		Status:   constants.DocStatusActive,
	}
	nodes := []*models.KGNode{docNode}
	var edges []*models.KGEdge
	if domain != "" {
		topicNode := &models.KGNode{
			ID:       constants.PrefixTopicNode + domain,
			NodeType: constants.KGNodeTypeTopic,
			Name:     domain,
			Code:     domain,
			Status:   constants.DocStatusActive,
		}
		nodes = append(nodes, topicNode)
		edges = append(edges, &models.KGEdge{
			ID:           fmt.Sprintf(constants.PrefixEdgeTopic+"%s:%s", doc.ID, domain),
			SourceNodeID: docNode.ID,
			TargetNodeID: topicNode.ID,
			RelationType: constants.KGRelationCoversTopic,
			Weight:       1.0,
			Status:       constants.DocStatusActive,
		})
	}

	return s.ragService.IngestDocument(ctx, doc, chunks, nodes, edges)
}

func (s *jobService) Queued(ctx context.Context, j worker.Job) error {
	_, err := s.repo.CreateJob(ctx, j.ID, j.Type, constants.JobStatusPending, j.Payload)
	return err
}

func (s *jobService) Running(ctx context.Context, j worker.Job) error {
	_, err := s.repo.UpdateJobStatus(ctx, j.ID, constants.JobStatusRunning, "")
	return err
}

func (s *jobService) Completed(ctx context.Context, j worker.Job) error {
	_, err := s.repo.UpdateJobStatus(ctx, j.ID, constants.JobStatusCompleted, "")
	return err
}

func (s *jobService) Failed(ctx context.Context, j worker.Job, err error) error {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	_, err = s.repo.UpdateJobStatus(ctx, j.ID, constants.JobStatusFailed, errMsg)
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
		constants.JobTypeCleanCorpus,
		constants.JobTypeCrawlNotices,
		constants.JobTypeCrawlNews,
		constants.JobTypeCrawlStaticPages,
		constants.JobTypeSupersedeCheck,
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
	case "corpus.clean", "clean.corpus", "wipe", "clean_corpus":
		return constants.JobTypeCleanCorpus
	case "crawler.notices", "crawl.notices", "notices", "crawl_notices":
		return constants.JobTypeCrawlNotices
	case "crawler.news", "crawl.news", "news", "crawl_news":
		return constants.JobTypeCrawlNews
	case "crawler.static", "crawl.static", "static", "crawl_static_pages":
		return constants.JobTypeCrawlStaticPages
	case "supersede.check", "supersede", "supersede_check":
		return constants.JobTypeSupersedeCheck
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

func isCrawlerType(t string) bool {
	return t == constants.JobTypeCrawlNotices || t == constants.JobTypeCrawlNews || t == constants.JobTypeCrawlStaticPages
}

func (s *jobService) triggerWithID(ctx context.Context, jobID, jobType, payload string) (*response.JobResponse, error) {
	canonical := s.canonicalTaskType(jobType)
	if isCrawlerType(canonical) {
		runningJobs, _, _ := s.repo.ListJobs(ctx, constants.JobStatusRunning, canonical, 1, 0)
		if len(runningJobs) > 0 {
			return nil, apperrors.New(apperrors.ErrConflict, fmt.Sprintf("task %s is already running", canonical))
		}
		pendingJobs, _, _ := s.repo.ListJobs(ctx, constants.JobStatusPending, canonical, 1, 0)
		if len(pendingJobs) > 0 {
			return nil, apperrors.New(apperrors.ErrConflict, fmt.Sprintf("task %s is already queued", canonical))
		}
	}

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
		Type:     canonical,
		Status:   constants.JobStatusPending,
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

func extractTitleFromMarkdown(content, fallback string) string {
	lines := strings.Split(content, "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(l, "# "))
		}
	}
	return strings.ReplaceAll(fallback, "_", " ")
}

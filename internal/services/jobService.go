package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"tluagent-web/internal/agent"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/config"
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
	notifRepo  repositories.NotificationRepository
	userRepo   repositories.UserRepository
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
			constants.JobTypeCleanCorpus:        "Wipe RAG knowledge base and conflict audit logs",
			constants.JobTypeCrawlNotices:       "End-to-end in-memory crawl, extraction, embedding, and supersession of announcements",
			constants.JobTypeCrawlNews:          "End-to-end in-memory crawl, extraction, and embedding of news",
			constants.JobTypeCrawlStaticPages:   "End-to-end in-memory crawl of student handbook and regulation static pages",
			constants.JobTypeSupersedeCheck:     "Agent-driven regulation conflict, amendment, and supersession evaluation",
			constants.JobTypeReextractDocuments: "Re-extract stored announcement attachments with table-aware parsing and vision OCR for corrupted text layers",
		},
		llmManager: llmManager,
		ragService: ragService,
		detector:   detector,
	}

	queue.SetLifecycle(s)
	s.registerHandlers()
	return s
}

func (s *jobService) SetFailureNotifier(notifRepo repositories.NotificationRepository, userRepo repositories.UserRepository) {
	s.notifRepo = notifRepo
	s.userRepo = userRepo
}

type crawlPayload struct {
	MaxPages         int   `json:"max_pages"`
	Limit            int   `json:"limit"`
	Pages            int   `json:"pages"`
	MaxTimeDays      int   `json:"max_time_days"`
	MaxTime          int   `json:"max_time"`
	Days             int   `json:"days"`
	FetchAttachments *bool `json:"fetch_attachments"`
}

// parseCrawlOptions reads the optional payload switches of the crawl jobs.
func parseAuditFull(payload string) bool {
	var p struct {
		Full bool `json:"full"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &p); err != nil {
		return false
	}
	return p.Full
}

func parseCrawlOptions(payload string) (fetchAttachments bool) {
	var p crawlPayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &p); err != nil {
		return true
	}
	if p.FetchAttachments == nil {
		return true
	}
	return *p.FetchAttachments
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
		return s.crawlCMSCollections(ctx, jobID, []cmsCrawlSource{noticeCrawlSource(parseCrawlOptions(payload))}, maxPages, maxTimeDays)
	})

	s.queue.RegisterHandler(constants.JobTypeCrawlNews, func(ctx context.Context, jobID, payload string) error {
		maxPages, maxTimeDays := parseCrawlPayload(payload)
		return s.crawlCMSCollections(ctx, jobID, []cmsCrawlSource{
			{
				Collection:     crawler.CMSCollectionPosts,
				DocType:        constants.DocTypeNews,
				IDPrefix:       "DOC_TIN_TUC_",
				MinBody:        80,
				AllowTitleOnly: true,
			},
			{
				Collection:     crawler.CMSCollectionEvents,
				DocType:        constants.DocTypeNews,
				IDPrefix:       "DOC_SU_KIEN_",
				MinBody:        80,
				AllowTitleOnly: true,
			},
		}, maxPages, maxTimeDays)
	})

	s.queue.RegisterHandler(constants.JobTypeCrawlStaticPages, func(ctx context.Context, jobID, payload string) error {
		maxPages, maxTimeDays := parseCrawlPayload(payload)
		return s.crawlCMSCollections(ctx, jobID, []cmsCrawlSource{{
			Collection:       crawler.CMSCollectionPages,
			DocType:          constants.DocTypeStatic,
			IDPrefix:         "DOC_STATIC_",
			MinBody:          200,
			SkipSlugSuffixes: []string{"-en"},
			SkipSlugs:        []string{"navbar", "footer", "header"},
		}}, maxPages, maxTimeDays)
	})

	s.queue.RegisterHandler(constants.JobTypeReextractDocuments, s.reextractDocuments)

	s.queue.RegisterHandler(constants.JobTypeSupersedeCheck, func(ctx context.Context, jobID, payload string) error {
		if s.detector == nil || s.detector.Engine() == nil {
			return nil
		}
		if s.ragService != nil {
			if _, err := s.ragService.RepairDocumentGraph(ctx, 500); err != nil {
				log.Warn().Err(err).Msg("Knowledge graph repair before supersession audit failed")
			}
		}
		return s.detector.Engine().AuditCorpus(ctx, AuditOptions{Full: parseAuditFull(payload)}, func(cur, total int64) {
			_, _ = s.repo.UpdateJobProgress(ctx, jobID, cur, total)
		})
	})
}

const attachmentOverlapRunes = 150

func noticeCrawlSource(fetchAttachments bool) cmsCrawlSource {
	return cmsCrawlSource{
		Collection:      crawler.CMSCollectionAnnouncements,
		DocType:         constants.DocTypeNotice,
		IDPrefix:        "DOC_THONG_BAO_",
		MinBody:         120,
		FetchAttachment: fetchAttachments,
		AllowTitleOnly:  true,
	}
}

func newCMSClient() (*crawler.CMSClient, error) {
	return crawler.NewCMSClient(crawler.CMSConfig{
		BaseURL: config.GetConfigWithDefault("CMS_API_BASE_URL", ""),
		SiteURL: config.GetConfigWithDefault("CMS_SITE_URL", ""),
		Locale:  config.GetConfigWithDefault("CMS_LOCALE", crawler.DefaultCMSLocale),
		DelayMs: 60,
	})
}

func (s *jobService) newAttachmentCrawler(client *crawler.CMSClient) (*crawler.Crawler, error) {
	if strings.TrimSpace(client.SiteURL()) == "" {
		return nil, fmt.Errorf("CMS_SITE_URL is required to download attachments")
	}
	return crawler.NewCrawler(crawler.Config{
		BaseURL:         client.SiteURL(),
		DelayMs:         60,
		Concurrency:     3,
		MaxRetries:      5,
		IgnoredDriveIDs: TLUIgnoredDriveIDs,
		VisionOCR:       s.visionOCRHandler(),
	})
}

func cmsItemBody(ctx context.Context, source cmsCrawlSource, item *crawler.CMSItem, attachments *crawler.Crawler) string {
	body := strings.TrimSpace(item.Body)
	if source.FetchAttachment && attachments != nil && item.AttachmentURL != "" {
		if attachmentText := attachments.AttachmentText(ctx, item.AttachmentURL); attachmentText != "" {
			body = mergeBodyAndAttachment(body, attachmentText)
		}
	}
	return body
}

// mergeBodyAndAttachment joins the page body with its attachment text unless one already contains the other.
func mergeBodyAndAttachment(body, attachment string) string {
	attachment = strings.TrimSpace(attachment)
	bodyKey, attachmentKey := documentTitleKey(body), documentTitleKey(attachment)
	switch {
	case attachmentKey == "":
		return body
	case bodyKey == "":
		return attachment
	case strings.Contains(bodyKey, attachmentKey):
		return body
	case strings.Contains(attachmentKey, bodyKey):
		return attachment
	}
	if prefix := []rune(attachmentKey); len(prefix) > attachmentOverlapRunes && strings.Contains(bodyKey, string(prefix[:attachmentOverlapRunes])) {
		return body
	}
	return body + "\n\n" + attachment
}

type cmsCrawlSource struct {
	Collection       string
	DocType          string
	IDPrefix         string
	MinBody          int
	FetchAttachment  bool
	AllowTitleOnly   bool
	SkipSlugSuffixes []string
	SkipSlugs        []string
}

func (s cmsCrawlSource) skips(slug string) bool {
	slug = strings.ToLower(strings.TrimSpace(slug))
	for _, exact := range s.SkipSlugs {
		if slug == exact {
			return true
		}
	}
	for _, suffix := range s.SkipSlugSuffixes {
		if suffix != "" && strings.HasSuffix(slug, suffix) {
			return true
		}
	}
	return false
}

func (s *jobService) crawlCMSCollections(ctx context.Context, jobID string, sources []cmsCrawlSource, maxPages, maxTimeDays int) error {
	if len(sources) == 0 {
		return nil
	}

	client, err := newCMSClient()
	if err != nil {
		return err
	}

	var attachmentCrawler *crawler.Crawler
	if needsAttachments(sources) {
		attachmentCrawler, err = s.newAttachmentCrawler(client)
		if err != nil {
			return err
		}
	}

	skipChecker := s.crawlSkipChecker(ctx)

	if maxPages <= 0 {
		maxPages = 5
	}

	var cutoff time.Time
	if maxTimeDays > 0 {
		cutoff = time.Now().AddDate(0, 0, -maxTimeDays)
	}

	pageSize := client.PageSize()
	var discovered, ingested, skipped int64
	for _, source := range sources {
		for page := 0; page < maxPages; page++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			items, pagination, err := client.List(ctx, source.Collection, page*pageSize)
			if err != nil {
				return fmt.Errorf("list %s page %d: %w", source.Collection, page, err)
			}
			if len(items) == 0 {
				break
			}

			for _, item := range items {
				discovered++
				if !cutoff.IsZero() && crawler.CMSItemOlderThan(item, cutoff) {
					continue
				}
				if item.URL == "" || item.Title == "" {
					continue
				}
				if source.skips(item.Slug) {
					skipped++
					continue
				}
				if skipChecker != nil && skipChecker(ctx, "", item.URL, item.Title) {
					skipped++
					continue
				}

				detail, err := client.Detail(ctx, source.Collection, item.ID)
				if err != nil {
					log.Warn().Err(err).Str("collection", source.Collection).Str("item_id", item.ID).Msg("Skipped CMS item without detail")
					continue
				}
				if detail.Title != "" {
					item = detail
				}

				body := cmsItemBody(ctx, source, item, attachmentCrawler)
				if len([]rune(body)) < source.MinBody {
					if !source.AllowTitleOnly {
						skipped++
						continue
					}
					body = item.Title
				}

				doc := &crawler.CrawledDocument{
					ID:          crawler.CMSDocID(source.IDPrefix, item.ID),
					Category:    source.DocType,
					URL:         item.URL,
					Title:       item.Title,
					PublishDate: item.PublishDate,
					Content:     body,
				}
				if err := s.processCrawledDocumentInMemory(ctx, doc); err != nil {
					log.Warn().Err(err).Str("doc_id", doc.ID).Msg("Failed to ingest crawled CMS item")
					continue
				}
				ingested++

				if _, err := s.repo.UpdateJobProgress(ctx, jobID, ingested, discovered); err != nil {
					log.Warn().Err(err).Msg("Failed to update crawl job progress")
				}
				if attachmentCrawler != nil {
					time.Sleep(60 * time.Millisecond)
				}
			}

			if len(items) < pageSize || !pagination.HasMore {
				break
			}
		}
	}

	if discovered == 0 {
		return fmt.Errorf("no items returned by CMS for %d collection(s)", len(sources))
	}

	log.Info().
		Int64("discovered", discovered).
		Int64("ingested", ingested).
		Int64("skipped", skipped).
		Msg("CMS crawl finished")
	return nil
}

func needsAttachments(sources []cmsCrawlSource) bool {
	for _, source := range sources {
		if source.FetchAttachment {
			return true
		}
	}
	return false
}

func (s *jobService) crawlSkipChecker(ctx context.Context) crawler.SkipChecker {
	var docFilter func(id, rawURL, title string) bool
	if s.ragService != nil {
		docFilter, _ = s.ragService.BuildDocumentFilter(ctx)
	}
	return func(ctx context.Context, id, rawURL, title string) bool {
		if docFilter != nil && docFilter(id, rawURL, title) {
			return true
		}
		if s.ragService == nil {
			return false
		}
		return s.ragService.DocumentExistsByURL(ctx, rawURL)
	}
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
			Model:            provider.DefaultModel(),
			ThinkingDisabled: true,
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

func effectiveFromFor(publishDate, academicYear string) string {
	if strings.TrimSpace(publishDate) != "" {
		return publishDate
	}
	return constants.DeriveEffectiveFromAcademicYear(academicYear)
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
	if cohort == constants.DocCohortAll && strings.HasPrefix(item.ID, noticeDocIDPrefix) {
		if scoped := extractScopeCohortFromBody(item.Content); scoped != "" {
			cohort = scoped
		}
	}

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
		EffectiveFrom:    effectiveFromFor(item.PublishDate, academicYear),
		ValidityTier:     constants.DefaultValidityTier(docType),
		FilePath:         item.URL,
		CharCount:        len(item.Content),
	}

	chunks := buildContentChunks(ctx, s.llmManager, doc, item.Content, constants.DocStatusActive)

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
	_, updateErr := s.repo.UpdateJobStatus(ctx, j.ID, constants.JobStatusFailed, errMsg)
	s.notifyRepeatedFailure(ctx, j, errMsg)
	return updateErr
}

func (s *jobService) notifyRepeatedFailure(ctx context.Context, j worker.Job, errMsg string) {
	if s.notifRepo == nil || s.userRepo == nil {
		return
	}
	statuses, err := s.repo.ListRecentJobStatusesByType(ctx, j.Type, 2)
	if err != nil || len(statuses) < 2 {
		return
	}
	for _, status := range statuses {
		if status != constants.JobStatusFailed {
			return
		}
	}
	adminIDs, err := s.userRepo.ListAdminUserIDs(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load admin recipients for job failure alert")
		return
	}
	for _, adminID := range adminIDs {
		_, _ = s.notifRepo.CreateNotification(ctx, &models.UserNotification{
			ID:      "notif_" + uuid.NewString(),
			UserID:  adminID,
			Title:   "Tác vụ nền thất bại liên tiếp",
			Content: fmt.Sprintf("Tác vụ %s đã thất bại 2 lần liên tiếp. Lỗi gần nhất: %s", j.Type, errMsg),
			Type:    constants.NotifTypeJobFailure,
		})
	}
	log.Warn().Str("job_type", j.Type).Int("admins", len(adminIDs)).Msg("Dispatched repeated job failure alerts")
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
		constants.JobTypeReextractDocuments,
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
	case "reextract.documents", "reextract", "reextract_documents":
		return constants.JobTypeReextractDocuments
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
	return t == constants.JobTypeCrawlNotices || t == constants.JobTypeCrawlNews || t == constants.JobTypeCrawlStaticPages || t == constants.JobTypeReextractDocuments
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

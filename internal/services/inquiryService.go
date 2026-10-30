package services

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"tluagent-web/internal/agent"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/convert"
	"tluagent-web/pkg/llm"
)

// InquiryService handles student inquiry routing, lecturer answers, and self-evolution.
type InquiryService interface {
	CreateInquiry(ctx context.Context, userID string, req *request.CreateInquiryRequest) (*response.InquiryResponse, error)
	GetInquiry(ctx context.Context, id string) (*response.InquiryResponse, error)
	ListStudentInquiries(ctx context.Context, userID string, dto *request.ListStudentInquiriesDto) ([]*response.InquiryResponse, int, error)
	ListTeacherInquiries(ctx context.Context, dto *request.ListTeacherInquiriesDto) ([]*response.InquiryResponse, int, error)
	AnswerInquiry(ctx context.Context, teacherID string, inquiryID string, req *request.AnswerInquiryRequest) (*response.InquiryResponse, error)
	ExpireInquiry(ctx context.Context, inquiryID, supersededByDocID, reason string) error
	CheckAndExpireOutdatedQA(ctx context.Context, doc *models.Document) (int, error)
}

type inquiryService struct {
	inquiryRepo      repositories.InquiryRepository
	ragRepo          repositories.RAGRepository
	userRepo         repositories.UserRepository
	notificationRepo repositories.NotificationRepository
	llmManager       *llm.Manager
	engine           *RegulatoryDissectionEngine
}

// NewInquiryService creates an instance of InquiryService.
func NewInquiryService(
	inquiryRepo repositories.InquiryRepository,
	ragRepo repositories.RAGRepository,
	userRepo repositories.UserRepository,
	notificationRepo repositories.NotificationRepository,
	llmManager *llm.Manager,
) InquiryService {
	engine := NewRegulatoryDissectionEngine(EngineParams{
		RAGRepo:     ragRepo,
		InquiryRepo: inquiryRepo,
		NotifRepo:   notificationRepo,
		UserRepo:    userRepo,
		LLMManager:  llmManager,
	})
	return &inquiryService{
		inquiryRepo:      inquiryRepo,
		ragRepo:          ragRepo,
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
		llmManager:       llmManager,
		engine:           engine,
	}
}

func (s *inquiryService) CreateInquiry(ctx context.Context, userID string, req *request.CreateInquiryRequest) (*response.InquiryResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "user not found")
	}

	studentName := strings.TrimSpace(req.StudentName)
	if studentName == "" {
		studentName = user.FullName
	}
	studentCode := strings.TrimSpace(req.StudentCode)
	if studentCode == "" {
		studentCode = user.StudentCode
	} else if user.StudentCode != "" && !strings.EqualFold(studentCode, user.StudentCode) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Cannot create inquiry for another student ID")
	}
	studentClass := strings.TrimSpace(req.StudentClass)

	id := "inq_" + uuid.NewString()
	inq := &models.AdvisoryInquiry{
		ID:             id,
		UserID:         userID,
		ConversationID: strings.TrimSpace(req.ConversationID),
		StudentName:    studentName,
		StudentCode:    studentCode,
		StudentClass:   studentClass,
		Question:       strings.TrimSpace(req.Question),
		Context:        strings.TrimSpace(req.Context),
		Status:         models.InquiryStatusPending,
	}

	created, err := s.inquiryRepo.CreateInquiry(ctx, inq)
	if err != nil {
		return nil, err
	}
	return created.ToResponse(), nil
}

func (s *inquiryService) GetInquiry(ctx context.Context, id string) (*response.InquiryResponse, error) {
	inq, err := s.inquiryRepo.GetInquiryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if inq == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "inquiry not found")
	}
	return inq.ToResponse(), nil
}

func (s *inquiryService) ListStudentInquiries(ctx context.Context, userID string, dto *request.ListStudentInquiriesDto) ([]*response.InquiryResponse, int, error) {
	limit := 20
	offset := 0
	if dto != nil {
		limit = dto.GetLimit(20)
		offset = dto.GetOffset(20)
	}
	items, total, err := s.inquiryRepo.ListInquiriesByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return models.InquiriesToResponse(items), total, nil
}

func (s *inquiryService) ListTeacherInquiries(ctx context.Context, dto *request.ListTeacherInquiriesDto) ([]*response.InquiryResponse, int, error) {
	limit := 20
	offset := 0
	status := ""
	search := ""
	if dto != nil {
		limit = dto.GetLimit(20)
		offset = dto.GetOffset(20)
		status = dto.Status
		search = dto.Search
	}
	items, total, err := s.inquiryRepo.ListInquiries(ctx, status, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return models.InquiriesToResponse(items), total, nil
}

func (s *inquiryService) AnswerInquiry(ctx context.Context, teacherID string, inquiryID string, req *request.AnswerInquiryRequest) (*response.InquiryResponse, error) {
	inq, err := s.GetInquiry(ctx, inquiryID)
	if err != nil {
		return nil, err
	}
	if inq.Status != string(models.InquiryStatusPending) {
		return nil, apperrors.New(apperrors.ErrBadRequest, "inquiry is not in pending status")
	}

	teacherName := "Academic Advisor"
	if teacher, err := s.userRepo.GetByID(ctx, teacherID); err == nil && teacher != nil && teacher.FullName != "" {
		teacherName = teacher.FullName
	}

	qaDocID := constants.DocIDTeacherQA
	qaDoc, err := s.ragRepo.GetDocumentByID(ctx, qaDocID)
	if err != nil || qaDoc == nil {
		_, _ = s.ragRepo.UpsertDocument(ctx, &models.Document{
			ID:               qaDocID,
			DocCode:          constants.DocCodeTeacherQA,
			Type:             constants.DocTypeAdvisoryQA,
			Title:            "Academic Advisor Q&A Handbook",
			Domain:           constants.DomainAdvisoryQA,
			ValidityTier:     constants.ValidityTierCorePermanent,
			Status:           constants.DocStatusActive,
			DepartmentCode:   "P.QLDT",
			PublishDate:      time.Now().Format("2006-01-02"),
			AcademicYear:     constants.CurrentAcademicYear(),
			Semester:         "1",
			ApplicableCohort: constants.DocCohortAll,
			PriorityLevel:    85,
			FilePath:         "internal://teacher_qa",
		})
	}

	chunkID := fmt.Sprintf(constants.PrefixQAChunk+"%s", inquiryID)
	chunkContent := fmt.Sprintf("Question: %s\nAnswer (%s): %s", inq.Question, teacherName, strings.TrimSpace(req.Reply))

	var chunkEmbedding []float32
	if s.llmManager != nil {
		if embs, err := s.llmManager.ExecuteEmbedding(ctx, []string{chunkContent}); err == nil && len(embs) > 0 {
			chunkEmbedding = embs[0]
		}
	}

	qaChunk := &models.DocumentChunk{
		ID:               chunkID,
		DocumentID:       qaDocID,
		ChunkIndex:       0,
		ChunkType:        constants.ChunkTypeTeacherQA,
		Content:          chunkContent,
		Status:           constants.DocStatusActive,
		Domain:           constants.DomainAdvisoryQA,
		ApplicableCohort: constants.DocCohortAll,
		AcademicYear:     constants.CurrentAcademicYear(),
		PriorityLevel:    85,
		Embedding:        chunkEmbedding,
	}

	if _, err := s.ragRepo.UpsertChunk(ctx, qaChunk); err != nil {
		return nil, fmt.Errorf("failed to persist self-evolving QA chunk: %w", err)
	}

	answered, err := s.inquiryRepo.AnswerInquiry(ctx, inquiryID, teacherID, teacherName, strings.TrimSpace(req.Reply), time.Now(), chunkID)
	if err != nil {
		return nil, err
	}

	notifID := "notif_" + uuid.NewString()
	notif := &models.UserNotification{
		ID:        notifID,
		UserID:    inq.UserID,
		InquiryID: inquiryID,
		Title:     "Academic Advisor answered your inquiry",
		Content:   fmt.Sprintf("Question: %s\n\nAnswer: %s", inq.Question, strings.TrimSpace(req.Reply)),
		Type:      constants.NotifTypeInquiryAnswered,
	}
	_, _ = s.notificationRepo.CreateNotification(ctx, notif)

	return answered.ToResponse(), nil
}

func (s *inquiryService) ExpireInquiry(ctx context.Context, inquiryID, supersededByDocID, reason string) error {
	inq, err := s.GetInquiry(ctx, inquiryID)
	if err != nil {
		return err
	}

	if inq.KnowledgeChunkID != "" {
		_ = s.ragRepo.UpdateChunkStatus(ctx, inq.KnowledgeChunkID, constants.InquiryStatusExpired)
	}

	activeDocID := supersededByDocID
	if activeDocID == "" {
		activeDocID = "SYSTEM_EXPIRY"
	}
	_ = s.ragRepo.RecordSupersessionLog(ctx, activeDocID, inquiryID, constants.RuleTypeTeacherQAExpired, reason)

	_, err = s.inquiryRepo.ExpireInquiry(ctx, inquiryID, supersededByDocID, reason, time.Now())
	if err != nil {
		return err
	}

	if s.notificationRepo != nil && inq.UserID != "" {
		content := fmt.Sprintf("Câu hỏi: %s\nLý do hết hiệu lực: %s", inq.Question, reason)
		_, _ = s.notificationRepo.CreateNotification(ctx, &models.UserNotification{
			ID:        "notif_" + uuid.NewString(),
			UserID:    inq.UserID,
			InquiryID: inquiryID,
			Title:     "Thắc mắc học vụ của bạn đã hết hiệu lực",
			Content:   content,
			Type:      constants.NotifTypeInquiryExpired,
		})
	}
	return nil
}

func (s *inquiryService) CheckAndExpireOutdatedQA(ctx context.Context, doc *models.Document) (int, error) {
	if doc == nil {
		return 0, nil
	}

	activeInquiries, err := s.inquiryRepo.ListActiveAnsweredInquiries(ctx)
	if err != nil {
		return 0, err
	}

	expiredCount := 0

	for _, inq := range activeInquiries {
		shouldExpire := false
		reason := ""

		if s.engine != nil && s.llmManager != nil && len(s.llmManager.List()) > 0 {
			isOutdated, explanation, err := s.engine.AuditInquiryOutdatedness(ctx, doc, inq)
			if err == nil && isOutdated {
				shouldExpire = true
				reason = explanation
			}
		} else if s.llmManager != nil && len(s.llmManager.List()) > 0 {
			prompt := agent.PromptInquiryAuditor(doc.Title, doc.DocCode, inq.Question, inq.TeacherReply)
			lowTemp := 0.1
			resp, err := s.llmManager.Chat(ctx, &llm.ChatRequest{
				Messages: []llm.Message{
					{Role: llm.RoleSystem, Content: prompt},
				},
				Temperature: &lowTemp,
			})
			if err == nil && resp != nil && strings.HasPrefix(strings.TrimSpace(resp.Message.Content), "EXPIRED:") {
				shouldExpire = true
				reason = strings.TrimSpace(strings.TrimPrefix(resp.Message.Content, "EXPIRED:"))
			}
		}

		if !shouldExpire {
			tokens := extractSubstantiveTokens(doc.Title + " " + doc.DocCode)
			qaText := strings.ToLower(convert.StripAccents(inq.Question + " " + inq.TeacherReply))
			overlap := 0
			for _, tok := range tokens {
				if strings.Contains(qaText, tok) {
					overlap++
				}
			}
			if overlap >= 2 {
				shouldExpire = true
				reason = fmt.Sprintf("Superseded by new document: %s (%s)", doc.Title, doc.DocCode)
			}
		}

		if shouldExpire {
			if reason == "" {
				reason = fmt.Sprintf("Superseded by new document: %s (%s)", doc.Title, doc.DocCode)
			}
			if err := s.ExpireInquiry(ctx, inq.ID, doc.ID, reason); err == nil {
				expiredCount++
			}
		}
	}

	return expiredCount, nil
}

func extractSubstantiveTokens(text string) []string {
	clean := strings.ToLower(convert.StripAccents(text))
	fields := strings.FieldsFunc(clean, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	seen := make(map[string]bool)
	var tokens []string
	for _, f := range fields {
		if len(f) >= 3 && !seen[f] {
			seen[f] = true
			tokens = append(tokens, f)
		}
	}
	return tokens
}

package services

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/llm"
)

// InquiryService handles student inquiry routing, lecturer answers, and self-evolution.
type InquiryService interface {
	SetLLMManager(mgr *llm.Manager)
	CreateInquiry(ctx context.Context, userID string, req *request.CreateInquiryRequest) (*models.AdvisoryInquiry, error)
	GetInquiry(ctx context.Context, id string) (*models.AdvisoryInquiry, error)
	ListStudentInquiries(ctx context.Context, userID string, dto *request.ListStudentInquiriesDto) ([]*models.AdvisoryInquiry, int, error)
	ListTeacherInquiries(ctx context.Context, dto *request.ListTeacherInquiriesDto) ([]*models.AdvisoryInquiry, int, error)
	AnswerInquiry(ctx context.Context, teacherID string, inquiryID string, req *request.AnswerInquiryRequest) (*models.AdvisoryInquiry, error)
	ExpireInquiry(ctx context.Context, inquiryID, supersededByDocID, reason string) error
	CheckAndExpireOutdatedQA(ctx context.Context, doc *models.Document) (int, error)
}

type inquiryService struct {
	inquiryRepo      repositories.InquiryRepository
	ragRepo          repositories.RAGRepository
	userRepo         repositories.UserRepository
	notificationRepo repositories.NotificationRepository
	llmManager       *llm.Manager
}

// NewInquiryService creates an instance of InquiryService.
func NewInquiryService(
	inquiryRepo repositories.InquiryRepository,
	ragRepo repositories.RAGRepository,
	userRepo repositories.UserRepository,
	notificationRepo repositories.NotificationRepository,
) InquiryService {
	return &inquiryService{
		inquiryRepo:      inquiryRepo,
		ragRepo:          ragRepo,
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
	}
}

func (s *inquiryService) SetLLMManager(mgr *llm.Manager) {
	s.llmManager = mgr
}

func (s *inquiryService) CreateInquiry(ctx context.Context, userID string, req *request.CreateInquiryRequest) (*models.AdvisoryInquiry, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	studentName := strings.TrimSpace(req.StudentName)
	if studentName == "" {
		studentName = user.FullName
	}
	studentCode := strings.TrimSpace(req.StudentCode)
	if studentCode == "" {
		studentCode = user.StudentCode
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

	return s.inquiryRepo.CreateInquiry(ctx, inq)
}

func (s *inquiryService) GetInquiry(ctx context.Context, id string) (*models.AdvisoryInquiry, error) {
	inq, err := s.inquiryRepo.GetInquiryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if inq == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "Inquiry not found")
	}
	return inq, nil
}

func (s *inquiryService) ListStudentInquiries(ctx context.Context, userID string, dto *request.ListStudentInquiriesDto) ([]*models.AdvisoryInquiry, int, error) {
	limit := 20
	offset := 0
	if dto != nil {
		limit = dto.GetLimit(20)
		offset = dto.GetOffset(20)
	}
	return s.inquiryRepo.ListInquiriesByUserID(ctx, userID, limit, offset)
}

func (s *inquiryService) ListTeacherInquiries(ctx context.Context, dto *request.ListTeacherInquiriesDto) ([]*models.AdvisoryInquiry, int, error) {
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
	return s.inquiryRepo.ListInquiries(ctx, status, search, limit, offset)
}

func (s *inquiryService) AnswerInquiry(ctx context.Context, teacherID string, inquiryID string, req *request.AnswerInquiryRequest) (*models.AdvisoryInquiry, error) {
	inq, err := s.GetInquiry(ctx, inquiryID)
	if err != nil {
		return nil, err
	}
	if inq.Status != models.InquiryStatusPending {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Inquiry is not in PENDING status")
	}

	teacherName := "Cố vấn học tập"
	if teacher, err := s.userRepo.GetByID(ctx, teacherID); err == nil && teacher != nil && teacher.FullName != "" {
		teacherName = teacher.FullName
	}

	qaDocID := "doc_teacher_qa"
	qaDoc, err := s.ragRepo.GetDocumentByID(ctx, qaDocID)
	if err != nil || qaDoc == nil {
		_, _ = s.ragRepo.UpsertDocument(ctx, &models.Document{
			ID:               qaDocID,
			DocCode:          "QA-ADVISORY-TLU",
			Type:             "ADVISORY_QA",
			Title:            "Sổ tay hỏi đáp Cố vấn học tập",
			Domain:           "ADVISORY_QA",
			ValidityTier:     "CORE_PERMANENT",
			Status:           "ACTIVE",
			DepartmentCode:   "P.QLDT",
			PublishDate:      time.Now().Format("2006-01-02"),
			AcademicYear:     "2025-2026",
			Semester:         "1",
			ApplicableCohort: "ALL",
			PriorityLevel:    85,
			FilePath:         "internal://teacher_qa",
		})
	}

	chunkID := fmt.Sprintf("chk_qa_%s", inquiryID)
	chunkContent := fmt.Sprintf("Hỏi: %s\nTrả lời của Cố vấn học tập (%s): %s", inq.Question, teacherName, strings.TrimSpace(req.Reply))

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
		ChunkType:        "TEACHER_QA",
		Content:          chunkContent,
		Status:           "ACTIVE",
		Domain:           "ADVISORY_QA",
		ApplicableCohort: "ALL",
		AcademicYear:     "2025-2026",
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
		Title:     "Cố vấn học tập đã giải đáp thắc mắc của bạn",
		Content:   fmt.Sprintf("Câu hỏi: %s\n\nGiải đáp: %s", inq.Question, strings.TrimSpace(req.Reply)),
		Type:      "INQUIRY_ANSWERED",
	}
	_, _ = s.notificationRepo.CreateNotification(ctx, notif)

	return answered, nil
}

func (s *inquiryService) ExpireInquiry(ctx context.Context, inquiryID, supersededByDocID, reason string) error {
	inq, err := s.GetInquiry(ctx, inquiryID)
	if err != nil {
		return err
	}

	if inq.KnowledgeChunkID != "" {
		_ = s.ragRepo.UpdateChunkStatus(ctx, inq.KnowledgeChunkID, "EXPIRED")
	}

	activeDocID := supersededByDocID
	if activeDocID == "" {
		activeDocID = "SYSTEM_EXPIRY"
	}
	_ = s.ragRepo.RecordSupersessionLog(ctx, activeDocID, inquiryID, "TEACHER_QA_EXPIRED", reason)

	_, err = s.inquiryRepo.ExpireInquiry(ctx, inquiryID, supersededByDocID, reason, time.Now())
	return err
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

		if s.llmManager != nil {
			prompt := fmt.Sprintf("You are an academic regulation auditor. Does the following new document update, supersede, replace, or alter the validity of this prior academic advice given by a lecturer?\n\nNew Document:\nTitle: %s\nCode: %s\n\nPrior Lecturer Advice:\nQuestion: %s\nAnswer: %s\n\nIf this new document makes the prior advice outdated or invalid, respond ONLY with 'EXPIRED: <brief explanation>'. Otherwise, respond ONLY with 'VALID'.", doc.Title, doc.DocCode, inq.Question, inq.TeacherReply)
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
			docNorm := strings.ToLower(stripAccents(doc.Title + " " + doc.DocCode))
			qaNorm := strings.ToLower(stripAccents(inq.Question + " " + inq.TeacherReply))

			keywords := []string{
				"chuan dau ra", "tieng anh", "toeic", "hoc phi", "hoc lai",
				"bao luu", "tot nghiep", "diem ren luyen", "gdtc", "the chat", "thuc tap",
			}
			matchedCount := 0
			for _, kw := range keywords {
				if strings.Contains(docNorm, kw) && strings.Contains(qaNorm, kw) {
					matchedCount++
				}
			}
			if matchedCount >= 2 {
				shouldExpire = true
				reason = fmt.Sprintf("Nội dung bị thay thế bởi văn bản mới: %s (%s)", doc.Title, doc.DocCode)
			}
		}

		if shouldExpire {
			if reason == "" {
				reason = fmt.Sprintf("Bị thay thế bởi văn bản mới: %s (%s)", doc.Title, doc.DocCode)
			}
			if err := s.ExpireInquiry(ctx, inq.ID, doc.ID, reason); err == nil {
				expiredCount++
			}
		}
	}

	return expiredCount, nil
}

func stripAccents(text string) string {
	decomposed := norm.NFD.String(text)
	var out strings.Builder
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if r == 'đ' || r == 'Đ' {
			out.WriteRune('d')
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

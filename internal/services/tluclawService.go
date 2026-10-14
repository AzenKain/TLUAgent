package services

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"tluagent-web/internal/agent"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/compactor"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/docparser"
	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/tuition"
)

// TLUClawService manages conversational interactions, tool invocations, and two-phase commits for administrators.
type TLUClawService interface {
	CreateSession(ctx context.Context, adminID, title string) (*response.TLUClawSessionResponse, error)
	ListSessions(ctx context.Context, adminID string, limit, offset int32) ([]*response.TLUClawSessionResponse, error)
	GetSessionDetail(ctx context.Context, sessionID, adminID string) (*response.TLUClawSessionDetailResponse, error)
	DeleteSession(ctx context.Context, sessionID, adminID string) error

	ProcessAdminMessage(ctx context.Context, adminID string, req *request.TLUClawChatRequest) (*response.TLUClawChatResponse, error)
	ConfirmSnapshot(ctx context.Context, adminID string, req *request.TLUClawConfirmRequest) (*response.TLUClawActionResponse, error)
	CancelSnapshot(ctx context.Context, adminID string, req *request.TLUClawCancelRequest) (*response.TLUClawActionResponse, error)
	RollbackBatch(ctx context.Context, adminID, batchID string) (*response.TLUClawActionResponse, error)
	ListSnapshots(ctx context.Context, status string, limit, offset int32) (*response.TLUClawSnapshotListResponse, error)
}

// TLUClawParams encapsulates dependencies for TLUClaw services and tool engine.
type TLUClawParams struct {
	TLUClawRepo    repositories.TLUClawRepository
	CurriculumRepo repositories.CurriculumRepository
	RAGRepo        repositories.RAGRepository
	RAGService     RAGService
	RAMCache       cache.Cache
	LLMManager     *llm.Manager
	SkillLoader    agent.SkillLoader
	MemoryService  MemoryService
	UserRepo       repositories.UserRepository
	RoleRepo       repositories.RoleRepository
	TranscriptRepo repositories.TranscriptRepository
	InquiryRepo    repositories.InquiryRepository
	LLMConfigSvc   LLMConfigService
	MemoryRepo     repositories.MemoryRepository
	NotifRepo      repositories.NotificationRepository
	JobService     JobService
	TuitionClient  *tuition.Client
	RoleService      RoleService
	SettingsRepo     repositories.SettingsRepository
	DissectionEngine *RegulatoryDissectionEngine
}

type tluclawService struct {
	tluclawRepo      repositories.TLUClawRepository
	curriculumRepo   repositories.CurriculumRepository
	ragRepo          repositories.RAGRepository
	notifRepo        repositories.NotificationRepository
	memoryService    MemoryService
	userRepo         repositories.UserRepository
	roleRepo         repositories.RoleRepository
	transcriptRepo   repositories.TranscriptRepository
	inquiryRepo      repositories.InquiryRepository
	llmConfigSvc     LLMConfigService
	memoryRepo       repositories.MemoryRepository
	jobSvc           JobService
	tuitionClient    *tuition.Client
	roleSvc          RoleService
	ragService       RAGService
	toolEngine       TLUClawToolEngine
	c                cache.Cache
	llmManager       *llm.Manager
	settingsRepo     repositories.SettingsRepository
	dissectionEngine *RegulatoryDissectionEngine
}

// NewTLUClawService creates a new administrative assistant service.
func NewTLUClawService(params TLUClawParams) TLUClawService {
	dissectionEngine := params.DissectionEngine
	if dissectionEngine == nil {
		dissectionEngine = NewRegulatoryDissectionEngine(EngineParams{
			RAGRepo:        params.RAGRepo,
			CurriculumRepo: params.CurriculumRepo,
			InquiryRepo: params.InquiryRepo,
			NotifRepo:   params.NotifRepo,
			UserRepo:    params.UserRepo,
			LLMManager:  params.LLMManager,
		})
	}
	params.DissectionEngine = dissectionEngine
	engine := NewTLUClawToolEngine(params)
	tuitionClient := params.TuitionClient
	if tuitionClient == nil {
		tuitionClient = tuition.NewClient(10 * time.Second)
	}
	return &tluclawService{
		tluclawRepo:      params.TLUClawRepo,
		curriculumRepo:   params.CurriculumRepo,
		ragRepo:          params.RAGRepo,
		notifRepo:        params.NotifRepo,
		memoryService:    params.MemoryService,
		userRepo:         params.UserRepo,
		roleRepo:         params.RoleRepo,
		transcriptRepo:   params.TranscriptRepo,
		inquiryRepo:      params.InquiryRepo,
		llmConfigSvc:     params.LLMConfigSvc,
		memoryRepo:       params.MemoryRepo,
		jobSvc:           params.JobService,
		tuitionClient:    tuitionClient,
		roleSvc:          params.RoleService,
		ragService:       params.RAGService,
		toolEngine:       engine,
		c:                params.RAMCache,
		llmManager:       params.LLMManager,
		settingsRepo:     params.SettingsRepo,
		dissectionEngine: dissectionEngine,
	}
}

func (s *tluclawService) CreateSession(ctx context.Context, adminID, title string) (*response.TLUClawSessionResponse, error) {
	sessionID := "sess_" + uuid.NewString()[:8]
	if strings.TrimSpace(title) == "" {
		title = "Phiên làm việc mới"
	}
	sess, err := s.tluclawRepo.CreateSession(ctx, sqlc.CreateTLUClawSessionParams{
		ID:          sessionID,
		AdminUserID: adminID,
		Title:       title,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return sess.ToResponse(), nil
}

func (s *tluclawService) ListSessions(ctx context.Context, adminID string, limit, offset int32) ([]*response.TLUClawSessionResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	sessions, err := s.tluclawRepo.ListSessionsByAdmin(ctx, adminID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]*response.TLUClawSessionResponse, 0, len(sessions))
	for _, sess := range sessions {
		out = append(out, sess.ToResponse())
	}
	return out, nil
}

func (s *tluclawService) GetSessionDetail(ctx context.Context, sessionID, adminID string) (*response.TLUClawSessionDetailResponse, error) {
	sess, err := s.tluclawRepo.GetSessionByID(ctx, sessionID)
	if err != nil || sess == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, fmt.Sprintf("session not found: %s", sessionID))
	}
	if sess.AdminUserID != adminID {
		return nil, apperrors.New(apperrors.ErrForbidden, "access denied to this session")
	}

	messages, err := s.tluclawRepo.ListMessagesBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	snapshots, err := s.tluclawRepo.ListSnapshotsBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	msgResponses := make([]*response.TLUClawMessageResponse, len(messages))
	for i, m := range messages {
		msgResponses[i] = m.ToResponse()
	}

	cardResponses := make([]*response.TLUClawPreviewCardResponse, len(snapshots))
	for i, snp := range snapshots {
		cardResponses[i] = snp.ToPreviewResponse()
	}

	return &response.TLUClawSessionDetailResponse{
		Session:  sess.ToResponse(),
		Messages: msgResponses,
		Cards:    cardResponses,
	}, nil
}

func (s *tluclawService) DeleteSession(ctx context.Context, sessionID, adminID string) error {
	sess, err := s.tluclawRepo.GetSessionByID(ctx, sessionID)
	if err != nil || sess == nil {
		return apperrors.New(apperrors.ErrNotFound, fmt.Sprintf("session not found: %s", sessionID))
	}
	if sess.AdminUserID != adminID {
		return apperrors.New(apperrors.ErrForbidden, "access denied to this session")
	}
	return s.tluclawRepo.DeleteSession(ctx, sessionID, adminID)
}

func (s *tluclawService) buildSessionOperationalStatePromptContext(ctx context.Context, sessionID string) (string, error) {
	if sessionID == "" {
		return "", nil
	}

	snapshots, err := s.tluclawRepo.ListSnapshotsBySessionID(ctx, sessionID)
	if err != nil {
		return "", err
	}

	var stagedSnapshots []*models.TLUClawActionSnapshot
	for _, snp := range snapshots {
		if snp.Status == "STAGED" {
			stagedSnapshots = append(stagedSnapshots, snp)
		}
	}

	var sb strings.Builder
	sb.WriteString("[SESSION OPERATIONAL STATE & ACTIVE PENDING STAGES]\n")
	fmt.Fprintf(&sb, "- Current Session ID: %s\n", sessionID)
	if len(stagedSnapshots) == 0 {
		sb.WriteString("- Pending Staged Snapshots: None (Clean state, ready for new operational workflows).\n")
		return sb.String(), nil
	}

	fmt.Fprintf(&sb, "- Pending Staged Snapshots (%d awaiting confirmation or modification):\n", len(stagedSnapshots))
	for _, snp := range stagedSnapshots {
		expiresStr := "No expiry"
		if snp.NonceExpiresAt != nil {
			expiresStr = snp.NonceExpiresAt.Format("15:04:05 02/01/2006")
		}
		fmt.Fprintf(&sb, "  * Snapshot ID: %s | Batch ID: %s | Action: %s\n", snp.ID, snp.BatchID, snp.ActionType)
		fmt.Fprintf(&sb, "    Summary: %s\n", snp.Summary)
		fmt.Fprintf(&sb, "    Status: STAGED | Confirmation Nonce: %s (Expires: %s)\n", snp.ConfirmationNonce, expiresStr)
	}
	sb.WriteString("(Note: If the administrator requests to confirm, cancel, or modify the staged changes, you have full context of this pending stage).\n")

	return sb.String(), nil
}

func (s *tluclawService) ProcessAdminMessage(ctx context.Context, adminID string, req *request.TLUClawChatRequest) (*response.TLUClawChatResponse, error) {
	sessionID := req.SessionID
	if sessionID == "" {
		title := req.Message
		if len([]rune(title)) > 40 {
			title = string([]rune(title)[:40]) + "..."
		}
		newSess, err := s.CreateSession(ctx, adminID, title)
		if err != nil {
			return nil, fmt.Errorf("create session: %w", err)
		}
		sessionID = newSess.ID
	} else {
		sess, err := s.tluclawRepo.GetSessionByID(ctx, sessionID)
		if err != nil || sess == nil {
			return nil, apperrors.New(apperrors.ErrNotFound, fmt.Sprintf("session not found: %s", sessionID))
		}
		if sess.AdminUserID != adminID {
			return nil, apperrors.New(apperrors.ErrForbidden, "access denied to this session")
		}
	}

	userMsgParams := sqlc.CreateTLUClawMessageParams{
		ID:               "msg_" + uuid.NewString()[:8],
		SessionID:        sessionID,
		Role:             "user",
		Content:          req.Message,
		ReasoningContent: "",
		ToolCallsJson:    "[]",
		ToolCallID:       "",
		MetadataJson:     "{}",
		CreatedAt:        time.Now(),
	}
	if _, err := s.tluclawRepo.CreateMessage(ctx, userMsgParams); err != nil {
		return nil, fmt.Errorf("save user message: %w", err)
	}

	historyMsgs, _ := s.tluclawRepo.ListMessagesBySessionID(ctx, sessionID)
	var dialogue []compactor.Message
	if len(historyMsgs) > 1 {
		for _, hm := range historyMsgs[:len(historyMsgs)-1] {
			role := compactor.RoleUser
			if hm.Role == "assistant" {
				role = compactor.RoleAssistant
			} else if hm.Role == "system" {
				role = compactor.RoleSystem
			}
			dialogue = append(dialogue, compactor.Message{
				Role:    role,
				Content: hm.Content,
			})
		}
	}

	contextLimit := 250000
	thresholdRatio := 0.80
	keepTurns := 10

	if s.settingsRepo != nil {
		if raw, err := s.settingsRepo.GetAppSetting(ctx, "compactor_settings"); err == nil && raw != "" {
			var customCfg request.UpdateCompactorSettingsRequest
			if err := json.Unmarshal([]byte(raw), &customCfg); err == nil {
				if customCfg.MaxContextTokens >= 1000 {
					contextLimit = customCfg.MaxContextTokens
				}
				if customCfg.CompactThresholdRatio > 0 && customCfg.CompactThresholdRatio <= 1.0 {
					thresholdRatio = customCfg.CompactThresholdRatio
				}
				if customCfg.KeepRecentTurns >= 2 {
					keepTurns = customCfg.KeepRecentTurns
				}
			}
		}
	}

	cfg := compactor.DynamicConfig(contextLimit, keepTurns)
	cfg.CompactThresholdRatio = thresholdRatio
	if s.llmManager != nil {
		cfg.Summarizer = compactor.CreateLLMSummarizer(func(c context.Context, p string) (string, error) {
			resp, err := s.llmManager.Chat(c, &llm.ChatRequest{
				Messages: []llm.Message{
					{Role: llm.RoleSystem, Content: p},
				},
			})
			if err != nil || resp == nil {
				return "", err
			}
			return resp.Message.Content, nil
		})
	}
	compacted, didCompact, _ := compactor.CompactHistory(ctx, dialogue, cfg)

	nonce, _ := crypto.GenerateRandomHex(8)
	if nonce == "" {
		nonce = uuid.NewString()[:8]
	}

	var adminMemContext string
	if s.memoryService != nil {
		adminMemContext, _ = s.memoryService.BuildAdminLongTermMemoryPromptContext(ctx, adminID)
	}

	sessionStateContext, _ := s.buildSessionOperationalStatePromptContext(ctx, sessionID)

	var sb strings.Builder
	sb.WriteString(agent.PromptTLUClawBaseSystem())

	if adminMemContext != "" {
		fmt.Fprintf(&sb, "\n\n<admin_memory nonce=\"%s\">\n%s\n</admin_memory>", nonce, adminMemContext)
	}

	if sessionStateContext != "" {
		fmt.Fprintf(&sb, "\n\n<session_operational_state nonce=\"%s\">\n%s\n</session_operational_state>", nonce, sessionStateContext)
	}

	systemPrompt := sb.String()

	userPromptWithAttachments := req.Message
	if len(req.Attachments) > 0 {
		var attBuilder strings.Builder
		attBuilder.WriteString("\n\n[Đính kèm tài liệu đã bóc tách]:\n")
		for _, att := range req.Attachments {
			rawBytes := []byte(att.Data)
			if idx := strings.Index(att.Data, ";base64,"); idx != -1 {
				if dec, err := base64.StdEncoding.DecodeString(att.Data[idx+8:]); err == nil {
					rawBytes = dec
				}
			} else if dec, err := base64.StdEncoding.DecodeString(att.Data); err == nil && len(dec) > 0 {
				rawBytes = dec
			}

			parsed, err := docparser.ParseBytes(att.FileName, rawBytes)
			if err == nil && parsed != nil {
				attBuilder.WriteString(fmt.Sprintf("--- Tập tin: %s (%s) ---\n", att.FileName, parsed.FileType))
				if len(parsed.Courses) > 0 {
					coursesJSON, _ := json.Marshal(parsed.Courses)
					attBuilder.WriteString(fmt.Sprintf("Tự động trích xuất %d môn học (Độ chính xác 100%%):\n```json\n%s\n```\n\n", len(parsed.Courses), string(coursesJSON)))
				} else if parsed.TextContent != "" {
					textSnippet := parsed.TextContent
					if len(textSnippet) > 8000 {
						textSnippet = textSnippet[:8000] + "\n...[Đã cắt bớt văn bản quá dài]"
					}
					attBuilder.WriteString(fmt.Sprintf("Nội dung văn bản:\n%s\n\n", textSnippet))
				}
			} else {
				attBuilder.WriteString(fmt.Sprintf("--- File: %s (%s) ---\n%s\n\n", att.FileName, att.FileType, att.Data))
			}
		}
		userPromptWithAttachments += attBuilder.String()
	}

	userPromptWithAttachments = fmt.Sprintf("<admin_query nonce=\"%s\">\n%s\n</admin_query>", nonce, userPromptWithAttachments)

	var llmMessages []llm.Message
	llmMessages = append(llmMessages, llm.SystemMessage(systemPrompt))

	for _, m := range compacted {
		role := llm.RoleUser
		switch m.Role {
		case compactor.RoleAssistant:
			role = llm.RoleAssistant
		case compactor.RoleSystem:
			role = llm.RoleSystem
		}
		llmMessages = append(llmMessages, llm.Message{
			Role:    role,
			Content: m.Content,
		})
	}

	if len(req.Images) > 0 {
		for _, img := range req.Images {
			llmMessages = append(llmMessages, llm.UserMessageWithImageURL(userPromptWithAttachments, img))
		}
	} else {
		llmMessages = append(llmMessages, llm.UserMessage(userPromptWithAttachments))
	}

	if s.llmManager == nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "chat system not configured; please try again later")
	}

	provider, err := s.llmManager.Default()
	if err != nil {
		log.Warn().Err(err).Msg("tluclaw ProcessAdminMessage: resolve default llm provider failed")
		return nil, apperrors.New(apperrors.ErrInternalError, "failed to connect to AI model; please try again later")
	}

	temp := 0.2
	maxTurns := 6
	var previewCard *response.TLUClawPreviewCardResponse
	var assistantReplyText string
	var toolOutputs []string

	for turn := 0; turn < maxTurns; turn++ {
		chatReq := &llm.ChatRequest{
			Model:       provider.DefaultModel(),
			Messages:    llmMessages,
			Tools:       s.toolEngine.GetToolDefinitions(),
			Temperature: &temp,
		}

		chatResp, err := provider.Chat(ctx, chatReq)
		if err != nil {
			log.Warn().Err(err).Msg("tluclaw ProcessAdminMessage: provider chat failed")
			break
		}
		if chatResp == nil {
			break
		}

		if len(chatResp.Message.ToolCalls) > 0 {
			llmMessages = append(llmMessages, chatResp.Message)
			for _, tc := range chatResp.Message.ToolCalls {
				resStr, card, toolErr := s.toolEngine.ExecuteTool(ctx, sessionID, adminID, tc)
				if toolErr != nil {
					resStr = fmt.Sprintf("Lỗi khi thực thi công cụ: %v", toolErr)
				}
				if card != nil {
					previewCard = card
				}
				toolOutputs = append(toolOutputs, resStr)
				llmMessages = append(llmMessages, llm.Message{
					Role:       llm.RoleTool,
					ToolCallID: tc.ID,
					Content:    resStr,
				})
			}
			continue
		}

		assistantReplyText = chatResp.Message.Content
		break
	}

	if assistantReplyText == "" {
		if len(toolOutputs) > 0 {
			assistantReplyText = strings.Join(toolOutputs, "\n\n")
		} else {
			assistantReplyText = "Đã tiếp nhận yêu cầu từ Quản trị viên."
		}
	}

	metadataJSON := "{}"
	if didCompact {
		metadataJSON = "{\"did_compact\":true}"
	}
	botMsgParams := sqlc.CreateTLUClawMessageParams{
		ID:               "msg_" + uuid.NewString()[:8],
		SessionID:        sessionID,
		Role:             "assistant",
		Content:          assistantReplyText,
		ReasoningContent: "",
		ToolCallsJson:    "[]",
		ToolCallID:       "",
		MetadataJson:     metadataJSON,
		CreatedAt:        time.Now(),
	}
	botMsg, _ := s.tluclawRepo.CreateMessage(ctx, botMsgParams)

	if s.memoryService != nil {
		go func() {
			_ = s.memoryService.ExtractAndSaveAdminMemories(context.Background(), adminID, req.Message, assistantReplyText)
		}()
	}

	return &response.TLUClawChatResponse{
		SessionID:        sessionID,
		MessageID:        botMsg.ID,
		Reply:            assistantReplyText,
		ReasoningContent: "",
		PreviewCard:      previewCard,
		DidCompact:       didCompact,
		CreatedAt:        time.Now(),
	}, nil
}

func (s *tluclawService) ConfirmSnapshot(ctx context.Context, adminID string, req *request.TLUClawConfirmRequest) (*response.TLUClawActionResponse, error) {
	snapshot, err := s.tluclawRepo.GetSnapshotByNonce(ctx, req.Nonce)
	if err != nil || snapshot == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, fmt.Sprintf("snapshot preview not found for nonce: %s", req.Nonce))
	}

	if snapshot.ID != req.SnapshotID {
		return nil, apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("snapshot id mismatch: expected %s, got %s", snapshot.ID, req.SnapshotID))
	}

	if snapshot.OperatorID != adminID {
		return nil, apperrors.New(apperrors.ErrForbidden, "access denied to snapshot preview")
	}

	if snapshot.Status != "STAGED" {
		return nil, apperrors.New(apperrors.ErrConflict, fmt.Sprintf("snapshot is in status '%s', cannot confirm", snapshot.Status))
	}

	if snapshot.NonceExpiresAt != nil && time.Now().After(*snapshot.NonceExpiresAt) {
		return nil, apperrors.New(apperrors.ErrBadRequest, "confirmation nonce has expired")
	}

	switch snapshot.ActionType {
	case "academic_structure":
		var state request.TLUClawAcademicStructureSnapshotState
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil {
			if state.Cohort != "" {
				_, _ = s.curriculumRepo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
					ID:           "CURR_" + state.Cohort,
					Cohort:       state.Cohort,
					AcademicYear: state.AcademicYear,
					MajorName:    "GENERAL",
					MajorCode:    "GEN",
					FacultyCode:  "ALL",
					Aliases:      "",
					TotalCourses: 0,
					TotalCredits: 0,
				})
			}
			if state.MajorCode != "" {
				aliasesJoined := strings.Join(state.Aliases, ",")
				_ = s.curriculumRepo.UpdateMajorMetadataByName(ctx, state.MajorNameVi, state.MajorCode, state.FacultyCode, aliasesJoined)
				_, _ = s.curriculumRepo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
					ID:           "CURR_MAJ_" + state.MajorCode,
					Cohort:       constants.DocCohortAll,
					AcademicYear: constants.CurrentAcademicYear(),
					MajorName:    state.MajorNameVi,
					MajorCode:    state.MajorCode,
					FacultyCode:  state.FacultyCode,
					Aliases:      aliasesJoined,
					TotalCourses: 0,
					TotalCredits: 0,
				})
				s.curriculumRepo.InvalidateCurriculumCache(ctx, "", "")
			}
		}

	case "curriculum_update":
		var state request.TLUClawCurriculumSnapshotState
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil {
			currID := fmt.Sprintf("CURR_%s_%s", state.Cohort, state.MajorCode)
			_, _ = s.curriculumRepo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
				ID:           currID,
				Cohort:       state.Cohort,
				AcademicYear: state.Cohort,
				MajorName:    state.MajorCode,
				MajorCode:    state.MajorCode,
				FacultyCode:  "ALL",
				Aliases:      "",
				TotalCourses: int64(len(state.IncomingCourses)),
				TotalCredits: 0,
			})

			existingCourses, _ := s.curriculumRepo.GetCurriculumCoursesByCohortAndMajor(ctx, state.Cohort, state.MajorCode)
			existingMap := make(map[string]*models.CurriculumCourse)
			for _, ec := range existingCourses {
				existingMap[strings.ToUpper(strings.TrimSpace(ec.CourseCode))] = ec
			}

			selectedMap := make(map[string]bool)
			if len(req.SelectedKeys) > 0 {
				for _, k := range req.SelectedKeys {
					selectedMap[strings.ToUpper(strings.TrimSpace(k))] = true
				}
			}

			for _, inc := range state.IncomingCourses {
				cleanCode := strings.ToUpper(strings.TrimSpace(inc.CourseCode))
				if len(req.SelectedKeys) > 0 && !selectedMap[cleanCode] {
					continue
				}

				courseID := fmt.Sprintf("C_%s_%s_%s", state.Cohort, state.MajorCode, cleanCode)
				if existing, exists := existingMap[cleanCode]; exists {
					courseID = existing.ID
				}
				_, _ = s.curriculumRepo.UpsertCurriculumCourse(ctx, sqlc.UpsertCurriculumCourseParams{
					ID:                courseID,
					CurriculumID:      currID,
					Cohort:            state.Cohort,
					MajorName:         state.MajorCode,
					CourseCode:        cleanCode,
					CourseName:        inc.CourseName,
					Credits:           inc.Credits,
					Weight:            inc.Weight,
					Hours:             inc.Hours,
					KnowledgeBlock:    inc.KnowledgeBlock,
					PrerequisitesText: inc.PrerequisitesText,
					SuggestedSemester: inc.SuggestedSemester,
					IsElearning:       inc.IsElearning,
				})
			}
			s.curriculumRepo.InvalidateCurriculumCache(ctx, state.Cohort, state.MajorCode)
			if s.c != nil {
				_ = s.c.DelByPattern(ctx, "curriculum:*")
				_ = s.c.DelByPattern(ctx, "course:*")
			}
		}

	case "document_supersession", "document_ingestion":
		var state request.TLUClawDocumentSnapshotState
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil {
			if state.PlanJSON != "" && s.dissectionEngine != nil {
				var plan GranularSupersessionPlan
				if err := json.Unmarshal([]byte(state.PlanJSON), &plan); err == nil {
					_ = s.dissectionEngine.ApplyPlan(ctx, &plan, &DissectionRequest{
						NewDocID:         state.NewDocID,
						Title:            state.Title,
						DocCode:          state.NewDocID,
						DocType:          state.DocType,
						Domain:           state.Domain,
						ApplicableCohort: state.Cohort,
						AcademicYear:     state.AcademicYear,
						Content:          state.Content,
					})
					break
				}
			}
			docModel := &models.Document{
				ID:               state.NewDocID,
				Title:            state.Title,
				Type:             state.DocType,
				Domain:           state.Domain,
				ApplicableCohort: state.Cohort,
				AcademicYear:     state.AcademicYear,
				Status:           "ACTIVE",
				PriorityLevel:    10,
				CreatedAt:        time.Now().Format(time.RFC3339),
			}
			_, _ = s.ragRepo.UpsertDocument(ctx, docModel)

			if strings.TrimSpace(state.Content) != "" {
				chunkTexts := splitIntoChunks(state.Content, 1200, 150)
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
						ID:               fmt.Sprintf("chk_%s_%d", state.NewDocID, idx),
						DocumentID:       state.NewDocID,
						ChunkIndex:       idx,
						ChunkType:        "CONTENT_CHUNK",
						Content:          text,
						Status:           "ACTIVE",
						Domain:           state.Domain,
						ApplicableCohort: state.Cohort,
						AcademicYear:     state.AcademicYear,
						PriorityLevel:    10,
						Embedding:        emb,
					})
				}
				if len(chunks) > 0 {
					_ = s.ragRepo.UpsertChunksBatch(ctx, chunks)
				}
			}

			docNode := &models.KGNode{
				ID:       "doc:" + state.NewDocID,
				NodeType: "DOCUMENT",
				Name:     state.Title,
				Code:     state.NewDocID,
				Status:   "ACTIVE",
			}
			_, _ = s.ragRepo.UpsertKGNode(ctx, docNode)

			if state.Cohort != "" && state.Cohort != "ALL" {
				cohortNode := &models.KGNode{
					ID:       "cohort:" + state.Cohort,
					NodeType: "COHORT",
					Name:     "Cohort " + state.Cohort,
					Code:     state.Cohort,
					Status:   "ACTIVE",
				}
				_, _ = s.ragRepo.UpsertKGNode(ctx, cohortNode)
				_, _ = s.ragRepo.UpsertKGEdge(ctx, &models.KGEdge{
					ID:           fmt.Sprintf("edge:cohort:%s:%s", state.NewDocID, state.Cohort),
					SourceNodeID: docNode.ID,
					TargetNodeID: cohortNode.ID,
					RelationType: "APPLIES_TO_COHORT",
					Weight:       1.0,
					Status:       "ACTIVE",
				})
			}

			for _, oldID := range state.SupersededDocIDs {
				_ = s.ragRepo.UpdateDocumentStatus(ctx, oldID, "SUPERSEDED", state.NewDocID)
				_ = s.ragRepo.RecordSupersessionLog(ctx, state.NewDocID, oldID, "DOCUMENT_REPLACEMENT", "Superseded via TLUClaw administrative action")

				edge := &models.KGEdge{
					ID:           "edge_" + uuid.NewString()[:8],
					SourceNodeID: "doc:" + state.NewDocID,
					TargetNodeID: "doc:" + oldID,
					RelationType: "SUPERSEDES",
					Weight:       1.0,
				}
				_, _ = s.ragRepo.UpsertKGEdge(ctx, edge)
			}

			if s.c != nil {
				_ = s.c.DelByPattern(ctx, "rag_search:*")
				_ = s.c.DelByPattern(ctx, "rag_candidates:*")
			}
		}

	case "substitution_rule":
		var state request.TLUClawSubstitutionSnapshotState
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil {
			var isTransitiveInt int64
			_, _ = s.curriculumRepo.UpsertCourseSubstitution(ctx, sqlc.UpsertCourseSubstitutionParams{
				ID:                    state.ID,
				OriginalCourseCode:    state.OriginalCourseCode,
				OriginalCourseName:    state.OriginalCourseName,
				ReplacementCourseCode: state.ReplacementCourseCode,
				ReplacementCourseName: state.ReplacementCourseName,
				ScopeCohort:           fmt.Sprintf("K%d..K%d", state.MinCohort, state.MaxCohort),
				ScopeMajor:            state.ScopeMajor,
				MinCohort:             state.MinCohort,
				MaxCohort:             state.MaxCohort,
				IsBidirectional:       state.IsBidirectional,
				IsTransitive:          isTransitiveInt,
				CreditsOriginal:       state.CreditsOriginal,
				CreditsReplacement:    state.CreditsReplacement,
				RuleType:              "COURSE_SUBSTITUTION",
				RuleNote:              state.RuleNote,
				SourceReference:       "TLUClaw Operation",
			})
			if state.IsBidirectional == 1 {
				reverseID := fmt.Sprintf("%s_rev", state.ID)
				_, _ = s.curriculumRepo.UpsertCourseSubstitution(ctx, sqlc.UpsertCourseSubstitutionParams{
					ID:                    reverseID,
					OriginalCourseCode:    state.ReplacementCourseCode,
					OriginalCourseName:    state.ReplacementCourseName,
					ReplacementCourseCode: state.OriginalCourseCode,
					ReplacementCourseName: state.OriginalCourseName,
					ScopeCohort:           fmt.Sprintf("K%d..K%d", state.MinCohort, state.MaxCohort),
					ScopeMajor:            state.ScopeMajor,
					MinCohort:             state.MinCohort,
					MaxCohort:             state.MaxCohort,
					IsBidirectional:       state.IsBidirectional,
					IsTransitive:          isTransitiveInt,
					CreditsOriginal:       state.CreditsReplacement,
					CreditsReplacement:    state.CreditsOriginal,
					RuleType:              "COURSE_SUBSTITUTION",
					RuleNote:              state.RuleNote,
					SourceReference:       "TLUClaw Operation (Bidirectional)",
				})
			}
			s.curriculumRepo.InvalidateCurriculumCache(ctx, "", "")
			if s.c != nil {
				_ = s.c.DelByPattern(ctx, "curr*")
			}
		}

	case "prerequisite_rule":
		var state request.TLUClawPrerequisiteSnapshotState
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil {
			minGradeVal, _ := strconv.ParseFloat(state.MinGrade, 64)
			if minGradeVal <= 0 {
				minGradeVal = 1.0
			}
			_, _ = s.curriculumRepo.UpsertCoursePrerequisite(ctx, sqlc.UpsertCoursePrerequisiteParams{
				ID:                     state.ID,
				CurriculumCourseID:     state.CurriculumCourseID,
				Cohort:                 state.Cohort,
				MajorName:              state.MajorName,
				CourseCode:             state.CourseCode,
				PrerequisiteCourseCode: state.PrerequisiteCourseCode,
				MinGrade:               minGradeVal,
				ConditionType:          state.ConditionType,
				RawExpression:          state.RawExpression,
			})
			s.curriculumRepo.InvalidateCurriculumCache(ctx, state.Cohort, "")
			if s.c != nil {
				_ = s.c.DelByPattern(ctx, "curr*")
			}
		}

	case "advisor_note":
		var state request.TLUClawAdvisorNoteSnapshotState
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil {
			if s.memoryRepo != nil {
				profile, _ := s.memoryRepo.GetStudentProfile(ctx, state.UserID)
				if profile == nil {
					profile = &models.StudentProfile{
						UserID:           state.UserID,
						Major:            "",
						Cohort:           "",
						AcademicStanding: "NORMAL",
						AdvisorNotes:     state.AdvisorNote,
						UpdatedAt:        time.Now(),
					}
				} else {
					profile.AdvisorNotes = state.AdvisorNote
					profile.UpdatedAt = time.Now()
				}
				_, _ = s.memoryRepo.UpsertStudentProfile(ctx, profile)
				s.memoryRepo.InvalidateProfileCache(ctx, state.UserID)
			}
		}

	case "user_status":
		var state request.TLUClawUserStatusSnapshotState
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil {
			if state.Status == "BANNED" {
				if s.roleRepo != nil {
					bannedRole, _ := s.roleRepo.GetByName(ctx, "BANNED")
					if bannedRole != nil {
						_ = s.roleRepo.CreateUserRole(ctx, state.UserID, bannedRole.ID)
					}
				}
				if s.userRepo != nil {
					_ = s.userRepo.RevokeSessions(ctx, state.UserID)
				}
			} else if state.Status == "ACTIVE" {
				if s.userRepo != nil {
					_ = s.userRepo.Restore(ctx, state.UserID)
				}
				if s.roleRepo != nil && s.userRepo != nil {
					u, _ := s.userRepo.GetByID(ctx, state.UserID)
					if u != nil {
						_ = s.roleRepo.BulkDeleteRolesFromUser(ctx, state.UserID)
						for _, r := range u.Roles {
							if r.Name != "BANNED" {
								_ = s.roleRepo.CreateUserRole(ctx, state.UserID, r.ID)
							}
						}
					}
				}
			}
			if s.userRepo != nil {
				s.userRepo.InvalidateUserCache(ctx, state.UserID, state.Email)
			}
		}

	case "user_role":
		var state request.TLUClawUserRoleChangeSnapshotState
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil {
			if s.roleRepo != nil {
				if state.RoleAction == "ASSIGN" {
					_ = s.roleRepo.CreateUserRole(ctx, state.UserID, state.RoleID)
				} else if state.RoleAction == "REMOVE" {
					if s.userRepo != nil {
						u, _ := s.userRepo.GetByID(ctx, state.UserID)
						if u != nil {
							_ = s.roleRepo.BulkDeleteRolesFromUser(ctx, state.UserID)
							for _, r := range u.Roles {
								if r.ID != state.RoleID {
									_ = s.roleRepo.CreateUserRole(ctx, state.UserID, r.ID)
								}
							}
						}
					}
				}
			}
			if s.userRepo != nil {
				_ = s.userRepo.RevokeSessions(ctx, state.UserID)
				s.userRepo.InvalidateUserCache(ctx, state.UserID, state.Email)
			}
		}

	case "inquiry_reply":
		var state request.TLUClawInquiryReplySnapshotState
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil {
			if s.inquiryRepo != nil {
				teacherName := "TLU Administrator"
				if s.userRepo != nil {
					if adminUser, err := s.userRepo.GetByID(ctx, adminID); err == nil && adminUser != nil {
						teacherName = adminUser.FullName
					}
				}
				_, _ = s.inquiryRepo.AnswerInquiry(ctx, state.InquiryID, adminID, teacherName, state.Reply, time.Now(), state.ChunkID)
				s.inquiryRepo.InvalidateInquiryCache(ctx, state.InquiryID, "")
			}
		}

	case "llm_config":
		var state request.TLUClawLLMConfigSnapshotState
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil {
			if s.llmConfigSvc != nil {
				isDef := true
				_, _ = s.llmConfigSvc.UpdateModel(ctx, state.ModelID, request.UpdateLLMModelRequest{
					IsDefault: &isDef,
				})
			}
			if s.c != nil {
				_ = s.c.DelByPattern(ctx, "llm*")
			}
		}

	case "document_status":
		var state request.TLUClawDocumentStatusSnapshotState
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil && s.ragRepo != nil {
			_ = s.ragRepo.UpdateDocumentStatus(ctx, state.DocumentID, state.Status, "")
			if s.c != nil {
				_ = s.c.DelByPattern(ctx, "rag_search:*")
				_ = s.c.DelByPattern(ctx, "rag_candidates:*")
				_ = s.c.Del(ctx, cache.BuildKey("document", "id", state.DocumentID))
			}
		}

	case "notification_dispatch":
		var state request.TLUClawNotificationDispatchSnapshotState
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil && s.notifRepo != nil {
			for _, n := range state.Notifications {
				_, _ = s.notifRepo.CreateNotification(ctx, &models.UserNotification{
					ID:      n.ID,
					UserID:  n.UserID,
					Title:   n.Title,
					Content: n.Content,
					Type:    n.Type,
				})
			}
		}

	case "job_trigger":
		var state request.TLUClawJobTriggerSnapshotState
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil && s.jobSvc != nil {
			_, _ = s.jobSvc.Trigger(ctx, state.TaskType, state.Payload)
		}
	}

	appliedTime := sql.NullTime{Time: time.Now(), Valid: true}
	if err := s.tluclawRepo.UpdateSnapshotStatus(ctx, snapshot.ID, "COMMITTED", appliedTime); err != nil {
		return nil, fmt.Errorf("update snapshot status: %w", err)
	}

	if s.notifRepo != nil {
		var blast models.TLUClawBlastRadius
		_ = json.Unmarshal([]byte(snapshot.BlastRadiusJSON), &blast)
		if len(blast.AffectedCohorts) > 0 || len(blast.AffectedMajors) > 0 {
			notifTitle := fmt.Sprintf("Thông báo cập nhật học vụ: %s", snapshot.Summary)
			notifContent := fmt.Sprintf("Hệ thống vừa cập nhật dữ liệu đào tạo ảnh hưởng đến: %s. Chi tiết: %s", strings.Join(blast.AffectedCohorts, ", "), blast.ImpactSummary)
			_, _ = s.notifRepo.CreateNotification(ctx, &models.UserNotification{
				ID:        "notif_" + uuid.NewString()[:8],
				UserID:    "BROADCAST",
				Title:     notifTitle,
				Content:   notifContent,
				Type:      "ACADEMIC_ALERT",
				IsRead:    false,
				CreatedAt: time.Now(),
			})
		}
	}

	return &response.TLUClawActionResponse{
		Success:    true,
		Message:    "Đã xác nhận và áp dụng thay đổi thành công vào cơ sở dữ liệu.",
		SnapshotID: snapshot.ID,
		BatchID:    snapshot.BatchID,
		Status:     "COMMITTED",
	}, nil
}

func (s *tluclawService) CancelSnapshot(ctx context.Context, adminID string, req *request.TLUClawCancelRequest) (*response.TLUClawActionResponse, error) {
	snapshot, err := s.tluclawRepo.GetSnapshotByNonce(ctx, req.Nonce)
	if err != nil || snapshot == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, fmt.Sprintf("snapshot preview not found for nonce: %s", req.Nonce))
	}

	if snapshot.ID != req.SnapshotID {
		return nil, apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("snapshot id mismatch: expected %s, got %s", snapshot.ID, req.SnapshotID))
	}

	if snapshot.OperatorID != adminID {
		return nil, apperrors.New(apperrors.ErrForbidden, "access denied to snapshot preview")
	}

	if snapshot.Status != "STAGED" {
		return nil, apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("snapshot is in status '%s', cannot cancel", snapshot.Status))
	}

	if err := s.tluclawRepo.UpdateSnapshotStatus(ctx, snapshot.ID, "CANCELLED", sql.NullTime{Valid: false}); err != nil {
		return nil, fmt.Errorf("update snapshot status: %w", err)
	}

	return &response.TLUClawActionResponse{
		Success:    true,
		Message:    "Đã hủy bỏ bản xem trước thay đổi thành công.",
		SnapshotID: snapshot.ID,
		BatchID:    snapshot.BatchID,
		Status:     "CANCELLED",
	}, nil
}

func (s *tluclawService) RollbackBatch(ctx context.Context, adminID, batchID string) (*response.TLUClawActionResponse, error) {
	snapshot, err := s.tluclawRepo.GetSnapshotByBatchID(ctx, batchID)
	if err != nil || snapshot == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, fmt.Sprintf("batch not found: %s", batchID))
	}

	if snapshot.OperatorID != adminID {
		return nil, apperrors.New(apperrors.ErrForbidden, "access denied: batch belongs to another operator")
	}

	if snapshot.Status != "COMMITTED" {
		return nil, apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("batch status is '%s', only COMMITTED batches can be rolled back", snapshot.Status))
	}

	if snapshot.ActionType == "document_supersession" || snapshot.ActionType == "document_ingestion" {
		var state request.TLUClawDocumentSnapshotState
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil {
			_ = s.ragRepo.UpdateDocumentStatus(ctx, state.NewDocID, "SUPERSEDED", "")
			for _, oldID := range state.SupersededDocIDs {
				_ = s.ragRepo.UpdateDocumentStatus(ctx, oldID, "ACTIVE", "")
			}
			if s.c != nil {
				_ = s.c.DelByPattern(ctx, "rag_search:*")
				_ = s.c.DelByPattern(ctx, "rag_candidates:*")
			}
		}
	} else if snapshot.ActionType == "curriculum_update" {
		var state request.TLUClawCurriculumSnapshotState
		_ = json.Unmarshal([]byte(snapshot.NewStateJSON), &state)

		var priorState models.TLUClawCurriculumPriorSnapshotState
		_ = json.Unmarshal([]byte(snapshot.PreviousStateJSON), &priorState)

		priorMap := make(map[string]*models.CurriculumCourse)
		for _, ec := range priorState.ExistingCourses {
			priorMap[strings.ToUpper(strings.TrimSpace(ec.CourseCode))] = ec
		}

		for _, inc := range state.IncomingCourses {
			cleanCode := strings.ToUpper(strings.TrimSpace(inc.CourseCode))
			if _, existed := priorMap[cleanCode]; !existed {
				courseID := fmt.Sprintf("C_%s_%s_%s", state.Cohort, state.MajorCode, cleanCode)
				_ = s.curriculumRepo.DeleteCurriculumCourse(ctx, courseID)
			}
		}

		currID := fmt.Sprintf("CURR_%s_%s", priorState.Cohort, priorState.MajorCode)
		for _, ec := range priorState.ExistingCourses {
			var isElearningInt int64
			if ec.IsElearning {
				isElearningInt = 1
			}
			_, _ = s.curriculumRepo.UpsertCurriculumCourse(ctx, sqlc.UpsertCurriculumCourseParams{
				ID:                ec.ID,
				CurriculumID:      currID,
				Cohort:            ec.Cohort,
				MajorName:         ec.MajorName,
				CourseCode:        ec.CourseCode,
				CourseName:        ec.CourseName,
				Credits:           ec.Credits,
				Weight:            ec.Weight,
				Hours:             ec.Hours,
				KnowledgeBlock:    ec.KnowledgeBlock,
				PrerequisitesText: ec.PrerequisitesText,
				SuggestedSemester: int64(ec.SuggestedSemester),
				IsElearning:       isElearningInt,
			})
		}

		s.curriculumRepo.InvalidateCurriculumCache(ctx, state.Cohort, state.MajorCode)
		if s.c != nil {
			_ = s.c.DelByPattern(ctx, "curriculum:*")
			_ = s.c.DelByPattern(ctx, "course:*")
		}
	} else if snapshot.ActionType == "substitution_rule" {
		var state struct {
			ID              string `json:"id"`
			IsBidirectional int64  `json:"is_bidirectional"`
		}
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil {
			if snapshot.PreviousStateJSON == "" || snapshot.PreviousStateJSON == "{}" {
				_ = s.curriculumRepo.DeleteCourseSubstitution(ctx, state.ID)
				if state.IsBidirectional == 1 {
					_ = s.curriculumRepo.DeleteCourseSubstitution(ctx, fmt.Sprintf("%s_rev", state.ID))
				}
			} else {
				var prevState sqlc.UpsertCourseSubstitutionParams
				if err := json.Unmarshal([]byte(snapshot.PreviousStateJSON), &prevState); err == nil {
					_, _ = s.curriculumRepo.UpsertCourseSubstitution(ctx, prevState)
				}
			}
			s.curriculumRepo.InvalidateCurriculumCache(ctx, "", "")
		}
	} else if snapshot.ActionType == "prerequisite_rule" {
		var state struct {
			ID     string `json:"id"`
			Cohort string `json:"cohort"`
		}
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil {
			_ = s.curriculumRepo.DeleteCoursePrerequisite(ctx, state.ID)
			s.curriculumRepo.InvalidateCurriculumCache(ctx, state.Cohort, "")
		}
	} else if snapshot.ActionType == "advisor_note" {
		var prevState struct {
			UserID       string `json:"user_id"`
			AdvisorNotes string `json:"advisor_notes"`
		}
		if err := json.Unmarshal([]byte(snapshot.PreviousStateJSON), &prevState); err == nil && prevState.UserID != "" {
			if s.memoryRepo != nil {
				profile, _ := s.memoryRepo.GetStudentProfile(ctx, prevState.UserID)
				if profile != nil {
					profile.AdvisorNotes = prevState.AdvisorNotes
					_, _ = s.memoryRepo.UpsertStudentProfile(ctx, profile)
					s.memoryRepo.InvalidateProfileCache(ctx, prevState.UserID)
				}
			}
		}
	} else if snapshot.ActionType == "user_status" {
		var prevState struct {
			UserID string `json:"user_id"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal([]byte(snapshot.PreviousStateJSON), &prevState); err == nil && prevState.UserID != "" {
			if prevState.Status == "ACTIVE" && s.roleRepo != nil && s.userRepo != nil {
				_ = s.userRepo.Restore(ctx, prevState.UserID)
				u, _ := s.userRepo.GetByID(ctx, prevState.UserID)
				if u != nil {
					_ = s.roleRepo.BulkDeleteRolesFromUser(ctx, prevState.UserID)
					for _, r := range u.Roles {
						if r.Name != "BANNED" {
							_ = s.roleRepo.CreateUserRole(ctx, prevState.UserID, r.ID)
						}
					}
				}
				s.userRepo.InvalidateUserCache(ctx, prevState.UserID, "")
			}
		}
	} else if snapshot.ActionType == "user_role" {
		var prevState struct {
			UserID string   `json:"user_id"`
			Roles  []string `json:"roles"`
		}
		if err := json.Unmarshal([]byte(snapshot.PreviousStateJSON), &prevState); err == nil && prevState.UserID != "" {
			if s.roleRepo != nil {
				_ = s.roleRepo.BulkDeleteRolesFromUser(ctx, prevState.UserID)
				for _, rName := range prevState.Roles {
					if r, err := s.roleRepo.GetByName(ctx, rName); err == nil && r != nil {
						_ = s.roleRepo.CreateUserRole(ctx, prevState.UserID, r.ID)
					}
				}
			}
			if s.userRepo != nil {
				_ = s.userRepo.RevokeSessions(ctx, prevState.UserID)
				s.userRepo.InvalidateUserCache(ctx, prevState.UserID, "")
			}
		}
	} else if snapshot.ActionType == "document_status" {
		var prevState struct {
			DocumentID string `json:"document_id"`
			Status     string `json:"status"`
		}
		if err := json.Unmarshal([]byte(snapshot.PreviousStateJSON), &prevState); err == nil && prevState.DocumentID != "" && s.ragRepo != nil {
			_ = s.ragRepo.UpdateDocumentStatus(ctx, prevState.DocumentID, prevState.Status, "")
			if s.c != nil {
				_ = s.c.DelByPattern(ctx, "rag_search:*")
				_ = s.c.DelByPattern(ctx, "rag_candidates:*")
				_ = s.c.Del(ctx, cache.BuildKey("document", "id", prevState.DocumentID))
			}
		}
	} else if snapshot.ActionType == "notification_dispatch" {
		var state struct {
			Notifications []struct {
				ID string `json:"id"`
			} `json:"notifications"`
		}
		if err := json.Unmarshal([]byte(snapshot.NewStateJSON), &state); err == nil && s.notifRepo != nil {
			for _, n := range state.Notifications {
				_ = s.notifRepo.DeleteNotification(ctx, n.ID)
			}
		}
	}

	if err := s.tluclawRepo.UpdateSnapshotStatus(ctx, snapshot.ID, "ROLLED_BACK", sql.NullTime{Time: time.Now(), Valid: true}); err != nil {
		return nil, fmt.Errorf("update snapshot status to rolled back: %w", err)
	}

	return &response.TLUClawActionResponse{
		Success:    true,
		Message:    fmt.Sprintf("Đã hoàn tác toàn bộ thay đổi của phiên %s về trạng thái trước đó.", batchID),
		SnapshotID: snapshot.ID,
		BatchID:    snapshot.BatchID,
		Status:     "ROLLED_BACK",
	}, nil
}

func (s *tluclawService) ListSnapshots(ctx context.Context, status string, limit, offset int32) (*response.TLUClawSnapshotListResponse, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	snps, total, err := s.tluclawRepo.ListSnapshotsPaginated(ctx, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	items := make([]*response.TLUClawSnapshotItemResponse, len(snps))
	for i, snp := range snps {
		items[i] = snp.ToSnapshotItemResponse()
	}
	return &response.TLUClawSnapshotListResponse{
		Snapshots: items,
		Total:     total,
	}, nil
}

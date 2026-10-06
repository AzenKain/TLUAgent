package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tluagent-web/internal/agent/defaults"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/compactor"
	"tluagent-web/pkg/guardrail"
	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/memory"
	"tluagent-web/pkg/tuition"

	"github.com/rs/zerolog/log"
)

var (
	cohortRegex    = regexp.MustCompile(`(?i)\b(?:k|khoá|khóa)\s*([0-9]{2})\b`)
	studentIDRegex = regexp.MustCompile(`(?i)\b([A-Z][0-9]{5})\b`)
)

func extractCohortFromText(text string) string {
	matches := cohortRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		return "K" + matches[1]
	}
	return ""
}

func extractStudentID(text string) string {
	matches := studentIDRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.ToUpper(matches[1])
	}
	return ""
}

func isTuitionQuery(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "học phí") ||
		strings.Contains(lower, "hoc phi") ||
		strings.Contains(lower, "hocphi") ||
		strings.Contains(lower, "nợ học phí") ||
		strings.Contains(lower, "tiền học") ||
		strings.Contains(lower, "đóng học phí") ||
		strings.Contains(lower, "nộp học phí") ||
		strings.Contains(lower, "nợ") ||
		strings.Contains(lower, "hóa đơn") ||
		strings.Contains(lower, "khoản phí")
}

const escalateTag = "[CAN_ESCALATE]"

func processEscalation(reply string) (string, bool) {
	hasTag := strings.Contains(reply, escalateTag)
	clean := strings.TrimSpace(strings.ReplaceAll(reply, escalateTag, ""))
	if hasTag {
		return clean, true
	}
	lower := strings.ToLower(clean)
	unresolved := strings.Contains(lower, "chưa có dữ liệu") ||
		strings.Contains(lower, "chưa thể cung cấp") ||
		strings.Contains(lower, "chưa có thông tin") ||
		strings.Contains(lower, "không tìm thấy thông tin") ||
		strings.Contains(lower, "không có trong tài liệu")
	return clean, unresolved
}

// StreamEvent represents an individual real-time streaming event sent over SSE.
type StreamEvent struct {
	Type             string        `json:"type"`
	Content          string        `json:"content,omitempty"`
	ReasoningContent string        `json:"reasoning_content,omitempty"`
	ToolCall         *ToolCallInfo `json:"tool_call,omitempty"`
	Sources          []string      `json:"sources,omitempty"`
	Timestamp        string        `json:"timestamp,omitempty"`
	SessionID        string        `json:"session_id,omitempty"`
	Message          string        `json:"message,omitempty"`
}

// ToolCallInfo describes a knowledge retrieval or function execution event.
type ToolCallInfo struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Title  string `json:"title,omitempty"`
	Result string `json:"result,omitempty"`
}

// AdvisoryService coordinates academic rule consultation and LLM interactions.
type AdvisoryService interface {
	Consult(ctx context.Context, userID string, dto *request.ChatRequest) (*response.ChatResponse, error)
	ConsultStream(ctx context.Context, userID string, dto *request.ChatRequest, onEvent func(evt StreamEvent) error) (*response.ChatResponse, error)
	SetRAGService(ragService RAGService)
	SetAgentService(agentService AgentService)
	SetMemoryService(memoryService MemoryService)
	SetChatRepository(chatRepo repositories.ChatRepository)
	SetSettingsRepository(settingsRepo repositories.SettingsRepository)
	SetTuitionClient(tuitionClient *tuition.Client)
	SetRamCache(ramCache cache.Cache)
	RecordGuestExchange(ctx context.Context, sessionID, userQuery, botReply string)
	HasGuestSession(ctx context.Context, sessionID string) bool
	EvictGuestSession(ctx context.Context, sessionID string)
}

type advisoryService struct {
	ruleRepo      repositories.RuleRepository
	llmManager    *llm.Manager
	userRepo      repositories.UserRepository
	llmRepo       repositories.LLMRepository
	ragService    RAGService
	agentService  AgentService
	memoryService MemoryService
	chatRepo      repositories.ChatRepository
	settingsRepo  repositories.SettingsRepository
	tuitionClient *tuition.Client
	ramCache      cache.Cache
}

// NewAdvisoryService creates an advisory service instance.
func NewAdvisoryService(ruleRepo repositories.RuleRepository, llmManager *llm.Manager, userRepo repositories.UserRepository, llmRepo repositories.LLMRepository) AdvisoryService {
	return &advisoryService{
		ruleRepo:      ruleRepo,
		llmManager:    llmManager,
		userRepo:      userRepo,
		llmRepo:       llmRepo,
		tuitionClient: tuition.NewClient(10 * time.Second),
	}
}

func (s *advisoryService) SetRAGService(ragService RAGService) {
	s.ragService = ragService
}

func (s *advisoryService) SetAgentService(agentService AgentService) {
	s.agentService = agentService
}

func (s *advisoryService) SetMemoryService(memoryService MemoryService) {
	s.memoryService = memoryService
}

func (s *advisoryService) SetChatRepository(chatRepo repositories.ChatRepository) {
	s.chatRepo = chatRepo
}

func (s *advisoryService) SetSettingsRepository(settingsRepo repositories.SettingsRepository) {
	s.settingsRepo = settingsRepo
}

func (s *advisoryService) SetTuitionClient(tuitionClient *tuition.Client) {
	s.tuitionClient = tuitionClient
}

func (s *advisoryService) SetRamCache(ramCache cache.Cache) {
	s.ramCache = ramCache
}

func (s *advisoryService) RecordGuestExchange(ctx context.Context, sessionID, userQuery, botReply string) {
	if s.ramCache == nil || sessionID == "" {
		return
	}
	var history []compactor.Message
	_ = s.ramCache.Get(ctx, "guest:history:"+sessionID, &history)
	history = append(history,
		compactor.Message{Role: compactor.RoleUser, Content: userQuery},
		compactor.Message{Role: compactor.RoleAssistant, Content: botReply},
	)
	if len(history) > 30 {
		history = history[len(history)-30:]
	}
	_ = s.ramCache.Set(ctx, "guest:history:"+sessionID, history, 2*time.Hour)
}

func (s *advisoryService) HasGuestSession(ctx context.Context, sessionID string) bool {
	if s.ramCache == nil || sessionID == "" {
		return false
	}
	var history []compactor.Message
	err := s.ramCache.Get(ctx, "guest:history:"+sessionID, &history)
	return err == nil && len(history) > 0
}

func (s *advisoryService) EvictGuestSession(ctx context.Context, sessionID string) {
	if s.ramCache == nil || sessionID == "" {
		return
	}
	_ = s.ramCache.Del(ctx, "guest:history:"+sessionID)
}

func (s *advisoryService) resolveModelAndProvider(ctx context.Context, modelID string) (llm.Provider, string, *sqlc.GetLLMModelWithProviderRow) {
	if modelID == "" || s.llmRepo == nil {
		return nil, "", nil
	}
	row, err := s.llmRepo.GetModelWithProvider(ctx, modelID)
	if err != nil || row.ModelKey == "" {
		return nil, "", nil
	}
	if s.llmManager != nil && row.ProviderName != "" {
		if p, err := s.llmManager.Get(row.ProviderName); err == nil && p != nil {
			return p, row.ModelKey, &row
		}
	}
	return nil, row.ModelKey, &row
}

func (s *advisoryService) buildChatRequest(dto *request.ChatRequest, modelKey string, row *sqlc.GetLLMModelWithProviderRow, messages []llm.Message) *llm.ChatRequest {
	req := &llm.ChatRequest{
		Model:    modelKey,
		Messages: messages,
	}
	if row != nil {
		enabled := row.ThinkingEnabled == 1
		req.ThinkingEnabled = &enabled
		if row.ThinkingBudget > 0 {
			b := int(row.ThinkingBudget)
			req.ThinkingBudget = &b
		}
		if row.EffortLevel != "" {
			req.ReasoningEffort = row.EffortLevel
		}
	}
	if dto.ThinkingEnabled != nil {
		req.ThinkingEnabled = dto.ThinkingEnabled
	}
	if dto.ThinkingBudget != nil {
		req.ThinkingBudget = dto.ThinkingBudget
	}
	if dto.EffortLevel != "" {
		req.ReasoningEffort = dto.EffortLevel
	}
	return req
}

func (s *advisoryService) fallbackReply(ctx context.Context, query string, sources []string, timestamp string) *response.ChatResponse {
	if s.ragService != nil {
		if ragResults, err := s.ragService.Search(ctx, models.RAGSearchParams{
			Query:         query,
			StudentCohort: "",
			TopK:          1,
		}, nil); err == nil && len(ragResults) > 0 {
			top := ragResults[0]
			if top.DocTitle != "" {
				sources = append(sources, top.DocTitle)
			}
			return &response.ChatResponse{
				Query:     query,
				Reply:     fmt.Sprintf("**%s**\n\n%s", top.DocTitle, top.Content),
				Sources:   sources,
				Timestamp: timestamp,
			}
		}
	}
	if s.ruleRepo != nil {
		if rule, _ := s.ruleRepo.FindByQuery(ctx, query); rule != nil {
			return rule.ToResponse(query, timestamp)
		}
	}
	return &response.ChatResponse{
		Query:     query,
		Reply:     fmt.Sprintf("Your inquiry regarding \"%s\" has been received. Please consult the official academic portal or Academic Affairs Office for further assistance.", query),
		Sources:   sources,
		Timestamp: timestamp,
	}
}

func (s *advisoryService) buildMessages(ctx context.Context, userID string, dto *request.ChatRequest, tuitionInfo *tuition.TuitionInfo) ([]llm.Message, []string) {
	var sources []string

	var systemPrompt strings.Builder
	var studentCohort = ""

	var profile *memory.StudentAcademicProfile
	if userID != "" && userID != "0" && s.userRepo != nil {
		if user, err := s.userRepo.GetByID(ctx, userID); err == nil && user != nil {
			profile = &memory.StudentAcademicProfile{
				UserID:    user.ID,
				StudentID: user.StudentCode,
				FullName:  user.FullName,
			}
			if s.memoryService != nil {
				if dbProf, err := s.memoryService.GetStudentProfile(ctx, userID); err == nil && dbProf != nil {
					profile.Major = dbProf.Major
					if dbProf.Cohort != "" {
						studentCohort = dbProf.Cohort
					}
					profile.AcademicStanding = memory.AcademicStanding(dbProf.AcademicStanding)
					profile.CompletedCredits = dbProf.CompletedCredits
					profile.CumulativeGPA = dbProf.CumulativeGPA
				}
				if mems, err := s.memoryService.ListUserMemories(ctx, userID); err == nil && len(mems) > 0 {
					profile.Preferences = make(map[string]string)
					for _, m := range mems {
						profile.Preferences[m.MemoryKey] = m.MemoryValue
					}
				}
			}
		}
	}

	var dialogue []compactor.Message
	if len(dto.History) > 0 {
		for _, h := range dto.History {
			role := compactor.RoleUser
			if h.Role == "assistant" {
				role = compactor.RoleAssistant
			}
			dialogue = append(dialogue, compactor.Message{
				Role:    role,
				Content: h.Content,
			})
		}
		if dto.SessionID != "" && s.ramCache != nil && (userID == "" || userID == "0") {
			_ = s.ramCache.Set(ctx, "guest:history:"+dto.SessionID, dialogue, 2*time.Hour)
		}
	} else if dto.SessionID != "" {
		if s.chatRepo != nil && userID != "" && userID != "0" {
			if pastMsgs, err := s.chatRepo.ListMessagesByConversationID(ctx, dto.SessionID); err == nil && len(pastMsgs) > 0 {
				for _, m := range pastMsgs {
					role := compactor.RoleUser
					if m.Sender == "assistant" {
						role = compactor.RoleAssistant
					}
					dialogue = append(dialogue, compactor.Message{
						Role:    role,
						Content: m.Content,
					})
				}
			}
		}
		if len(dialogue) == 0 && s.ramCache != nil {
			var guestMsgs []compactor.Message
			if err := s.ramCache.Get(ctx, "guest:history:"+dto.SessionID, &guestMsgs); err == nil && len(guestMsgs) > 0 {
				dialogue = append(dialogue, guestMsgs...)
				_ = s.ramCache.Set(ctx, "guest:history:"+dto.SessionID, guestMsgs, 2*time.Hour)
			}
		}
	}

	explicitCohort := extractCohortFromText(dto.Query)
	if explicitCohort == "" && len(dialogue) > 0 {
		for i := len(dialogue) - 1; i >= 0; i-- {
			if dialogue[i].Role == compactor.RoleUser {
				if c := extractCohortFromText(dialogue[i].Content); c != "" {
					explicitCohort = c
					break
				}
			}
		}
	}
	if explicitCohort != "" {
		studentCohort = explicitCohort
	}

	explicitID := extractStudentID(dto.Query)
	if explicitID == "" && len(dialogue) > 0 {
		for i := len(dialogue) - 1; i >= 0; i-- {
			if dialogue[i].Role == compactor.RoleUser {
				if id := extractStudentID(dialogue[i].Content); id != "" {
					explicitID = id
					break
				}
			}
		}
	}

	if profile == nil && (studentCohort != "" || explicitID != "") {
		profile = &memory.StudentAcademicProfile{}
	}
	if profile != nil {
		if explicitID != "" {
			profile.StudentID = explicitID
		}
		if studentCohort != "" {
			cleanCohort := strings.TrimPrefix(strings.ToUpper(studentCohort), "K")
			if n, err := strconv.Atoi(cleanCohort); err == nil {
				profile.CohortYear = n
			}
		}
	}

	searchQuery := dto.Query
	if len(dialogue) > 0 {
		var prevContext []string
		for i := len(dialogue) - 1; i >= 0; i-- {
			if dialogue[i].Role == compactor.RoleUser {
				prevContext = append(prevContext, dialogue[i].Content)
				if len(prevContext) >= 1 {
					break
				}
			}
		}
		if len(prevContext) > 0 && len(strings.Fields(dto.Query)) <= 5 {
			searchQuery = dto.Query + " " + strings.Join(prevContext, " ")
		}
	}

	var ragItems []*models.RAGSearchResultItem
	if s.ragService != nil {
		if ragResults, err := s.ragService.Search(ctx, models.RAGSearchParams{
			Query:         searchQuery,
			StudentCohort: studentCohort,
			TopK:          8,
		}, nil); err == nil && len(ragResults) > 0 {
			ragItems = ragResults
		}
	}

	if s.agentService != nil {
		prompt, promptSources, err := s.agentService.BuildSystemPrompt(ctx, profile, ragItems)
		if err == nil && prompt != "" {
			systemPrompt.WriteString(prompt)
			if len(promptSources) > 0 {
				sources = append(sources, promptSources...)
			}
		}
	} else {
		if soul, err := defaults.ReadDefaultFile("soul.md"); err == nil {
			systemPrompt.WriteString(soul)
			systemPrompt.WriteString("\n\n---\n\n")
		}
		if rule, err := defaults.ReadDefaultFile("rule.md"); err == nil {
			systemPrompt.WriteString(rule)
		}
		if profile != nil {
			systemPrompt.WriteString("\n\n---\n\n")
			systemPrompt.WriteString(memory.FormatStudentPromptContext(profile))
		}
		if len(ragItems) > 0 {
			systemPrompt.WriteString("\n\n---\n\n# RETRIEVED REGULATORY DOCUMENTS:\n\n")
			for _, item := range ragItems {
				fmt.Fprintf(&systemPrompt, "- Document: %s\nContent: %s\n\n", item.DocTitle, item.Content)
				if item.DocTitle != "" {
					sources = append(sources, item.DocTitle)
				}
			}
		}
		if s.ruleRepo != nil {
			if rule, err := s.ruleRepo.FindByQuery(ctx, dto.Query); err == nil && rule != nil {
				fmt.Fprintf(&systemPrompt, "\n\n[SYSTEM ADVISORY RULE]\n- Topic: %s\n- Content: %s\n- Sources: %s\n", rule.Title, rule.Content, strings.Join(rule.Sources, ", "))
				if len(rule.Sources) > 0 {
					sources = append(sources, rule.Sources...)
				}
			}
		}
		systemPrompt.WriteString("\n---\n\n# MANDATORY CITATION & VERIFICATION DIRECTIVE:\n")
		systemPrompt.WriteString("1. CITATION REQUIREMENT: At the very end OF EVERY response, you MUST append a 'Nguồn trích dẫn / Căn cứ pháp lý:' section clearly citing the document titles and specific articles used so the student can verify.\n")
		systemPrompt.WriteString("2. STRICT ZERO-HALLUCINATION POLICY: Never invent or guess rules, policies, dates, or tuition fees not present in the retrieved context. If an inquiry cannot be answered by the official documents provided, clearly declare that the information is not found in official TLU documents and guide the student to contact the Academic Affairs Department (Phòng Đào tạo / Nhà T).\n")
		systemPrompt.WriteString("3. ADVISOR ESCALATION DIRECTIVE: If and ONLY if you DO NOT have sufficient official documents in the provided context to answer the student's question, OR if the question involves special personal requests requiring advisor approval/intervention (e.g. grade appeal, deferred exam, special study plan approval), you MUST append the exact token `[CAN_ESCALATE]` at the very end of your response. If you have enough official documents to answer clearly, DO NOT include `[CAN_ESCALATE]`.\n")
	}

	if s.memoryService != nil && userID != "" && userID != "0" {
		if memPrompt, err := s.memoryService.BuildLongTermMemoryPromptContext(ctx, userID); err == nil && memPrompt != "" {
			systemPrompt.WriteString("\n\n---\n\n")
			systemPrompt.WriteString(memPrompt)
		}
	}

	if tuitionInfo != nil && tuitionInfo.Found {
		systemPrompt.WriteString("\n\n---\n\n")
		systemPrompt.WriteString(tuition.FormatPromptContext(tuitionInfo))
		sources = append(sources, "Cổng thanh toán học phí Đại học Thăng Long (hocphi.thanglong.edu.vn)")
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

	if contextLimit == 250000 && dto.ModelID != "" && s.llmRepo != nil {
		if m, err := s.llmRepo.GetModelByID(ctx, dto.ModelID); err == nil && m.ContextLength > 1000 {
			contextLimit = int(m.ContextLength)
		}
	} else if contextLimit == 250000 && s.llmRepo != nil {
		if defModel, err := s.llmRepo.GetDefaultModelByType(ctx, "chat"); err == nil && defModel.ContextLength > 1000 {
			contextLimit = int(defModel.ContextLength)
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
	compacted, _, _ := compactor.CompactHistory(ctx, dialogue, cfg)

	messages := make([]llm.Message, 0, len(compacted)+2)
	messages = append(messages, llm.Message{
		Role:    llm.RoleSystem,
		Content: systemPrompt.String(),
	})

	for _, m := range compacted {
		role := llm.RoleUser
		switch m.Role {
		case compactor.RoleAssistant:
			role = llm.RoleAssistant
		case compactor.RoleSystem:
			role = llm.RoleSystem
		}
		messages = append(messages, llm.Message{
			Role:    role,
			Content: m.Content,
		})
	}

	if len(dto.Images) > 0 {
		parts := make([]llm.ContentPart, 0, len(dto.Images)+1)
		parts = append(parts, llm.ContentPart{
			Type: llm.ContentPartText,
			Text: dto.Query,
		})
		for _, img := range dto.Images {
			parts = append(parts, llm.ContentPart{
				Type: llm.ContentPartImage,
				Image: &llm.ImageSource{
					URL: img,
				},
			})
		}
		messages = append(messages, llm.Message{
			Role:  llm.RoleUser,
			Parts: parts,
		})
	} else {
		messages = append(messages, llm.Message{
			Role:    llm.RoleUser,
			Content: dto.Query,
		})
	}

	return messages, sources
}

func (s *advisoryService) Consult(ctx context.Context, userID string, dto *request.ChatRequest) (*response.ChatResponse, error) {
	timestamp := time.Now().Format("15:04:05")

	chk := guardrail.CheckQuery(dto.Query)
	if chk.Decision != guardrail.DecisionAllow {
		return &response.ChatResponse{
			Query:       dto.Query,
			Reply:       chk.Response,
			Sources:     nil,
			Timestamp:   timestamp,
			CanEscalate: false,
		}, nil
	}

	var tuitionInfo *tuition.TuitionInfo
	targetStudentID := extractStudentID(dto.Query)
	if targetStudentID == "" && s.userRepo != nil && userID != "" && userID != "0" {
		if u, err := s.userRepo.GetByID(ctx, userID); err == nil && u != nil {
			targetStudentID = extractStudentID(u.StudentCode)
		}
	}
	if targetStudentID != "" && isTuitionQuery(dto.Query) && s.tuitionClient != nil {
		if info, err := s.tuitionClient.LookupTuition(ctx, targetStudentID); err == nil && info != nil && info.Found {
			tuitionInfo = info
		}
	}

	messages, sources := s.buildMessages(ctx, userID, dto, tuitionInfo)

	if s.llmManager != nil && len(s.llmManager.List()) > 0 {
		prov, modelKey, modelRow := s.resolveModelAndProvider(ctx, dto.ModelID)
		llmReq := s.buildChatRequest(dto, modelKey, modelRow, messages)

		if prov != nil {
			resp, err := prov.Chat(ctx, llmReq)
			if err == nil && resp != nil && strings.TrimSpace(resp.Message.Content) != "" {
				cleanReply, canEsc := processEscalation(resp.Message.Content)
				reply := guardrail.SanitizeOutput(cleanReply)
				res := &response.ChatResponse{
					Query:       dto.Query,
					Reply:       reply,
					Sources:     sources,
					Timestamp:   timestamp,
					CanEscalate: canEsc,
				}
				if s.memoryService != nil && userID != "" && userID != "0" {
					go func() {
						_ = s.memoryService.ExtractAndSaveMemories(context.Background(), userID, dto.Query, res.Reply)
					}()
				}
				return res, nil
			}
			log.Warn().Err(err).Str("modelKey", modelKey).Msg("direct model chat failed, trying fallback chain")
		}

		resp, err := s.llmManager.Chat(ctx, llmReq)
		if err != nil {
			log.Error().Err(err).Str("modelKey", modelKey).Msg("advisory Consult: Chat failed")
		} else if resp != nil && strings.TrimSpace(resp.Message.Content) != "" {
			cleanReply, canEsc := processEscalation(resp.Message.Content)
			reply := guardrail.SanitizeOutput(cleanReply)
			res := &response.ChatResponse{
				Query:       dto.Query,
				Reply:       reply,
				Sources:     sources,
				Timestamp:   timestamp,
				CanEscalate: canEsc,
			}
			if s.memoryService != nil && userID != "" && userID != "0" {
				go func() {
					_ = s.memoryService.ExtractAndSaveMemories(context.Background(), userID, dto.Query, res.Reply)
				}()
			}
			return res, nil
		}
	}

	fb := s.fallbackReply(ctx, dto.Query, sources, timestamp)
	fb.Reply = guardrail.SanitizeOutput(fb.Reply)
	fb.CanEscalate = true
	return fb, nil
}

func (s *advisoryService) ConsultStream(ctx context.Context, userID string, dto *request.ChatRequest, onEvent func(evt StreamEvent) error) (*response.ChatResponse, error) {
	timestamp := time.Now().Format("15:04:05")

	chk := guardrail.CheckQuery(dto.Query)
	if chk.Decision != guardrail.DecisionAllow {
		if onEvent != nil {
			_ = onEvent(StreamEvent{
				Type:    "chunk",
				Content: chk.Response,
			})
		}
		return &response.ChatResponse{
			Query:       dto.Query,
			Reply:       chk.Response,
			Sources:     nil,
			Timestamp:   timestamp,
			CanEscalate: false,
		}, nil
	}

	var tuitionInfo *tuition.TuitionInfo
	targetStudentID := extractStudentID(dto.Query)
	if targetStudentID == "" && s.userRepo != nil && userID != "" && userID != "0" {
		if u, err := s.userRepo.GetByID(ctx, userID); err == nil && u != nil {
			targetStudentID = extractStudentID(u.StudentCode)
		}
	}

	if targetStudentID != "" && isTuitionQuery(dto.Query) && s.tuitionClient != nil {
		if onEvent != nil {
			_ = onEvent(StreamEvent{
				Type: "tool_call",
				ToolCall: &ToolCallInfo{
					ID:     "call_tuition_" + targetStudentID,
					Name:   "lookup_tuition_fee",
					Status: "running",
					Result: targetStudentID,
				},
			})
		}
		if info, err := s.tuitionClient.LookupTuition(ctx, targetStudentID); err == nil && info != nil && info.Found {
			tuitionInfo = info
			if onEvent != nil {
				_ = onEvent(StreamEvent{
					Type: "tool_call",
					ToolCall: &ToolCallInfo{
						ID:     "call_tuition_" + targetStudentID,
						Name:   "lookup_tuition_fee",
						Status: "completed",
						Result: fmt.Sprintf("Mã SV: %s | %s | Nợ: %s VNĐ", info.StudentID, info.StudentName, info.TotalDebt),
					},
				})
			}
		} else if onEvent != nil {
			_ = onEvent(StreamEvent{
				Type: "tool_call",
				ToolCall: &ToolCallInfo{
					ID:     "call_tuition_" + targetStudentID,
					Name:   "lookup_tuition_fee",
					Status: "completed",
					Result: "Không tìm thấy dữ liệu học phí cho mã: " + targetStudentID,
				},
			})
		}
	}

	if onEvent != nil {
		_ = onEvent(StreamEvent{
			Type: "tool_call",
			ToolCall: &ToolCallInfo{
				ID:     "call_rag_search",
				Name:   "rag_knowledge_search",
				Status: "running",
			},
		})
	}

	messages, sources := s.buildMessages(ctx, userID, dto, tuitionInfo)

	if onEvent != nil {
		_ = onEvent(StreamEvent{
			Type: "tool_call",
			ToolCall: &ToolCallInfo{
				ID:     "call_rag_search",
				Name:   "rag_knowledge_search",
				Status: "completed",
				Result: fmt.Sprintf("%d", len(sources)),
			},
			Sources: sources,
		})
	}

	if s.llmManager != nil && len(s.llmManager.List()) > 0 {
		prov, modelKey, modelRow := s.resolveModelAndProvider(ctx, dto.ModelID)
		llmReq := s.buildChatRequest(dto, modelKey, modelRow, messages)

		var stream llm.StreamReader
		var err error

		if prov != nil {
			stream, err = prov.ChatStream(ctx, llmReq)
			if err != nil {
				log.Warn().Err(err).Str("modelKey", modelKey).Msg("direct model stream failed, trying fallback chain")
				stream = nil
			}
		}

		if stream == nil {
			stream, err = s.llmManager.ChatStream(ctx, llmReq)
			if err != nil {
				log.Error().Err(err).Str("modelKey", modelKey).Msg("advisory ConsultStream: ChatStream failed")
			}
		}

		if stream != nil {
			defer stream.Close()
			var full strings.Builder
			for {
				chunk, recvErr := stream.Recv()
				if errors.Is(recvErr, io.EOF) || errors.Is(recvErr, llm.ErrStreamClosed) {
					break
				}
				if recvErr != nil {
					log.Error().Err(recvErr).Msg("advisory ConsultStream: stream.Recv failed")
					break
				}
				if chunk == nil {
					continue
				}

				if chunk.Delta.ReasoningContent != "" {
					if onEvent != nil {
						if err := onEvent(StreamEvent{
							Type:             "reasoning",
							ReasoningContent: chunk.Delta.ReasoningContent,
							Content:          chunk.Delta.ReasoningContent,
						}); err != nil {
							return nil, err
						}
					}
				}

				if chunk.Delta.Content != "" {
					full.WriteString(chunk.Delta.Content)
					if onEvent != nil {
						contentToSend := strings.ReplaceAll(chunk.Delta.Content, escalateTag, "")
						if contentToSend != "" {
							if err := onEvent(StreamEvent{
								Type:    "chunk",
								Content: contentToSend,
							}); err != nil {
								return nil, err
							}
						}
					}
				}

				if len(chunk.Delta.ToolCalls) > 0 {
					for _, tc := range chunk.Delta.ToolCalls {
						if onEvent != nil {
							_ = onEvent(StreamEvent{
								Type: "tool_call",
								ToolCall: &ToolCallInfo{
									ID:     tc.ID,
									Name:   tc.Function.Name,
									Status: "running",
									Result: tc.Function.Arguments,
								},
							})
						}
					}
				}
			}

			cleanReply, canEsc := processEscalation(full.String())
			reply := guardrail.SanitizeOutput(strings.TrimSpace(cleanReply))
			if reply != "" {
				if onEvent != nil && canEsc {
					_ = onEvent(StreamEvent{
						Type:    "can_escalate",
						Content: "true",
					})
				}
				res := &response.ChatResponse{
					Query:       dto.Query,
					Reply:       reply,
					Sources:     sources,
					Timestamp:   timestamp,
					CanEscalate: canEsc,
				}
				if s.memoryService != nil && userID != "" && userID != "0" {
					go func() {
						_ = s.memoryService.ExtractAndSaveMemories(context.Background(), userID, dto.Query, reply)
					}()
				}
				return res, nil
			}
		}
	}

	fallback := s.fallbackReply(ctx, dto.Query, sources, timestamp)
	fallback.Reply = guardrail.SanitizeOutput(fallback.Reply)
	fallback.CanEscalate = true
	if onEvent != nil {
		_ = onEvent(StreamEvent{
			Type:    "chunk",
			Content: fallback.Reply,
		})
		_ = onEvent(StreamEvent{
			Type:    "can_escalate",
			Content: "true",
		})
	}
	return fallback, nil
}

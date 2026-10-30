package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tluagent-web/internal/agent"
	"tluagent-web/internal/agent/defaults"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/compactor"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/guardrail"
	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/memory"
	"tluagent-web/pkg/tuition"

	"github.com/rs/zerolog/log"
)

var (
	cohortRegex     = regexp.MustCompile(`(?i)\b(?:k|khoá|khóa)\s*([0-9]{2})\b`)
	studentIDRegex  = regexp.MustCompile(`(?i)\b([A-Z][0-9]{5})\b`)
	courseCodeRegex = regexp.MustCompile(`(?i)\b([a-z]{2,4}[0-9]{3,4})\b`)
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

func extractCourseCodes(text string) []string {
	matches := courseCodeRegex.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var res []string
	for _, m := range matches {
		upper := strings.ToUpper(m)
		if !seen[upper] {
			seen[upper] = true
			res = append(res, upper)
		}
	}
	return res
}

func deduplicateSources(sources []string) []string {
	if len(sources) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(sources))
	out := make([]string, 0, len(sources))
	for _, s := range sources {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			out = append(out, trimmed)
		}
	}
	return out
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

type streamFilterBuffer struct {
	buf       strings.Builder
	keepBytes int
	isBlocked bool
	onFlush   func(string) error
}

func newStreamFilterBuffer(keepBytes int, onFlush func(string) error) *streamFilterBuffer {
	return &streamFilterBuffer{
		keepBytes: keepBytes,
		onFlush:   onFlush,
	}
}

func (b *streamFilterBuffer) Reset() {
	b.buf.Reset()
	b.isBlocked = false
}

type advisoryToolCallAccumulator struct {
	id             string
	callType       string
	name           string
	arguments      strings.Builder
	runningEmitted bool
}

func (b *streamFilterBuffer) Feed(chunk string) error {
	if b.isBlocked {
		return nil
	}
	b.buf.WriteString(chunk)
	str := b.buf.String()

	cleaned, blocked := guardrail.SanitizeStreamChunk(strings.ReplaceAll(str, escalateTag, ""))
	if blocked {
		b.isBlocked = true
		b.buf.Reset()
		if b.onFlush != nil {
			return b.onFlush(cleaned)
		}
		return nil
	}
	if cleaned != str {
		b.buf.Reset()
		b.buf.WriteString(cleaned)
		str = cleaned
	}

	if len(str) <= b.keepBytes {
		return nil
	}

	flushLen := len(str) - b.keepBytes

	lastOpenTag := strings.LastIndex(str[:flushLen+b.keepBytes], "<")
	lastCloseTag := strings.LastIndex(str[:flushLen+b.keepBytes], ">")
	if lastOpenTag >= 0 && lastOpenTag > lastCloseTag && (len(str)-lastOpenTag) < 120 {
		if lastOpenTag < flushLen {
			flushLen = lastOpenTag
		}
	}

	lastOpenSq := strings.LastIndex(str[:flushLen+b.keepBytes], "[")
	lastCloseSq := strings.LastIndex(str[:flushLen+b.keepBytes], "]")
	if lastOpenSq >= 0 && lastOpenSq > lastCloseSq && (len(str)-lastOpenSq) < 60 {
		if lastOpenSq < flushLen {
			flushLen = lastOpenSq
		}
	}

	for flushLen > 0 && (str[flushLen]&0xC0) == 0x80 {
		flushLen--
	}

	if flushLen <= 0 {
		return nil
	}

	toFlush := str[:flushLen]
	remaining := str[flushLen:]
	b.buf.Reset()
	b.buf.WriteString(remaining)

	if toFlush != "" && b.onFlush != nil {
		return b.onFlush(toFlush)
	}
	return nil
}

func (b *streamFilterBuffer) Flush() error {
	if b.isBlocked {
		return nil
	}
	str := b.buf.String()
	b.buf.Reset()
	if str == "" {
		return nil
	}
	cleaned := strings.ReplaceAll(str, escalateTag, "")
	safeChunk, blocked := guardrail.SanitizeStreamChunk(cleaned)
	if blocked {
		b.isBlocked = true
		if b.onFlush != nil {
			return b.onFlush(safeChunk)
		}
		return nil
	}
	if safeChunk != "" && b.onFlush != nil {
		return b.onFlush(safeChunk)
	}
	return nil
}

// StreamEvent represents an individual real-time streaming event sent over SSE.
type StreamEvent struct {
	Type             string                                      `json:"type"`
	Content          string                                      `json:"content,omitempty"`
	ReasoningContent string                                      `json:"reasoning_content,omitempty"`
	ToolCall         *ToolCallInfo                               `json:"tool_call,omitempty"`
	Sources          []string                                    `json:"sources,omitempty"`
	Timestamp        string                                      `json:"timestamp,omitempty"`
	SessionID        string                                      `json:"session_id,omitempty"`
	Message          string                                      `json:"message,omitempty"`
	Roadmap          *response.RoadmapOptimizationResultResponse `json:"roadmap,omitempty"`
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
	RecordGuestExchange(ctx context.Context, sessionID, userQuery, botReply string)
	HasGuestSession(ctx context.Context, sessionID string) bool
	EvictGuestSession(ctx context.Context, sessionID string)
}

// AdvisoryParams encapsulates dependencies for AdvisoryService.
type AdvisoryParams struct {
	RuleRepo         repositories.RuleRepository
	LLMManager       *llm.Manager
	UserRepo         repositories.UserRepository
	LLMRepo          repositories.LLMRepository
	RAGService       RAGService
	AgentService     AgentService
	MemoryService    MemoryService
	ChatRepo         repositories.ChatRepository
	SettingsRepo     repositories.SettingsRepository
	TuitionClient    *tuition.Client
	RamCache         cache.Cache
	CurriculumSvc    CurriculumService
	TranscriptSvc    TranscriptService
	RoadmapOptimizer RoadmapOptimizer
}

type advisoryService struct {
	ruleRepo         repositories.RuleRepository
	llmManager       *llm.Manager
	userRepo         repositories.UserRepository
	llmRepo          repositories.LLMRepository
	ragService       RAGService
	agentService     AgentService
	memoryService    MemoryService
	chatRepo         repositories.ChatRepository
	settingsRepo     repositories.SettingsRepository
	tuitionClient    *tuition.Client
	ramCache         cache.Cache
	curriculumSvc    CurriculumService
	transcriptSvc    TranscriptService
	roadmapOptimizer RoadmapOptimizer
}

// NewAdvisoryService creates an advisory service instance.
func NewAdvisoryService(params AdvisoryParams) AdvisoryService {
	tuitionClient := params.TuitionClient
	if tuitionClient == nil {
		tuitionClient = tuition.NewClient(10 * time.Second)
	}
	return &advisoryService{
		ruleRepo:         params.RuleRepo,
		llmManager:       params.LLMManager,
		userRepo:         params.UserRepo,
		llmRepo:          params.LLMRepo,
		ragService:       params.RAGService,
		agentService:     params.AgentService,
		memoryService:    params.MemoryService,
		chatRepo:         params.ChatRepo,
		settingsRepo:     params.SettingsRepo,
		tuitionClient:    tuitionClient,
		ramCache:         params.RamCache,
		curriculumSvc:    params.CurriculumSvc,
		transcriptSvc:    params.TranscriptSvc,
		roadmapOptimizer: params.RoadmapOptimizer,
	}
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
	if s.llmManager != nil {
		if row.ProviderID != "" {
			if p, err := s.llmManager.Get(row.ProviderID); err == nil && p != nil {
				return p, row.ModelKey, &row
			}
		}
		if row.ProviderName != "" {
			if p, err := s.llmManager.Get(row.ProviderName); err == nil && p != nil {
				return p, row.ModelKey, &row
			}
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
				Sources:   deduplicateSources(sources),
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
		Reply:     fmt.Sprintf("Rất tiếc, hệ thống chưa thể trả lời câu hỏi \"%s\" lúc này. Vui lòng thử lại sau, tra cứu trên Cổng thông tin sinh viên hoặc liên hệ Phòng Đào tạo để được hỗ trợ.", query),
		Sources:   deduplicateSources(sources),
		Timestamp: timestamp,
	}
}

func (s *advisoryService) getToolDefinitions(ctx context.Context) []llm.ToolDefinition {
	if s.agentService != nil {
		if tools, err := s.agentService.GetActiveToolDefinitions(ctx); err == nil && len(tools) > 0 {
			var advisorTools []llm.ToolDefinition
			for _, t := range tools {
				if !strings.HasPrefix(t.Function.Name, "tluclaw_") {
					advisorTools = append(advisorTools, t)
				}
			}
			return advisorTools
		}
	}
	return agent.GetDefaultAdvisorToolDefinitions()
}

func (s *advisoryService) executeAdvisorTool(ctx context.Context, userID string, profile *memory.StudentAcademicProfile, tc llm.ToolCall) (*agent.ToolResult, error) {
	name := tc.Function.Name
	argsStr := tc.Function.Arguments

	switch name {
	case "search_regulations", "rag_knowledge_search":
		var args request.AdvisorSearchRegulationsArgs
		_ = json.Unmarshal([]byte(argsStr), &args)
		query := strings.TrimSpace(args.Query)
		cohort := strings.TrimSpace(args.Cohort)
		domain := strings.TrimSpace(args.Domain)
		if query == "" {
			query = argsStr
		}
		if cohort == "" && profile != nil && profile.CohortYear > 0 {
			cohort = fmt.Sprintf("K%d", profile.CohortYear)
		}

		if s.ragService == nil {
			return &agent.ToolResult{
				Content: "Dịch vụ tra cứu quy chế hiện chưa sẵn sàng.",
				Title:   "Tra cứu quy chế",
			}, nil
		}

		ragResults, err := s.ragService.Search(ctx, models.RAGSearchParams{
			Query:         query,
			StudentCohort: cohort,
			Domain:        domain,
			TopK:          5,
		}, nil)
		if (err != nil || len(ragResults) == 0) && (domain != "" || cohort != "") {
			ragResults, err = s.ragService.Search(ctx, models.RAGSearchParams{
				Query: query,
				TopK:  5,
			}, nil)
		}
		code := courseCodeRegex.FindString(query)
		if code == "" {
			code = courseCodeRegex.FindString(argsStr)
		}
		if code != "" {
			upperCode := strings.ToUpper(code)
			hasCode := false
			for _, r := range ragResults {
				if strings.Contains(strings.ToUpper(r.Content), upperCode) {
					hasCode = true
					break
				}
			}
			if !hasCode {
				codeResults, _ := s.ragService.Search(ctx, models.RAGSearchParams{
					Query: upperCode,
					TopK:  5,
				}, nil)
				if len(codeResults) > 0 {
					ragResults = append(codeResults, ragResults...)
					if len(ragResults) > 5 {
						ragResults = ragResults[:5]
					}
				}
			}
		}
		if err != nil || len(ragResults) == 0 {
			return &agent.ToolResult{
				Content: fmt.Sprintf("Không tìm thấy văn bản quy chế nào phù hợp với từ khóa: '%s'.", query),
				Title:   "Tra cứu cơ sở quy chế",
			}, nil
		}

		var b strings.Builder
		var sources []string
		seenTitles := make(map[string]bool)
		fmt.Fprintf(&b, "# KẾT QUẢ TRA CỨU QUY CHẾ VĂN BẢN CHÍNH THỨC (%d văn bản liên quan):\n\n", len(ragResults))
		for _, item := range ragResults {
			fmt.Fprintf(&b, "--- VĂN BẢN: %s ---\n%s\n\n", item.DocTitle, item.Content)
			if item.DocTitle != "" && !seenTitles[item.DocTitle] {
				seenTitles[item.DocTitle] = true
				sources = append(sources, item.DocTitle)
			}
		}

		return &agent.ToolResult{
			Content: b.String(),
			Title:   fmt.Sprintf("Tra cứu cơ sở quy chế: %s", query),
			Sources: sources,
		}, nil

	case "lookup_curriculum", "lookup_curriculum_database":
		var args request.AdvisorLookupCurriculumArgs
		_ = json.Unmarshal([]byte(argsStr), &args)
		cohort := strings.TrimSpace(args.Cohort)
		major := strings.TrimSpace(args.Major)
		if cohort == "" && profile != nil && profile.CohortYear > 0 {
			cohort = fmt.Sprintf("K%d", profile.CohortYear)
		}
		if major == "" && profile != nil && profile.Major != "" {
			major = profile.Major
		}
		if s.curriculumSvc == nil {
			return &agent.ToolResult{Content: "Dịch vụ chương trình đào tạo chưa sẵn sàng.", Title: "Tra cứu CTĐT"}, nil
		}
		_, courses, err := s.curriculumSvc.GetCurriculum(ctx, cohort, major)
		if err != nil || len(courses) == 0 {
			return &agent.ToolResult{
				Content: fmt.Sprintf("Không tìm thấy chương trình đào tạo cho khóa %s ngành %s.", cohort, major),
				Title:   fmt.Sprintf("Tra cứu CTĐT %s %s", cohort, major),
			}, nil
		}
		srcTitle := fmt.Sprintf("Khung CTĐT %s ngành %s (ĐH Thăng Long)", cohort, major)
		return &agent.ToolResult{
			Content: fmt.Sprintf("# CHƯƠNG TRÌNH ĐÀO TẠO CHÍNH THỨC (%s - %s):\n\n%s", cohort, major, s.curriculumSvc.FormatCurriculumContext(courses)),
			Title:   fmt.Sprintf("Tra cứu CTĐT %s ngành %s", cohort, major),
			Sources: []string{srcTitle},
		}, nil

	case "lookup_course_substitutions":
		var args request.AdvisorLookupSubstitutionsArgs
		_ = json.Unmarshal([]byte(argsStr), &args)
		code := strings.ToUpper(strings.TrimSpace(args.CourseCode))
		cohort := strings.TrimSpace(args.Cohort)
		major := strings.TrimSpace(args.Major)
		if cohort == "" && profile != nil && profile.CohortYear > 0 {
			cohort = fmt.Sprintf("K%d", profile.CohortYear)
		}
		if major == "" && profile != nil && profile.Major != "" {
			major = profile.Major
		}
		if s.curriculumSvc == nil {
			return &agent.ToolResult{Content: "Dịch vụ môn thay thế chưa sẵn sàng.", Title: "Tra cứu môn thay thế"}, nil
		}
		subs, grps, err := s.curriculumSvc.GetSubstitutions(ctx, code, cohort, major)
		if err != nil || (len(subs) == 0 && len(grps) == 0) {
			return &agent.ToolResult{
				Content: fmt.Sprintf("Không tìm thấy môn thay thế chính thức nào cho môn %s trong phạm vi khóa %s.", code, cohort),
				Title:   fmt.Sprintf("Tra cứu môn thay thế: %s", code),
			}, nil
		}
		return &agent.ToolResult{
			Content: s.curriculumSvc.FormatSubstitutionsContext(code, subs, grps),
			Title:   fmt.Sprintf("Tra cứu môn thay thế: %s", code),
			Sources: []string{"Cơ sở dữ liệu Môn thay thế Khoa CNTT (ĐH Thăng Long)"},
		}, nil

	case "lookup_course_prerequisites":
		var args request.AdvisorLookupPrerequisitesArgs
		_ = json.Unmarshal([]byte(argsStr), &args)
		code := strings.ToUpper(strings.TrimSpace(args.CourseCode))
		cohort := strings.TrimSpace(args.Cohort)
		if cohort == "" && profile != nil && profile.CohortYear > 0 {
			cohort = fmt.Sprintf("K%d", profile.CohortYear)
		}
		if s.curriculumSvc == nil {
			return &agent.ToolResult{Content: "Dịch vụ điều kiện tiên quyết chưa sẵn sàng.", Title: "Tra cứu tiên quyết"}, nil
		}
		prereqs, dependents, err := s.curriculumSvc.GetPrerequisites(ctx, code, cohort)
		if err != nil || (len(prereqs) == 0 && len(dependents) == 0) {
			return &agent.ToolResult{
				Content: fmt.Sprintf("Không tìm thấy thông tin điều kiện tiên quyết cho môn %s.", code),
				Title:   fmt.Sprintf("Tra cứu tiên quyết: %s", code),
			}, nil
		}
		return &agent.ToolResult{
			Content: s.curriculumSvc.FormatPrerequisitesContext(code, cohort, prereqs, dependents),
			Title:   fmt.Sprintf("Tra cứu điều kiện tiên quyết: %s", code),
			Sources: []string{"Cơ sở dữ liệu Tiên quyết CTĐT (ĐH Thăng Long)"},
		}, nil

	case "lookup_student_transcript_and_roadmap", "lookup_student_transcript":
		if userID == "" || userID == "0" {
			return &agent.ToolResult{
				Content: "Sinh viên đang ở chế độ Khách (chưa đăng nhập). Vui lòng đăng nhập tài khoản sinh viên Thăng Long để tra cứu bảng điểm và CPA/GPA cá nhân.",
				Title:   "Tra cứu bảng điểm cá nhân",
			}, nil
		}
		if s.transcriptSvc == nil {
			return &agent.ToolResult{Content: "Dịch vụ bảng điểm chưa sẵn sàng.", Title: "Tra cứu bảng điểm"}, nil
		}
		summary, err := s.transcriptSvc.GetTranscriptSummary(ctx, userID)
		if err != nil || summary == nil || len(summary.Grades) == 0 {
			return &agent.ToolResult{
				Content: "Chưa có dữ liệu bảng điểm cá nhân được lưu trong hệ thống cho tài khoản sinh viên này.",
				Title:   "Tra cứu bảng điểm cá nhân",
			}, nil
		}
		var b strings.Builder
		b.WriteString("# BẢNG ĐIỂM HỌC TẬP CÁ NHÂN (DỮ LIỆU ĐÃ XÁC THỰC TLU):\n")
		fmt.Fprintf(&b, "- Tổng tín chỉ tích lũy: %.1f / %.1f\n", summary.TotalCreditsCompleted, summary.TotalCreditsAttempted)
		fmt.Fprintf(&b, "- Điểm trung bình CPA (Hệ 10): %.2f | (Hệ 4): %.2f\n", summary.CumulativeGPA10, summary.CumulativeGPA4)
		fmt.Fprintf(&b, "- Số học phần đã đạt: %d | Số học phần cần học lại: %d\n\n", summary.PassedCoursesCount, summary.FailedCoursesCount)
		for _, g := range summary.Grades {
			statusStr := "ĐẠT"
			if !g.IsPassed || g.GradeLetter == "F" {
				statusStr = "CẦN HỌC LẠI"
			}
			fmt.Fprintf(&b, "- [%s] %s (%.1f TC): Điểm số %.1f (%s) - %s\n", g.CourseCode, g.CourseName, g.Credits, g.Score10, g.GradeLetter, statusStr)
		}
		return &agent.ToolResult{
			Content: b.String(),
			Title:   "Tra cứu bảng điểm cá nhân & Phân tích lộ trình học tập",
			Sources: []string{"Hồ sơ bảng điểm học tập cá nhân sinh viên (Hệ thống TLU - Đã xác thực)"},
		}, nil

	case "sync_student_transcript_grades", "update_student_transcript":
		if userID == "" || userID == "0" {
			return &agent.ToolResult{
				Content: "Bạn đang ở chế độ Khách (chưa đăng nhập). Vui lòng đăng nhập để đồng bộ bảng điểm học tập cá nhân.",
				Title:   "Đồng bộ bảng điểm cá nhân",
			}, nil
		}
		if s.transcriptSvc == nil {
			return &agent.ToolResult{Content: "Dịch vụ bảng điểm chưa sẵn sàng.", Title: "Đồng bộ bảng điểm"}, nil
		}
		var args request.AdvisorSyncTranscriptGradesArgs
		_ = json.Unmarshal([]byte(argsStr), &args)
		if len(args.Grades) == 0 {
			var directList []request.AdvisorTranscriptGradeItem
			if err := json.Unmarshal([]byte(argsStr), &directList); err == nil && len(directList) > 0 {
				args.Grades = directList
			} else {
				var altWrapper struct {
					Items        []request.AdvisorTranscriptGradeItem `json:"items"`
					CourseGrades []request.AdvisorTranscriptGradeItem `json:"course_grades"`
				}
				if err2 := json.Unmarshal([]byte(argsStr), &altWrapper); err2 == nil {
					if len(altWrapper.Items) > 0 {
						args.Grades = altWrapper.Items
					} else if len(altWrapper.CourseGrades) > 0 {
						args.Grades = altWrapper.CourseGrades
					}
				}
			}
		}
		if len(args.Grades) == 0 {
			return &agent.ToolResult{
				Content: "Danh sách môn học cần đồng bộ trống.",
				Title:   "Đồng bộ bảng điểm cá nhân",
			}, nil
		}
		var gradeRecords []*models.StudentCourseGradeRecord
		for _, g := range args.Grades {
			code := strings.ToUpper(strings.TrimSpace(g.CourseCode))
			if code == "" {
				continue
			}
			gradeRecords = append(gradeRecords, &models.StudentCourseGradeRecord{
				CourseCode:  code,
				CourseName:  strings.TrimSpace(g.CourseName),
				Credits:     g.Credits,
				Score10:     g.Score10,
				Score4:      g.Score4,
				GradeLetter: strings.ToUpper(strings.TrimSpace(g.GradeLetter)),
				IsPassed:    g.IsPassed,
				IsRetake:    g.IsRetake,
				TermTaken:   strings.TrimSpace(g.TermTaken),
			})
		}
		summary, err := s.transcriptSvc.BatchSaveTranscript(ctx, userID, gradeRecords)
		if err != nil {
			return &agent.ToolResult{
				Content: fmt.Sprintf("Đồng bộ bảng điểm thất bại: %v", err),
				Title:   "Đồng bộ bảng điểm cá nhân",
			}, nil
		}
		return &agent.ToolResult{
			Content: fmt.Sprintf("Đã đồng bộ thành công %d môn học vào hồ sơ học vụ dài hạn.\n- Tổng tín chỉ tích lũy mới: %.1f TC\n- CPA Hệ 10: %.2f | Hệ 4: %.2f (Đã đạt %d môn, cần học lại %d môn).",
				len(gradeRecords), summary.TotalCreditsCompleted, summary.CumulativeGPA10, summary.CumulativeGPA4, summary.PassedCoursesCount, summary.FailedCoursesCount),
			Title:   fmt.Sprintf("Đồng bộ bảng điểm cá nhân (%d môn học)", len(gradeRecords)),
			Sources: []string{"Hồ sơ bảng điểm học tập cá nhân sinh viên (Hệ thống TLU - Đã xác thực)"},
		}, nil

	case "lookup_tuition_fee":
		if userID == "" || userID == "0" {
			return &agent.ToolResult{
				Content: "Bạn đang ở chế độ Khách (chưa đăng nhập). Vui lòng đăng nhập tài khoản sinh viên Thăng Long để tra cứu thông tin học phí cá nhân.",
				Title:   "Tra cứu học phí",
			}, nil
		}
		var args request.AdvisorLookupTuitionArgs
		_ = json.Unmarshal([]byte(argsStr), &args)
		targetStudentID := strings.TrimSpace(args.StudentID)
		userStudentCode := ""
		if profile != nil {
			userStudentCode = profile.StudentID
		}
		if userStudentCode == "" && s.userRepo != nil {
			if u, err := s.userRepo.GetByID(ctx, userID); err == nil && u != nil {
				userStudentCode = u.StudentCode
			}
		}
		if targetStudentID == "" {
			targetStudentID = userStudentCode
		}
		if targetStudentID == "" {
			return &agent.ToolResult{
				Content: "Tài khoản của bạn chưa được liên kết với mã sinh viên Thăng Long nào. Vui lòng cập nhật hồ sơ cá nhân để tra cứu học phí.",
				Title:   "Tra cứu học phí",
			}, nil
		}
		if userStudentCode != "" && !strings.EqualFold(targetStudentID, userStudentCode) {
			return &agent.ToolResult{
				Content: "Bạn chỉ có thể tra cứu thông tin học phí cho mã sinh viên của chính mình (" + userStudentCode + "). Không được phép tra cứu thông tin học phí của sinh viên khác.",
				Title:   "Tra cứu học phí",
			}, nil
		}
		if s.tuitionClient == nil {
			return &agent.ToolResult{Content: "Dịch vụ tra cứu học phí chưa sẵn sàng.", Title: "Tra cứu học phí"}, nil
		}
		info, err := s.tuitionClient.LookupTuition(ctx, targetStudentID)
		if err != nil || info == nil || !info.Found {
			return &agent.ToolResult{
				Content: fmt.Sprintf("Không tìm thấy thông tin công nợ học phí cho mã sinh viên %s trên cổng thanh toán.", targetStudentID),
				Title:   fmt.Sprintf("Tra cứu học phí: %s", targetStudentID),
			}, nil
		}
		return &agent.ToolResult{
			Content: tuition.FormatPromptContext(info),
			Title:   fmt.Sprintf("Tra cứu học phí SV: %s (%s)", info.StudentID, info.TotalDebt),
			Sources: []string{fmt.Sprintf("Cổng thanh toán học phí trực tuyến của Trường (%s)", tuition.HostLabel())},
		}, nil

	case "optimize_study_roadmap":
		var args request.AdvisorOptimizeRoadmapArgs
		_ = json.Unmarshal([]byte(argsStr), &args)
		cohort := strings.TrimSpace(args.Cohort)
		major := strings.TrimSpace(args.Major)
		if cohort == "" && profile != nil && profile.CohortYear > 0 {
			cohort = fmt.Sprintf("K%d", profile.CohortYear)
		}
		if major == "" && profile != nil && profile.Major != "" {
			major = profile.Major
		}
		var assumedDefaults []string
		curSem := args.CurrentSemester
		if curSem <= 0 {
			curSem = 1
			assumedDefaults = append(assumedDefaults, "học kỳ hiện tại được giả định là 1")
		}
		targetTerm := args.TargetSemesters
		if targetTerm <= 0 {
			targetTerm = 8
			assumedDefaults = append(assumedDefaults, "số học kỳ mục tiêu được giả định là 8")
		}
		pacePref := strings.TrimSpace(args.PacePreference)
		summerWillingness := args.SummerWillingness
		primaryStrength := strings.TrimSpace(args.PrimaryStrength)
		deliveryPref := strings.TrimSpace(args.DeliveryPreference)

		if userID != "" && userID != "0" && s.memoryService != nil {
			if mems, err := s.memoryService.ListUserMemories(ctx, userID); err == nil && len(mems) > 0 {
				for _, m := range mems {
					k := strings.ToUpper(strings.TrimSpace(m.MemoryKey))
					v := strings.TrimSpace(m.MemoryValue)
					if targetTerm == 8 && (k == "TARGET_SEMESTERS" || strings.Contains(k, "TARGET_TERM") || strings.Contains(k, "GRADUATION")) {
						if strings.Contains(v, "7") || strings.Contains(v, "3.5") {
							targetTerm = 7
						} else if strings.Contains(v, "9") || strings.Contains(v, "4.5") {
							targetTerm = 9
						}
					}
					if pacePref == "" && (k == "PACE_PREFERENCE" || strings.Contains(k, "PACE")) {
						pacePref = strings.ToLower(v)
					}
					if !summerWillingness && (k == "SUMMER_WILLINGNESS" || strings.Contains(k, "SUMMER")) {
						summerWillingness = strings.EqualFold(v, "true") || strings.EqualFold(v, "1") || strings.Contains(strings.ToLower(v), "sẵn sàng")
					}
					if primaryStrength == "" && (k == "PRIMARY_STRENGTH" || k == "SPECIALIZATION" || strings.Contains(k, "TRACK")) {
						primaryStrength = v
					}
					if deliveryPref == "" && (k == "DELIVERY_PREFERENCE" || strings.Contains(k, "DELIVERY")) {
						deliveryPref = v
					}
				}
			}
		}

		if s.roadmapOptimizer == nil {
			return &agent.ToolResult{Content: "Dịch vụ tối ưu lộ trình DAG chưa sẵn sàng.", Title: "Tối ưu hóa lộ trình"}, nil
		}
		var transcriptList []models.StudentCourseGrade
		if s.transcriptSvc != nil && userID != "" && userID != "0" {
			if sum, err := s.transcriptSvc.GetTranscriptSummary(ctx, userID); err == nil && sum != nil {
				for _, g := range sum.Grades {
					transcriptList = append(transcriptList, models.StudentCourseGrade{
						CourseCode: g.CourseCode,
						CourseName: g.CourseName,
						Credits:    g.Credits,
						Grade:      g.GradeLetter,
						Score:      g.Score10,
						IsRetake:   g.IsRetake,
					})
				}
			}
		}

		var quizAnswers *models.PersonalityQuizAnswers
		if pacePref != "" || summerWillingness || primaryStrength != "" || deliveryPref != "" {
			quizAnswers = &models.PersonalityQuizAnswers{
				PacePreference:     pacePref,
				SummerWillingness:  summerWillingness,
				PrimaryStrength:    primaryStrength,
				DeliveryPreference: deliveryPref,
			}
		}

		res, err := s.roadmapOptimizer.OptimizeRoadmap(ctx, &models.RoadmapOptimizationRequest{
			Cohort:               cohort,
			MajorName:            major,
			CurrentSemester:      curSem,
			TargetGraduationTerm: targetTerm,
			Transcript:           transcriptList,
			QuizAnswers:          quizAnswers,
		})
		if err != nil || res == nil {
			log.Warn().Err(err).Msg("advisory executeAdvisorTool: roadmap optimization failed")
			return &agent.ToolResult{
				Content: fmt.Sprintf("Không thể tối ưu lộ trình cho khóa %s ngành %s. Vui lòng thử lại sau.", cohort, major),
				Title:   "Tối ưu hóa lộ trình học tập",
			}, nil
		}
		var b strings.Builder
		b.WriteString("## ĐỀ XUẤT LỘ TRÌNH ĐĂNG KÝ HỌC KỲ TIẾP THEO (TỐI ƯU HÓA DAG):\n")
		if len(assumedDefaults) > 0 {
			fmt.Fprintf(&b, "- Lưu ý: %s do hồ sơ học vụ chưa có thông tin; hãy cung cấp để lộ trình chính xác hơn.\n", strings.Join(assumedDefaults, ", "))
		}
		fmt.Fprintf(&b, "- Tiến độ tích lũy: %.1f / %.1f tín chỉ (Còn lại: %.1f tín chỉ)\n", res.AccumulatedCredits, res.TotalCurriculumCredits, res.RemainingCredits)
		if len(res.CriticalPathCourses) > 0 {
			fmt.Fprintf(&b, "- Môn đường găng tiên quyết: %s\n", strings.Join(res.CriticalPathCourses, ", "))
		}
		if len(res.Semesters) > 0 {
			nextSem := res.Semesters[0]
			fmt.Fprintf(&b, "- Gợi ý đăng ký Học kỳ %d (Tổng %.1f tín chỉ):\n", nextSem.SemesterNumber, nextSem.TotalCredits)
			for _, sc := range nextSem.Courses {
				note := ""
				if sc.IsRetake {
					note = " [HỌC LẠI]"
				}
				fmt.Fprintf(&b, "  + [%s] %s (%.1f TC)%s\n", sc.CourseCode, sc.CourseName, sc.Credits, note)
			}
		}
		return &agent.ToolResult{
			Content: b.String(),
			Title:   "Tối ưu hóa tiến độ học tập & đồ thị tiên quyết DAG",
			Sources: []string{"Thuật toán tối ưu tiến độ học tập DAG & Đồ thị tiên quyết (TLU Roadmap Optimizer)"},
			Data:    res,
		}, nil

	case "update_student_academic_preferences", "save_student_roadmap_preferences":
		if userID == "" || userID == "0" {
			return &agent.ToolResult{
				Content: "Bạn đang ở chế độ Khách (chưa đăng nhập). Vui lòng đăng nhập tài khoản sinh viên Thăng Long để lưu trữ sở thích và lộ trình học tập vào Hồ sơ dài hạn.",
				Title:   "Lưu sở thích học vụ",
			}, nil
		}
		if s.memoryService == nil {
			return &agent.ToolResult{Content: "Dịch vụ bộ nhớ học vụ chưa sẵn sàng.", Title: "Lưu sở thích"}, nil
		}

		var args request.AdvisorUpdatePreferencesArgs
		_ = json.Unmarshal([]byte(argsStr), &args)
		var savedItems []string

		targetSems := args.TargetSemesters
		if targetSems > 0 {
			years := float64(targetSems) / 2.0
			val := fmt.Sprintf("%d kỳ (khoảng %.1f năm)", targetSems, years)
			_, _ = s.memoryService.SaveUserMemory(ctx, &models.UserMemoryItem{
				UserID:      userID,
				Category:    "ROADMAP_PREFERENCE",
				MemoryKey:   "TARGET_SEMESTERS",
				MemoryValue: val,
				Confidence:  1.0,
				Source:      "AGENT_CONFIRMED",
			})
			savedItems = append(savedItems, fmt.Sprintf("Mục tiêu tốt nghiệp: %s", val))
		}

		pace := strings.TrimSpace(args.PacePreference)
		if pace != "" {
			_, _ = s.memoryService.SaveUserMemory(ctx, &models.UserMemoryItem{
				UserID:      userID,
				Category:    "ROADMAP_PREFERENCE",
				MemoryKey:   "PACE_PREFERENCE",
				MemoryValue: pace,
				Confidence:  1.0,
				Source:      "AGENT_CONFIRMED",
			})
			savedItems = append(savedItems, fmt.Sprintf("Nhịp độ học tập: %s", pace))
		}

		if args.SummerWillingness {
			_, _ = s.memoryService.SaveUserMemory(ctx, &models.UserMemoryItem{
				UserID:      userID,
				Category:    "ROADMAP_PREFERENCE",
				MemoryKey:   "SUMMER_WILLINGNESS",
				MemoryValue: "true",
				Confidence:  1.0,
				Source:      "AGENT_CONFIRMED",
			})
			savedItems = append(savedItems, "Kế hoạch học hè: Sẵn sàng học kỳ hè")
		}

		workSched := strings.TrimSpace(args.WorkSchedule)
		if workSched != "" {
			_, _ = s.memoryService.SaveUserMemory(ctx, &models.UserMemoryItem{
				UserID:      userID,
				Category:    "WORK_SCHEDULE",
				MemoryKey:   "WORK_COMMITMENT",
				MemoryValue: workSched,
				Confidence:  1.0,
				Source:      "AGENT_CONFIRMED",
			})
			savedItems = append(savedItems, fmt.Sprintf("Lịch làm việc ngoài giờ: %s", workSched))
		}

		track := strings.TrimSpace(args.FocusTrack)
		if track != "" {
			_, _ = s.memoryService.SaveUserMemory(ctx, &models.UserMemoryItem{
				UserID:      userID,
				Category:    "STUDY_TRACK",
				MemoryKey:   "SPECIALIZATION",
				MemoryValue: track,
				Confidence:  1.0,
				Source:      "AGENT_CONFIRMED",
			})
			savedItems = append(savedItems, fmt.Sprintf("Định hướng chuyên sâu: %s", track))
		}

		notes := strings.TrimSpace(args.CustomNotes)
		if notes != "" {
			_, _ = s.memoryService.SaveUserMemory(ctx, &models.UserMemoryItem{
				UserID:      userID,
				Category:    "COURSE_OVERRIDE",
				MemoryKey:   "ADVISOR_NOTES",
				MemoryValue: notes,
				Confidence:  1.0,
				Source:      "AGENT_CONFIRMED",
			})
			savedItems = append(savedItems, fmt.Sprintf("Ghi chú điều chỉnh: %s", notes))
		}

		if len(savedItems) == 0 {
			return &agent.ToolResult{
				Content: "Không có thông tin sở thích học tập nào được cung cấp để lưu trữ.",
				Title:   "Cập nhật sở thích học vụ",
			}, nil
		}

		resultText := fmt.Sprintf("Đã cập nhật thành công các sở thích học vụ vào Bộ nhớ dài hạn (Long-Term Memory):\n- %s\n\nCố vấn đã ghi nhớ các thông tin này và sẽ tự động áp dụng vào các bản dự thảo lộ trình tiếp theo.", strings.Join(savedItems, "\n- "))
		return &agent.ToolResult{
			Content: resultText,
			Title:   "Cập nhật sở thích học vụ vào Bộ nhớ dài hạn",
			Sources: []string{"Hồ sơ bộ nhớ dài hạn của sinh viên (TLU Long-Term Memory)"},
		}, nil

	default:
		return &agent.ToolResult{
			Content: fmt.Sprintf("Công cụ không được hỗ trợ: %s", name),
			Title:   name,
		}, nil
	}
}

func (s *advisoryService) executeOfflineFallback(ctx context.Context, userID string, profile *memory.StudentAcademicProfile, dto *request.ChatRequest, onEvent func(evt StreamEvent) error, timestamp string, sources []string) (*response.ChatResponse, error) {
	lower := strings.ToLower(dto.Query)
	if s.transcriptSvc != nil && userID != "" && userID != "0" && (strings.Contains(lower, "bảng điểm") || strings.Contains(lower, "lộ trình") || strings.Contains(lower, "tin chi")) {
		tc := llm.ToolCall{
			ID:   "call_transcript_fallback",
			Type: "function",
			Function: llm.FunctionCall{
				Name:      "lookup_student_transcript_and_roadmap",
				Arguments: "{}",
			},
		}
		if onEvent != nil {
			_ = onEvent(StreamEvent{
				Type: "tool_call",
				ToolCall: &ToolCallInfo{
					ID:     tc.ID,
					Name:   tc.Function.Name,
					Status: "running",
					Title:  "Tra cứu bảng điểm cá nhân & Phân tích lộ trình học tập",
				},
			})
		}
		res, _ := s.executeAdvisorTool(ctx, userID, profile, tc)
		if len(res.Sources) > 0 {
			sources = append(sources, res.Sources...)
		}
		var rMap *response.RoadmapOptimizationResultResponse
		if res != nil && res.Data != nil {
			if r, ok := res.Data.(*response.RoadmapOptimizationResultResponse); ok {
				rMap = r
			}
		}
		if onEvent != nil {
			_ = onEvent(StreamEvent{
				Type: "tool_call",
				ToolCall: &ToolCallInfo{
					ID:     tc.ID,
					Name:   tc.Function.Name,
					Status: "completed",
					Title:  res.Title,
					Result: "Đã hoàn thành phân tích",
				},
				Sources: deduplicateSources(sources),
			})
			if rMap != nil {
				_ = onEvent(StreamEvent{
					Type:    "roadmap",
					Roadmap: rMap,
				})
			}
			_ = onEvent(StreamEvent{
				Type:    "chunk",
				Content: res.Content,
			})
		}
		return &response.ChatResponse{
			Query:       dto.Query,
			Reply:       res.Content,
			Sources:     deduplicateSources(sources),
			Timestamp:   timestamp,
			CanEscalate: false,
			Roadmap:     rMap,
		}, nil
	}

	fb := s.fallbackReply(ctx, dto.Query, sources, timestamp)
	fb.Reply = guardrail.SanitizeOutput(fb.Reply)
	fb.CanEscalate = true
	if onEvent != nil {
		_ = onEvent(StreamEvent{
			Type:    "chunk",
			Content: fb.Reply,
		})
		_ = onEvent(StreamEvent{
			Type:    "can_escalate",
			Content: "true",
		})
	}
	return fb, nil
}

func (s *advisoryService) buildMessages(ctx context.Context, userID string, dto *request.ChatRequest, nonce string) ([]llm.Message, []string, *memory.StudentAcademicProfile, bool) {
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
			conv, err := s.chatRepo.GetConversationByID(ctx, dto.SessionID)
			if err == nil && conv != nil && conv.UserID == userID {
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
		}
		if (userID == "" || userID == "0") && len(dialogue) == 0 && s.ramCache != nil {
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

	var explicitMajor string
	if s.curriculumSvc != nil {
		explicitMajor = s.curriculumSvc.FindMajorInText(ctx, dto.Query)
		if explicitMajor == "" && len(dialogue) > 0 {
			for i := len(dialogue) - 1; i >= 0; i-- {
				if dialogue[i].Role == compactor.RoleUser {
					if m := s.curriculumSvc.FindMajorInText(ctx, dialogue[i].Content); m != "" {
						explicitMajor = m
						break
					}
				}
			}
		}
	}

	if profile != nil {
		if studentCohort != "" && profile.CohortYear == 0 {
			cleanCohort := strings.TrimPrefix(strings.ToUpper(studentCohort), "K")
			if n, err := strconv.Atoi(cleanCohort); err == nil {
				profile.CohortYear = n
			}
		}
		if explicitMajor != "" && profile.Major == "" {
			profile.Major = explicitMajor
		}
	} else if studentCohort != "" || explicitMajor != "" {
		profile = &memory.StudentAcademicProfile{
			Major: explicitMajor,
		}
		if studentCohort != "" {
			cleanCohort := strings.TrimPrefix(strings.ToUpper(studentCohort), "K")
			if n, err := strconv.Atoi(cleanCohort); err == nil {
				profile.CohortYear = n
			}
		}
	}

	if s.agentService != nil {
		prompt, promptSources, err := s.agentService.BuildSystemPrompt(ctx, profile, nil, nonce)
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
			fmt.Fprintf(&systemPrompt, "<student_memory nonce=\"%s\">\n%s\n</student_memory>\n", nonce, memory.FormatStudentPromptContext(profile))
		}
		if s.ruleRepo != nil {
			if rule, err := s.ruleRepo.FindByQuery(ctx, dto.Query); err == nil && rule != nil {
				fmt.Fprintf(&systemPrompt, "\n\n[SYSTEM ADVISORY RULE]\n- Topic: %s\n- Content: %s\n- Sources: %s\n", rule.Title, rule.Content, strings.Join(rule.Sources, ", "))
				if len(rule.Sources) > 0 {
					sources = append(sources, rule.Sources...)
				}
			}
		}
	}

	systemPrompt.WriteString(agent.BuildAdvisoryDirectivesPrompt(nonce))

	if s.memoryService != nil && userID != "" && userID != "0" {
		if memPrompt, err := s.memoryService.BuildLongTermMemoryPromptContext(ctx, userID); err == nil && memPrompt != "" {
			systemPrompt.WriteString("\n\n---\n\n")
			fmt.Fprintf(&systemPrompt, "<student_memory nonce=\"%s\">\n%s\n</student_memory>\n", nonce, memPrompt)
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
	compacted, didCompact, _ := compactor.CompactHistory(ctx, dialogue, cfg)

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

	wrappedQuery := fmt.Sprintf("<user_query nonce=\"%s\">\n%s\n</user_query>", nonce, dto.Query)
	if len(dto.Images) > 0 {
		parts := make([]llm.ContentPart, 0, len(dto.Images)+1)
		parts = append(parts, llm.ContentPart{
			Type: llm.ContentPartText,
			Text: wrappedQuery,
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
			Content: wrappedQuery,
		})
	}

	return messages, deduplicateSources(sources), profile, didCompact
}

func (s *advisoryService) Consult(ctx context.Context, userID string, dto *request.ChatRequest) (*response.ChatResponse, error) {
	timestamp := time.Now().Format("15:04:05")

	if userID != "" && userID != "0" && dto.SessionID != "" && s.chatRepo != nil {
		conv, err := s.chatRepo.GetConversationByID(ctx, dto.SessionID)
		if err == nil && conv != nil && conv.UserID != userID {
			return nil, apperrors.New(apperrors.ErrForbidden, "conversation does not belong to user")
		}
	}

	chk := guardrail.CheckQueryContext(ctx, dto.Query)
	if chk.Decision != guardrail.DecisionAllow {
		return &response.ChatResponse{
			Query:       dto.Query,
			Reply:       chk.Response,
			Sources:     nil,
			Timestamp:   timestamp,
			CanEscalate: false,
		}, nil
	}

	for _, h := range dto.History {
		if hChk := guardrail.CheckQueryContext(ctx, h.Content); hChk.Decision == guardrail.DecisionBlockedInjection {
			return &response.ChatResponse{
				Query:       dto.Query,
				Reply:       hChk.Response,
				Sources:     nil,
				Timestamp:   timestamp,
				CanEscalate: false,
			}, nil
		}
	}

	nonce, _ := crypto.GenerateRandomHex(8)
	messages, initialSources, profile, didCompact := s.buildMessages(ctx, userID, dto, nonce)
	sources := initialSources

	tools := s.getToolDefinitions(ctx)

	if s.llmManager != nil && len(s.llmManager.List()) > 0 {
		prov, modelKey, modelRow := s.resolveModelAndProvider(ctx, dto.ModelID)
		maxTurns := 6
		var lastRoadmap *response.RoadmapOptimizationResultResponse

		for turn := 0; turn < maxTurns; turn++ {
			llmReq := s.buildChatRequest(dto, modelKey, modelRow, messages)
			llmReq.Tools = tools

			var resp *llm.ChatResponse
			var err error

			if prov != nil {
				resp, err = prov.Chat(ctx, llmReq)
			}
			if resp == nil && s.llmManager != nil {
				resp, err = s.llmManager.Chat(ctx, llmReq)
			}

			if err != nil || resp == nil {
				log.Warn().Err(err).Msg("advisory Consult: Chat failed in turn")
				break
			}

			if len(resp.Message.ToolCalls) > 0 {
				messages = append(messages, resp.Message)
				for _, tc := range resp.Message.ToolCalls {
					res, _ := s.executeAdvisorTool(ctx, userID, profile, tc)
					content := ""
					if res != nil {
						content = res.Content
						if len(res.Sources) > 0 {
							sources = append(sources, res.Sources...)
						}
						if res.Data != nil {
							if rMap, ok := res.Data.(*response.RoadmapOptimizationResultResponse); ok {
								lastRoadmap = rMap
							}
						}
					}
					messages = append(messages, llm.Message{
						Role:       llm.RoleTool,
						ToolCallID: tc.ID,
						Content:    content,
					})
				}
				continue
			}

			cleanReply, canEsc := processEscalation(resp.Message.Content)
			reply := guardrail.SanitizeOutput(cleanReply)
			res := &response.ChatResponse{
				Query:       dto.Query,
				Reply:       reply,
				Sources:     deduplicateSources(sources),
				Timestamp:   timestamp,
				CanEscalate: canEsc,
				Roadmap:     lastRoadmap,
				DidCompact:  didCompact,
			}
			if s.memoryService != nil && userID != "" && userID != "0" {
				go func() {
					_ = s.memoryService.ExtractAndSaveMemories(context.Background(), userID, dto.Query, res.Reply)
				}()
			}
			return res, nil
		}
	}

	return s.executeOfflineFallback(ctx, userID, profile, dto, nil, timestamp, sources)
}

func (s *advisoryService) ConsultStream(ctx context.Context, userID string, dto *request.ChatRequest, onEvent func(evt StreamEvent) error) (*response.ChatResponse, error) {
	timestamp := time.Now().Format("15:04:05")

	if userID != "" && userID != "0" && dto.SessionID != "" && s.chatRepo != nil {
		conv, err := s.chatRepo.GetConversationByID(ctx, dto.SessionID)
		if err == nil && conv != nil && conv.UserID != userID {
			return nil, apperrors.New(apperrors.ErrForbidden, "conversation does not belong to user")
		}
	}

	chk := guardrail.CheckQueryContext(ctx, dto.Query)
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

	for _, h := range dto.History {
		if hChk := guardrail.CheckQueryContext(ctx, h.Content); hChk.Decision == guardrail.DecisionBlockedInjection {
			if onEvent != nil {
				_ = onEvent(StreamEvent{
					Type:    "chunk",
					Content: hChk.Response,
				})
			}
			return &response.ChatResponse{
				Query:       dto.Query,
				Reply:       hChk.Response,
				Sources:     nil,
				Timestamp:   timestamp,
				CanEscalate: false,
			}, nil
		}
	}

	nonce, _ := crypto.GenerateRandomHex(8)
	messages, initialSources, profile, didCompact := s.buildMessages(ctx, userID, dto, nonce)
	sources := initialSources

	if didCompact && onEvent != nil {
		_ = onEvent(StreamEvent{
			Type:    "compact",
			Content: "true",
		})
	}

	tools := s.getToolDefinitions(ctx)

	if s.llmManager != nil && len(s.llmManager.List()) > 0 {
		prov, modelKey, modelRow := s.resolveModelAndProvider(ctx, dto.ModelID)
		maxTurns := 6
		var lastRoadmap *response.RoadmapOptimizationResultResponse

		for turn := 0; turn < maxTurns; turn++ {
			llmReq := s.buildChatRequest(dto, modelKey, modelRow, messages)
			llmReq.Tools = tools
			llmReq.Stream = true

			var stream llm.StreamReader
			var streamErr error

			if prov != nil {
				stream, streamErr = prov.ChatStream(ctx, llmReq)
			}
			if (stream == nil || streamErr != nil) && s.llmManager != nil {
				stream, streamErr = s.llmManager.ChatStream(ctx, llmReq)
			}

			if stream != nil && streamErr == nil {
				var fullContent strings.Builder
				var fullReasoning strings.Builder
				accumulators := make(map[int]*advisoryToolCallAccumulator)
				var accOrder []int

				filterBuf := newStreamFilterBuffer(200, func(chunkToSend string) error {
					if onEvent != nil {
						return onEvent(StreamEvent{
							Type:    "chunk",
							Content: chunkToSend,
						})
					}
					return nil
				})

				for {
					chunk, err := stream.Recv()
					if err == io.EOF {
						break
					}
					if err != nil {
						log.Warn().Err(err).Msg("advisory ConsultStream: error receiving stream chunk")
						break
					}
					if chunk == nil {
						continue
					}

					if chunk.Delta.ReasoningContent != "" {
						fullReasoning.WriteString(chunk.Delta.ReasoningContent)
						if onEvent != nil {
							safeReasoning, _ := guardrail.SanitizeStreamChunk(chunk.Delta.ReasoningContent)
							if safeReasoning != "" {
								_ = onEvent(StreamEvent{
									Type:             "reasoning",
									ReasoningContent: safeReasoning,
									Content:          safeReasoning,
								})
							}
						}
					}

					if len(chunk.Delta.ToolCalls) > 0 {
						for _, tcChunk := range chunk.Delta.ToolCalls {
							acc, ok := accumulators[tcChunk.Index]
							if !ok {
								acc = &advisoryToolCallAccumulator{callType: "function"}
								accumulators[tcChunk.Index] = acc
								accOrder = append(accOrder, tcChunk.Index)
							}
							if tcChunk.ID != "" {
								acc.id = tcChunk.ID
							}
							if tcChunk.Type != "" {
								acc.callType = tcChunk.Type
							}
							if tcChunk.Function.Name != "" {
								acc.name = tcChunk.Function.Name
							}
							if tcChunk.Function.Arguments != "" {
								acc.arguments.WriteString(tcChunk.Function.Arguments)
							}
							if !acc.runningEmitted && acc.name != "" {
								acc.runningEmitted = true
								if acc.id == "" {
									acc.id = fmt.Sprintf("call_%d", tcChunk.Index)
								}
								if onEvent != nil {
									_ = onEvent(StreamEvent{
										Type: "tool_call",
										ToolCall: &ToolCallInfo{
											ID:     acc.id,
											Name:   acc.name,
											Status: "running",
											Title:  "Đang thực hiện tra cứu...",
										},
									})
								}
							}
						}
					}

					if chunk.Delta.Content != "" {
						fullContent.WriteString(chunk.Delta.Content)
						if len(accumulators) == 0 {
							_ = filterBuf.Feed(chunk.Delta.Content)
						}
					}
				}
				_ = stream.Close()

				if len(accOrder) > 0 {
					filterBuf.Reset()
					var assembledToolCalls []llm.ToolCall
					for _, idx := range accOrder {
						acc := accumulators[idx]
						if acc.id == "" {
							acc.id = fmt.Sprintf("call_%d", idx)
						}
						if !acc.runningEmitted && onEvent != nil {
							_ = onEvent(StreamEvent{
								Type: "tool_call",
								ToolCall: &ToolCallInfo{
									ID:     acc.id,
									Name:   acc.name,
									Status: "running",
									Title:  "Đang thực hiện tra cứu...",
								},
							})
						}
						assembledToolCalls = append(assembledToolCalls, llm.ToolCall{
							ID:   acc.id,
							Type: acc.callType,
							Function: llm.FunctionCall{
								Name:      acc.name,
								Arguments: acc.arguments.String(),
							},
						})
					}

					messages = append(messages, llm.Message{
						Role:             llm.RoleAssistant,
						Content:          "",
						ReasoningContent: fullReasoning.String(),
						ToolCalls:        assembledToolCalls,
					})

					for _, tc := range assembledToolCalls {
						res, _ := s.executeAdvisorTool(ctx, userID, profile, tc)
						content := ""
						title := "Đang xử lý kết quả"
						if res != nil {
							content = res.Content
							title = res.Title
							if len(res.Sources) > 0 {
								sources = append(sources, res.Sources...)
							}
							if res.Data != nil {
								if rMap, ok := res.Data.(*response.RoadmapOptimizationResultResponse); ok {
									lastRoadmap = rMap
									if onEvent != nil {
										_ = onEvent(StreamEvent{
											Type:    "roadmap",
											Roadmap: rMap,
										})
									}
								}
							}
						}

						if onEvent != nil {
							_ = onEvent(StreamEvent{
								Type: "tool_call",
								ToolCall: &ToolCallInfo{
									ID:     tc.ID,
									Name:   tc.Function.Name,
									Status: "completed",
									Title:  title,
									Result: "Đã hoàn thành tra cứu",
								},
								Sources: deduplicateSources(sources),
							})
						}

						messages = append(messages, llm.Message{
							Role:       llm.RoleTool,
							ToolCallID: tc.ID,
							Content:    content,
						})
					}
					continue
				}

				_ = filterBuf.Flush()
				cleanReply, canEsc := processEscalation(fullContent.String())
				reply := guardrail.SanitizeOutput(strings.TrimSpace(cleanReply))
				if reply != "" {
					if canEsc && onEvent != nil {
						_ = onEvent(StreamEvent{
							Type:    "can_escalate",
							Content: "true",
						})
					}
					dedupSources := deduplicateSources(sources)
					if len(dedupSources) > 0 && onEvent != nil {
						_ = onEvent(StreamEvent{
							Type:    "sources",
							Sources: dedupSources,
						})
					}
					res := &response.ChatResponse{
						Query:       dto.Query,
						Reply:       reply,
						Sources:     deduplicateSources(sources),
						Timestamp:   timestamp,
						CanEscalate: canEsc,
						Roadmap:     lastRoadmap,
						DidCompact:  didCompact,
					}
					if s.memoryService != nil && userID != "" && userID != "0" {
						go func() {
							_ = s.memoryService.ExtractAndSaveMemories(context.Background(), userID, dto.Query, reply)
						}()
					}
					return res, nil
				}
				break
			}

			llmReq.Stream = false
			var chatResp *llm.ChatResponse
			var chatErr error
			if prov != nil {
				chatResp, chatErr = prov.Chat(ctx, llmReq)
			}
			if chatResp == nil && s.llmManager != nil {
				chatResp, chatErr = s.llmManager.Chat(ctx, llmReq)
			}
			if chatErr != nil || chatResp == nil {
				log.Warn().Err(chatErr).Msg("advisory ConsultStream: Chat fallback failed in turn")
				break
			}

			if len(chatResp.Message.ToolCalls) > 0 {
				chatResp.Message.Content = ""
				messages = append(messages, chatResp.Message)
				for _, tc := range chatResp.Message.ToolCalls {
					if onEvent != nil {
						_ = onEvent(StreamEvent{
							Type: "tool_call",
							ToolCall: &ToolCallInfo{
								ID:     tc.ID,
								Name:   tc.Function.Name,
								Status: "running",
								Title:  "Đang thực hiện tra cứu...",
							},
						})
					}

					res, _ := s.executeAdvisorTool(ctx, userID, profile, tc)
					content := ""
					title := "Đang xử lý kết quả"
					if res != nil {
						content = res.Content
						title = res.Title
						if len(res.Sources) > 0 {
							sources = append(sources, res.Sources...)
						}
						if res.Data != nil {
							if rMap, ok := res.Data.(*response.RoadmapOptimizationResultResponse); ok {
								lastRoadmap = rMap
								if onEvent != nil {
									_ = onEvent(StreamEvent{
										Type:    "roadmap",
										Roadmap: rMap,
									})
								}
							}
						}
					}

					if onEvent != nil {
						_ = onEvent(StreamEvent{
							Type: "tool_call",
							ToolCall: &ToolCallInfo{
								ID:     tc.ID,
								Name:   tc.Function.Name,
								Status: "completed",
								Title:  title,
								Result: "Đã hoàn thành tra cứu",
							},
							Sources: deduplicateSources(sources),
						})
					}

					messages = append(messages, llm.Message{
						Role:       llm.RoleTool,
						ToolCallID: tc.ID,
						Content:    content,
					})
				}
				continue
			}

			if chatResp.Message.ReasoningContent != "" {
				if onEvent != nil {
					safeReasoning, _ := guardrail.SanitizeStreamChunk(chatResp.Message.ReasoningContent)
					if safeReasoning != "" {
						_ = onEvent(StreamEvent{
							Type:             "reasoning",
							ReasoningContent: safeReasoning,
							Content:          safeReasoning,
						})
					}
				}
			}

			cleanReply, canEsc := processEscalation(chatResp.Message.Content)
			reply := guardrail.SanitizeOutput(strings.TrimSpace(cleanReply))
			if reply != "" {
				if onEvent != nil {
					filterBuf := newStreamFilterBuffer(80, func(chunkToSend string) error {
						return onEvent(StreamEvent{
							Type:    "chunk",
							Content: chunkToSend,
						})
					})
					runes := []rune(reply)
					chunkSize := 16
					for i := 0; i < len(runes); i += chunkSize {
						end := i + chunkSize
						if end > len(runes) {
							end = len(runes)
						}
						_ = filterBuf.Feed(string(runes[i:end]))
						time.Sleep(15 * time.Millisecond)
					}
					_ = filterBuf.Flush()

					if canEsc {
						_ = onEvent(StreamEvent{
							Type:    "can_escalate",
							Content: "true",
						})
					}
					dedupSources := deduplicateSources(sources)
					if len(dedupSources) > 0 {
						_ = onEvent(StreamEvent{
							Type:    "sources",
							Sources: dedupSources,
						})
					}
				}
				res := &response.ChatResponse{
					Query:       dto.Query,
					Reply:       reply,
					Sources:     deduplicateSources(sources),
					Timestamp:   timestamp,
					CanEscalate: canEsc,
					Roadmap:     lastRoadmap,
					DidCompact:  didCompact,
				}
				if s.memoryService != nil && userID != "" && userID != "0" {
					go func() {
						_ = s.memoryService.ExtractAndSaveMemories(context.Background(), userID, dto.Query, reply)
					}()
				}
				return res, nil
			}
			break
		}
	}

	return s.executeOfflineFallback(ctx, userID, profile, dto, onEvent, timestamp, sources)
}

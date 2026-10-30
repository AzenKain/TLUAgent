package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/agent"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/config"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/convert"
	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/supersede"
)

// SupersessionAction denotes the resolution category for regulatory supersede analysis.
type SupersessionAction string

const (
	ActionNone            SupersessionAction = "NONE"
	ActionFullSupersede   SupersessionAction = "FULL_SUPERSEDE"
	ActionPartialAmend    SupersessionAction = "PARTIAL_AMEND"
	ActionCoexistByCohort SupersessionAction = "COEXIST_BY_COHORT"
)

// DissectionRequest captures input parameters for analyzing a regulatory document.
type DissectionRequest struct {
	NewDocID         string             `json:"new_doc_id"`
	Title            string             `json:"title"`
	DocCode          string             `json:"doc_code"`
	DocType          string             `json:"doc_type"`
	Domain           string             `json:"domain"`
	ApplicableCohort string             `json:"applicable_cohort"`
	AcademicYear     string             `json:"academic_year"`
	Content          string             `json:"content"`
	PublishDate      string             `json:"publish_date,omitempty"`
	FilePath         string             `json:"file_path,omitempty"`
	PriorityLevel    int                `json:"priority_level,omitempty"`
	Candidates       []*models.Document `json:"-"`
	FullContent      string             `json:"-"`
}

// SupersededArticleItem encapsulates an individual rule or clause marked for supersession.
type SupersededArticleItem struct {
	RuleID        string `json:"rule_id"`
	ArticleNumber string `json:"article_number"`
	Title         string `json:"title"`
	Reason        string `json:"reason"`
}

// RetainedArticleItem represents an existing rule that remains active and unchanged.
type RetainedArticleItem struct {
	RuleID        string `json:"rule_id"`
	ArticleNumber string `json:"article_number"`
	Title         string `json:"title"`
}

// GranularSupersessionPlan provides the structured outcome of multi-turn regulatory analysis.
type GranularSupersessionPlan struct {
	ActionType         SupersessionAction        `json:"action_type"`
	TargetDocID        string                    `json:"target_doc_id,omitempty"`
	TargetDocTitle     string                    `json:"target_doc_title,omitempty"`
	Reason             string                    `json:"reason"`
	SupersededArticles []SupersededArticleItem   `json:"superseded_articles"`
	RetainedArticles   []RetainedArticleItem     `json:"retained_articles"`
	NewArticles        []supersede.ParsedArticle `json:"new_articles"`
	AffectedCohorts    []string                  `json:"affected_cohorts"`
	ImpactSummary      string                    `json:"impact_summary"`
	RiskLevel          string                    `json:"risk_level"`
	InquiryQuestion    string                    `json:"inquiry_question,omitempty"`
	DiffItems          []models.TLUClawDiffItem  `json:"diff_items"`
	BlastRadius        models.TLUClawBlastRadius `json:"blast_radius"`
}

// EngineParams specifies constructor dependencies for RegulatoryDissectionEngine.
type EngineParams struct {
	DB             sqlc.DBTX
	RAGRepo        repositories.RAGRepository
	CurriculumRepo repositories.CurriculumRepository
	InquiryRepo    repositories.InquiryRepository
	NotifRepo      repositories.NotificationRepository
	UserRepo       repositories.UserRepository
	LLMManager     *llm.Manager
}

// RegulatoryDissectionEngine orchestrates multi-turn tool calling for regulatory dissection and supersession auditing.
type RegulatoryDissectionEngine struct {
	q              *sqlc.Queries
	ragRepo        repositories.RAGRepository
	curriculumRepo repositories.CurriculumRepository
	inquiryRepo    repositories.InquiryRepository
	notifRepo      repositories.NotificationRepository
	userRepo       repositories.UserRepository
	mgr            *llm.Manager
	sfg            *singleflight.Group

	dissectFn      func(ctx context.Context, req *DissectionRequest) (*GranularSupersessionPlan, error)
	auditFailed    *sync.Map
	auditBatchSize int
}

const (
	candidateExcerptRunes = 400
	auditDocTimeout       = 8 * time.Minute
)

const (
	DefaultAuditBatchSize = 200
	auditConcurrency      = 5
)

// NewRegulatoryDissectionEngine creates a new RegulatoryDissectionEngine instance.
func NewRegulatoryDissectionEngine(params EngineParams) *RegulatoryDissectionEngine {
	var q *sqlc.Queries
	if params.DB != nil {
		q = sqlc.New(params.DB)
	}
	return &RegulatoryDissectionEngine{
		auditFailed:    &sync.Map{},
		q:              q,
		ragRepo:        params.RAGRepo,
		curriculumRepo: params.CurriculumRepo,
		inquiryRepo:    params.InquiryRepo,
		notifRepo:      params.NotifRepo,
		userRepo:       params.UserRepo,
		mgr:            params.LLMManager,
		sfg:            &singleflight.Group{},
	}
}

// GetToolDefinitions produces LLM tool specifications for regulatory analysis.
func (e *RegulatoryDissectionEngine) GetToolDefinitions() []llm.ToolDefinition {
	return agent.GetRegulatoryDissectionToolDefinitions()
}

func compactDissectionHistory(ctx context.Context, msgs []llm.Message) []llm.Message {
	out := make([]llm.Message, len(msgs))
	for i, m := range msgs {
		out[i] = m
		if len(m.Content) > 4000 && m.Role == llm.RoleTool {
			out[i].Content = m.Content[:4000] + "\n...[truncated]"
		}
	}
	return out
}

// ExecuteDissectionLoop executes multi-turn tool verification to construct an article-level supersession plan.
func needsInspection(p *request.RegulatoryProposeGranularSupersessionArgs, inspected map[string]bool) bool {
	action := SupersessionAction(strings.TrimSpace(p.ActionType))
	if action != ActionFullSupersede && action != ActionPartialAmend {
		return false
	}
	return !inspected[strings.TrimSpace(p.TargetDocID)]
}

func (e *RegulatoryDissectionEngine) candidateExcerpt(ctx context.Context, docID string) string {
	if e.ragRepo == nil {
		return ""
	}
	chunks, err := e.ragRepo.GetChunksByDocumentID(ctx, docID)
	if err != nil || len(chunks) == 0 {
		return ""
	}
	text := strings.Join(strings.Fields(chunks[0].Content), " ")
	if runes := []rune(text); len(runes) > candidateExcerptRunes {
		text = string(runes[:candidateExcerptRunes]) + "..."
	}
	return text
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (e *RegulatoryDissectionEngine) ExecuteDissectionLoop(ctx context.Context, req *DissectionRequest) (*GranularSupersessionPlan, error) {
	if req == nil {
		return nil, fmt.Errorf("nil dissection request")
	}

	parsedArticles := supersede.ParseArticles(req.Content)

	if e.mgr == nil || len(e.mgr.List()) == 0 {
		var candidateIDs []string
		var candidateTitles []string
		if len(req.Candidates) > 0 {
			for _, c := range req.Candidates {
				candidateIDs = append(candidateIDs, c.ID)
				candidateTitles = append(candidateTitles, c.Title)
			}
		} else if e.ragRepo != nil && req.Domain != "" {
			docs, _, _ := e.ragRepo.ListDocuments(ctx, &request.ListDocumentsDto{
				Status: "ACTIVE",
				Domain: req.Domain,
				Limit:  50,
			})
			for _, doc := range docs {
				if doc.ID != req.NewDocID {
					candidateIDs = append(candidateIDs, doc.ID)
					candidateTitles = append(candidateTitles, doc.Title)
				}
			}
		}
		fallbackAction, fallbackNote := e.reviewSupersessionAction(ctx, req, ActionFullSupersede, firstOrEmpty(candidateIDs))
		if fallbackAction == ActionNone {
			return e.buildDefaultPlan(req, parsedArticles, ActionNone, "", "", fallbackNote, nil, nil, []string{req.ApplicableCohort}, "LOW"), nil
		}
		if len(candidateIDs) > 0 {
			risk := "MEDIUM"
			if len(candidateIDs) > 1 {
				risk = "HIGH"
			}
			action := fallbackAction
			plan := e.buildDefaultPlan(req, parsedArticles, action, candidateIDs[0], candidateTitles[0], "Bãi bỏ các quy chế đang hiệu lực cùng lĩnh vực", nil, nil, []string{req.ApplicableCohort}, risk)
			plan.BlastRadius.SupersededDocIDs = candidateIDs
			plan.BlastRadius.SupersededDocNames = candidateTitles
			plan.BlastRadius.RiskLevel = risk
			for i := 1; i < len(candidateIDs); i++ {
				plan.DiffItems = append(plan.DiffItems, models.TLUClawDiffItem{
					EntityType: "DOCUMENT",
					Action:     "SUPERSEDE",
					Key:        candidateIDs[i],
					Name:       candidateTitles[i],
					OldValue:   "ACTIVE",
					NewValue:   "SUPERSEDED",
					Note:       fmt.Sprintf("Bị bãi bỏ toàn phần bởi '%s'", req.Title),
				})
			}
			return plan, nil
		}
		return e.buildDefaultPlan(req, parsedArticles, ActionNone, "", "", "Không có nhà cung cấp LLM nào đang hoạt động", nil, nil, []string{req.ApplicableCohort}, "LOW"), nil
	}

	if len(req.Candidates) == 0 && e.ragRepo != nil {
		req.Candidates = e.retrieveCandidatesDataDriven(ctx, req.NewDocID, req.Title, req.DocType, req.Domain, req.PublishDate, nil)
	}

	systemPrompt := agent.PromptRegulatoryDissection(req.NewDocID)

	evidenceText := req.FullContent
	if evidenceText == "" {
		evidenceText = req.Content
	}
	clauses := extractReplacementClauses(evidenceText)
	clauseMatches := matchClausesToCandidates(clauses, req.Candidates, func(id string) string { return e.documentText(ctx, id) })
	pinned := make(map[string]bool, len(clauseMatches))
	for _, m := range clauseMatches {
		pinned[m.Candidate.ID] = true
	}
	req.Candidates = e.rankCandidates(ctx, req.NewDocID, req.Candidates, pinned)

	candSummary := ""
	if len(req.Candidates) > 0 {
		var sb strings.Builder
		sb.WriteString("\n\nCANDIDATE EARLIER ACTIVE REGULATIONS (FOUND IN DATABASE FOR COMPARISON):\n")
		for _, c := range req.Candidates {
			sb.WriteString(fmt.Sprintf("- ID: %s | Title: %s | Published: %s | Status: %s | Cohort: %s | AcademicYear: %s\n  Excerpt: %s\n", c.ID, c.Title, c.PublishDate, c.Status, c.ApplicableCohort, c.AcademicYear, e.candidateExcerpt(ctx, c.ID)))
		}
		sb.WriteString("\nEvaluate whether incoming document supersedes (FULL_SUPERSEDE), partially amends (PARTIAL_AMEND), or coexists with (COEXIST_BY_COHORT) any of the above candidates. Submit your decision using propose_granular_supersession.\n")
		sb.WriteString(formatEvidence(clauses, clauseMatches))
		candSummary = sb.String()
	}

	initialUserPrompt := fmt.Sprintf(`Analyze the following incoming document for supersession:
ID: %s
Title: %s
Code: %s
Type: %s
Domain: %s
Applicable Cohort: %s
Academic Year: %s

Parsed Incoming Articles Count: %d
Full Document Content:
%s%s`, req.NewDocID, req.Title, req.DocCode, req.DocType, req.Domain, req.ApplicableCohort, req.AcademicYear, len(parsedArticles), req.Content, candSummary)

	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: systemPrompt},
		{Role: llm.RoleUser, Content: initialUserPrompt},
	}

	maxTurns := 4
	inspected := map[string]bool{}
	var finalProposal *request.RegulatoryProposeGranularSupersessionArgs
	temp := 0.1

	for turn := 0; turn < maxTurns; turn++ {
		messages = compactDissectionHistory(ctx, messages)
		if turn == maxTurns-1 && finalProposal == nil {
			messages = append(messages, llm.Message{
				Role:    llm.RoleUser,
				Content: "Inspection phase concluded. You MUST now call propose_granular_supersession with your final decision (FULL_SUPERSEDE, PARTIAL_AMEND, COEXIST_BY_COHORT, or NONE).",
			})
		}

		chatReq := &llm.ChatRequest{
			Messages:    messages,
			Tools:       e.GetToolDefinitions(),
			Temperature: &temp,
		}

		resp, err := e.mgr.ChatWithChain(ctx, llm.ChainTypeAudit, chatReq)
		if err != nil || resp == nil {
			if err != nil {
				log.Warn().Err(err).Str("doc_id", req.NewDocID).Msg("Supersession audit LLM call failed")
			}
			if finalProposal == nil {
				return nil, fmt.Errorf("audit model unavailable for %s: %v", req.NewDocID, err)
			}
			break
		}

		messages = append(messages, resp.Message)

		if len(resp.Message.ToolCalls) == 0 {
			if finalProposal == nil {
				txt := strings.ToUpper(resp.Message.Content)
				if strings.Contains(txt, "FULL_SUPERSEDE") || strings.Contains(txt, "PARTIAL_AMEND") {
					for _, c := range req.Candidates {
						if strings.Contains(resp.Message.Content, c.ID) {
							act := ActionFullSupersede
							if strings.Contains(txt, "PARTIAL_AMEND") {
								act = ActionPartialAmend
							}
							finalProposal = &request.RegulatoryProposeGranularSupersessionArgs{
								ActionType:  string(act),
								TargetDocID: c.ID,
								Reason:      resp.Message.Content,
							}
							break
						}
					}
				}
			}
			break
		}

		for _, tc := range resp.Message.ToolCalls {
			if tc.Function.Name == "inspect_document_articles" {
				var args request.RegulatoryInspectDocumentArticlesArgs
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
				if args.DocumentID == "" && len(req.Candidates) > 0 {
					args.DocumentID = req.Candidates[0].ID
				}
				inspected[args.DocumentID] = true
			}
			toolOut, isProposal := e.executeToolCall(ctx, req, tc)
			if isProposal {
				var p request.RegulatoryProposeGranularSupersessionArgs
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &p); err == nil {
					if needsInspection(&p, inspected) && turn < maxTurns-1 {
						toolOut = `{"status":"REJECTED","error":"Call inspect_document_articles for the target document and quote the conflicting provisions of both documents in reason before proposing FULL_SUPERSEDE or PARTIAL_AMEND."}`
					} else {
						finalProposal = &p
					}
				}
			}
			messages = append(messages, llm.Message{
				Role:       llm.RoleTool,
				ToolCallID: tc.ID,
				Content:    toolOut,
			})
		}

		if finalProposal != nil {
			break
		}
	}

	if finalProposal != nil {
		actionType := SupersessionAction(strings.TrimSpace(finalProposal.ActionType))
		if actionType == "" {
			actionType = ActionNone
		}
		targetDocID := strings.TrimSpace(finalProposal.TargetDocID)
		reason := strings.TrimSpace(finalProposal.Reason)
		impact := strings.TrimSpace(finalProposal.ImpactSummary)
		risk := strings.TrimSpace(finalProposal.RiskLevel)
		if risk == "" {
			risk = "LOW"
		}
		inquiryQ := strings.TrimSpace(finalProposal.InquiryQuestion)
		superNums := finalProposal.SupersededArticleNumbers
		retainNums := finalProposal.RetainedArticleNumbers
		cohorts := finalProposal.AffectedCohorts
		if len(cohorts) == 0 && req.ApplicableCohort != "" {
			cohorts = []string{req.ApplicableCohort}
		}

		if targetDocID == req.NewDocID {
			targetDocID = ""
			actionType = ActionNone
		}
		if actionType != ActionNone && needsInspection(finalProposal, inspected) {
			reason = strings.TrimSpace(reason + " Ignored: the target document was never inspected, so no supersession is justified.")
			targetDocID = ""
			actionType = ActionNone
		}

		targetTitle := ""
		if targetDocID != "" && e.ragRepo != nil {
			if doc, err := e.ragRepo.GetDocumentByID(ctx, targetDocID); err == nil && doc != nil && doc.ID != req.NewDocID {
				targetTitle = doc.Title
			} else {
				targetDocID = ""
				actionType = ActionNone
			}
		}

		if reviewed, note := e.reviewSupersessionAction(ctx, req, actionType, targetDocID); reviewed != actionType {
			actionType = reviewed
			reason = strings.TrimSpace(reason + " " + note)
			if actionType == ActionNone {
				targetDocID, targetTitle = "", ""
			}
		}

		if actionType == ActionFullSupersede && e.dissectFn == nil && e.ragRepo != nil {
			if target, err := e.ragRepo.GetDocumentByID(ctx, targetDocID); err == nil {
				if verdict, why, ok := e.verifyFullSupersession(ctx, req, target); ok && verdict != ActionFullSupersede {
					actionType = verdict
					reason = strings.TrimSpace(reason + " Second opinion: " + why)
					if actionType == ActionNone {
						targetDocID, targetTitle = "", ""
					}
				}
			}
		}

		plan := e.buildDefaultPlan(req, parsedArticles, actionType, targetDocID, targetTitle, reason, superNums, retainNums, cohorts, risk)
		plan.ImpactSummary = impact
		plan.InquiryQuestion = inquiryQ
		return plan, nil
	}

	return e.buildDefaultPlan(req, parsedArticles, ActionNone, "", "", "Không phát hiện xung đột bãi bỏ quy chế", nil, nil, []string{req.ApplicableCohort}, "LOW"), nil
}

func (e *RegulatoryDissectionEngine) executeToolCall(ctx context.Context, req *DissectionRequest, tc llm.ToolCall) (string, bool) {
	switch tc.Function.Name {
	case "search_regulations_fts":
		var args request.RegulatorySearchRegulationsFTSArgs
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		if args.Limit <= 0 {
			args.Limit = 5
		}

		res := response.RegulatorySearchResultResponse{}

		if e.ragRepo != nil {
			docs, _, _ := e.ragRepo.ListDocuments(ctx, &request.ListDocumentsDto{
				Search: args.Query,
				Status: "ACTIVE",
				Cohort: args.Cohort,
				Domain: args.Domain,
				Limit:  args.Limit * 2,
			})
			for _, d := range docs {
				if d.ID == req.NewDocID {
					continue
				}
				if req.PublishDate != "" && d.PublishDate != "" && d.PublishDate > req.PublishDate {
					continue
				}
				res.Documents = append(res.Documents, d.ToResponse())
				if len(res.Documents) >= args.Limit {
					break
				}
			}

			ftsQuery := sanitizeFTS5Query(args.Query)
			var chunkIDs []string
			var err error
			if ftsQuery != "" {
				chunkIDs, err = e.ragRepo.SearchActiveChunkIDsFTS(ctx, ftsQuery, args.Cohort, args.Domain, args.Limit*2)
			}
			if err == nil && len(chunkIDs) > 0 {
				chunks, _ := e.ragRepo.GetChunksByIDs(ctx, chunkIDs)
				for _, ch := range chunks {
					if ch.DocumentID == req.NewDocID {
						continue
					}
					res.Chunks = append(res.Chunks, ch.ToResponse())
					if len(res.Chunks) >= args.Limit {
						break
					}
				}
			}
		}

		out, _ := json.Marshal(res)
		return string(out), false

	case "inspect_document_articles":
		var args request.RegulatoryInspectDocumentArticlesArgs
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		if args.DocumentID == "" && len(req.Candidates) > 0 {
			args.DocumentID = req.Candidates[0].ID
		}

		if e.ragRepo == nil {
			out, _ := json.Marshal(response.RegulatoryInspectDocumentArticlesResponse{DocumentID: args.DocumentID})
			return string(out), false
		}

		rules, err := e.ragRepo.ListRulesByDocumentID(ctx, args.DocumentID)
		if err != nil || len(rules) == 0 {
			chunks, _ := e.ragRepo.GetChunksByDocumentID(ctx, args.DocumentID)
			if len(chunks) > 5 {
				chunks = chunks[:5]
			}
			var chunkResponses []*response.DocumentChunkResponse
			for _, ch := range chunks {
				chunkResponses = append(chunkResponses, ch.ToResponse())
			}
			out, _ := json.Marshal(response.RegulatoryInspectDocumentArticlesResponse{
				DocumentID: args.DocumentID,
				Chunks:     chunkResponses,
			})
			return string(out), false
		}

		var ruleResponses []*response.DocumentRuleResponse
		for _, r := range rules {
			if args.ArticleNumber == "" || strings.Contains(strings.ToLower(r.ArticleNumber), strings.ToLower(args.ArticleNumber)) {
				ruleResponses = append(ruleResponses, r.ToResponse())
			}
		}
		out, _ := json.Marshal(response.RegulatoryInspectDocumentArticlesResponse{
			DocumentID: args.DocumentID,
			Rules:      ruleResponses,
		})
		return string(out), false

	case "compare_article_diff":
		var args request.RegulatoryCompareArticleDiffArgs
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		if args.OldDocumentID != "" {
			return structuralArticleDiff(e.documentText(ctx, args.OldDocumentID), req.Content, args.ArticleNumber), false
		}
		return structuralArticleDiff(args.OldArticleText, args.NewArticleText, args.ArticleNumber), false

	case "search_knowledge_graph":
		var args request.RegulatorySearchKnowledgeGraphArgs
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)

		if e.ragRepo == nil {
			out, _ := json.Marshal(response.RegulatorySearchKnowledgeGraphResponse{})
			return string(out), false
		}

		edges, _ := e.ragRepo.ListAllKGEdges(ctx)
		var matched []*response.KGEdgeResponse
		term := strings.ToLower(args.EntityName)
		for _, ed := range edges {
			if strings.Contains(strings.ToLower(ed.SourceNodeID), term) || strings.Contains(strings.ToLower(ed.TargetNodeID), term) {
				matched = append(matched, ed.ToResponse())
			}
		}
		out, _ := json.Marshal(response.RegulatorySearchKnowledgeGraphResponse{
			MatchedEdges: matched,
		})
		return string(out), false

	case "inspect_cohort_applicability":
		var args request.RegulatoryInspectCohortApplicabilityArgs
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)

		if e.ragRepo == nil {
			out, _ := json.Marshal(response.RegulatoryInspectCohortApplicabilityResponse{IsDisjoint: false})
			return string(out), false
		}

		d1, _ := e.ragRepo.GetDocumentByID(ctx, args.DocID1)
		d2, _ := e.ragRepo.GetDocumentByID(ctx, args.DocID2)
		isDisjoint := false
		if d1 != nil && d2 != nil && d1.ApplicableCohort != "ALL" && d2.ApplicableCohort != "ALL" && d1.ApplicableCohort != d2.ApplicableCohort {
			isDisjoint = true
		}
		res := response.RegulatoryInspectCohortApplicabilityResponse{
			IsDisjoint: isDisjoint,
		}
		if d1 != nil {
			res.Doc1 = d1.ToResponse()
		}
		if d2 != nil {
			res.Doc2 = d2.ToResponse()
		}
		out, _ := json.Marshal(res)
		return string(out), false

	case "propose_granular_supersession":
		out, _ := json.Marshal(response.RegulatoryProposeStatusResponse{
			Status: "PROPOSAL_RECEIVED",
		})
		return string(out), true

	default:
		return `{"error":"công cụ không được hỗ trợ"}`, false
	}
}

func (e *RegulatoryDissectionEngine) buildDefaultPlan(
	req *DissectionRequest,
	parsedArticles []supersede.ParsedArticle,
	action SupersessionAction,
	targetDocID, targetTitle, reason string,
	superNums, retainNums []string,
	cohorts []string,
	risk string,
) *GranularSupersessionPlan {
	var diffItems []models.TLUClawDiffItem

	diffItems = append(diffItems, models.TLUClawDiffItem{
		EntityType: "DOCUMENT",
		Action:     "INGEST",
		Key:        req.NewDocID,
		Name:       req.Title,
		OldValue:   "-",
		NewValue:   "ACTIVE",
		Note:       fmt.Sprintf("Type: %s | Domain: %s | Cohort: %s", req.DocType, req.Domain, req.ApplicableCohort),
	})

	for _, art := range parsedArticles {
		diffItems = append(diffItems, models.TLUClawDiffItem{
			EntityType: "ARTICLE",
			Action:     "CREATE",
			Key:        fmt.Sprintf("%s_%s", req.NewDocID, art.ArticleNumber),
			Name:       fmt.Sprintf("%s. %s", art.ArticleNumber, art.Title),
			OldValue:   "-",
			NewValue:   "ACTIVE",
			Note:       art.Chapter,
		})
	}

	var supersededArticles []SupersededArticleItem
	var retainedArticles []RetainedArticleItem
	var supersededIDs []string
	var supersededNames []string

	if targetDocID != "" && action != ActionNone && action != ActionCoexistByCohort {
		supersededIDs = append(supersededIDs, targetDocID)
		if targetTitle != "" {
			supersededNames = append(supersededNames, targetTitle)
		}

		if action == ActionFullSupersede {
			diffItems = append(diffItems, models.TLUClawDiffItem{
				EntityType: "DOCUMENT",
				Action:     "SUPERSEDE",
				Key:        targetDocID,
				Name:       targetTitle,
				OldValue:   "ACTIVE",
				NewValue:   "SUPERSEDED",
				Note:       fmt.Sprintf("Fully superseded by '%s'", req.Title),
			})

			diffItems = append(diffItems, models.TLUClawDiffItem{
				EntityType: "KG_EDGE",
				Action:     "CREATE",
				Key:        "SUPERSEDES",
				Name:       fmt.Sprintf("(doc:%s) -> (doc:%s)", req.NewDocID, targetDocID),
				OldValue:   "-",
				NewValue:   "SUPERSEDES",
				Note:       "Supersession link on Knowledge Graph",
			})
		} else if action == ActionPartialAmend {
			diffItems = append(diffItems, models.TLUClawDiffItem{
				EntityType: "DOCUMENT",
				Action:     "PARTIALLY_AMEND",
				Key:        targetDocID,
				Name:       targetTitle,
				OldValue:   "ACTIVE",
				NewValue:   "PARTIALLY_AMENDED",
				Note:       fmt.Sprintf("Partially amended by '%s'", req.Title),
			})

			diffItems = append(diffItems, models.TLUClawDiffItem{
				EntityType: "KG_EDGE",
				Action:     "CREATE",
				Key:        "PARTIALLY_AMENDS",
				Name:       fmt.Sprintf("(doc:%s) -> (doc:%s)", req.NewDocID, targetDocID),
				OldValue:   "-",
				NewValue:   "PARTIALLY_AMENDS",
				Note:       "Partial amendment link on Knowledge Graph",
			})

			for _, num := range superNums {
				item := SupersededArticleItem{
					RuleID:        fmt.Sprintf("%s_%s", targetDocID, num),
					ArticleNumber: num,
					Title:         num,
					Reason:        reason,
				}
				supersededArticles = append(supersededArticles, item)

				diffItems = append(diffItems, models.TLUClawDiffItem{
					EntityType: "ARTICLE",
					Action:     "SUPERSEDE",
					Key:        item.RuleID,
					Name:       num,
					OldValue:   "ACTIVE",
					NewValue:   "SUPERSEDED",
					Note:       fmt.Sprintf("Amended by incoming '%s'", req.Title),
				})
			}

			for _, num := range retainNums {
				item := RetainedArticleItem{
					RuleID:        fmt.Sprintf("%s_%s", targetDocID, num),
					ArticleNumber: num,
					Title:         num,
				}
				retainedArticles = append(retainedArticles, item)

				diffItems = append(diffItems, models.TLUClawDiffItem{
					EntityType: "ARTICLE",
					Action:     "RETAIN",
					Key:        item.RuleID,
					Name:       num,
					OldValue:   "ACTIVE",
					NewValue:   "ACTIVE",
					Note:       "Retains active legal force",
				})
			}
		}
	} else if action == ActionCoexistByCohort && targetDocID != "" {
		diffItems = append(diffItems, models.TLUClawDiffItem{
			EntityType: "DOCUMENT",
			Action:     "RETAIN",
			Key:        targetDocID,
			Name:       targetTitle,
			OldValue:   "ACTIVE",
			NewValue:   "ACTIVE",
			Note:       "Coexists in parallel for distinct cohort",
		})
	}

	impactSummary := reason
	if impactSummary == "" {
		if action == ActionFullSupersede {
			impactSummary = fmt.Sprintf("Fully supersedes previous document %s", targetDocID)
		} else if action == ActionPartialAmend {
			impactSummary = fmt.Sprintf("Partially amends %d articles in document %s while retaining unchanged articles", len(supersededArticles), targetDocID)
		} else if action == ActionCoexistByCohort {
			impactSummary = fmt.Sprintf("Coexists in parallel with document %s for target cohort %s", targetDocID, req.ApplicableCohort)
		} else {
			impactSummary = fmt.Sprintf("Ingests '%s' into knowledge base with no conflicts", req.Title)
		}
	}

	blastRadius := models.TLUClawBlastRadius{
		AffectedCohorts:    cohorts,
		SupersededDocIDs:   supersededIDs,
		SupersededDocNames: supersededNames,
		ImpactSummary:      impactSummary,
		RiskLevel:          risk,
	}

	return &GranularSupersessionPlan{
		ActionType:         action,
		TargetDocID:        targetDocID,
		TargetDocTitle:     targetTitle,
		Reason:             reason,
		SupersededArticles: supersededArticles,
		RetainedArticles:   retainedArticles,
		NewArticles:        parsedArticles,
		AffectedCohorts:    cohorts,
		ImpactSummary:      impactSummary,
		RiskLevel:          risk,
		DiffItems:          diffItems,
		BlastRadius:        blastRadius,
	}
}

// ApplyPlan executes the granular supersession modifications atomically across documents, rules, chunks, and graph.
func (e *RegulatoryDissectionEngine) WithRepositories(ragRepo repositories.RAGRepository, curriculumRepo repositories.CurriculumRepository) *RegulatoryDissectionEngine {
	clone := *e
	if ragRepo != nil {
		clone.ragRepo = ragRepo
	}
	if curriculumRepo != nil {
		clone.curriculumRepo = curriculumRepo
	}
	return &clone
}

func (e *RegulatoryDissectionEngine) ApplyPlan(ctx context.Context, plan *GranularSupersessionPlan, req *DissectionRequest) error {
	if plan == nil || req == nil || e.ragRepo == nil {
		return nil
	}

	existing, _ := e.ragRepo.GetDocumentByID(ctx, req.NewDocID)

	publishDate := req.PublishDate
	if publishDate == "" && existing != nil {
		publishDate = existing.PublishDate
	}
	if publishDate == "" {
		publishDate = time.Now().Format("2006-01-02")
	}

	filePath := req.FilePath
	if filePath == "" && existing != nil {
		filePath = existing.FilePath
	}

	priorityLevel := req.PriorityLevel
	if priorityLevel == 0 && existing != nil {
		priorityLevel = existing.PriorityLevel
	}
	if priorityLevel == 0 {
		priorityLevel = 50
	}

	validityTier := "OFFICIAL"
	if existing != nil && existing.ValidityTier != "" {
		validityTier = existing.ValidityTier
	}

	status := "ACTIVE"
	if existing != nil && existing.Status != "" {
		status = existing.Status
	}

	departmentCode := "DAOTAO"
	if existing != nil && existing.DepartmentCode != "" {
		departmentCode = existing.DepartmentCode
	}

	academicYear := req.AcademicYear
	if academicYear == "" && existing != nil {
		academicYear = existing.AcademicYear
	}

	cohort := req.ApplicableCohort
	if cohort == "" && existing != nil {
		cohort = existing.ApplicableCohort
	}

	createdAt := time.Now().Format(time.RFC3339)
	if existing != nil && existing.CreatedAt != "" {
		createdAt = existing.CreatedAt
	}
	now := time.Now().Format(time.RFC3339)

	docModel := &models.Document{
		ID:               req.NewDocID,
		DocCode:          req.DocCode,
		Type:             req.DocType,
		Title:            req.Title,
		Domain:           req.Domain,
		ValidityTier:     validityTier,
		Status:           status,
		DepartmentCode:   departmentCode,
		PublishDate:      publishDate,
		AcademicYear:     academicYear,
		Semester:         "1",
		ApplicableCohort: cohort,
		PriorityLevel:    priorityLevel,
		FilePath:         filePath,
		CharCount:        len(req.Content),
		CreatedAt:        createdAt,
		UpdatedAt:        now,
	}
	_, err := e.ragRepo.UpsertDocument(ctx, docModel)
	if err != nil {
		return err
	}

	existingChunks, _ := e.ragRepo.GetChunksByDocumentID(ctx, req.NewDocID)
	if len(existingChunks) == 0 && len(plan.NewArticles) > 0 {
		var chunks []*models.DocumentChunk
		for _, art := range plan.NewArticles {
			ruleID := fmt.Sprintf("rule_%s_%d", req.NewDocID, art.Index)
			ruleModel := &models.DocumentRule{
				ID:            ruleID,
				DocumentID:    req.NewDocID,
				RuleIndex:     art.Index,
				Chapter:       art.Chapter,
				ArticleNumber: art.ArticleNumber,
				Title:         art.Title,
				Content:       art.Content,
				Status:        "ACTIVE",
			}
			_, _ = e.ragRepo.UpsertRule(ctx, ruleModel)

			chkID := fmt.Sprintf("chk_%s_%d", req.NewDocID, art.Index)
			chk := &models.DocumentChunk{
				ID:               chkID,
				DocumentID:       req.NewDocID,
				RuleID:           ruleID,
				ChunkIndex:       art.Index,
				ChunkType:        "CONTENT_CHUNK",
				Content:          fmt.Sprintf("%s. %s\n%s", art.ArticleNumber, art.Title, art.Content),
				Status:           "ACTIVE",
				Domain:           req.Domain,
				ApplicableCohort: cohort,
				AcademicYear:     academicYear,
				PriorityLevel:    priorityLevel,
			}
			if e.mgr != nil {
				embs, _ := e.mgr.ExecuteEmbedding(ctx, []string{chk.Content})
				if len(embs) > 0 {
					chk.Embedding = embs[0]
				}
			}
			chunks = append(chunks, chk)
		}

		if len(chunks) > 0 {
			_ = e.ragRepo.UpsertChunksBatch(ctx, chunks)
		}
	}

	docNode := &models.KGNode{
		ID:       "doc:" + req.NewDocID,
		NodeType: "DOCUMENT",
		Name:     req.Title,
		Code:     req.NewDocID,
		Status:   "ACTIVE",
	}
	_, _ = e.ragRepo.UpsertKGNode(ctx, docNode)

	if req.ApplicableCohort != "" && req.ApplicableCohort != "ALL" {
		cohortNode := &models.KGNode{
			ID:       "cohort:" + req.ApplicableCohort,
			NodeType: "COHORT",
			Name:     "Cohort " + req.ApplicableCohort,
			Code:     req.ApplicableCohort,
			Status:   "ACTIVE",
		}
		_, _ = e.ragRepo.UpsertKGNode(ctx, cohortNode)
		_, _ = e.ragRepo.UpsertKGEdge(ctx, &models.KGEdge{
			ID:           fmt.Sprintf(constants.PrefixEdgeCohort+"%s:%s", req.NewDocID, req.ApplicableCohort),
			SourceNodeID: docNode.ID,
			TargetNodeID: cohortNode.ID,
			RelationType: constants.KGRelationAppliesToCohort,
			Weight:       1.0,
			Status:       constants.DocStatusActive,
		})
	}

	if plan.TargetDocID != "" {
		if plan.ActionType == ActionFullSupersede {
			_ = e.ragRepo.UpdateDocumentStatus(ctx, plan.TargetDocID, constants.DocStatusSuperseded, req.NewDocID)
			_ = e.ragRepo.UpdateChunksByDocIDStatus(ctx, plan.TargetDocID, constants.DocStatusSuperseded)

			oldRules, _ := e.ragRepo.ListRulesByDocumentID(ctx, plan.TargetDocID)
			for _, r := range oldRules {
				_ = e.ragRepo.UpdateRuleStatus(ctx, r.ID, constants.DocStatusSuperseded, req.NewDocID)
			}

			_ = e.ragRepo.RecordSupersessionLog(ctx, req.NewDocID, plan.TargetDocID, constants.RuleTypeDocumentReplacement, plan.Reason)
			_, _ = e.ragRepo.UpsertKGEdge(ctx, &models.KGEdge{
				ID:           fmt.Sprintf(constants.PrefixEdgeSuper+"%s:%s", req.NewDocID, plan.TargetDocID),
				SourceNodeID: constants.PrefixDocNode + req.NewDocID,
				TargetNodeID: constants.PrefixDocNode + plan.TargetDocID,
				RelationType: constants.KGRelationSupersedes,
				Weight:       1.0,
				Status:       constants.DocStatusActive,
			})
		} else if plan.ActionType == ActionPartialAmend {
			_ = e.ragRepo.UpdateDocumentStatus(ctx, plan.TargetDocID, constants.DocStatusPartiallyAmended, req.NewDocID)

			for _, superArt := range plan.SupersededArticles {
				if superArt.RuleID != "" {
					_ = e.ragRepo.UpdateRuleStatus(ctx, superArt.RuleID, constants.DocStatusSuperseded, req.NewDocID)
					_ = e.ragRepo.UpdateChunksByRuleIDStatus(ctx, superArt.RuleID, constants.DocStatusSuperseded)
				}
			}

			_ = e.ragRepo.RecordSupersessionLog(ctx, req.NewDocID, plan.TargetDocID, constants.RuleTypePartialAmendment, plan.Reason)
			_, _ = e.ragRepo.UpsertKGEdge(ctx, &models.KGEdge{
				ID:           fmt.Sprintf(constants.PrefixEdgeAmend+"%s:%s", req.NewDocID, plan.TargetDocID),
				SourceNodeID: constants.PrefixDocNode + req.NewDocID,
				TargetNodeID: constants.PrefixDocNode + plan.TargetDocID,
				RelationType: constants.KGRelationPartiallyAmends,
				Weight:       1.0,
				Status:       constants.DocStatusActive,
			})
		}
	}

	e.ragRepo.InvalidateDocumentCache(ctx, req.NewDocID)
	if plan.TargetDocID != "" {
		e.ragRepo.InvalidateDocumentCache(ctx, plan.TargetDocID)
	}

	return nil
}

var (
	cohortRangeRe  = regexp.MustCompile(`(?i)(?:\b(?:k|kh\x{00f3}a|kho\x{00e1})\s*(\d{2})\s*(?:[-–—]|\x{0111}\x{1ebf}n|t\x{1edb}i)\s*(?:k|kh\x{00f3}a|kho\x{00e1})?\s*(\d{2})\b)`)
	cohortLeRe     = regexp.MustCompile(`(?i)(?:(?:kh\x{00f3}a|kho\x{00e1}|k)\s*(?:<=|≤|<)\s*(\d{2})|(?:<=|≤)\s*(?:kh\x{00f3}a|kho\x{00e1}|k)\s*(\d{2})|(?:kh\x{00f3}a|kho\x{00e1}|k)\s*(\d{2})\s*tr\x{1edf}\s*v\x{1ec1}\s*tr\x{01b0}\x{1edb}c)`)
	cohortGeRe     = regexp.MustCompile(`(?i)(?:(?:kh\x{00f3}a|kho\x{00e1}|k)\s*(?:>=|≥|>)\s*(\d{2})|(?:>=|≥)\s*(?:kh\x{00f3}a|kho\x{00e1}|k)\s*(\d{2})|(?:t\x{1eeb}\s*)?(?:kh\x{00f3}a|kho\x{00e1}|k)\s*(\d{2})\s*tr\x{1edf}\s*(?:l\x{00ea}n|\x{0111}i))`)
	cohortSingleRe = regexp.MustCompile(`(?i)\b(?:k|kh\x{00f3}a|kho\x{00e1}|nj)\s*(\d{2})\b`)
	acadYearPairRe = regexp.MustCompile(`(20[12]\d)\s*[-–—/]\s*(20[12]\d)`)
	singleYearRe   = regexp.MustCompile(`(?i)n\x{0103}m(?:\s+h\x{1ecd}c)?\s+(20[12]\d)\b`)
)

func extractCrawledDocumentMetadata(docID, title, category, publishDate string) (string, string, string, string) {
	docType := category
	if docType == "" {
		docType = constants.DocTypeNotice
	}
	domain := constants.DomainTraining
	if category == constants.DocTypeNews {
		docType = constants.DocTypeNews
		domain = constants.DomainNews
	} else if strings.HasPrefix(docID, "DOC_STATIC") {
		docType = constants.DocTypeGuideline
		domain = constants.DomainRegulation
	}

	cohort := extractNormalizedCohortList(title)
	if cohort == "" {
		cohort = constants.DocCohortAll
	}

	academicYear := constants.CurrentAcademicYear()
	if m := acadYearPairRe.FindStringSubmatch(title); len(m) > 2 {
		y1, _ := strconv.Atoi(m[1])
		y2, _ := strconv.Atoi(m[2])
		if y2 == y1+1 {
			academicYear = fmt.Sprintf("%d-%d", y1, y2)
		}
	} else if len(publishDate) >= 7 {
		y, err1 := strconv.Atoi(publishDate[:4])
		mo, err2 := strconv.Atoi(publishDate[5:7])
		if err1 == nil && err2 == nil && y >= 2000 && y <= 2100 {
			if mo >= 8 {
				academicYear = fmt.Sprintf("%d-%d", y, y+1)
			} else {
				academicYear = fmt.Sprintf("%d-%d", y-1, y)
			}
		}
	} else if m := singleYearRe.FindStringSubmatch(title); len(m) > 1 {
		if y, err := strconv.Atoi(m[1]); err == nil {
			academicYear = fmt.Sprintf("%d-%d", y, y+1)
		}
	}

	return docType, domain, cohort, academicYear
}

// AnalyzeAndNotify serves Entrypoint 1 (Worker Crawlers) by staging supersession proposals and alerting advisors.
func (e *RegulatoryDissectionEngine) AnalyzeAndNotify(ctx context.Context, newDocID, newDocTitle, newDocContent string) (*GranularSupersessionPlan, error) {
	docType, domain, cohort, academicYear := extractCrawledDocumentMetadata(newDocID, newDocTitle, constants.DocTypeNotice, "")
	if e.ragRepo != nil {
		if existing, err := e.ragRepo.GetDocumentByID(ctx, newDocID); err == nil && existing != nil {
			if existing.Type != "" {
				docType = existing.Type
			}
			if existing.Domain != "" {
				domain = existing.Domain
			}
			if existing.ApplicableCohort != "" {
				cohort = existing.ApplicableCohort
			}
			if existing.AcademicYear != "" {
				academicYear = existing.AcademicYear
			}
		}
	}

	req := &DissectionRequest{
		NewDocID:         newDocID,
		Title:            newDocTitle,
		DocCode:          newDocID,
		DocType:          docType,
		Domain:           domain,
		ApplicableCohort: cohort,
		AcademicYear:     academicYear,
		Content:          newDocContent,
	}

	plan, err := e.ExecuteDissectionLoop(ctx, req)
	if err != nil || plan == nil {
		return nil, err
	}

	if plan.ActionType == ActionNone || plan.TargetDocID == "" {
		return plan, nil
	}

	inquiryID := "inq-" + uuid.Must(uuid.NewV7()).String()[:8]
	now := time.Now()
	if e.q != nil {
		inqQuestion := plan.InquiryQuestion
		if inqQuestion == "" {
			inqQuestion = fmt.Sprintf("Academic regulation change confirmation: %s", plan.ImpactSummary)
		}

		_, _ = e.q.CreateInquiry(ctx, sqlc.CreateInquiryParams{
			ID:             inquiryID,
			UserID:         "system",
			ConversationID: sql.NullString{},
			StudentName:    "Automated Audit Agent",
			StudentCode:    "AI_AGENT",
			Question:       inqQuestion,
			Context:        fmt.Sprintf("New document: %s (%s). Impact: %s", newDocTitle, newDocID, plan.ImpactSummary),
			CreatedAt:      now,
			UpdatedAt:      now,
		})

		advisorRole, err := e.q.GetRoleByName(ctx, string(constants.RoleTypeAdvisor))
		var recipientIDs []string
		if err == nil {
			recipientIDs, _ = e.q.SearchUserIDs(ctx, sqlc.SearchUserIDsParams{
				RoleID: advisorRole.ID,
				Limit:  20,
				Offset: 0,
			})
		}
		if len(recipientIDs) == 0 {
			adminRole, err := e.q.GetRoleByName(ctx, string(constants.RoleTypeAdmin))
			if err == nil {
				recipientIDs, _ = e.q.SearchUserIDs(ctx, sqlc.SearchUserIDsParams{
					RoleID: adminRole.ID,
					Limit:  20,
					Offset: 0,
				})
			}
		}

		for _, recID := range recipientIDs {
			_, _ = e.q.CreateNotification(ctx, sqlc.CreateNotificationParams{
				ID:        "notif-" + uuid.Must(uuid.NewV7()).String()[:8],
				UserID:    recID,
				InquiryID: convert.StringToNullString(inquiryID),
				Title:     "Academic regulation change confirmation",
				Content:   fmt.Sprintf("Document %s may impact document %s: %s", newDocTitle, plan.TargetDocID, plan.ImpactSummary),
				Type:      constants.NotifTypeSupersededProposed,
				CreatedAt: now,
			})
		}
	}

	return plan, nil
}

// AuditInquiryOutdatedness serves Entrypoint 3 (Advisory Inquiries) to dynamically assess if a Q&A advice is invalidated.
func (e *RegulatoryDissectionEngine) AuditInquiryOutdatedness(ctx context.Context, doc *models.Document, inq *models.AdvisoryInquiry) (bool, string, error) {
	if doc == nil || inq == nil {
		return false, "", nil
	}

	if e.ragRepo == nil || e.mgr == nil {
		return false, "", nil
	}

	ftsQuery := sanitizeFTS5Query(inq.Question)
	if ftsQuery == "" {
		return false, "", nil
	}
	chunkIDs, err := e.ragRepo.SearchActiveChunkIDsFTS(ctx, ftsQuery, "", doc.Domain, 5)
	if err != nil || len(chunkIDs) == 0 {
		return false, "", nil
	}

	chunks, err := e.ragRepo.GetChunksByIDs(ctx, chunkIDs)
	if err != nil || len(chunks) == 0 {
		return false, "", nil
	}

	var relevantSnippets []string
	for _, chk := range chunks {
		if chk.DocumentID == doc.ID {
			relevantSnippets = append(relevantSnippets, chk.Content)
		}
	}

	if len(relevantSnippets) == 0 {
		return false, "", nil
	}

	systemPrompt := agent.PromptRegulatoryAuditInquiry()

	userPrompt := fmt.Sprintf(`New Document:
Title: %s
Code: %s
Domain: %s
Relevant Clauses:
%s

Prior Student Inquiry & Advice:
Student Question: %s
Advisor Reply: %s`, doc.Title, doc.DocCode, doc.Domain, strings.Join(relevantSnippets, "\n---\n"), inq.Question, inq.TeacherReply)

	resp, err := e.mgr.ChatWithChain(ctx, llm.ChainTypeAudit, &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: systemPrompt},
			{Role: llm.RoleUser, Content: userPrompt},
		},
		JSONMode: true,
	})
	if err != nil || resp == nil {
		return false, "", err
	}

	var out response.AuditInquiryOutdatednessResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp.Message.Content)), &out); err == nil {
		return out.IsOutdated, out.Reason, nil
	}

	return false, "", nil
}

// SupersessionLineageMemory maintains active transitive supersession maps across a batch audit.
type SupersessionLineageMemory struct {
	mu           sync.RWMutex
	DocLineage   map[string]string
	RuleLineage  map[string]string
	ActiveDocIDs map[string]bool
}

// NewSupersessionLineageMemory initializes an empty lineage memory store.
func NewSupersessionLineageMemory() *SupersessionLineageMemory {
	return &SupersessionLineageMemory{
		DocLineage:   make(map[string]string),
		RuleLineage:  make(map[string]string),
		ActiveDocIDs: make(map[string]bool),
	}
}

// RecordSupersession tracks a supersession link and resolves transitive chains.
func (m *SupersessionLineageMemory) RecordSupersession(oldDocID, newDocID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DocLineage[oldDocID] = newDocID
	delete(m.ActiveDocIDs, oldDocID)
	m.ActiveDocIDs[newDocID] = true

	for doc, target := range m.DocLineage {
		if target == oldDocID {
			m.DocLineage[doc] = newDocID
		}
	}
}

// ResolveTerminalActiveID returns the latest superseding document ID, or original if active.
func (m *SupersessionLineageMemory) ResolveTerminalActiveID(docID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	curr := docID
	for {
		if next, ok := m.DocLineage[curr]; ok && next != "" {
			curr = next
		} else {
			break
		}
	}
	return curr
}

// IsSuperseded checks whether a document ID has already been superseded by a newer regulation.
func (m *SupersessionLineageMemory) IsSuperseded(docID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.DocLineage[docID]
	return ok
}

func (e *RegulatoryDissectionEngine) resolveMajorFromDB(ctx context.Context, text string, majorRows []sqlc.ListAllMajorMetadataRow) string {
	if text == "" {
		return ""
	}
	if len(majorRows) == 0 && e.curriculumRepo != nil {
		majorRows, _ = e.curriculumRepo.ListAllMajorMetadata(ctx)
	}
	lowerText := strings.ToLower(text)
	bestCode := ""
	bestLen := 0
	for _, m := range majorRows {
		nameLower := strings.ToLower(strings.TrimSpace(m.MajorName))
		if nameLower != "" && strings.Contains(lowerText, nameLower) && len(nameLower) > bestLen {
			bestLen = len(nameLower)
			if m.MajorCode != "" {
				bestCode = m.MajorCode
			} else {
				bestCode = m.MajorName
			}
		}
		if m.Aliases != "" {
			for _, al := range strings.Split(m.Aliases, ",") {
				alTrim := strings.ToLower(strings.TrimSpace(al))
				if len([]rune(alTrim)) >= 4 && strings.Contains(lowerText, alTrim) && len(alTrim) > bestLen {
					bestLen = len(alTrim)
					if m.MajorCode != "" {
						bestCode = m.MajorCode
					} else {
						bestCode = m.MajorName
					}
				}
			}
		}
	}
	return bestCode
}

func (e *RegulatoryDissectionEngine) retrieveCandidatesDataDriven(
	ctx context.Context,
	newDocID, title, docType, domain, publishDate string,
	priorDocs []*models.Document,
) []*models.Document {
	var candidates []*models.Document
	seen := map[string]bool{newDocID: true}

	var majorRows []sqlc.ListAllMajorMetadataRow
	if e.curriculumRepo != nil {
		majorRows, _ = e.curriculumRepo.ListAllMajorMetadata(ctx)
	}
	docMajor := e.resolveMajorFromDB(ctx, title, majorRows)

	for i := len(priorDocs) - 1; i >= 0; i-- {
		cand := priorDocs[i]
		if seen[cand.ID] || cand.Status == constants.DocStatusSuperseded {
			continue
		}
		if cand.Domain != domain && cand.Type != docType {
			continue
		}
		if docType == constants.DocTypeCurriculum || cand.Type == constants.DocTypeCurriculum {
			if cand.Type != docType {
				continue
			}
			candMajor := e.resolveMajorFromDB(ctx, cand.Title, majorRows)
			if docMajor != candMajor {
				continue
			}
		}
		seen[cand.ID] = true
		candidates = append(candidates, cand)
		if len(candidates) >= 6 {
			break
		}
	}

	if len(candidates) < 8 && e.ragRepo != nil && title != "" {
		ftsQuery := sanitizeFTS5Query(title)
		if ftsQuery != "" {
			chunkIDs, err := e.ragRepo.SearchActiveChunkIDsFTS(ctx, ftsQuery, "", domain, 20)
			if err == nil && len(chunkIDs) > 0 {
				chunks, _ := e.ragRepo.GetChunksByIDs(ctx, chunkIDs)
				var docIDs []string
				for _, ch := range chunks {
					if !seen[ch.DocumentID] {
						seen[ch.DocumentID] = true
						docIDs = append(docIDs, ch.DocumentID)
					}
				}
				if len(docIDs) > 0 {
					docs, _ := e.ragRepo.GetDocumentsByIDs(ctx, docIDs)
					for _, d := range docs {
						if d == nil || d.Status == constants.DocStatusSuperseded {
							continue
						}
						if publishDate != "" && d.PublishDate != "" && d.PublishDate > publishDate {
							continue
						}
						if d.Type == constants.DocTypeNotice || d.Type == constants.DocTypeNews || d.Type == constants.DocTypeGuideline {
							continue
						}
						if docType == constants.DocTypeCurriculum || d.Type == constants.DocTypeCurriculum {
							if d.Type != docType {
								continue
							}
							candMajor := e.resolveMajorFromDB(ctx, d.Title, majorRows)
							if docMajor != candMajor {
								continue
							}
						}
						candidates = append(candidates, d)
						if len(candidates) >= 8 {
							break
						}
					}
				}
			}
		}
	}

	return candidates
}

func (e *RegulatoryDissectionEngine) auditBucketKey(ctx context.Context, doc *models.Document, majorRows []sqlc.ListAllMajorMetadataRow) string {
	if doc.Domain == constants.DomainRegulation && (doc.Type == constants.DocTypeRegulation || doc.Type == constants.DocTypeRule) {
		return constants.DomainRegulation + ":REG_RULE"
	}
	if doc.Type == constants.DocTypeCurriculum {
		return "CTDT:" + e.resolveMajorFromDB(ctx, doc.Title, majorRows)
	}
	return doc.Domain + ":" + doc.Type
}

func (e *RegulatoryDissectionEngine) auditBatchSizeOrDefault() int {
	if e.auditBatchSize > 0 {
		return e.auditBatchSize
	}
	if n := config.GetIntConfigWithDefault("AUDIT_BATCH_SIZE", DefaultAuditBatchSize); n > 0 {
		return n
	}
	return DefaultAuditBatchSize
}

func (e *RegulatoryDissectionEngine) dissectDocument(ctx context.Context, req *DissectionRequest) (*GranularSupersessionPlan, error) {
	if e.dissectFn != nil {
		return e.dissectFn(ctx, req)
	}
	return e.ExecuteDissectionLoop(ctx, req)
}

// AuditCorpusChronological runs one batch per call and persists its cursor so interrupted audits resume.
func (e *RegulatoryDissectionEngine) AuditCorpusChronological(ctx context.Context, onProgress func(cur, total int64)) error {
	if e.ragRepo == nil {
		return fmt.Errorf("rag repo unavailable for audit")
	}

	docs, err := e.ragRepo.ListAllRegulationsChronological(ctx)
	if err != nil {
		return fmt.Errorf("failed to list regulations: %w", err)
	}
	total := int64(len(docs))

	run, err := e.ragRepo.GetRunningAuditRun(ctx)
	if err != nil {
		return fmt.Errorf("failed to load running audit run: %w", err)
	}

	if run == nil {
		if err := e.ragRepo.ResetRegulationSupersessions(ctx); err != nil {
			return fmt.Errorf("failed to reset supersession state: %w", err)
		}
		if err := e.ragRepo.ClearAuditFingerprints(ctx); err != nil {
			return fmt.Errorf("failed to clear audit fingerprints: %w", err)
		}
		run, err = e.ragRepo.CreateAuditRun(ctx, uuid.NewString(), "", total)
		if err != nil {
			return fmt.Errorf("failed to create audit run: %w", err)
		}
		log.Info().Str("run_id", run.ID).Int64("total", total).Msg("Supersession audit run started (supersession state reset once)")
	} else {
		log.Info().
			Str("run_id", run.ID).
			Str("cursor", run.CursorDocID).
			Int64("processed", run.Processed).
			Int64("total", total).
			Msg("Resuming supersession audit run")
	}

	if total == 0 {
		return e.finishFullRun(ctx, run.ID, docs)
	}

	startIdx := 0
	if run.CursorDocID != "" {
		for i, doc := range docs {
			if doc.ID == run.CursorDocID {
				startIdx = i + 1
				break
			}
		}
	}
	if startIdx > len(docs) {
		startIdx = len(docs)
	}
	processed := int64(startIdx)
	if err := e.ragRepo.UpdateAuditRunProgress(ctx, run.ID, run.CursorDocID, processed, total); err != nil {
		log.Warn().Err(err).Str("run_id", run.ID).Msg("Failed to persist audit run progress")
	}
	if onProgress != nil {
		onProgress(processed, total)
	}

	memory := NewSupersessionLineageMemory()
	for _, doc := range docs {
		if doc.Status == constants.DocStatusActive {
			memory.ActiveDocIDs[doc.ID] = true
		} else if doc.Status == constants.DocStatusSuperseded && doc.SupersededByID != "" {
			memory.RecordSupersession(doc.ID, doc.SupersededByID)
		}
	}

	var majorRows []sqlc.ListAllMajorMetadataRow
	if e.curriculumRepo != nil {
		majorRows, _ = e.curriculumRepo.ListAllMajorMetadata(ctx)
	}

	buckets := make(map[string][]*models.Document, len(docs))
	bucketKeyByDoc := make(map[string]string, len(docs))
	for i, doc := range docs {
		key := e.auditBucketKey(ctx, doc, majorRows)
		bucketKeyByDoc[doc.ID] = key
		if i < startIdx {
			buckets[key] = append(buckets[key], doc)
		}
	}

	batchSize := e.auditBatchSizeOrDefault()

	startedAt := time.Now()
	deadline, hasDeadline := ctx.Deadline()
	var timeBudget time.Duration
	if hasDeadline {
		timeBudget = time.Until(deadline)
	}
	budgetExhausted := func() bool {
		if !hasDeadline {
			return false
		}
		if timeBudget <= 0 {
			return true
		}
		return time.Since(startedAt) >= time.Duration(float64(timeBudget)*0.8)
	}

	var applyMu sync.Mutex
	sem := make(chan struct{}, auditConcurrency)

	if startIdx >= len(docs) {
		return e.finishFullRun(ctx, run.ID, docs)
	}

	if budgetExhausted() || ctx.Err() != nil {
		log.Info().
			Str("run_id", run.ID).
			Int64("processed", processed).
			Int64("total", total).
			Msg("Supersession audit paused before batch; will resume on next tick")
		return nil
	}

	windowStart := startIdx
	windowEnd := windowStart + batchSize
	if windowEnd > len(docs) {
		windowEnd = len(docs)
	}
	window := docs[windowStart:windowEnd]

	grouped := make(map[string][]*models.Document, len(window))
	groupOrder := make([]string, 0, len(window))
	windowIdx := make(map[string]int, len(window))
	for i, doc := range window {
		key := bucketKeyByDoc[doc.ID]
		if _, ok := grouped[key]; !ok {
			groupOrder = append(groupOrder, key)
		}
		grouped[key] = append(grouped[key], doc)
		windowIdx[doc.ID] = i
	}

	windowDone := make([]atomic.Bool, len(window))
	var progressMu sync.Mutex
	lastPersisted := 0
	persistContiguous := func() {
		progressMu.Lock()
		defer progressMu.Unlock()
		advance := lastPersisted
		for advance < len(window) && windowDone[advance].Load() {
			advance++
		}
		if advance == lastPersisted {
			return
		}
		lastPersisted = advance
		cur := window[advance-1].ID
		p := int64(windowStart + advance)
		if err := e.ragRepo.UpdateAuditRunProgress(ctx, run.ID, cur, p, total); err != nil {
			log.Warn().Err(err).Str("run_id", run.ID).Msg("Failed to persist audit run cursor")
		}
		if onProgress != nil {
			onProgress(p, total)
		}
	}

	var wg sync.WaitGroup
	for _, key := range groupOrder {
		groupDocs := grouped[key]
		bucketKey := key
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, doc := range groupDocs {
				if budgetExhausted() || ctx.Err() != nil {
					return
				}
				select {
				case <-ctx.Done():
					return
				case sem <- struct{}{}:
				}
				e.auditOneDocument(ctx, doc, buckets[bucketKey], memory, &applyMu, false)
				<-sem
				buckets[bucketKey] = append(buckets[bucketKey], doc)
				windowDone[windowIdx[doc.ID]].Store(true)
				persistContiguous()
			}
		}()
	}
	wg.Wait()

	progressMu.Lock()
	advanced := lastPersisted
	progressMu.Unlock()

	if advanced < len(window) {
		log.Info().
			Str("run_id", run.ID).
			Int64("processed", processed).
			Int64("total", total).
			Msg("Supersession audit paused mid-batch; will resume on next tick")
		return nil
	}

	startIdx += advanced
	processed = int64(startIdx)

	if startIdx < len(docs) {
		log.Info().
			Str("run_id", run.ID).
			Int64("processed", processed).
			Int64("total", total).
			Msg("Supersession audit batch finished; next tick resumes from cursor")
		return nil
	}

	log.Info().Str("run_id", run.ID).Int64("total", total).Msg("Supersession audit run finished processing all documents")
	return e.finishFullRun(ctx, run.ID, docs)
}

func (e *RegulatoryDissectionEngine) completeAuditRun(ctx context.Context, runID string) error {
	if err := e.ragRepo.CompleteAuditRun(ctx, runID); err != nil {
		return fmt.Errorf("failed to complete audit run: %w", err)
	}
	if latest, err := e.ragRepo.GetLatestAuditRun(ctx); err == nil && latest != nil {
		log.Info().
			Str("run_id", latest.ID).
			Int64("processed", latest.Processed).
			Int64("total", latest.TotalDocs).
			Msg("Supersession audit run completed")
	}
	return nil
}

func (e *RegulatoryDissectionEngine) auditOneDocument(ctx context.Context, doc *models.Document, visited []*models.Document, memory *SupersessionLineageMemory, applyMu *sync.Mutex, recheck bool) {
	if memory.IsSuperseded(doc.ID) {
		return
	}
	if !recheck {
		if logged, err := e.ragRepo.DocumentHasAuditLog(ctx, doc.ID); err == nil && logged {
			return
		}
	}

	chunks, _ := e.ragRepo.GetChunksByDocumentID(ctx, doc.ID)
	chunkTexts := make([]string, 0, len(chunks))
	for _, ch := range chunks {
		chunkTexts = append(chunkTexts, ch.Content)
	}
	if isTitleOnlyContent(strings.Join(chunkTexts, "\n"), doc.Title) {
		return
	}

	var activeVisited []*models.Document
	for _, v := range visited {
		if !memory.IsSuperseded(v.ID) {
			activeVisited = append(activeVisited, v)
		}
	}

	candidates := e.retrieveCandidatesDataDriven(ctx, doc.ID, doc.Title, doc.Type, doc.Domain, doc.PublishDate, activeVisited)
	if len(candidates) == 0 {
		return
	}

	var contentBuilder strings.Builder
	for ci, ch := range chunks {
		if ci > 0 {
			contentBuilder.WriteString("\n\n")
		}
		contentBuilder.WriteString(ch.Content)
		if contentBuilder.Len() > 8000 {
			break
		}
	}
	fullContent := contentBuilder.String()
	if strings.TrimSpace(fullContent) == "" {
		fullContent = doc.Title
	}

	req := &DissectionRequest{
		NewDocID:         doc.ID,
		Title:            doc.Title,
		DocCode:          doc.DocCode,
		DocType:          doc.Type,
		Domain:           doc.Domain,
		ApplicableCohort: doc.ApplicableCohort,
		AcademicYear:     doc.AcademicYear,
		Content:          fullContent,
		FullContent:      strings.Join(chunkTexts, "\n"),
		PublishDate:      doc.PublishDate,
		FilePath:         doc.FilePath,
		PriorityLevel:    doc.PriorityLevel,
		Candidates:       candidates,
	}

	docCtx, cancelDoc := context.WithTimeout(ctx, auditDocTimeout)
	plan, err := e.dissectDocument(docCtx, req)
	cancelDoc()
	if err != nil {
		log.Warn().Err(err).Str("doc_id", doc.ID).Msg("Supersession dissection failed for document")
		e.auditFailed.Store(doc.ID, true)
		return
	}
	if plan == nil || plan.TargetDocID == "" || plan.TargetDocID == doc.ID {
		return
	}
	if plan.ActionType != ActionFullSupersede && plan.ActionType != ActionPartialAmend {
		return
	}
	if exists, err := e.ragRepo.AuditLogPairExists(ctx, doc.ID, plan.TargetDocID); err == nil && exists {
		return
	}

	log.Info().
		Str("action", string(plan.ActionType)).
		Str("new_doc_id", doc.ID).
		Str("new_title", doc.Title).
		Str("target_doc_id", plan.TargetDocID).
		Str("target_title", plan.TargetDocTitle).
		Str("reason", plan.Reason).
		Msg("Regulatory supersession applied")

	applyMu.Lock()
	applyErr := e.ApplyPlan(ctx, plan, req)
	applyMu.Unlock()
	if applyErr != nil {
		log.Error().Err(applyErr).Str("new_doc_id", doc.ID).Str("target_doc_id", plan.TargetDocID).Msg("Failed to apply supersession plan")
		return
	}
	if plan.ActionType == ActionFullSupersede {
		memory.RecordSupersession(plan.TargetDocID, doc.ID)
	}
}

func extractNormalizedCohortList(text string) string {
	numsMap := make(map[int]bool)

	for _, m := range cohortRangeRe.FindAllStringSubmatch(text, -1) {
		if len(m) > 2 {
			n1, err1 := strconv.Atoi(m[1])
			n2, err2 := strconv.Atoi(m[2])
			if err1 == nil && err2 == nil && n1 >= 24 && n2 <= 45 && n1 <= n2 {
				for i := n1; i <= n2; i++ {
					numsMap[i] = true
				}
			}
		}
	}

	for _, m := range cohortLeRe.FindAllStringSubmatch(text, -1) {
		numStr := m[1]
		if numStr == "" {
			numStr = m[2]
		}
		if numStr == "" {
			numStr = m[3]
		}
		if n, err := strconv.Atoi(numStr); err == nil && n >= 24 && n <= 45 {
			for i := 24; i <= n; i++ {
				numsMap[i] = true
			}
		}
	}

	for _, m := range cohortGeRe.FindAllStringSubmatch(text, -1) {
		numStr := m[1]
		if numStr == "" {
			numStr = m[2]
		}
		if numStr == "" {
			numStr = m[3]
		}
		if n, err := strconv.Atoi(numStr); err == nil && n >= 24 && n <= 39 {
			for i := n; i <= 39; i++ {
				numsMap[i] = true
			}
		}
	}

	for _, m := range cohortSingleRe.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			if num, err := strconv.Atoi(m[1]); err == nil && num >= 24 && num <= 45 {
				numsMap[num] = true
			}
		}
	}

	if len(numsMap) == 0 {
		return ""
	}
	if len(numsMap) >= 16 {
		return constants.DocCohortAll
	}
	var sorted []int
	for k := range numsMap {
		sorted = append(sorted, k)
	}
	sort.Ints(sorted)
	cohorts := make([]string, 0, len(sorted))
	for _, k := range sorted {
		cohorts = append(cohorts, fmt.Sprintf("K%d", k))
	}
	return strings.Join(cohorts, ",")
}

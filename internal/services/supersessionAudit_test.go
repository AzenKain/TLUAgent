package services

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
	"tluagent-web/pkg/llm"
)

type scriptedProvider struct {
	replies []llm.Message
	calls   atomic.Int32
}

func (p *scriptedProvider) Name() string         { return "scripted" }
func (p *scriptedProvider) DefaultModel() string { return "scripted-model" }
func (p *scriptedProvider) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	i := int(p.calls.Add(1)) - 1
	if i >= len(p.replies) {
		i = len(p.replies) - 1
	}
	return &llm.ChatResponse{Message: p.replies[i]}, nil
}
func (p *scriptedProvider) ChatStream(ctx context.Context, req *llm.ChatRequest) (llm.StreamReader, error) {
	return nil, fmt.Errorf("not supported")
}

func toolReply(name, args string) llm.Message {
	return llm.Message{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-" + name, Type: "function", Function: llm.FunctionCall{Name: name, Arguments: args}}}}
}

func newScriptedEngine(t *testing.T, replies ...llm.Message) (*RegulatoryDissectionEngine, repositories.RAGRepository, *scriptedProvider, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	repo := repositories.NewRAGRepository(db, cache.NewRamCache())
	provider := &scriptedProvider{replies: replies}
	mgr := llm.NewManager()
	mgr.Register("scripted", provider)
	return NewRegulatoryDissectionEngine(EngineParams{DB: db, RAGRepo: repo, LLMManager: mgr}), repo, provider, db
}

func proposal(action, target string) string {
	return fmt.Sprintf(`{"action_type":"%s","target_doc_id":"%s","reason":"quoted provisions"}`, action, target)
}

func TestExecuteDissectionLoop_RequiresInspectionBeforeSupersession(t *testing.T) {
	engine, repo, provider, db := newScriptedEngine(t,
		toolReply("propose_granular_supersession", proposal("FULL_SUPERSEDE", "DOC_OLD")),
		toolReply("inspect_document_articles", `{"document_id":"DOC_OLD"}`),
		toolReply("propose_granular_supersession", proposal("FULL_SUPERSEDE", "DOC_OLD")),
	)
	defer db.Close()
	target := seedGuardTarget(t, repo, "DOC_OLD", "ALL")

	req := &DissectionRequest{
		NewDocID: "DOC_NEW", Title: "New decision", DocType: "QUY_DINH", Domain: "CHUAN_DAU_RA", ApplicableCohort: "ALL",
		Content: "New decision\nArticle 1. Substantive rule.", Candidates: []*models.Document{target},
	}
	plan, err := engine.ExecuteDissectionLoop(context.Background(), req)
	if err != nil {
		t.Fatalf("dissection failed: %v", err)
	}
	if plan.ActionType != ActionFullSupersede || plan.TargetDocID != "DOC_OLD" {
		t.Fatalf("expected the supersession after inspection, got %s -> %q", plan.ActionType, plan.TargetDocID)
	}
	if provider.calls.Load() != 4 {
		t.Fatalf("expected rejected proposal, inspection, accepted proposal and one verification call, calls=%d", provider.calls.Load())
	}
}

func TestExecuteDissectionLoop_NeverInspectedProposalIsDropped(t *testing.T) {
	engine, repo, _, db := newScriptedEngine(t, toolReply("propose_granular_supersession", proposal("FULL_SUPERSEDE", "DOC_OLD")))
	defer db.Close()
	target := seedGuardTarget(t, repo, "DOC_OLD", "ALL")

	req := &DissectionRequest{
		NewDocID: "DOC_NEW", Title: "New decision", DocType: "QUY_DINH", Domain: "CHUAN_DAU_RA", ApplicableCohort: "ALL",
		Content: "New decision\nArticle 1. Substantive rule.", Candidates: []*models.Document{target},
	}
	plan, err := engine.ExecuteDissectionLoop(context.Background(), req)
	if err != nil {
		t.Fatalf("dissection failed: %v", err)
	}
	if plan.ActionType != ActionNone || plan.TargetDocID != "" {
		t.Fatalf("a proposal made without inspecting the target must not be applied, got %s -> %q", plan.ActionType, plan.TargetDocID)
	}
}

func seedAuditCorpus(t *testing.T, repo repositories.RAGRepository, count int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("doc_%02d", i)
		doc := &models.Document{
			ID: id, DocCode: id, Title: fmt.Sprintf("Regulation number %d", i), Type: "QUY_CHE", Domain: "DAO_TAO",
			Status: "ACTIVE", PublishDate: fmt.Sprintf("2026-01-%02d", i+1), ApplicableCohort: "ALL", PriorityLevel: 80,
		}
		if _, err := repo.UpsertDocument(ctx, doc); err != nil {
			t.Fatalf("seed document: %v", err)
		}
		chunk := &models.DocumentChunk{
			ID: "chk_" + id, DocumentID: id, ChunkType: "CONTENT_CHUNK", Content: "Provisions of " + id, Status: "ACTIVE",
			Domain: "DAO_TAO", ApplicableCohort: "ALL", PriorityLevel: 80,
		}
		if err := repo.UpsertChunksBatch(ctx, []*models.DocumentChunk{chunk}); err != nil {
			t.Fatalf("seed chunk: %v", err)
		}
	}
}

func TestAuditCorpus_IncrementalAuditsOnlyNewOrChangedDocuments(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	repo := repositories.NewRAGRepository(db, cache.NewRamCache())
	ctx := context.Background()
	seedAuditCorpus(t, repo, 5)

	var audited []string
	engine := NewRegulatoryDissectionEngine(EngineParams{DB: db, RAGRepo: repo})
	engine.dissectFn = func(ctx context.Context, req *DissectionRequest) (*GranularSupersessionPlan, error) {
		audited = append(audited, req.NewDocID)
		return &GranularSupersessionPlan{ActionType: ActionNone}, nil
	}

	if err := engine.AuditCorpus(ctx, AuditOptions{}, nil); err != nil {
		t.Fatalf("first audit failed: %v", err)
	}
	if len(audited) != 4 {
		t.Fatalf("the first audit must evaluate the whole corpus (every doc with an earlier candidate), got %v", audited)
	}

	audited = nil
	if err := engine.AuditCorpus(ctx, AuditOptions{}, nil); err != nil {
		t.Fatalf("second audit failed: %v", err)
	}
	if len(audited) != 0 {
		t.Fatalf("nothing changed, so nothing may be sent to the LLM, got %v", audited)
	}

	chunk := &models.DocumentChunk{
		ID: "chk_doc_03", DocumentID: "doc_03", ChunkType: "CONTENT_CHUNK", Content: "Provisions of doc_03 amended", Status: "ACTIVE",
		Domain: "DAO_TAO", ApplicableCohort: "ALL", PriorityLevel: 80,
	}
	if err := repo.UpsertChunksBatch(ctx, []*models.DocumentChunk{chunk}); err != nil {
		t.Fatalf("change chunk: %v", err)
	}
	if err := engine.AuditCorpus(ctx, AuditOptions{}, nil); err != nil {
		t.Fatalf("third audit failed: %v", err)
	}
	if len(audited) != 1 || audited[0] != "doc_03" {
		t.Fatalf("only the changed document may be re-audited, got %v", audited)
	}

	audited = nil
	if err := engine.AuditCorpus(ctx, AuditOptions{Full: true}, nil); err != nil {
		t.Fatalf("full audit failed: %v", err)
	}
	if len(audited) != 4 {
		t.Fatalf("a full audit re-evaluates everything, got %v", audited)
	}
}

func TestAuditCorpus_TitleOnlyDocumentIsNeverSentToTheLLM(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	repo := repositories.NewRAGRepository(db, cache.NewRamCache())
	ctx := context.Background()
	seedAuditCorpus(t, repo, 2)
	stub := &models.Document{ID: "doc_stub", DocCode: "doc_stub", Title: "Announcement only", Type: "QUY_CHE", Domain: "DAO_TAO", Status: "ACTIVE", PublishDate: "2026-02-01", ApplicableCohort: "ALL"}
	if _, err := repo.UpsertDocument(ctx, stub); err != nil {
		t.Fatalf("seed stub: %v", err)
	}
	if err := repo.UpsertChunksBatch(ctx, []*models.DocumentChunk{{ID: "chk_stub", DocumentID: "doc_stub", ChunkType: "CONTENT_CHUNK", Content: "Announcement only", Status: "ACTIVE", Domain: "DAO_TAO", ApplicableCohort: "ALL"}}); err != nil {
		t.Fatalf("seed stub chunk: %v", err)
	}

	var audited []string
	engine := NewRegulatoryDissectionEngine(EngineParams{DB: db, RAGRepo: repo})
	engine.dissectFn = func(ctx context.Context, req *DissectionRequest) (*GranularSupersessionPlan, error) {
		audited = append(audited, req.NewDocID)
		return &GranularSupersessionPlan{ActionType: ActionNone}, nil
	}
	if err := engine.AuditCorpus(ctx, AuditOptions{}, nil); err != nil {
		t.Fatalf("audit failed: %v", err)
	}
	for _, id := range audited {
		if id == "doc_stub" {
			t.Fatalf("a title-only document must not cost an LLM call")
		}
	}
}

func textReply(content string) llm.Message {
	return llm.Message{Role: llm.RoleAssistant, Content: content}
}

func TestExecuteDissectionLoop_SecondOpinionCanDowngradeFullSupersession(t *testing.T) {
	engine, repo, provider, db := newScriptedEngine(t,
		toolReply("inspect_document_articles", `{"document_id":"DOC_OLD"}`),
		toolReply("propose_granular_supersession", proposal("FULL_SUPERSEDE", "DOC_OLD")),
		textReply(`{"verdict":"PARTIAL_AMEND","reason":"The incoming notice only changes the fee article."}`),
	)
	defer db.Close()
	target := seedGuardTarget(t, repo, "DOC_OLD", "ALL")

	req := &DissectionRequest{
		NewDocID: "DOC_NEW", Title: "New decision", DocType: "QUY_DINH", Domain: "CHUAN_DAU_RA", ApplicableCohort: "ALL",
		Content: "New decision\nSubstantive rule text.", Candidates: []*models.Document{target},
	}
	plan, err := engine.ExecuteDissectionLoop(context.Background(), req)
	if err != nil {
		t.Fatalf("dissection failed: %v", err)
	}
	if plan.ActionType != ActionPartialAmend || plan.TargetDocID != "DOC_OLD" {
		t.Fatalf("a second opinion of PARTIAL_AMEND must downgrade the plan, got %s -> %q", plan.ActionType, plan.TargetDocID)
	}
	if provider.calls.Load() != 3 {
		t.Fatalf("expected inspection, proposal and verification, calls=%d", provider.calls.Load())
	}
}

func TestParseVerdict(t *testing.T) {
	if action, reason, ok := parseVerdict("Sure! {\"verdict\":\"none\",\"reason\":\"no overlap\"} done"); !ok || action != ActionNone || reason != "no overlap" {
		t.Fatalf("lenient JSON parsing failed: %v %q %v", action, reason, ok)
	}
	if _, _, ok := parseVerdict("no json here"); ok {
		t.Fatalf("text without JSON must not parse")
	}
	if _, _, ok := parseVerdict(`{"verdict":"MAYBE"}`); ok {
		t.Fatalf("an unknown verdict must not parse")
	}
}

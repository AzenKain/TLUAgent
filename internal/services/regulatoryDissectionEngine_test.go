package services

import (
	"context"
	"testing"

	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/supersede"
)

func TestEnginePlanGeneration(t *testing.T) {
	engine := NewRegulatoryDissectionEngine(EngineParams{})

	parsed := []supersede.ParsedArticle{
		{Index: 0, ArticleNumber: "Điều 1", Title: "Phạm vi", Content: "Phạm vi điều chỉnh"},
		{Index: 1, ArticleNumber: "Điều 15", Title: "Chuẩn đầu ra", Content: "TOEIC 550 điểm"},
	}

	req := &DissectionRequest{
		NewDocID:         "DOC_NEW_2026",
		Title:            "Quy định chuẩn đầu ra mới 2026",
		DocCode:          "QD-2026",
		DocType:          "DECISION",
		Domain:           "ACADEMIC",
		ApplicableCohort: "K38",
		AcademicYear:     "2026-2027",
		Content:          "Nội dung quy định",
	}

	t.Run("Full Supersede Plan", func(t *testing.T) {
		plan := engine.buildDefaultPlan(req, parsed, ActionFullSupersede, "DOC_OLD_2020", "Quy định 2020", "Thay thế toàn bộ văn bản 2020", nil, nil, []string{"K38"}, "HIGH")
		if plan.ActionType != ActionFullSupersede {
			t.Fatalf("expected ActionFullSupersede, got %s", plan.ActionType)
		}
		if len(plan.BlastRadius.SupersededDocIDs) != 1 || plan.BlastRadius.SupersededDocIDs[0] != "DOC_OLD_2020" {
			t.Errorf("unexpected blast radius superseded IDs: %v", plan.BlastRadius.SupersededDocIDs)
		}
		foundSuperDoc := false
		for _, item := range plan.DiffItems {
			if item.EntityType == "DOCUMENT" && item.Action == "SUPERSEDE" && item.Key == "DOC_OLD_2020" {
				foundSuperDoc = true
			}
		}
		if !foundSuperDoc {
			t.Errorf("missing SUPERSEDE diff item for old document")
		}
	})

	t.Run("Partial Amend Plan", func(t *testing.T) {
		superNums := []string{"Điều 15"}
		retainNums := []string{"Điều 1", "Điều 2"}
		plan := engine.buildDefaultPlan(req, parsed, ActionPartialAmend, "DOC_OLD_2020", "Quy định 2020", "Sửa đổi riêng Điều 15", superNums, retainNums, []string{"K38"}, "MEDIUM")

		if plan.ActionType != ActionPartialAmend {
			t.Fatalf("expected ActionPartialAmend, got %s", plan.ActionType)
		}
		if len(plan.SupersededArticles) != 1 || plan.SupersededArticles[0].ArticleNumber != "Điều 15" {
			t.Errorf("unexpected superseded articles: %+v", plan.SupersededArticles)
		}
		if len(plan.RetainedArticles) != 2 {
			t.Errorf("expected 2 retained articles, got %d", len(plan.RetainedArticles))
		}

		foundPartiallyAmendDoc := false
		foundSuperArt := false
		foundRetainArt := false
		for _, item := range plan.DiffItems {
			if item.EntityType == "DOCUMENT" && item.Action == "PARTIALLY_AMEND" && item.Key == "DOC_OLD_2020" {
				foundPartiallyAmendDoc = true
			}
			if item.EntityType == "ARTICLE" && item.Action == "SUPERSEDE" && item.Name == "Điều 15" {
				foundSuperArt = true
			}
			if item.EntityType == "ARTICLE" && item.Action == "RETAIN" && item.Name == "Điều 1" {
				foundRetainArt = true
			}
		}

		if !foundPartiallyAmendDoc {
			t.Errorf("missing PARTIALLY_AMEND diff item for target document")
		}
		if !foundSuperArt {
			t.Errorf("missing SUPERSEDE diff item for article 15")
		}
		if !foundRetainArt {
			t.Errorf("missing RETAIN diff item for article 1")
		}
	})

	t.Run("Coexist By Cohort Plan", func(t *testing.T) {
		plan := engine.buildDefaultPlan(req, parsed, ActionCoexistByCohort, "DOC_OLD_K35", "Học phí K35", "Song song theo khóa", nil, nil, []string{"K38"}, "LOW")
		if plan.ActionType != ActionCoexistByCohort {
			t.Fatalf("expected ActionCoexistByCohort, got %s", plan.ActionType)
		}
		if len(plan.BlastRadius.SupersededDocIDs) != 0 {
			t.Errorf("coexist plan should not supersede doc IDs, got: %v", plan.BlastRadius.SupersededDocIDs)
		}
		foundRetainDoc := false
		for _, item := range plan.DiffItems {
			if item.EntityType == "DOCUMENT" && item.Action == "RETAIN" && item.Key == "DOC_OLD_K35" {
				foundRetainDoc = true
			}
		}
		if !foundRetainDoc {
			t.Errorf("missing RETAIN diff item for existing cohort doc")
		}
	})
}

func TestEngineToolDefinitions(t *testing.T) {
	engine := NewRegulatoryDissectionEngine(EngineParams{})
	defs := engine.GetToolDefinitions()
	if len(defs) != 6 {
		t.Fatalf("expected 6 tool definitions, got %d", len(defs))
	}

	names := map[string]bool{}
	for _, d := range defs {
		names[d.Function.Name] = true
	}

	expected := []string{
		"search_regulations_fts",
		"inspect_document_articles",
		"compare_article_diff",
		"search_knowledge_graph",
		"inspect_cohort_applicability",
		"propose_granular_supersession",
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing expected tool definition: %s", name)
		}
	}
}

func TestExecuteToolCallDiff(t *testing.T) {
	engine := NewRegulatoryDissectionEngine(EngineParams{})
	tc := llm.ToolCall{
		ID: "call_1",
		Function: llm.FunctionCall{
			Name:      "compare_article_diff",
			Arguments: `{"old_article_text":"old rule","new_article_text":"new rule"}`,
		},
	}
	out, isProp := engine.executeToolCall(context.Background(), &DissectionRequest{}, tc)
	if isProp {
		t.Errorf("compare_article_diff should not be proposal")
	}
	if out == "" {
		t.Errorf("expected non-empty diff output")
	}
}

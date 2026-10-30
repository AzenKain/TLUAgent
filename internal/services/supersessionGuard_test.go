package services

import (
	"context"
	"database/sql"
	"testing"

	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func TestIsTitleOnlyContent(t *testing.T) {
	if !isTitleOnlyContent("  Notice  OF exam ", "notice of exam") {
		t.Fatalf("content equal to the title must count as title-only")
	}
	if !isTitleOnlyContent("   ", "notice of exam") {
		t.Fatalf("blank content must count as title-only")
	}
	if isTitleOnlyContent("notice of exam\nTarget: K36, K37", "notice of exam") {
		t.Fatalf("content with a body is not title-only")
	}
}

func TestCohortCovers(t *testing.T) {
	cases := []struct {
		newCohort, oldCohort string
		want                 bool
	}{
		{"ALL", "K26,K27,K28", true},
		{"", "K26", true},
		{"K36,K37", "K26,K27,K36,K37", false},
		{"K26,K27,K28,K29", "K27,K28", true},
		{"K36,K37", "ALL", false},
		{"k36, k37", "K36,K37", true},
	}
	for _, tc := range cases {
		if got := cohortCovers(tc.newCohort, tc.oldCohort); got != tc.want {
			t.Errorf("cohortCovers(%q, %q) = %v, want %v", tc.newCohort, tc.oldCohort, got, tc.want)
		}
	}
}

func newGuardEngine(t *testing.T) (*RegulatoryDissectionEngine, repositories.RAGRepository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	repo := repositories.NewRAGRepository(db, cache.NewRamCache())
	return NewRegulatoryDissectionEngine(EngineParams{DB: db, RAGRepo: repo}), repo, db
}

func seedGuardTarget(t *testing.T, repo repositories.RAGRepository, id, cohort string) *models.Document {
	t.Helper()
	doc := &models.Document{
		ID: id, DocCode: id, Title: "Old exit standard " + id, Type: "QUY_DINH", Domain: "CHUAN_DAU_RA",
		ValidityTier: "OFFICIAL", Status: "ACTIVE", PublishDate: "2023-04-25", AcademicYear: "2022-2023",
		ApplicableCohort: cohort, PriorityLevel: 85,
	}
	if _, err := repo.UpsertDocument(context.Background(), doc); err != nil {
		t.Fatalf("seed target: %v", err)
	}
	chunk := &models.DocumentChunk{
		ID: "chk_" + id + "_0", DocumentID: id, ChunkIndex: 0, ChunkType: "CONTENT_CHUNK",
		Content: "Article 1. Scope and objects.\nArticle 2. English exit standard for graduation.", Status: "ACTIVE",
		Domain: "CHUAN_DAU_RA", ApplicableCohort: cohort, AcademicYear: "2022-2023", PriorityLevel: 85,
	}
	if err := repo.UpsertChunksBatch(context.Background(), []*models.DocumentChunk{chunk}); err != nil {
		t.Fatalf("seed target chunk: %v", err)
	}
	return doc
}

func TestExecuteDissectionLoop_TitleOnlyDocumentCannotSupersede(t *testing.T) {
	engine, repo, db := newGuardEngine(t)
	defer db.Close()
	target := seedGuardTarget(t, repo, "DOC_OLD", "K26,K27,K28,K29,K30,K31,K32,K33,K34,K35,K36,K37,K38,K39")

	req := &DissectionRequest{
		NewDocID: "DOC_STUB", Title: "Notice on internal TOEIC standard", DocType: "QUY_DINH", Domain: "CHUAN_DAU_RA",
		ApplicableCohort: "ALL", Content: "Notice on internal TOEIC standard", Candidates: []*models.Document{target},
	}
	plan, err := engine.ExecuteDissectionLoop(context.Background(), req)
	if err != nil {
		t.Fatalf("dissection failed: %v", err)
	}
	if plan.ActionType != ActionNone || plan.TargetDocID != "" {
		t.Fatalf("a title-only document must not supersede anything, got %s -> %q", plan.ActionType, plan.TargetDocID)
	}
}

func TestExecuteDissectionLoop_NarrowerCohortOnlyAmendsPartially(t *testing.T) {
	engine, repo, db := newGuardEngine(t)
	defer db.Close()
	target := seedGuardTarget(t, repo, "DOC_OLD", "K26,K27,K28,K29,K30,K31,K32,K33,K34,K35,K36,K37,K38,K39")

	req := &DissectionRequest{
		NewDocID: "DOC_NEW", Title: "Notice for two cohorts", DocType: "QUY_DINH", Domain: "CHUAN_DAU_RA",
		ApplicableCohort: "K36,K37", Content: "Notice for two cohorts\nTarget students: K36, K37.", Candidates: []*models.Document{target},
	}
	plan, err := engine.ExecuteDissectionLoop(context.Background(), req)
	if err != nil {
		t.Fatalf("dissection failed: %v", err)
	}
	if plan.ActionType != ActionPartialAmend || plan.TargetDocID != "DOC_OLD" {
		t.Fatalf("a narrower document may only amend partially, got %s -> %q", plan.ActionType, plan.TargetDocID)
	}

	req.ApplicableCohort = "ALL"
	plan, err = engine.ExecuteDissectionLoop(context.Background(), req)
	if err != nil {
		t.Fatalf("dissection failed: %v", err)
	}
	if plan.ActionType != ActionFullSupersede {
		t.Fatalf("a document covering every cohort may supersede fully, got %s", plan.ActionType)
	}
}

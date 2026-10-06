package services_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

// TestRAGService_LexPosteriorEmergencyOverride verifies that emergency notices supersede previous notices.
func TestRAGService_LexPosteriorEmergencyOverride(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	ragRepo := repositories.NewRAGRepository(db, cache.NewRamCache())
	ragSvc := services.NewRAGService(ragRepo)
	ctx := context.Background()

	docOriginal := &models.Document{
		ID:               "doc_dkh_orig",
		Title:            "Ke hoach dang ky hoc goc HK1",
		Type:             "THONG_BAO",
		Domain:           "DANG_KY_HOC",
		ValidityTier:     "SEMESTER_DYNAMIC",
		Status:           "SUPERSEDED",
		PublishDate:      "2026-08-15",
		AcademicYear:     "2026-2027",
		Semester:         "HK1",
		ApplicableCohort: "ALL",
		PriorityLevel:    60,
		FilePath:         "dkh_orig.md",
	}

	docUrgent := &models.Document{
		ID:               "doc_dkh_urgent",
		Title:            "Thong bao khan thay doi lich dang ky hoc HK1",
		Type:             "THONG_BAO",
		Domain:           "DANG_KY_HOC",
		ValidityTier:     "EMERGENCY_OVERRIDE",
		Status:           "ACTIVE",
		PublishDate:      "2026-08-20",
		AcademicYear:     "2026-2027",
		Semester:         "HK1",
		ApplicableCohort: "ALL",
		PriorityLevel:    95,
		FilePath:         "dkh_urgent.md",
	}

	chunks := []*models.DocumentChunk{
		{
			ID:               "chk_dkh_orig",
			DocumentID:       "doc_dkh_orig",
			ChunkIndex:       0,
			ChunkType:        "NOTICE_BODY",
			Content:          "Thoi gian dang ky hoc tu ngay 20 den 25 thang 8",
			Status:           "ACTIVE",
			Domain:           "DANG_KY_HOC",
			ApplicableCohort: "ALL",
			AcademicYear:     "2026-2027",
			PriorityLevel:    60,
			Embedding:        []float32{0.8, 0.2, 0.1},
		},
		{
			ID:               "chk_dkh_urgent",
			DocumentID:       "doc_dkh_urgent",
			ChunkIndex:       0,
			ChunkType:        "NOTICE_BODY",
			Content:          "Thong bao khan hoa lich dang ky hoc sang ngay 28 thang 8 do su co he thong",
			Status:           "ACTIVE",
			Domain:           "DANG_KY_HOC",
			ApplicableCohort: "ALL",
			AcademicYear:     "2026-2027",
			PriorityLevel:    95,
			Embedding:        []float32{0.85, 0.25, 0.15},
		},
	}

	nodes := []*models.KGNode{
		{ID: "doc:doc_dkh_orig", NodeType: "DOCUMENT", Name: docOriginal.Title, Status: "SUPERSEDED"},
		{ID: "doc:doc_dkh_urgent", NodeType: "DOCUMENT", Name: docUrgent.Title, Status: "ACTIVE"},
	}

	edges := []*models.KGEdge{
		{
			ID:           "e_super_dkh",
			SourceNodeID: "doc:doc_dkh_urgent",
			TargetNodeID: "doc:doc_dkh_orig",
			RelationType: "SUPERSEDES",
			Weight:       1.0,
			Status:       "ACTIVE",
		},
	}

	if err := ragSvc.IngestDocument(ctx, docOriginal, []*models.DocumentChunk{chunks[0]}, []*models.KGNode{nodes[0]}, nil); err != nil {
		t.Fatalf("failed to ingest original doc: %v", err)
	}

	if err := ragSvc.IngestDocument(ctx, docUrgent, []*models.DocumentChunk{chunks[1]}, []*models.KGNode{nodes[1]}, edges); err != nil {
		t.Fatalf("failed to ingest urgent doc: %v", err)
	}

	queryVec := []float32{0.82, 0.22, 0.12}
	searchParams := models.RAGSearchParams{
		Query:         "lich dang ky hoc bi doi khi nao",
		StudentCohort: "K37",
		Domain:        "DANG_KY_HOC",
		TopK:          5,
	}

	results, err := ragSvc.Search(ctx, searchParams, queryVec)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("expected search results, got 0")
	}

	if results[0].ChunkID != "chk_dkh_urgent" {
		t.Fatalf("expected top result to be chk_dkh_urgent, got %s", results[0].ChunkID)
	}

	for _, item := range results {
		if item.ChunkID == "chk_dkh_orig" {
			t.Fatalf("superseded chunk chk_dkh_orig was returned in results")
		}
	}
}

// TestRAGService_LexSpecialisCohortSpecific verifies cohort specialization filtering.
func TestRAGService_LexSpecialisCohortSpecific(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	ragRepo := repositories.NewRAGRepository(db, cache.NewRamCache())
	ragSvc := services.NewRAGService(ragRepo)
	ctx := context.Background()

	docGeneric := &models.Document{
		ID:               "doc_toeic_generic",
		Title:            "Quy dinh chuan ngoai ngu chung 450 TOEIC",
		Type:             "QUYET_DINH",
		Domain:           "NGOAI_NGU",
		ValidityTier:     "CORE_PERMANENT",
		Status:           "ACTIVE",
		PublishDate:      "2020-01-01",
		AcademicYear:     "2020-2021",
		Semester:         "ALL",
		ApplicableCohort: "ALL",
		PriorityLevel:    50,
		FilePath:         "toeic_gen.md",
	}

	docSpecial := &models.Document{
		ID:               "doc_toeic_k36_k37",
		Title:            "Quyet dinh chuan TOEIC K36 va K37 500 diem",
		Type:             "QUYET_DINH",
		Domain:           "NGOAI_NGU",
		ValidityTier:     "CORE_PERMANENT",
		Status:           "ACTIVE",
		PublishDate:      "2026-07-28",
		AcademicYear:     "2026-2027",
		Semester:         "ALL",
		ApplicableCohort: "K36,K37",
		PriorityLevel:    75,
		FilePath:         "toeic_k36_k37.md",
	}

	chkGen := &models.DocumentChunk{
		ID:               "chk_toeic_gen",
		DocumentID:       "doc_toeic_generic",
		ChunkIndex:       0,
		ChunkType:        "RULE_BODY",
		Content:          "Chuan dau ra tieng Anh TOEIC toan truong la 450 diem",
		Status:           "ACTIVE",
		Domain:           "NGOAI_NGU",
		ApplicableCohort: "ALL",
		AcademicYear:     "2020-2021",
		PriorityLevel:    50,
		Embedding:        []float32{0.7, 0.3, 0.1},
	}

	chkSpecial := &models.DocumentChunk{
		ID:               "chk_toeic_k36_k37",
		DocumentID:       "doc_toeic_k36_k37",
		ChunkIndex:       0,
		ChunkType:        "RULE_BODY",
		Content:          "Chuan dau ra tieng Anh TOEIC doi voi sinh vien khoa 36 va khoa 37 la 500 diem",
		Status:           "ACTIVE",
		Domain:           "NGOAI_NGU",
		ApplicableCohort: "K36,K37",
		AcademicYear:     "2026-2027",
		PriorityLevel:    75,
		Embedding:        []float32{0.75, 0.35, 0.15},
	}

	if err := ragSvc.IngestDocument(ctx, docGeneric, []*models.DocumentChunk{chkGen}, nil, nil); err != nil {
		t.Fatalf("failed to ingest generic doc: %v", err)
	}

	if err := ragSvc.IngestDocument(ctx, docSpecial, []*models.DocumentChunk{chkSpecial}, nil, nil); err != nil {
		t.Fatalf("failed to ingest special doc: %v", err)
	}

	queryVec := []float32{0.73, 0.33, 0.13}

	searchParamsK37 := models.RAGSearchParams{
		Query:         "chuan dau ra tieng anh toeic bao nhieu diem",
		StudentCohort: "K37",
		Domain:        "NGOAI_NGU",
		TopK:          5,
	}

	resultsK37, err := ragSvc.Search(ctx, searchParamsK37, queryVec)
	if err != nil {
		t.Fatalf("failed to search for K37: %v", err)
	}

	if len(resultsK37) == 0 || resultsK37[0].ChunkID != "chk_toeic_k36_k37" {
		t.Fatalf("expected top result for K37 to be chk_toeic_k36_k37, got %+v", resultsK37)
	}

	searchParamsK38 := models.RAGSearchParams{
		Query:         "chuan dau ra tieng anh toeic bao nhieu diem",
		StudentCohort: "K38",
		Domain:        "NGOAI_NGU",
		TopK:          5,
	}

	resultsK38, err := ragSvc.Search(ctx, searchParamsK38, queryVec)
	if err != nil {
		t.Fatalf("failed to search for K38: %v", err)
	}

	for _, item := range resultsK38 {
		if item.ChunkID == "chk_toeic_k36_k37" {
			t.Fatalf("K36_K37 rule should not be returned for K38 student")
		}
	}
}

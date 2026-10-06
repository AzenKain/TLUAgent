package repositories_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

// TestRAGRepository_DocumentAndChunks verifies insertion, retrieval, and batch chunk storage.
func TestRAGRepository_DocumentAndChunks(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	ramCache := cache.NewRamCache()
	repo := repositories.NewRAGRepository(db, ramCache)
	ctx := context.Background()

	doc := &models.Document{
		ID:               "doc_test_1",
		DocCode:          "2608/QD-DHTL",
		Type:             "QUY_CHE",
		Title:            "Quy che dao tao dai hoc 2026",
		Domain:           "DAO_TAO",
		ValidityTier:     "CORE_PERMANENT",
		Status:           "ACTIVE",
		DepartmentCode:   "PHONG_DAO_TAO",
		PublishDate:      "2026-08-01",
		AcademicYear:     "2026-2027",
		Semester:         "ALL",
		ApplicableCohort: "ALL",
		PriorityLevel:    80,
		FilePath:         "data/test.md",
		CharCount:        5000,
	}

	if _, err := repo.UpsertDocument(ctx, doc); err != nil {
		t.Fatalf("failed to insert document: %v", err)
	}

	fetchedDoc, err := repo.GetDocumentByID(ctx, "doc_test_1")
	if err != nil {
		t.Fatalf("failed to fetch document: %v", err)
	}
	if fetchedDoc.Title != doc.Title || fetchedDoc.PriorityLevel != 80 {
		t.Fatalf("document data mismatch: got %+v", fetchedDoc)
	}

	chunks := []*models.DocumentChunk{
		{
			ID:               "chk_test_1",
			DocumentID:       "doc_test_1",
			ChunkIndex:       0,
			ChunkType:        "RULE_BODY",
			Content:          "Hoc phi mot tin chi nam 2026 la 500000 dong",
			Status:           "ACTIVE",
			Domain:           "DAO_TAO",
			ApplicableCohort: "ALL",
			AcademicYear:     "2026-2027",
			PriorityLevel:    80,
			Embedding:        []float32{0.1, 0.2, 0.3},
		},
		{
			ID:               "chk_test_2",
			DocumentID:       "doc_test_1",
			ChunkIndex:       1,
			ChunkType:        "RULE_BODY",
			Content:          "Dieu kien canh bao hoc tap",
			Status:           "ACTIVE",
			Domain:           "DAO_TAO",
			ApplicableCohort: "ALL",
			AcademicYear:     "2026-2027",
			PriorityLevel:    80,
			Embedding:        []float32{0.4, 0.5, 0.6},
		},
	}

	if err := repo.UpsertChunksBatch(ctx, chunks); err != nil {
		t.Fatalf("failed to insert batch chunks: %v", err)
	}

	chk1, err := repo.GetChunkByID(ctx, "chk_test_1")
	if err != nil {
		t.Fatalf("failed to get chunk by id: %v", err)
	}
	if len(chk1.Embedding) != 3 || chk1.Embedding[0] != 0.1 {
		t.Fatalf("chunk embedding mismatch: got %+v", chk1.Embedding)
	}

	candidates, err := repo.QueryActiveCandidates(ctx, "K37", "DAO_TAO")
	if err != nil {
		t.Fatalf("failed to query candidates: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
}

// TestRAGRepository_RecursiveCTE verifies multi-hop graph invalidation traversal.
func TestRAGRepository_RecursiveCTE(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	ramCache := cache.NewRamCache()
	repo := repositories.NewRAGRepository(db, ramCache)
	ctx := context.Background()

	nodes := []*models.KGNode{
		{ID: "doc:A", NodeType: "DOCUMENT", Name: "Doc A", Status: "SUPERSEDED"},
		{ID: "doc:B", NodeType: "DOCUMENT", Name: "Doc B", Status: "SUPERSEDED"},
		{ID: "doc:C", NodeType: "DOCUMENT", Name: "Doc C", Status: "SUPERSEDED"},
		{ID: "doc:D", NodeType: "DOCUMENT", Name: "Doc D", Status: "ACTIVE"},
	}
	for _, n := range nodes {
		if _, err := repo.UpsertKGNode(ctx, n); err != nil {
			t.Fatalf("failed to insert node %s: %v", n.ID, err)
		}
	}

	edges := []*models.KGEdge{
		{ID: "e_dc", SourceNodeID: "doc:D", TargetNodeID: "doc:C", RelationType: "SUPERSEDES", Status: "ACTIVE"},
		{ID: "e_cb", SourceNodeID: "doc:C", TargetNodeID: "doc:B", RelationType: "SUPERSEDES", Status: "ACTIVE"},
		{ID: "e_ba", SourceNodeID: "doc:B", TargetNodeID: "doc:A", RelationType: "SUPERSEDES", Status: "ACTIVE"},
	}
	for _, e := range edges {
		if _, err := repo.UpsertKGEdge(ctx, e); err != nil {
			t.Fatalf("failed to insert edge %s: %v", e.ID, err)
		}
	}

	supersededMap, err := repo.FindSupersededDocIDs(ctx)
	if err != nil {
		t.Fatalf("failed to find superseded doc ids: %v", err)
	}

	if !supersededMap["C"] || !supersededMap["B"] || !supersededMap["A"] {
		t.Fatalf("expected C, B, and A to be marked superseded, got %+v", supersededMap)
	}
	if supersededMap["D"] {
		t.Fatalf("doc D is active and should not be marked superseded")
	}
}

// TestRAGRepository_CacheAndSingleflight verifies cache population, cache hit, and invalidation.
func TestRAGRepository_CacheAndSingleflight(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	ramCache := cache.NewRamCache()
	repo := repositories.NewRAGRepository(db, ramCache)
	ctx := context.Background()

	doc := &models.Document{
		ID:               "doc_cache_test",
		Title:            "Cache Test Title",
		Type:             "QUY_CHE",
		Domain:           "DAO_TAO",
		ValidityTier:     "CORE_PERMANENT",
		Status:           "ACTIVE",
		DepartmentCode:   "PHONG_DAO_TAO",
		PublishDate:      "2026-09-01",
		AcademicYear:     "2026-2027",
		Semester:         "ALL",
		ApplicableCohort: "ALL",
		PriorityLevel:    70,
		FilePath:         "cache_test.md",
	}

	if _, err := repo.UpsertDocument(ctx, doc); err != nil {
		t.Fatalf("failed to upsert doc: %v", err)
	}

	fetched1, err := repo.GetDocumentByID(ctx, "doc_cache_test")
	if err != nil || fetched1 == nil {
		t.Fatalf("failed first fetch: %v", err)
	}

	var cachedDoc models.Document
	cacheKey := cache.BuildKey("rag_doc", "id", "doc_cache_test")
	if err := ramCache.Get(ctx, cacheKey, &cachedDoc); err != nil || cachedDoc.Title != doc.Title {
		t.Fatalf("expected document to be cached in RAM, err: %v", err)
	}

	fetched2, err := repo.GetDocumentByID(ctx, "doc_cache_test")
	if err != nil || fetched2.Title != doc.Title {
		t.Fatalf("failed second fetch via cache: %v", err)
	}

	doc.Title = "Updated Cache Test Title"
	if _, err := repo.UpsertDocument(ctx, doc); err != nil {
		t.Fatalf("failed to update doc: %v", err)
	}

	if err := ramCache.Get(ctx, cacheKey, &cachedDoc); err == nil {
		t.Fatalf("expected cache key to be invalidated after update")
	}

	fetched3, err := repo.GetDocumentByID(ctx, "doc_cache_test")
	if err != nil || fetched3.Title != "Updated Cache Test Title" {
		t.Fatalf("expected fresh title after invalidation, got: %s", fetched3.Title)
	}

	docs, total, err := repo.ListDocuments(ctx, &request.ListDocumentsDto{
		Domain: "DAO_TAO",
		Status: "ACTIVE",
		PaginationDto: request.PaginationDto{
			Limit: 10,
		},
	})
	if err != nil || total < 1 || len(docs) < 1 {
		t.Fatalf("failed to list documents via Cache IDs pattern: %v", err)
	}
	if docs[0].ID != "doc_cache_test" {
		t.Fatalf("expected doc_cache_test in list, got %s", docs[0].ID)
	}
}

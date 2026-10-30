package services_test

import (
	"context"
	"database/sql"
	"testing"

	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"

	_ "modernc.org/sqlite"
)

func setupGraphRepairDB(t *testing.T) (*sql.DB, repositories.RAGRepository, services.RAGService) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}
	ragRepo := repositories.NewRAGRepository(db, cache.NewRamCache())
	return db, ragRepo, services.NewRAGService(ragRepo, nil, nil, nil)
}

func seedRepairDoc(t *testing.T, repo repositories.RAGRepository, id, status, supersededBy string) {
	t.Helper()
	doc := &models.Document{
		ID:               id,
		DocCode:          id,
		Title:            "Hướng dẫn học tập " + id,
		Type:             "SO_TAY",
		Domain:           "DAO_TAO",
		ValidityTier:     "OFFICIAL",
		Status:           status,
		SupersededByID:   supersededBy,
		PublishDate:      "2024-09-01",
		EffectiveFrom:    "2024-09-01",
		AcademicYear:     "2024-2025",
		ApplicableCohort: "K36",
		PriorityLevel:    50,
		FilePath:         id + ".md",
	}
	if _, err := repo.UpsertDocument(context.Background(), doc); err != nil {
		t.Fatalf("failed to seed document %s: %v", id, err)
	}
}

func TestRepairDocumentGraph_RestoresMissingNodesAndEdges(t *testing.T) {
	db, ragRepo, ragSvc := setupGraphRepairDB(t)
	defer db.Close()
	ctx := context.Background()

	seedRepairDoc(t, ragRepo, "DOC_NB_HDHT_K36", "ACTIVE", "")

	res, err := ragSvc.RepairDocumentGraph(ctx, 100)
	if err != nil {
		t.Fatalf("repair failed: %v", err)
	}
	if res.DocsScanned != 1 || res.NodesCreated != 3 || res.EdgesCreated != 2 {
		t.Fatalf("unexpected repair result: %+v", res)
	}

	if node, err := ragRepo.GetKGNodeByID(ctx, "doc:DOC_NB_HDHT_K36"); err != nil || node == nil {
		t.Fatalf("expected document node to be restored, got %v (err: %v)", node, err)
	}
	if node, err := ragRepo.GetKGNodeByID(ctx, "cohort:K36"); err != nil || node == nil {
		t.Fatalf("expected cohort node to be restored, got %v (err: %v)", node, err)
	}
	if node, err := ragRepo.GetKGNodeByID(ctx, "topic:DAO_TAO"); err != nil || node == nil {
		t.Fatalf("expected topic node to be restored, got %v (err: %v)", node, err)
	}

	again, err := ragSvc.RepairDocumentGraph(ctx, 100)
	if err != nil {
		t.Fatalf("second repair failed: %v", err)
	}
	if again.NodesCreated != 0 || again.EdgesCreated != 0 {
		t.Fatalf("expected repair to be idempotent, got %+v", again)
	}
}

func TestRepairDocumentGraph_RestoresSupersessionEdge(t *testing.T) {
	db, ragRepo, ragSvc := setupGraphRepairDB(t)
	defer db.Close()
	ctx := context.Background()

	seedRepairDoc(t, ragRepo, "DOC_NB_HDHT_K32", "SUPERSEDED", "DOC_NB_HDHT_K35")
	seedRepairDoc(t, ragRepo, "DOC_NB_HDHT_K35", "ACTIVE", "")

	res, err := ragSvc.RepairDocumentGraph(ctx, 100)
	if err != nil {
		t.Fatalf("repair failed: %v", err)
	}
	if res.EdgesCreated != 5 {
		t.Fatalf("expected 4 document edges plus 1 supersession edge, got %+v", res)
	}

	var relation string
	if err := db.QueryRowContext(ctx, `
		SELECT relation_type FROM kg_edges
		WHERE source_node_id = 'doc:DOC_NB_HDHT_K35' AND target_node_id = 'doc:DOC_NB_HDHT_K32'`).Scan(&relation); err != nil {
		t.Fatalf("expected restored supersession edge: %v", err)
	}
	if relation != "SUPERSEDES" {
		t.Fatalf("expected SUPERSEDES relation, got %s", relation)
	}

	again, err := ragSvc.RepairDocumentGraph(ctx, 100)
	if err != nil {
		t.Fatalf("second repair failed: %v", err)
	}
	if again.EdgesCreated != 0 {
		t.Fatalf("expected no duplicate edge on second pass, got %+v", again)
	}
}

func TestRepairDocumentGraph_RebuildsPartialAmendmentFromAuditLog(t *testing.T) {
	db, ragRepo, ragSvc := setupGraphRepairDB(t)
	defer db.Close()
	ctx := context.Background()

	seedRepairDoc(t, ragRepo, "DOC_NB_HDHT_K32", "SUPERSEDED", "DOC_NB_HDHT_K35")
	seedRepairDoc(t, ragRepo, "DOC_NB_HDHT_K35", "ACTIVE", "")
	seedRepairDoc(t, ragRepo, "DOC_THONG_BAO_15", "ACTIVE", "")

	if err := ragRepo.RecordSupersessionLog(ctx, "DOC_THONG_BAO_15", "DOC_NB_HDHT_K32", "PARTIAL_AMENDMENT", "Chỉ sửa đổi phụ lục học phí"); err != nil {
		t.Fatalf("failed to seed audit log: %v", err)
	}

	if _, err := ragSvc.RepairDocumentGraph(ctx, 100); err != nil {
		t.Fatalf("repair failed: %v", err)
	}

	var relation string
	if err := db.QueryRowContext(ctx, `
		SELECT relation_type FROM kg_edges
		WHERE source_node_id = 'doc:DOC_THONG_BAO_15' AND target_node_id = 'doc:DOC_NB_HDHT_K32'`).Scan(&relation); err != nil {
		t.Fatalf("expected restored partial-amendment edge: %v", err)
	}
	if relation != "PARTIALLY_AMENDS" {
		t.Fatalf("expected PARTIALLY_AMENDS relation, got %s", relation)
	}
}

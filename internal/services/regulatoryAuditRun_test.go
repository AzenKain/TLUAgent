package services

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func TestAuditCorpusChronological_ResumesWithoutResettingState(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	repo := repositories.NewRAGRepository(db, cache.NewRamCache())
	ctx := context.Background()

	const docCount = 5
	const totalDocs = docCount + 2
	for i := 0; i < docCount; i++ {
		doc := &models.Document{
			ID:               fmt.Sprintf("doc_audit_%d", i),
			DocCode:          fmt.Sprintf("QC-%02d", i),
			Title:            fmt.Sprintf("Quy che dao tao so %d", i),
			Type:             "QUY_CHE",
			Domain:           "DAO_TAO",
			Status:           "ACTIVE",
			PublishDate:      fmt.Sprintf("2026-01-%02d", i+1),
			ApplicableCohort: "ALL",
			PriorityLevel:    80,
			FilePath:         fmt.Sprintf("data/audit_%d.md", i),
		}
		if _, err := repo.UpsertDocument(ctx, doc); err != nil {
			t.Fatalf("failed to seed document: %v", err)
		}
	}

	for _, id := range []string{"seed_old", "seed_active"} {
		seeded := &models.Document{
			ID:               id,
			DocCode:          id,
			Title:            "Quy che dao tao " + id,
			Type:             "QUY_CHE",
			Domain:           "DAO_TAO",
			Status:           "ACTIVE",
			PublishDate:      "2026-06-01",
			ApplicableCohort: "ALL",
			PriorityLevel:    80,
			FilePath:         "data/" + id + ".md",
		}
		if _, err := repo.UpsertDocument(ctx, seeded); err != nil {
			t.Fatalf("failed to seed %s: %v", id, err)
		}
	}

	if err := repo.RecordSupersessionLog(ctx, "seed_active", "seed_old", "DOCUMENT_REPLACEMENT", "seed"); err != nil {
		t.Fatalf("failed to seed audit log: %v", err)
	}

	engine := NewRegulatoryDissectionEngine(EngineParams{DB: db, RAGRepo: repo})
	engine.auditBatchSize = 2
	engine.dissectFn = func(ctx context.Context, req *DissectionRequest) (*GranularSupersessionPlan, error) {
		return &GranularSupersessionPlan{ActionType: ActionNone}, nil
	}

	if err := engine.AuditCorpusChronological(ctx, nil); err != nil {
		t.Fatalf("first audit run failed: %v", err)
	}

	if exists, _ := repo.AuditLogPairExists(ctx, "seed_active", "seed_old"); exists {
		t.Fatalf("expected the first fresh run to clear pre-existing supersession logs")
	}

	run, err := repo.GetRunningAuditRun(ctx)
	if err != nil || run == nil {
		t.Fatalf("expected a RUNNING audit run after batch 1, got run=%+v err=%v", run, err)
	}
	if run.CursorDocID != "doc_audit_1" || run.Processed != 2 || run.TotalDocs != totalDocs {
		t.Fatalf("unexpected run progress after batch 1: %+v", run)
	}

	if err := repo.RecordSupersessionLog(ctx, "sentinel_active", "sentinel_old", "DOCUMENT_REPLACEMENT", "sentinel"); err != nil {
		t.Fatalf("failed to record sentinel log: %v", err)
	}

	if err := engine.AuditCorpusChronological(ctx, nil); err != nil {
		t.Fatalf("second audit run failed: %v", err)
	}
	if exists, _ := repo.AuditLogPairExists(ctx, "sentinel_active", "sentinel_old"); !exists {
		t.Fatalf("resumed run must not reset supersession logs")
	}
	run, err = repo.GetRunningAuditRun(ctx)
	if err != nil || run == nil {
		t.Fatalf("expected the audit run to still be RUNNING after batch 2, got run=%+v err=%v", run, err)
	}
	if run.CursorDocID != "doc_audit_3" || run.Processed != 4 {
		t.Fatalf("unexpected run progress after batch 2: %+v", run)
	}

	for i := 0; i < 2; i++ {
		if err := engine.AuditCorpusChronological(ctx, nil); err != nil {
			t.Fatalf("audit run %d failed: %v", i+3, err)
		}
	}
	run, err = repo.GetRunningAuditRun(ctx)
	if err != nil || run != nil {
		t.Fatalf("expected no RUNNING audit run after completion, got run=%+v err=%v", run, err)
	}
	latest, err := repo.GetLatestAuditRun(ctx)
	if err != nil || latest == nil {
		t.Fatalf("expected a latest audit run, got %+v err=%v", latest, err)
	}
	if latest.Status != models.AuditRunStatusCompleted {
		t.Fatalf("expected COMPLETED audit run, got %q", latest.Status)
	}
	if latest.Processed != latest.TotalDocs || latest.Processed != totalDocs {
		t.Fatalf("expected processed == total_docs == %d, got %+v", totalDocs, latest)
	}
}

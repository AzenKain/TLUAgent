package services_test

import (
	"context"
	"testing"

	"tluagent-web/internal/models"
	"tluagent-web/pkg/constants"
)

const bodyText = "Dieu 1 pham vi va doi tuong ap dung chuan dau ra tieng Anh tot nghiep hoc. "

func markSuperseded(t *testing.T, repo interface {
	UpdateDocumentStatus(ctx context.Context, id, status, supersededByID string) error
	UpdateChunksByDocIDStatus(ctx context.Context, docID, status string) error
	RecordSupersessionLog(ctx context.Context, activeID, supersededID, ruleType, reason string) error
	UpsertKGEdge(ctx context.Context, edge *models.KGEdge) (*models.KGEdge, error)
	UpsertKGNode(ctx context.Context, node *models.KGNode) (*models.KGNode, error)
}, activeID, targetID string) {
	t.Helper()
	ctx := context.Background()
	for _, id := range []string{activeID, targetID} {
		if _, err := repo.UpsertKGNode(ctx, &models.KGNode{ID: "doc:" + id, NodeType: "DOCUMENT", Name: id, Code: id, Status: "ACTIVE"}); err != nil {
			t.Fatalf("seed node: %v", err)
		}
	}
	if err := repo.UpdateDocumentStatus(ctx, targetID, "SUPERSEDED", activeID); err != nil {
		t.Fatalf("mark status: %v", err)
	}
	if err := repo.UpdateChunksByDocIDStatus(ctx, targetID, "SUPERSEDED"); err != nil {
		t.Fatalf("mark chunks: %v", err)
	}
	if err := repo.RecordSupersessionLog(ctx, activeID, targetID, constants.RuleTypeDocumentReplacement, "seed"); err != nil {
		t.Fatalf("seed log: %v", err)
	}
	if _, err := repo.UpsertKGEdge(ctx, &models.KGEdge{
		ID: "edge:super:" + activeID + ":" + targetID, SourceNodeID: "doc:" + activeID, TargetNodeID: "doc:" + targetID,
		RelationType: constants.KGRelationSupersedes, Weight: 1, Status: "ACTIVE",
	}); err != nil {
		t.Fatalf("seed edge: %v", err)
	}
}

func TestRepairInvalidSupersessions_RevertsReplacementByTitleOnlyDocument(t *testing.T) {
	db, repo, svc := setupGraphRepairDB(t)
	defer db.Close()
	ctx := context.Background()

	seedTextDoc(t, repo, "DOC_OLD", "ACTIVE", "", bodyText+bodyText+bodyText)
	seedTextDoc(t, repo, "DOC_STUB", "ACTIVE", "", "Exit standard DOC_STUB")
	markSuperseded(t, repo, "DOC_STUB", "DOC_OLD")

	res, err := svc.RepairInvalidSupersessions(ctx)
	if err != nil {
		t.Fatalf("repair failed: %v", err)
	}
	if res.Reverted != 1 || res.Downgraded != 0 {
		t.Fatalf("expected one reverted supersession, got %+v", res)
	}

	old, _ := repo.GetDocumentByID(ctx, "DOC_OLD")
	if old.Status != "ACTIVE" || old.SupersededByID != "" {
		t.Fatalf("old document must be active again, got %s by %q", old.Status, old.SupersededByID)
	}
	chunks, _ := repo.GetChunksByDocumentID(ctx, "DOC_OLD")
	if len(chunks) != 1 || chunks[0].Status != "ACTIVE" {
		t.Fatalf("old chunks must be searchable again")
	}
	if exists, _ := repo.AuditLogPairExists(ctx, "DOC_STUB", "DOC_OLD"); exists {
		t.Fatalf("audit log of the unjustified pair must be removed")
	}
	if _, err := repo.GetKGEdgeByID(ctx, "edge:super:DOC_STUB:DOC_OLD"); err == nil {
		t.Fatalf("supersedes edge must be removed")
	}

	again, err := svc.RepairInvalidSupersessions(ctx)
	if err != nil || again.Reverted != 0 {
		t.Fatalf("repair must be idempotent, got %+v err=%v", again, err)
	}
}

func TestRepairInvalidSupersessions_DowngradesNarrowerCohortToPartialAmendment(t *testing.T) {
	db, repo, svc := setupGraphRepairDB(t)
	defer db.Close()
	ctx := context.Background()

	seedTextDoc(t, repo, "DOC_WIDE", "ACTIVE", "", bodyText+bodyText)
	seedTextDoc(t, repo, "DOC_NARROW", "ACTIVE", "", bodyText+bodyText+"Target students: K36, K37.")
	for _, id := range []string{"DOC_WIDE", "DOC_NARROW"} {
		doc, _ := repo.GetDocumentByID(ctx, id)
		doc.ApplicableCohort = map[string]string{"DOC_WIDE": "K26,K27,K36,K37", "DOC_NARROW": "K36,K37"}[id]
		if _, err := repo.UpsertDocument(ctx, doc); err != nil {
			t.Fatalf("set cohort: %v", err)
		}
	}
	markSuperseded(t, repo, "DOC_NARROW", "DOC_WIDE")

	res, err := svc.RepairInvalidSupersessions(ctx)
	if err != nil {
		t.Fatalf("repair failed: %v", err)
	}
	if res.Downgraded != 1 {
		t.Fatalf("expected one downgrade, got %+v", res)
	}
	wide, _ := repo.GetDocumentByID(ctx, "DOC_WIDE")
	if wide.Status != "PARTIALLY_AMENDED" || wide.SupersededByID != "DOC_NARROW" {
		t.Fatalf("wide document must be partially amended by the narrow one, got %s by %q", wide.Status, wide.SupersededByID)
	}
	chunks, _ := repo.GetChunksByDocumentID(ctx, "DOC_WIDE")
	if chunks[0].Status != "ACTIVE" {
		t.Fatalf("chunks for the uncovered cohorts must stay searchable")
	}
	logs, _ := repo.ListAuditLogsByDocID(ctx, "DOC_NARROW")
	if len(logs) != 1 || logs[0].RuleType != constants.RuleTypePartialAmendment {
		t.Fatalf("audit log must record a partial amendment, got %+v", logs)
	}
}

func TestRepairInvalidSupersessions_KeepsJustifiedReplacement(t *testing.T) {
	db, repo, svc := setupGraphRepairDB(t)
	defer db.Close()
	ctx := context.Background()

	seedTextDoc(t, repo, "DOC_V1", "ACTIVE", "", bodyText+bodyText)
	seedTextDoc(t, repo, "DOC_V2", "ACTIVE", "", bodyText+bodyText+"Replaces the earlier decision.")
	markSuperseded(t, repo, "DOC_V2", "DOC_V1")

	res, err := svc.RepairInvalidSupersessions(ctx)
	if err != nil || res.Reverted != 0 || res.Downgraded != 0 {
		t.Fatalf("a justified replacement must stay untouched, got %+v err=%v", res, err)
	}
	v1, _ := repo.GetDocumentByID(ctx, "DOC_V1")
	if v1.Status != "SUPERSEDED" {
		t.Fatalf("status must stay superseded, got %s", v1.Status)
	}
}

func TestRepairInvalidSupersessions_TitleOnlyTargetDoesNotCountAsBroaderScope(t *testing.T) {
	db, repo, svc := setupGraphRepairDB(t)
	defer db.Close()
	ctx := context.Background()

	seedTextDoc(t, repo, "DOC_STUB_OLD", "ACTIVE", "", "Exit standard DOC_STUB_OLD")
	seedTextDoc(t, repo, "DOC_REPUBLISHED", "ACTIVE", "", bodyText+bodyText+"Target students: K36, K37.")
	doc, _ := repo.GetDocumentByID(ctx, "DOC_REPUBLISHED")
	doc.ApplicableCohort = "K36,K37"
	if _, err := repo.UpsertDocument(ctx, doc); err != nil {
		t.Fatalf("set cohort: %v", err)
	}
	markSuperseded(t, repo, "DOC_REPUBLISHED", "DOC_STUB_OLD")

	res, err := svc.RepairInvalidSupersessions(ctx)
	if err != nil || res.Reverted != 0 || res.Downgraded != 0 {
		t.Fatalf("re-publication of a title-only stub must stay valid, got %+v err=%v", res, err)
	}
}

func TestRepairDocumentCohorts_NarrowsNoticeAndItsChunks(t *testing.T) {
	db, repo, svc := setupGraphRepairDB(t)
	defer db.Close()
	ctx := context.Background()

	scope := "Đối tượng: sinh viên K36, K37 (trừ ngành Ngoại ngữ)."
	seedTextDoc(t, repo, "DOC_THONG_BAO_1", "ACTIVE", "", "Notice about the internal exam\n- "+scope+"\n- Fee: 900.000")
	seedTextDoc(t, repo, "DOC_TIN_TUC_1", "ACTIVE", "", "News\n"+scope)

	n, err := svc.RepairDocumentCohorts(ctx)
	if err != nil || n != 1 {
		t.Fatalf("expected exactly the notice to be updated, got %d err=%v", n, err)
	}
	notice, _ := repo.GetDocumentByID(ctx, "DOC_THONG_BAO_1")
	if notice.ApplicableCohort != "K36,K37" {
		t.Fatalf("notice cohort not narrowed: %q", notice.ApplicableCohort)
	}
	chunks, _ := repo.GetChunksByDocumentID(ctx, "DOC_THONG_BAO_1")
	if len(chunks) != 1 || chunks[0].ApplicableCohort != "K36,K37" {
		t.Fatalf("chunk cohort must follow the document, got %+v", chunks)
	}
	news, _ := repo.GetDocumentByID(ctx, "DOC_TIN_TUC_1")
	if news.ApplicableCohort != "ALL" {
		t.Fatalf("news must keep its cohort, got %q", news.ApplicableCohort)
	}
	if again, _ := svc.RepairDocumentCohorts(ctx); again != 0 {
		t.Fatalf("repair must be idempotent")
	}
}

func TestRevertSupersession_RestoresDocumentAndRejectsUnknownPair(t *testing.T) {
	db, repo, svc := setupGraphRepairDB(t)
	defer db.Close()
	ctx := context.Background()

	seedTextDoc(t, repo, "DOC_V1", "ACTIVE", "", bodyText+bodyText)
	seedTextDoc(t, repo, "DOC_V2", "ACTIVE", "", bodyText+bodyText+"Replaces the earlier decision.")
	markSuperseded(t, repo, "DOC_V2", "DOC_V1")

	if err := svc.RevertSupersession(ctx, "DOC_V2", "DOC_V1"); err != nil {
		t.Fatalf("revert failed: %v", err)
	}
	v1, _ := repo.GetDocumentByID(ctx, "DOC_V1")
	if v1.Status != "ACTIVE" || v1.SupersededByID != "" {
		t.Fatalf("the replaced document must be active again, got %s by %q", v1.Status, v1.SupersededByID)
	}
	if exists, _ := repo.AuditLogPairExists(ctx, "DOC_V2", "DOC_V1"); exists {
		t.Fatalf("the audit log of the reverted pair must be removed")
	}
	if err := svc.RevertSupersession(ctx, "DOC_V2", "DOC_V1"); err == nil {
		t.Fatalf("reverting a pair that is no longer audited must fail")
	}
}

func TestUpdateDocumentCohort_NormalizesAndValidates(t *testing.T) {
	db, repo, svc := setupGraphRepairDB(t)
	defer db.Close()
	ctx := context.Background()
	seedTextDoc(t, repo, "DOC_C", "ACTIVE", "", bodyText+bodyText)

	if err := svc.UpdateDocumentCohort(ctx, "DOC_C", " k36, k37 "); err != nil {
		t.Fatalf("valid cohort rejected: %v", err)
	}
	doc, _ := repo.GetDocumentByID(ctx, "DOC_C")
	chunks, _ := repo.GetChunksByDocumentID(ctx, "DOC_C")
	if doc.ApplicableCohort != "K36,K37" || chunks[0].ApplicableCohort != "K36,K37" {
		t.Fatalf("cohort must be normalized on the document and its chunks, got %q / %q", doc.ApplicableCohort, chunks[0].ApplicableCohort)
	}
	for _, bad := range []string{"", "K3", "36,37", "K36;K37", "ALL,K36"} {
		if err := svc.UpdateDocumentCohort(ctx, "DOC_C", bad); err == nil {
			t.Fatalf("cohort %q must be rejected", bad)
		}
	}
	if err := svc.UpdateDocumentCohort(ctx, "DOC_C", "all"); err != nil {
		t.Fatalf("ALL must be accepted: %v", err)
	}
	if err := svc.UpdateDocumentCohort(ctx, "missing", "K36"); err == nil {
		t.Fatalf("an unknown document must be rejected")
	}
}

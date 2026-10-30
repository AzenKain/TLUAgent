package services_test

import (
	"context"
	"testing"

	"tluagent-web/internal/models"
)

func seedDuplicateDoc(t *testing.T, ragRepo interface {
	UpsertDocument(ctx context.Context, doc *models.Document) (*models.Document, error)
}, id, title, publishDate, status string, charCount int) {
	t.Helper()
	doc := &models.Document{
		ID:               id,
		DocCode:          id,
		Title:            title,
		Type:             "THONG_BAO",
		Domain:           "DAO_TAO",
		ValidityTier:     "SEMESTER_DYNAMIC",
		Status:           status,
		PublishDate:      publishDate,
		AcademicYear:     "2025-2026",
		ApplicableCohort: "ALL",
		PriorityLevel:    50,
		FilePath:         "https://thanglong.edu.vn/vi/thong-bao/" + id,
		CharCount:        charCount,
	}
	if _, err := ragRepo.UpsertDocument(context.Background(), doc); err != nil {
		t.Fatalf("failed to seed document %s: %v", id, err)
	}
}

func TestRepairDuplicateDocuments_SupersedesCaseVariantCopies(t *testing.T) {
	db, ragRepo, ragSvc := setupGraphRepairDB(t)
	defer db.Close()
	ctx := context.Background()

	seedDuplicateDoc(t, ragRepo, "DOC_A", "Báo cáo công khai chất lượng năm 2025", "2025-06-27", "ACTIVE", 5918)
	seedDuplicateDoc(t, ragRepo, "DOC_B", "BÁO CÁO Công khai chất lượng năm 2025", "2025-06-27", "ACTIVE", 5926)
	seedDuplicateDoc(t, ragRepo, "DOC_C", "Thông báo khác ngày", "2025-06-27", "ACTIVE", 900)

	res, err := ragSvc.RepairDuplicateDocuments(ctx)
	if err != nil {
		t.Fatalf("repair failed: %v", err)
	}
	if res.Groups != 1 || res.Superseded != 1 {
		t.Fatalf("expected one group with one superseded copy, got %+v", res)
	}

	loser, err := ragRepo.GetDocumentByID(ctx, "DOC_A")
	if err != nil {
		t.Fatalf("load loser: %v", err)
	}
	if loser.Status != "SUPERSEDED" || loser.SupersededByID != "DOC_B" {
		t.Fatalf("expected DOC_A superseded by DOC_B, got status=%s by=%s", loser.Status, loser.SupersededByID)
	}
	winner, err := ragRepo.GetDocumentByID(ctx, "DOC_B")
	if err != nil {
		t.Fatalf("load winner: %v", err)
	}
	if winner.Status != "ACTIVE" {
		t.Fatalf("expected the longer copy to stay active, got %s", winner.Status)
	}
	if _, err := ragRepo.GetKGEdgeByID(ctx, "edge:super:DOC_B:DOC_A"); err != nil {
		t.Fatalf("expected supersession edge, got %v", err)
	}

	again, err := ragSvc.RepairDuplicateDocuments(ctx)
	if err != nil {
		t.Fatalf("second repair failed: %v", err)
	}
	if again.Superseded != 0 {
		t.Fatalf("expected the repair to be idempotent, superseded %d again", again.Superseded)
	}
}

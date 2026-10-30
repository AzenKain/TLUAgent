package services_test

import (
	"context"
	"strings"
	"testing"

	"tluagent-web/internal/models"
)

const garbledSample = "Di~u 1 Pharn vi va dBi nrong chu§n d~u ra Ti~ng Anh t6t nghiep h9c "
const cleanSample = "Dieu 1 pham vi va doi tuong ap dung chuan dau ra tieng Anh tot nghiep hoc "

func seedTextDoc(t *testing.T, ragRepo interface {
	UpsertDocument(ctx context.Context, doc *models.Document) (*models.Document, error)
	UpsertChunksBatch(ctx context.Context, chunks []*models.DocumentChunk) error
}, id, status, supersededBy, text string) {
	t.Helper()
	ctx := context.Background()
	doc := &models.Document{
		ID: id, DocCode: id, Title: "Exit standard " + id, Type: "QUY_DINH", Domain: "CHUAN_DAU_RA",
		ValidityTier: "OFFICIAL", Status: status, SupersededByID: supersededBy, PublishDate: "2026-08-06",
		AcademicYear: "2026-2027", ApplicableCohort: "ALL", PriorityLevel: 85, CharCount: len(text),
	}
	if _, err := ragRepo.UpsertDocument(ctx, doc); err != nil {
		t.Fatalf("seed document: %v", err)
	}
	chunk := &models.DocumentChunk{
		ID: "chk_" + id + "_0", DocumentID: id, ChunkIndex: 0, ChunkType: "CONTENT_CHUNK", Content: text,
		Status: "ACTIVE", Domain: "CHUAN_DAU_RA", ApplicableCohort: "ALL", AcademicYear: "2026-2027", PriorityLevel: 85,
	}
	if err := ragRepo.UpsertChunksBatch(ctx, []*models.DocumentChunk{chunk}); err != nil {
		t.Fatalf("seed chunk: %v", err)
	}
}

func TestReplaceDocumentContent_KeepsStatusAndGraphLinks(t *testing.T) {
	db, ragRepo, ragSvc := setupGraphRepairDB(t)
	defer db.Close()
	ctx := context.Background()

	seedTextDoc(t, ragRepo, "DOC_REPLACE", "PARTIALLY_AMENDED", "DOC_NEWER", strings.Repeat(garbledSample, 8))

	fresh := "| TT | Chung chi | Basic |\n| --- | --- | --- |\n| 4 | TOEIC | 450 |"
	if err := ragSvc.ReplaceDocumentContent(ctx, "DOC_REPLACE", fresh); err != nil {
		t.Fatalf("replace failed: %v", err)
	}

	doc, err := ragRepo.GetDocumentByID(ctx, "DOC_REPLACE")
	if err != nil {
		t.Fatalf("load document: %v", err)
	}
	if doc.Status != "PARTIALLY_AMENDED" || doc.SupersededByID != "DOC_NEWER" {
		t.Fatalf("status and supersession link must survive, got %s by %q", doc.Status, doc.SupersededByID)
	}
	if doc.CharCount != len(fresh) {
		t.Fatalf("char count not refreshed: %d", doc.CharCount)
	}
	chunks, err := ragRepo.GetChunksByDocumentID(ctx, "DOC_REPLACE")
	if err != nil {
		t.Fatalf("load chunks: %v", err)
	}
	if len(chunks) != 1 || chunks[0].Content != fresh {
		t.Fatalf("old chunks must be fully replaced, got %d chunk(s)", len(chunks))
	}
}

func TestReplaceDocumentContent_RejectsEmptyText(t *testing.T) {
	db, ragRepo, ragSvc := setupGraphRepairDB(t)
	defer db.Close()
	seedTextDoc(t, ragRepo, "DOC_KEEP", "ACTIVE", "", strings.Repeat(cleanSample, 4))

	if err := ragSvc.ReplaceDocumentContent(context.Background(), "DOC_KEEP", "   "); err == nil {
		t.Fatalf("empty replacement must be rejected")
	}
	chunks, _ := ragRepo.GetChunksByDocumentID(context.Background(), "DOC_KEEP")
	if len(chunks) != 1 {
		t.Fatalf("stored chunks must stay untouched on rejection")
	}
}

func TestListReextractCandidates_FindsGarbledAndHonoursExplicitIDs(t *testing.T) {
	db, ragRepo, ragSvc := setupGraphRepairDB(t)
	defer db.Close()
	ctx := context.Background()

	seedTextDoc(t, ragRepo, "DOC_BAD", "ACTIVE", "", strings.Repeat(garbledSample, 8))
	seedTextDoc(t, ragRepo, "DOC_GOOD", "ACTIVE", "", strings.Repeat(cleanSample, 8))

	found, err := ragSvc.ListReextractCandidates(ctx, nil, 0.02)
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}
	if len(found) != 1 || found[0].ID != "DOC_BAD" || found[0].GarbleRatio < 0.02 {
		t.Fatalf("only the corrupted document must be selected, got %+v", found)
	}

	explicit, err := ragSvc.ListReextractCandidates(ctx, []string{"DOC_GOOD"}, 0.02)
	if err != nil {
		t.Fatalf("list explicit: %v", err)
	}
	if len(explicit) != 1 || explicit[0].ID != "DOC_GOOD" {
		t.Fatalf("explicit ids must bypass the garble filter, got %+v", explicit)
	}
}

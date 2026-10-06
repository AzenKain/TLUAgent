package services_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func TestInquiryService_SelfEvolutionAndOutdatedExpiry(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	ramCache := cache.NewTheineCache(10 << 20)
	userRepo := repositories.NewUserRepository(db, ramCache)
	inquiryRepo := repositories.NewInquiryRepository(db, ramCache)
	notificationRepo := repositories.NewNotificationRepository(db, ramCache)
	ragRepo := repositories.NewRAGRepository(db, ramCache)

	ctx := context.Background()

	studentID := "student-evolve-1"
	_, err = userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           studentID,
		Email:        "sv_evolve@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Nguyen Van A", Valid: true},
		StudentCode:  sql.NullString{String: "A36888", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create student: %v", err)
	}

	teacherID := "teacher-evolve-1"
	_, err = userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           teacherID,
		Email:        "teacher_evolve@thanglong.edu.vn",
		FullName:     sql.NullString{String: "TS. Le Van C", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create teacher: %v", err)
	}

	inquirySvc := services.NewInquiryService(inquiryRepo, ragRepo, userRepo, notificationRepo)
	ragSvc := services.NewRAGService(ragRepo)
	ragSvc.SetInquiryService(inquirySvc)

	createdInq, err := inquirySvc.CreateInquiry(ctx, studentID, &request.CreateInquiryRequest{
		StudentName:  "Nguyen Van Sinh Vien",
		StudentCode:  "A36001",
		StudentClass: "CNTT01",
		Question:     "Sinh vien bi diem F mon GDTC thi hoc ky he co duoc hoc lai khong?",
		Context:      "Sinh vien K36 CNTT, ky truoc truot the chat 1",
	})
	if err != nil {
		t.Fatalf("Failed to create inquiry: %v", err)
	}
	if createdInq.Status != models.InquiryStatusPending {
		t.Fatalf("Expected status PENDING, got %s", createdInq.Status)
	}

	answered, err := inquirySvc.AnswerInquiry(ctx, teacherID, createdInq.ID, &request.AnswerInquiryRequest{
		Reply: "Sinh vien duoc phep dang ky hoc lai mon GDTC vao ky he neu nha truong co mo lop.",
	})
	if err != nil {
		t.Fatalf("Failed to answer inquiry: %v", err)
	}
	if answered.Status != models.InquiryStatusAnswered || answered.KnowledgeChunkID == "" {
		t.Fatalf("Expected answered inquiry with knowledge chunk, got chunkID=%s status=%s", answered.KnowledgeChunkID, answered.Status)
	}

	notifs, err := notificationRepo.ListNotificationsByUserID(ctx, studentID, 10, 0)
	if err != nil || len(notifs) != 1 {
		t.Fatalf("Expected 1 notification for student, got %d", len(notifs))
	}

	searchRes, err := ragSvc.Search(ctx, models.RAGSearchParams{
		Query: "diem F mon GDTC hoc ky he",
		TopK:  5,
	}, nil)
	if err != nil {
		t.Fatalf("RAG search error: %v", err)
	}
	foundSelfEvolved := false
	for _, res := range searchRes {
		if res.ChunkID == answered.KnowledgeChunkID {
			foundSelfEvolved = true
			break
		}
	}
	if !foundSelfEvolved {
		t.Fatalf("Expected RAG search to retrieve newly evolved teacher QA chunk %s", answered.KnowledgeChunkID)
	}

	newDoc := &models.Document{
		ID:               "doc_gdtc_2026",
		DocCode:          "QD-2026-GDTC",
		Type:             "QUY_CHE",
		Title:            "Quy định mới về chuẩn đầu ra và học lại môn GDTC học kỳ hè",
		Domain:           "ADVISORY_QA",
		ValidityTier:     "CORE_PERMANENT",
		Status:           "ACTIVE",
		DepartmentCode:   "BGH",
		PublishDate:      "2026-06-01",
		AcademicYear:     "2025-2026",
		Semester:         "3",
		ApplicableCohort: "ALL",
		PriorityLevel:    95,
	}

	if _, err := ragRepo.UpsertDocument(ctx, newDoc); err != nil {
		t.Fatalf("Failed to upsert new doc: %v", err)
	}

	expiredCount, err := inquirySvc.CheckAndExpireOutdatedQA(ctx, newDoc)
	if err != nil {
		t.Fatalf("CheckAndExpireOutdatedQA error: %v", err)
	}
	if expiredCount != 1 {
		t.Fatalf("Expected 1 inquiry to be expired, got %d", expiredCount)
	}

	refetchedInq, err := inquirySvc.GetInquiry(ctx, createdInq.ID)
	if err != nil || refetchedInq.Status != models.InquiryStatusExpired || !refetchedInq.IsExpired {
		t.Fatalf("Expected inquiry to be marked EXPIRED, got status=%s isExpired=%v", refetchedInq.Status, refetchedInq.IsExpired)
	}

	chunk, err := ragRepo.GetChunkByID(ctx, answered.KnowledgeChunkID)
	if err != nil || chunk.Status != "EXPIRED" {
		t.Fatalf("Expected RAG chunk to be marked EXPIRED, got status=%s", chunk.Status)
	}

	searchAfterExpire, err := ragSvc.Search(ctx, models.RAGSearchParams{
		Query: "diem F mon GDTC hoc ky he",
		TopK:  5,
	}, nil)
	if err != nil {
		t.Fatalf("RAG search error: %v", err)
	}
	for _, res := range searchAfterExpire {
		if res.ChunkID == answered.KnowledgeChunkID {
			t.Fatalf("Expired teacher QA chunk was unexpectedly retrieved by RAG search!")
		}
	}
}

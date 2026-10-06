package repositories_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func TestInquiryRepository_CRUDAndCache(t *testing.T) {
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
	repo := repositories.NewInquiryRepository(db, ramCache)
	ctx := context.Background()

	studentID := "student-001"
	_, err = userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           studentID,
		Email:        "student001@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Nguyen Van Sinh Vien", Valid: true},
		StudentCode:  sql.NullString{String: "A36001", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create student: %v", err)
	}

	teacherID := "teacher-001"
	_, err = userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           teacherID,
		Email:        "teacher001@thanglong.edu.vn",
		FullName:     sql.NullString{String: "TS. Tran Giang Vien", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create teacher: %v", err)
	}

	inq := &models.AdvisoryInquiry{
		ID:          "inq-001",
		UserID:      studentID,
		StudentName:  "Nguyen Van Sinh Vien",
		StudentCode:  "A36001",
		StudentClass: "CNTT01",
		Question:     "Khi nao thi duoc dang ky thi chuan dau ra tieng Anh?",
		Context:     "Sinh vien K36 CNTT da tich luy 90 tin chi",
	}

	created, err := repo.CreateInquiry(ctx, inq)
	if err != nil {
		t.Fatalf("Failed to create inquiry: %v", err)
	}
	if created.Status != models.InquiryStatusPending {
		t.Fatalf("Expected status PENDING, got %s", created.Status)
	}

	got, err := repo.GetInquiryByID(ctx, "inq-001")
	if err != nil || got == nil {
		t.Fatalf("Failed to get inquiry: %v", err)
	}
	if got.Question != inq.Question {
		t.Fatalf("Expected question %s, got %s", inq.Question, got.Question)
	}

	list, total, err := repo.ListInquiriesByUserID(ctx, studentID, 10, 0)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("Expected 1 inquiry in user list, got total=%d len=%d", total, len(list))
	}

	ragRepo := repositories.NewRAGRepository(db, ramCache)
	_, err = ragRepo.UpsertDocument(ctx, &models.Document{
		ID:               "doc-qa-source",
		Title:            "So tay hoi dap Co van hoc tap",
		Type:             "ADVISORY_QA",
		Domain:           "ADVISORY_QA",
		ValidityTier:     "CORE_PERMANENT",
		Status:           "ACTIVE",
		DepartmentCode:   "P.QLDT",
		PublishDate:      "2026-01-01",
		AcademicYear:     "2025-2026",
		Semester:         "1",
		ApplicableCohort: "ALL",
		PriorityLevel:    80,
	})
	if err != nil {
		t.Fatalf("Failed to create source doc: %v", err)
	}

	_, err = ragRepo.UpsertChunk(ctx, &models.DocumentChunk{
		ID:               "chk-qa-001",
		DocumentID:       "doc-qa-source",
		ChunkIndex:       0,
		ChunkType:        "TEACHER_QA",
		Content:          "Hoi dap thi chuan dau ra tieng Anh",
		Status:           "ACTIVE",
		Domain:           "ADVISORY_QA",
		ApplicableCohort: "ALL",
		AcademicYear:     "2025-2026",
		PriorityLevel:    80,
	})
	if err != nil {
		t.Fatalf("Failed to create test chunk: %v", err)
	}

	_, err = ragRepo.UpsertDocument(ctx, &models.Document{
		ID:               "doc-new-rules",
		Title:            "Quy che dao tao moi 2026",
		Type:             "QUY_CHE",
		Domain:           "DAO_TAO",
		ValidityTier:     "CORE_PERMANENT",
		Status:           "ACTIVE",
		DepartmentCode:   "BGH",
		PublishDate:      "2026-02-01",
		AcademicYear:     "2025-2026",
		Semester:         "2",
		ApplicableCohort: "ALL",
		PriorityLevel:    90,
	})
	if err != nil {
		t.Fatalf("Failed to create new rule doc: %v", err)
	}

	now := time.Now()
	answered, err := repo.AnswerInquiry(ctx, "inq-001", teacherID, "TS. Tran Giang Vien", "Sinh vien duoc dang ky thi vao dau moi hoc ky chinh.", now, "chk-qa-001")
	if err != nil {
		t.Fatalf("Failed to answer inquiry: %v", err)
	}
	if answered.Status != models.InquiryStatusAnswered || answered.TeacherReply == "" {
		t.Fatalf("Expected status ANSWERED with reply, got status=%s reply=%s", answered.Status, answered.TeacherReply)
	}

	activeAnswered, err := repo.ListActiveAnsweredInquiries(ctx)
	if err != nil || len(activeAnswered) != 1 {
		t.Fatalf("Expected 1 active answered inquiry, got %d", len(activeAnswered))
	}

	expired, err := repo.ExpireInquiry(ctx, "inq-001", "doc-new-rules", "Thay the boi quy che moi 2026", time.Now())
	if err != nil {
		t.Fatalf("Failed to expire inquiry: %v", err)
	}
	if expired.Status != models.InquiryStatusExpired || !expired.IsExpired {
		t.Fatalf("Expected status EXPIRED, got %s", expired.Status)
	}

	activeAnsweredAfter, err := repo.ListActiveAnsweredInquiries(ctx)
	if err != nil || len(activeAnsweredAfter) != 0 {
		t.Fatalf("Expected 0 active answered inquiries after expire, got %d", len(activeAnsweredAfter))
	}
}

package services_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/tuition"
	"tluagent-web/pkg/worker"

	_ "modernc.org/sqlite"
)

func setupTestTLUClawDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open test sqlite db: %v", err)
	}

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply database schema: %v", err)
	}

	for _, u := range []string{"admin_01", "admin_op", "admin_alice", "admin_bob", "admin_tester"} {
		if _, err := db.Exec(`INSERT OR IGNORE INTO users (id, email, full_name, password_hash)
			VALUES (?, ? || '@tlu.edu.vn', 'Test User', 'hash')`,
			u, u,
		); err != nil {
			t.Fatalf("failed to seed test user %s: %v", u, err)
		}
	}

	return db
}

func TestTLUClawSessionAndSnapshotLifecycle(t *testing.T) {
	ctx := context.Background()
	db := setupTestTLUClawDB(t)
	defer db.Close()

	memCache := cache.NewRamCache()
	tluclawRepo := repositories.NewTLUClawRepository(db, memCache)
	currRepo := repositories.NewCurriculumRepository(db, memCache)
	ragRepo := repositories.NewRAGRepository(db, memCache)
	ragSvc := services.NewRAGService(ragRepo, memCache, nil, nil)

	svc := services.NewTLUClawService(services.TLUClawParams{
		TLUClawRepo:    tluclawRepo,
		CurriculumRepo: currRepo,
		RAGRepo:        ragRepo,
		RAGService:     ragSvc,
		RAMCache:       memCache,
	})

	sess, err := svc.CreateSession(ctx, "admin_01", "Phiên điều hành K40")
	if err != nil {
		t.Fatalf("unexpected create session error: %v", err)
	}

	if sess.Title != "Phiên điều hành K40" {
		t.Fatalf("expected title 'Phiên điều hành K40', got '%s'", sess.Title)
	}

	sessions, err := svc.ListSessions(ctx, "admin_01", 10, 0)
	if err != nil || len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d (err: %v)", len(sessions), err)
	}

	exp := time.Now().Add(10 * time.Minute)
	snapParams := sqlc.CreateTLUClawSnapshotParams{
		ID:                "snp_test_01",
		BatchID:           "batch_01",
		SessionID:         sess.ID,
		OperatorID:        "admin_01",
		ActionType:        "curriculum_update",
		Summary:           "Thêm mới môn CS101 cho K40",
		DiffPreviewJson:   `[{"entity_type":"COURSE","action":"CREATE","key":"CS101"}]`,
		BlastRadiusJson:   `{"affected_cohorts":["K40"],"risk_level":"LOW"}`,
		PreviousStateJson: `{}`,
		NewStateJson:      `{"cohort":"K40","major_code":"CNTT","incoming_courses":[{"course_code":"CS101","course_name":"Nhập môn CNTT","credits":3.0}]}`,
		Status:            "STAGED",
		ConfirmationNonce: "nonce_abc_123",
		NonceExpiresAt:    sql.NullTime{Time: exp, Valid: true},
		AppliedAt:         sql.NullTime{Valid: false},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	snap, err := tluclawRepo.CreateSnapshot(ctx, snapParams)
	if err != nil || snap == nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	confirmReq := &request.TLUClawConfirmRequest{
		SnapshotID: "snp_test_01",
		Nonce:      "nonce_abc_123",
	}

	resp, err := svc.ConfirmSnapshot(ctx, "admin_01", confirmReq)
	if err != nil {
		t.Fatalf("failed to confirm snapshot: %v", err)
	}

	if !resp.Success || resp.Status != "COMMITTED" {
		t.Fatalf("expected COMMITTED response, got %+v", resp)
	}

	courses, err := currRepo.GetCurriculumCoursesByCohortAndMajor(ctx, "K40", "CNTT")
	if err != nil || len(courses) != 1 {
		t.Fatalf("expected 1 course in K40 CNTT after confirm, got %d (err: %v)", len(courses), err)
	}

	if courses[0].CourseCode != "CS101" {
		t.Fatalf("expected course code 'CS101', got '%s'", courses[0].CourseCode)
	}
}

func TestTLUClawToolEngineProposals(t *testing.T) {
	ctx := context.Background()
	db := setupTestTLUClawDB(t)
	defer db.Close()

	memCache := cache.NewRamCache()
	tluclawRepo := repositories.NewTLUClawRepository(db, memCache)
	currRepo := repositories.NewCurriculumRepository(db, memCache)
	ragRepo := repositories.NewRAGRepository(db, memCache)
	ragSvc := services.NewRAGService(ragRepo, memCache, nil, nil)

	engine := services.NewTLUClawToolEngine(services.TLUClawParams{
		TLUClawRepo:    tluclawRepo,
		CurriculumRepo: currRepo,
		RAGRepo:        ragRepo,
		RAGService:     ragSvc,
		RAMCache:       memCache,
	})

	_, _ = tluclawRepo.CreateSession(ctx, sqlc.CreateTLUClawSessionParams{
		ID:          "sess_01",
		AdminUserID: "admin_01",
		Title:       "Test Session",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	defs := engine.GetToolDefinitions()
	if len(defs) != 11 {
		t.Fatalf("expected 11 tool definitions, got %d", len(defs))
	}

	call := llm.ToolCall{
		ID:   "call_01",
		Type: "function",
		Function: llm.FunctionCall{
			Name: "tluclaw_curriculum_operations",
			Arguments: `{
				"action": "propose_cohort",
				"cohort": "K40",
				"academic_year": "2026-2030"
			}`,
		},
	}

	msg, card, err := engine.ExecuteTool(ctx, "sess_01", "admin_01", call)
	if err != nil {
		t.Fatalf("execute tool error: %v", err)
	}

	if card == nil {
		t.Fatalf("expected non-nil preview card for propose_cohort")
	}

	if card.ActionType != "academic_structure" {
		t.Fatalf("expected action_type 'academic_structure', got '%s'", card.ActionType)
	}

	if len(card.DiffItems) == 0 || card.DiffItems[0].Key != "K40" {
		t.Fatalf("expected diff item key 'K40', got %+v", card.DiffItems)
	}

	if card.Nonce == "" {
		t.Fatalf("expected non-empty confirmation nonce")
	}

	if msg == "" {
		t.Fatalf("expected non-empty preview explanation message")
	}
}

// TestTLUClaw_SelectiveConfirmAndTrueRollback tests selective approval and restoring prior state on rollback.
func TestTLUClaw_SelectiveConfirmAndTrueRollback(t *testing.T) {
	ctx := context.Background()
	db := setupTestTLUClawDB(t)
	defer db.Close()

	memCache := cache.NewRamCache()
	tluclawRepo := repositories.NewTLUClawRepository(db, memCache)
	currRepo := repositories.NewCurriculumRepository(db, memCache)
	ragRepo := repositories.NewRAGRepository(db, memCache)
	ragSvc := services.NewRAGService(ragRepo, memCache, nil, nil)
	notifRepo := repositories.NewNotificationRepository(db, memCache)

	svc := services.NewTLUClawService(services.TLUClawParams{
		TLUClawRepo:    tluclawRepo,
		CurriculumRepo: currRepo,
		RAGRepo:        ragRepo,
		RAGService:     ragSvc,
		RAMCache:       memCache,
		NotifRepo:      notifRepo,
	})

	sess, err := svc.CreateSession(ctx, "admin_01", "Phiên cập nhật chọn lọc")
	if err != nil {
		t.Fatalf("create session error: %v", err)
	}

	_, err = currRepo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
		ID:           "CURR_K40_CNTT",
		Cohort:       "K40",
		AcademicYear: "2026",
		MajorName:    "CNTT",
		MajorCode:    "CNTT",
		FacultyCode:  "ALL",
		Aliases:      "",
		TotalCourses: 1,
		TotalCredits: 3.0,
	})
	if err != nil {
		t.Fatalf("failed to insert test curriculum: %v", err)
	}

	_, err = currRepo.UpsertCurriculumCourse(ctx, sqlc.UpsertCurriculumCourseParams{
		ID:                "c_cs101_v1",
		CurriculumID:      "CURR_K40_CNTT",
		Cohort:            "K40",
		MajorName:         "CNTT",
		CourseCode:        "CS101",
		CourseName:        "Nhập môn CNTT",
		Credits:           3.0,
		Weight:            1.0,
		Hours:             "45",
		KnowledgeBlock:    "Cơ sở ngành",
		PrerequisitesText: "",
		SuggestedSemester: 1,
		IsElearning:       0,
	})
	if err != nil {
		t.Fatalf("failed to insert test course: %v", err)
	}

	exp := time.Now().Add(10 * time.Minute)
	snapParams := sqlc.CreateTLUClawSnapshotParams{
		ID:                "snp_selective_01",
		BatchID:           "batch_selective_01",
		SessionID:         sess.ID,
		OperatorID:        "admin_01",
		ActionType:        "curriculum_update",
		Summary:           "Cập nhật môn CS101 và bổ sung CS102",
		DiffPreviewJson:   `[{"entity_type":"COURSE","action":"UPDATE","key":"CS101"},{"entity_type":"COURSE","action":"CREATE","key":"CS102"}]`,
		BlastRadiusJson:   `{"affected_cohorts":["K40"],"risk_level":"LOW","impact_summary":"Cập nhật K40"}`,
		PreviousStateJson: `{"cohort":"K40","major_code":"CNTT","existing_courses":[{"id":"c_cs101_v1","curriculum_id":"CURR_K40_CNTT","cohort":"K40","major_name":"CNTT","course_code":"CS101","course_name":"Nhập môn CNTT","credits":3.0,"weight":1.0,"hours":"45","knowledge_block":"Cơ sở ngành","suggested_semester":1,"is_elearning":false}]}`,
		NewStateJson:      `{"cohort":"K40","major_code":"CNTT","incoming_courses":[{"course_code":"CS101","course_name":"Nhập môn CNTT","credits":4.0,"hours":"60","knowledge_block":"Cơ sở ngành","suggested_semester":1},{"course_code":"CS102","course_name":"Lập trình nâng cao","credits":3.0,"hours":"45","knowledge_block":"Cơ sở ngành","suggested_semester":2}]}`,
		Status:            "STAGED",
		ConfirmationNonce: "nonce_selective_999",
		NonceExpiresAt:    sql.NullTime{Time: exp, Valid: true},
		AppliedAt:         sql.NullTime{Valid: false},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	snap, err := tluclawRepo.CreateSnapshot(ctx, snapParams)
	if err != nil || snap == nil {
		t.Fatalf("failed to create staged snapshot: %v", err)
	}

	confirmReq := &request.TLUClawConfirmRequest{
		SnapshotID:   "snp_selective_01",
		Nonce:        "nonce_selective_999",
		SelectedKeys: []string{"CS101"},
	}

	resp, err := svc.ConfirmSnapshot(ctx, "admin_01", confirmReq)
	if err != nil {
		t.Fatalf("confirm snapshot error: %v", err)
	}

	if resp.Status != "COMMITTED" {
		t.Fatalf("expected COMMITTED, got %s", resp.Status)
	}

	courses, err := currRepo.GetCurriculumCoursesByCohortAndMajor(ctx, "K40", "CNTT")
	if err != nil || len(courses) != 1 {
		t.Fatalf("expected exactly 1 course (only CS101 selected), got %d (err: %v)", len(courses), err)
	}
	if courses[0].Credits != 4.0 {
		t.Fatalf("expected updated credits 4.0 for CS101, got %f", courses[0].Credits)
	}

	rbResp, err := svc.RollbackBatch(ctx, "admin_01", "batch_selective_01")
	if err != nil {
		t.Fatalf("rollback batch error: %v", err)
	}
	if rbResp.Status != "ROLLED_BACK" {
		t.Fatalf("expected ROLLED_BACK status, got %s", rbResp.Status)
	}

	restoredCourses, err := currRepo.GetCurriculumCoursesByCohortAndMajor(ctx, "K40", "CNTT")
	if err != nil || len(restoredCourses) != 1 {
		t.Fatalf("expected 1 course restored, got %d (err: %v)", len(restoredCourses), err)
	}
	if restoredCourses[0].Credits != 3.0 {
		t.Fatalf("expected restored credits 3.0 for CS101, got %f", restoredCourses[0].Credits)
	}
}

// TestTLUClaw_ListSnapshots tests retrieving paginated snapshots and status filtering.
func TestTLUClaw_ListSnapshots(t *testing.T) {
	ctx := context.Background()
	db := setupTestTLUClawDB(t)
	defer db.Close()

	memCache := cache.NewRamCache()
	tluclawRepo := repositories.NewTLUClawRepository(db, memCache)
	currRepo := repositories.NewCurriculumRepository(db, memCache)
	ragRepo := repositories.NewRAGRepository(db, memCache)
	ragSvc := services.NewRAGService(ragRepo, memCache, nil, nil)

	svc := services.NewTLUClawService(services.TLUClawParams{
		TLUClawRepo:    tluclawRepo,
		CurriculumRepo: currRepo,
		RAGRepo:        ragRepo,
		RAGService:     ragSvc,
		RAMCache:       memCache,
	})

	sess, _ := svc.CreateSession(ctx, "admin_01", "Session Snapshots")
	exp := time.Now().Add(10 * time.Minute)

	_, _ = tluclawRepo.CreateSnapshot(ctx, sqlc.CreateTLUClawSnapshotParams{
		ID:                "snp_list_01",
		BatchID:           "b_01",
		SessionID:         sess.ID,
		OperatorID:        "admin_01",
		ActionType:        "curriculum_update",
		Summary:           "Snapshot 1",
		DiffPreviewJson:   `[]`,
		BlastRadiusJson:   `{}`,
		PreviousStateJson: `{}`,
		NewStateJson:      `{}`,
		Status:            "COMMITTED",
		ConfirmationNonce: "n_01",
		NonceExpiresAt:    sql.NullTime{Time: exp, Valid: true},
		AppliedAt:         sql.NullTime{Time: time.Now(), Valid: true},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	})

	_, _ = tluclawRepo.CreateSnapshot(ctx, sqlc.CreateTLUClawSnapshotParams{
		ID:                "snp_list_02",
		BatchID:           "b_02",
		SessionID:         sess.ID,
		OperatorID:        "admin_01",
		ActionType:        "curriculum_update",
		Summary:           "Snapshot 2",
		DiffPreviewJson:   `[]`,
		BlastRadiusJson:   `{}`,
		PreviousStateJson: `{}`,
		NewStateJson:      `{}`,
		Status:            "ROLLED_BACK",
		ConfirmationNonce: "n_02",
		NonceExpiresAt:    sql.NullTime{Time: exp, Valid: true},
		AppliedAt:         sql.NullTime{Time: time.Now(), Valid: true},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	})

	allList, err := svc.ListSnapshots(ctx, "", 10, 0)
	if err != nil || allList.Total != 2 {
		t.Fatalf("expected total 2 snapshots, got %d (err: %v)", allList.Total, err)
	}

	rbList, err := svc.ListSnapshots(ctx, "ROLLED_BACK", 10, 0)
	if err != nil || rbList.Total != 1 {
		t.Fatalf("expected total 1 rolled back snapshot, got %d (err: %v)", rbList.Total, err)
	}
	if rbList.Snapshots[0].Status != "ROLLED_BACK" {
		t.Fatalf("expected status ROLLED_BACK, got %s", rbList.Snapshots[0].Status)
	}
}

// TestTLUClaw_InspectionToolsAndMemoryOps tests Stage 1 inspection tools and admin memory preferences.
func TestTLUClaw_InspectionToolsAndMemoryOps(t *testing.T) {
	ctx := context.Background()
	db := setupTestTLUClawDB(t)
	defer db.Close()

	memCache := cache.NewRamCache()
	tluclawRepo := repositories.NewTLUClawRepository(db, memCache)
	currRepo := repositories.NewCurriculumRepository(db, memCache)
	ragRepo := repositories.NewRAGRepository(db, memCache)
	ragSvc := services.NewRAGService(ragRepo, memCache, nil, nil)
	userRepo := repositories.NewUserRepository(db, memCache)
	memoryRepo := repositories.NewMemoryRepository(db, memCache)
	memorySvc := services.NewMemoryService(memoryRepo, userRepo, nil)

	engine := services.NewTLUClawToolEngine(services.TLUClawParams{
		TLUClawRepo:    tluclawRepo,
		CurriculumRepo: currRepo,
		RAGRepo:        ragRepo,
		RAGService:     ragSvc,
		RAMCache:       memCache,
		MemoryService:  memorySvc,
		UserRepo:       userRepo,
	})

	_, _ = currRepo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
		ID:           "CURR_K39",
		Cohort:       "K39",
		AcademicYear: "2025-2029",
		MajorName:    "Công nghệ thông tin",
		MajorCode:    "CNTT",
		FacultyCode:  "FITI",
		Aliases:      "IT, CNTT",
		TotalCourses: 1,
		TotalCredits: 3,
	})
	_, _ = currRepo.UpsertCurriculumCourse(ctx, sqlc.UpsertCurriculumCourseParams{
		ID:                "C_01",
		CurriculumID:      "CURR_K39",
		CourseCode:        "CS101",
		CourseName:        "Nhập môn CNTT",
		Credits:           3.0,
		PrerequisitesText: "None",
	})

	inspectCall := llm.ToolCall{
		ID:   "call_inspect",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_curriculum_operations",
			Arguments: `{"action": "inspect_structure"}`,
		},
	}
	resStr, _, err := engine.ExecuteTool(ctx, "sess_01", "admin_01", inspectCall)
	if err != nil || resStr == "" {
		t.Fatalf("inspect_structure failed: %v", err)
	}

	currCall := llm.ToolCall{
		ID:   "call_curr",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_curriculum_operations",
			Arguments: `{"action": "inspect_curriculum", "cohort": "K39", "major_code": "CNTT"}`,
		},
	}
	resCurr, _, err := engine.ExecuteTool(ctx, "sess_01", "admin_01", currCall)
	if err != nil || resCurr == "" {
		t.Fatalf("inspect_curriculum failed: %v", err)
	}

	savePrefCall := llm.ToolCall{
		ID:   "call_save_pref",
		Type: "function",
		Function: llm.FunctionCall{
			Name: "tluclaw_admin_memory_operations",
			Arguments: `{
				"action": "save_preference",
				"key": "DEFAULT_WORKING_COHORT",
				"value": "K39",
				"category": "OPERATIONAL_CONTEXT"
			}`,
		},
	}
	resPref, _, err := engine.ExecuteTool(ctx, "sess_01", "admin_01", savePrefCall)
	if err != nil || resPref == "" {
		t.Fatalf("save_preference failed: %v", err)
	}

	listPrefCall := llm.ToolCall{
		ID:   "call_list_pref",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_admin_memory_operations",
			Arguments: `{"action": "list_preferences"}`,
		},
	}
	resListPref, _, err := engine.ExecuteTool(ctx, "sess_01", "admin_01", listPrefCall)
	if err != nil || resListPref == "" {
		t.Fatalf("list_preferences failed: %v", err)
	}
}

// TestTLUClaw_ThreeMemoryTiers tests Tier 1, Tier 2, and Tier 3 memory contexts for TLUClaw.
func TestTLUClaw_ThreeMemoryTiers(t *testing.T) {
	ctx := context.Background()
	db := setupTestTLUClawDB(t)
	defer db.Close()

	memCache := cache.NewRamCache()
	tluclawRepo := repositories.NewTLUClawRepository(db, memCache)
	currRepo := repositories.NewCurriculumRepository(db, memCache)
	ragRepo := repositories.NewRAGRepository(db, memCache)
	ragSvc := services.NewRAGService(ragRepo, memCache, nil, nil)
	userRepo := repositories.NewUserRepository(db, memCache)
	memoryRepo := repositories.NewMemoryRepository(db, memCache)
	memorySvc := services.NewMemoryService(memoryRepo, userRepo, nil)

	svc := services.NewTLUClawService(services.TLUClawParams{
		TLUClawRepo:    tluclawRepo,
		CurriculumRepo: currRepo,
		RAGRepo:        ragRepo,
		RAGService:     ragSvc,
		RAMCache:       memCache,
		MemoryService:  memorySvc,
		UserRepo:       userRepo,
	})

	_, _ = memorySvc.SaveUserMemory(ctx, &models.UserMemoryItem{
		UserID:      "admin_01",
		Category:    "CATALOG_FOCUS",
		MemoryKey:   "MANAGED_FACULTY",
		MemoryValue: "Khoa CNTT",
		Confidence:  1.0,
		Source:      "ADMIN_TEST",
	})
	adminPrompt, err := memorySvc.BuildAdminLongTermMemoryPromptContext(ctx, "admin_01")
	if err != nil || adminPrompt == "" {
		t.Fatalf("expected non-empty admin memory context, got %v", err)
	}

	sess, _ := svc.CreateSession(ctx, "admin_01", "Phiên kiểm thử bộ nhớ")
	exp := time.Now().Add(15 * time.Minute)
	_, _ = tluclawRepo.CreateSnapshot(ctx, sqlc.CreateTLUClawSnapshotParams{
		ID:                "snp_tier2_test",
		BatchID:           "batch_tier2",
		SessionID:         sess.ID,
		OperatorID:        "admin_01",
		ActionType:        "curriculum_update",
		Summary:           "Staged update for K39",
		DiffPreviewJson:   `[]`,
		BlastRadiusJson:   `{}`,
		PreviousStateJson: `{}`,
		NewStateJson:      `{}`,
		Status:            "STAGED",
		ConfirmationNonce: "nonce_tier2_test",
		NonceExpiresAt:    sql.NullTime{Time: exp, Valid: true},
		AppliedAt:         sql.NullTime{Valid: false},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	})

	snaps, err := tluclawRepo.ListSnapshotsBySessionID(ctx, sess.ID)
	if err != nil || len(snaps) == 0 {
		t.Fatalf("expected staged snapshot in session, got %v", err)
	}
	if snaps[0].Status != "STAGED" {
		t.Fatalf("expected STAGED status, got %s", snaps[0].Status)
	}
}

// TestTLUClaw_FullOmnipotentOperations tests the complete administrative operational toolsets.
func TestTLUClaw_FullOmnipotentOperations(t *testing.T) {
	ctx := context.Background()
	db := setupTestTLUClawDB(t)
	defer db.Close()

	memCache := cache.NewRamCache()
	tluclawRepo := repositories.NewTLUClawRepository(db, memCache)
	currRepo := repositories.NewCurriculumRepository(db, memCache)
	ragRepo := repositories.NewRAGRepository(db, memCache)
	ragSvc := services.NewRAGService(ragRepo, memCache, nil, nil)
	userRepo := repositories.NewUserRepository(db, memCache)
	roleRepo := repositories.NewRoleRepository(db, memCache)
	transcriptRepo := repositories.NewTranscriptRepository(db, memCache)
	inquiryRepo := repositories.NewInquiryRepository(db, memCache)
	memoryRepo := repositories.NewMemoryRepository(db, memCache)
	memorySvc := services.NewMemoryService(memoryRepo, userRepo, nil)
	llmRepo := repositories.NewLLMRepository(db, memCache)
	llmConfigSvc := services.NewLLMConfigService(llmRepo, nil, nil)
	notifRepo := repositories.NewNotificationRepository(db, memCache)
	jobRepo := repositories.NewJobRepository(db, memCache)
	jobQueue := worker.NewQueue(2)
	jobSvc := services.NewJobService(jobRepo, jobQueue, nil, ragSvc, nil)
	tuitionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><body>
			<input type="text" id="masv" value="A44519" />
			<input type="text" id="hoten" value="Do Duy Khanh" />
			<input type="text" id="khoa" value="CNTT" />
			<span id="totalAmount">9.450.000</span> vnđ
			<table><tr>
				<td data-title="Mã hóa đơn:">4744643049</td>
				<td data-title="Ngày tạo:">08/03/2026</td>
				<td data-title="Ngày hết hạn:">09/03/2026</td>
				<td data-title="Tổng:">17.400.000</td>
				<td data-title="Thanh toán:">Đã thanh toán</td>
			</tr></table>
			</body></html>`))
	}))
	defer tuitionServer.Close()
	tuitionClient := tuition.NewClient(5*time.Second, tuitionServer.URL)
	roleSvc := services.NewRoleService(roleRepo, nil, nil)

	clawParams := services.TLUClawParams{
		TLUClawRepo:    tluclawRepo,
		CurriculumRepo: currRepo,
		RAGRepo:        ragRepo,
		RAGService:     ragSvc,
		RAMCache:       memCache,
		MemoryService:  memorySvc,
		UserRepo:       userRepo,
		RoleRepo:       roleRepo,
		TranscriptRepo: transcriptRepo,
		InquiryRepo:    inquiryRepo,
		LLMConfigSvc:   llmConfigSvc,
		MemoryRepo:     memoryRepo,
		NotifRepo:      notifRepo,
		JobService:     jobSvc,
		TuitionClient:  tuitionClient,
		RoleService:    roleSvc,
	}
	svc := services.NewTLUClawService(clawParams)
	_ = svc
	engine := services.NewTLUClawToolEngine(clawParams)

	defs := engine.GetToolDefinitions()
	if len(defs) < 9 {
		t.Fatalf("expected at least 9 tool definitions, got %d", len(defs))
	}

	sess, err := svc.CreateSession(ctx, "admin_01", "Phiên kiểm thử Omnipotent")
	if err != nil {
		t.Fatalf("create test session failed: %v", err)
	}
	sessionID := sess.ID

	_, _ = db.Exec(`INSERT INTO users (id, email, full_name, student_code, password_hash)
		VALUES ('stu_44519', 'khanh.dd@thanglong.edu.vn', 'Do Duy Khanh', 'A44519', 'hash')`)

	_, _ = memoryRepo.UpsertStudentProfile(ctx, &models.StudentProfile{
		UserID:           "stu_44519",
		Major:            "CNTT",
		Cohort:           "K35",
		AcademicStanding: "NORMAL",
		CompletedCredits: 80,
		CumulativeGPA:    3.25,
		TargetGPA:        3.5,
		AdvisorNotes:     "Hoc tap tot",
		UpdatedAt:        time.Now(),
	})

	_, _ = transcriptRepo.UpsertGrade(ctx, sqlc.UpsertTranscriptGradeParams{
		ID:          "grd_01",
		UserID:      "stu_44519",
		CourseCode:  "TT101",
		CourseName:  "Lap trinh Co ban",
		Credits:     3.0,
		Score10:     8.5,
		Score4:      3.5,
		GradeLetter: "B+",
		IsPassed:    1,
		IsRetake:    0,
		TermTaken:   "HK1-2023",
	})

	_, _ = ragRepo.UpsertDocument(ctx, &models.Document{
		ID:               "doc_test_qc",
		Title:            "Quy che dao tao tin chi K35",
		Type:             "QUY_CHE",
		Domain:           "DAO_TAO",
		ValidityTier:     "OFFICIAL",
		Status:           "ACTIVE",
		DepartmentCode:   "DT",
		PublishDate:      "2022-09-01",
		AcademicYear:     "2022-2023",
		Semester:         "HK1",
		ApplicableCohort: "K35",
		PriorityLevel:    100,
		FilePath:         "/docs/qc_k35.pdf",
		CharCount:        500,
	})

	_, _ = ragRepo.UpsertChunk(ctx, &models.DocumentChunk{
		ID:               "chk_qc_01",
		DocumentID:       "doc_test_qc",
		ChunkIndex:       0,
		ChunkType:        "CONTENT_CHUNK",
		Content:          "Sinh vien duoc dang ky hoc vuot toi da 24 tin chi moi hoc ky neu GPA dat tu 3.0 tro len.",
		Status:           "ACTIVE",
		Domain:           "DAO_TAO",
		ApplicableCohort: "K35",
		AcademicYear:     "2022-2023",
		PriorityLevel:    100,
	})

	_, _ = ragRepo.UpsertKGNode(ctx, &models.KGNode{
		ID:       "node_k35_rule",
		NodeType: "RULE",
		Name:     "Quy dinh dang ky hoc vuot K35",
		Code:     "QD_HV_K35",
		Status:   "ACTIVE",
	})

	propSubCall := llm.ToolCall{
		ID:   "call_sub",
		Type: "function",
		Function: llm.FunctionCall{
			Name: "tluclaw_substitution_operations",
			Arguments: `{
				"action": "propose_substitution",
				"original_course_code": "CS101",
				"replacement_course_code": "TT101",
				"min_cohort": 30,
				"max_cohort": 38,
				"scope_major": "TT",
				"is_bidirectional": true
			}`,
		},
	}
	subMsg, subCard, err := engine.ExecuteTool(ctx, sessionID, "admin_01", propSubCall)
	if err != nil || subCard == nil || subMsg == "" {
		t.Fatalf("propose_substitution failed: %v", err)
	}

	confirmSubResp, err := svc.ConfirmSnapshot(ctx, "admin_01", &request.TLUClawConfirmRequest{
		SnapshotID: subCard.SnapshotID,
		Nonce:      subCard.Nonce,
	})
	if err != nil || !confirmSubResp.Success {
		t.Fatalf("confirm substitution snapshot failed: %v", err)
	}

	inspectSubCall := llm.ToolCall{
		ID:   "call_inspect_sub",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_substitution_operations",
			Arguments: `{"action": "inspect_substitutions", "course_code": "CS101"}`,
		},
	}
	inspectSubRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", inspectSubCall)
	if err != nil || inspectSubRes == "" {
		t.Fatalf("inspect_substitutions failed: %v", err)
	}

	propPrereqCall := llm.ToolCall{
		ID:   "call_prereq",
		Type: "function",
		Function: llm.FunctionCall{
			Name: "tluclaw_substitution_operations",
			Arguments: `{
				"action": "propose_prerequisite",
				"course_code": "TT201",
				"prerequisite_course_code": "TT101",
				"cohort": "K35",
				"min_grade": "2.0"
			}`,
		},
	}
	_, prereqCard, err := engine.ExecuteTool(ctx, sessionID, "admin_01", propPrereqCall)
	if err != nil || prereqCard == nil {
		t.Fatalf("propose_prerequisite failed: %v", err)
	}
	confirmPrereqResp, err := svc.ConfirmSnapshot(ctx, "admin_01", &request.TLUClawConfirmRequest{
		SnapshotID: prereqCard.SnapshotID,
		Nonce:      prereqCard.Nonce,
	})
	if err != nil || !confirmPrereqResp.Success {
		t.Fatalf("confirm prerequisite failed: %v", err)
	}

	inspectProfileCall := llm.ToolCall{
		ID:   "call_profile",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_student_academic_operations",
			Arguments: `{"action": "inspect_student_profile", "student_code": "A44519"}`,
		},
	}
	profileRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", inspectProfileCall)
	if err != nil || profileRes == "" {
		t.Fatalf("inspect_student_profile failed: %v", err)
	}

	inspectTranscriptCall := llm.ToolCall{
		ID:   "call_trans",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_student_academic_operations",
			Arguments: `{"action": "inspect_student_transcript", "student_code": "A44519"}`,
		},
	}
	transRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", inspectTranscriptCall)
	if err != nil || transRes == "" {
		t.Fatalf("inspect_student_transcript failed: %v", err)
	}

	tuitionCall := llm.ToolCall{
		ID:   "call_tuition",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_student_academic_operations",
			Arguments: `{"action": "inspect_tuition_fee", "student_code": "A44519"}`,
		},
	}
	tuitionRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", tuitionCall)
	if err != nil || tuitionRes == "" {
		t.Fatalf("inspect_tuition_fee failed: %v", err)
	}

	propNoteCall := llm.ToolCall{
		ID:   "call_note",
		Type: "function",
		Function: llm.FunctionCall{
			Name: "tluclaw_student_academic_operations",
			Arguments: `{
				"action": "propose_advisor_note",
				"student_code": "A44519",
				"advisor_note": "Advisor reminder test"
			}`,
		},
	}
	_, noteCard, err := engine.ExecuteTool(ctx, sessionID, "admin_01", propNoteCall)
	if err != nil || noteCard == nil {
		t.Fatalf("propose_advisor_note failed: %v", err)
	}
	confirmNoteResp, err := svc.ConfirmSnapshot(ctx, "admin_01", &request.TLUClawConfirmRequest{
		SnapshotID: noteCard.SnapshotID,
		Nonce:      noteCard.Nonce,
	})
	if err != nil || !confirmNoteResp.Success {
		t.Fatalf("confirm advisor note failed: %v", err)
	}

	searchUserCall := llm.ToolCall{
		ID:   "call_search_user",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_user_operations",
			Arguments: `{"action": "search_users", "search_text": "Do Duy Khanh"}`,
		},
	}
	searchRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", searchUserCall)
	if err != nil || searchRes == "" {
		t.Fatalf("search_users failed: %v", err)
	}

	inspectUserCall := llm.ToolCall{
		ID:   "call_inspect_user",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_user_operations",
			Arguments: `{"action": "inspect_user_detail", "student_code": "A44519"}`,
		},
	}
	inspectUserRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", inspectUserCall)
	if err != nil || inspectUserRes == "" {
		t.Fatalf("inspect_user_detail failed: %v", err)
	}

	listRolesCall := llm.ToolCall{
		ID:   "call_list_roles",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_user_operations",
			Arguments: `{"action": "list_system_roles"}`,
		},
	}
	rolesRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", listRolesCall)
	if err != nil || rolesRes == "" {
		t.Fatalf("list_system_roles failed: %v", err)
	}

	listPermsCall := llm.ToolCall{
		ID:   "call_list_perms",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_user_operations",
			Arguments: `{"action": "list_system_permissions"}`,
		},
	}
	permsRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", listPermsCall)
	if err != nil || permsRes == "" {
		t.Fatalf("list_system_permissions failed: %v", err)
	}

	propStatusCall := llm.ToolCall{
		ID:   "call_status",
		Type: "function",
		Function: llm.FunctionCall{
			Name: "tluclaw_user_operations",
			Arguments: `{
				"action": "propose_user_status",
				"student_code": "A44519",
				"status": "BANNED",
				"reason": "Test suspension"
			}`,
		},
	}
	_, statusCard, err := engine.ExecuteTool(ctx, sessionID, "admin_01", propStatusCall)
	if err != nil || statusCard == nil {
		t.Fatalf("propose_user_status failed: %v", err)
	}
	confirmStatusResp, err := svc.ConfirmSnapshot(ctx, "admin_01", &request.TLUClawConfirmRequest{
		SnapshotID: statusCard.SnapshotID,
		Nonce:      statusCard.Nonce,
	})
	if err != nil || !confirmStatusResp.Success {
		t.Fatalf("confirm user status failed: %v", err)
	}

	searchDocCall := llm.ToolCall{
		ID:   "call_search_doc",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_document_operations",
			Arguments: `{"action": "search_knowledge", "query": "dang ky hoc vuot"}`,
		},
	}
	searchDocRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", searchDocCall)
	if err != nil || searchDocRes == "" {
		t.Fatalf("search_knowledge failed: %v", err)
	}

	kgCall := llm.ToolCall{
		ID:   "call_kg",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_document_operations",
			Arguments: `{"action": "inspect_kg_entity", "node_id": "node_k35_rule"}`,
		},
	}
	kgRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", kgCall)
	if err != nil || kgRes == "" {
		t.Fatalf("inspect_kg_entity failed: %v", err)
	}

	propDocStatusCall := llm.ToolCall{
		ID:   "call_doc_status",
		Type: "function",
		Function: llm.FunctionCall{
			Name: "tluclaw_document_operations",
			Arguments: `{
				"action": "propose_document_status",
				"document_id": "doc_test_qc",
				"status": "ARCHIVED",
				"reason": "Het hieu luc ap dung"
			}`,
		},
	}
	_, docStatusCard, err := engine.ExecuteTool(ctx, sessionID, "admin_01", propDocStatusCall)
	if err != nil || docStatusCard == nil {
		t.Fatalf("propose_document_status failed: %v", err)
	}
	confirmDocStatusResp, err := svc.ConfirmSnapshot(ctx, "admin_01", &request.TLUClawConfirmRequest{
		SnapshotID: docStatusCard.SnapshotID,
		Nonce:      docStatusCard.Nonce,
	})
	if err != nil || !confirmDocStatusResp.Success {
		t.Fatalf("confirm document_status failed: %v", err)
	}
	archivedDoc, err := ragRepo.GetDocumentByID(ctx, "doc_test_qc")
	if err != nil || archivedDoc.Status != "ARCHIVED" {
		t.Fatalf("expected doc status ARCHIVED, got %v", archivedDoc)
	}
	rollbackDocResp, err := svc.RollbackBatch(ctx, "admin_01", docStatusCard.BatchID)
	if err != nil || !rollbackDocResp.Success {
		t.Fatalf("rollback document_status failed: %v", err)
	}
	restoredDoc, err := ragRepo.GetDocumentByID(ctx, "doc_test_qc")
	if err != nil || restoredDoc.Status != "ACTIVE" {
		t.Fatalf("expected restored doc status ACTIVE, got %v", restoredDoc)
	}

	propNotifCall := llm.ToolCall{
		ID:   "call_prop_notif",
		Type: "function",
		Function: llm.FunctionCall{
			Name: "tluclaw_notification_operations",
			Arguments: `{
				"action": "propose_notification",
				"target_type": "STUDENT",
				"student_code": "A44519",
				"title": "Thong bao hoc phi",
				"content": "Han nop hoc phi",
				"notification_type": "ACADEMIC"
			}`,
		},
	}
	_, notifCard, err := engine.ExecuteTool(ctx, sessionID, "admin_01", propNotifCall)
	if err != nil || notifCard == nil {
		t.Fatalf("propose_notification failed: %v", err)
	}
	confirmNotifResp, err := svc.ConfirmSnapshot(ctx, "admin_01", &request.TLUClawConfirmRequest{
		SnapshotID: notifCard.SnapshotID,
		Nonce:      notifCard.Nonce,
	})
	if err != nil || !confirmNotifResp.Success {
		t.Fatalf("confirm notification failed: %v", err)
	}
	listNotifCall := llm.ToolCall{
		ID:   "call_list_notif",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_notification_operations",
			Arguments: `{"action": "list_notifications", "student_code": "A44519"}`,
		},
	}
	notifListRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", listNotifCall)
	if err != nil || notifListRes == "" {
		t.Fatalf("list_notifications failed: %v", err)
	}
	rollbackNotifResp, err := svc.RollbackBatch(ctx, "admin_01", notifCard.BatchID)
	if err != nil || !rollbackNotifResp.Success {
		t.Fatalf("rollback notification failed: %v", err)
	}

	listTasksCall := llm.ToolCall{
		ID:   "call_list_tasks",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_job_operations",
			Arguments: `{"action": "list_available_tasks"}`,
		},
	}
	tasksRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", listTasksCall)
	if err != nil || tasksRes == "" {
		t.Fatalf("list_available_tasks failed: %v", err)
	}

	propJobCall := llm.ToolCall{
		ID:   "call_prop_job",
		Type: "function",
		Function: llm.FunctionCall{
			Name: "tluclaw_job_operations",
			Arguments: `{
				"action": "propose_trigger_job",
				"task_type": "crawl_notices",
				"payload": "{}"
			}`,
		},
	}
	_, jobCard, err := engine.ExecuteTool(ctx, sessionID, "admin_01", propJobCall)
	if err != nil || jobCard == nil {
		t.Fatalf("propose_trigger_job failed: %v", err)
	}
	confirmJobResp, err := svc.ConfirmSnapshot(ctx, "admin_01", &request.TLUClawConfirmRequest{
		SnapshotID: jobCard.SnapshotID,
		Nonce:      jobCard.Nonce,
	})
	if err != nil || !confirmJobResp.Success {
		t.Fatalf("confirm job trigger failed: %v", err)
	}

	listRecentJobsCall := llm.ToolCall{
		ID:   "call_list_jobs",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_job_operations",
			Arguments: `{"action": "list_recent_jobs"}`,
		},
	}
	recentJobsRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", listRecentJobsCall)
	if err != nil || recentJobsRes == "" {
		t.Fatalf("list_recent_jobs failed: %v", err)
	}

	inq, err := inquiryRepo.CreateInquiry(ctx, &models.AdvisoryInquiry{
		ID:           "inq_test_01",
		UserID:       "stu_44519",
		StudentName:  "Do Duy Khanh",
		StudentCode:  "A44519",
		StudentClass: "TT35A",
		Question:     "Em co the hoc vuot khong?",
		Context:      "Hoi ve tin chi",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil || inq == nil {
		t.Fatalf("create test inquiry failed: %v", err)
	}

	listInqCall := llm.ToolCall{
		ID:   "call_list_inq",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_inquiry_operations",
			Arguments: `{"action": "list_pending_inquiries"}`,
		},
	}
	listInqRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", listInqCall)
	if err != nil || listInqRes == "" {
		t.Fatalf("list_pending_inquiries failed: %v", err)
	}

	propReplyCall := llm.ToolCall{
		ID:   "call_reply",
		Type: "function",
		Function: llm.FunctionCall{
			Name: "tluclaw_inquiry_operations",
			Arguments: fmt.Sprintf(`{
				"action": "propose_reply_inquiry",
				"inquiry_id": "%s",
				"reply": "Sinh vien co GPA >= 3.0 duoc phep hoc vuot toi da 24 tin chi."
			}`, inq.ID),
		},
	}
	_, replyCard, err := engine.ExecuteTool(ctx, sessionID, "admin_01", propReplyCall)
	if err != nil || replyCard == nil {
		t.Fatalf("propose_reply_inquiry failed: %v", err)
	}
	confirmReplyResp, err := svc.ConfirmSnapshot(ctx, "admin_01", &request.TLUClawConfirmRequest{
		SnapshotID: replyCard.SnapshotID,
		Nonce:      replyCard.Nonce,
	})
	if err != nil || !confirmReplyResp.Success {
		t.Fatalf("confirm inquiry reply failed: %v", err)
	}

	sysStatsCall := llm.ToolCall{
		ID:   "call_sys_stats",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_system_operations",
			Arguments: `{"action": "inspect_system_stats"}`,
		},
	}
	statsRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", sysStatsCall)
	if err != nil || statsRes == "" {
		t.Fatalf("inspect_system_stats failed: %v", err)
	}

	flushCacheCall := llm.ToolCall{
		ID:   "call_flush",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_system_operations",
			Arguments: `{"action": "flush_system_cache", "scope": "all"}`,
		},
	}
	flushRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", flushCacheCall)
	if err != nil || flushRes == "" {
		t.Fatalf("flush_system_cache failed: %v", err)
	}

	inspectLLMCall := llm.ToolCall{
		ID:   "call_llm_status",
		Type: "function",
		Function: llm.FunctionCall{
			Name:      "tluclaw_system_operations",
			Arguments: `{"action": "inspect_llm_status"}`,
		},
	}
	llmStatusRes, _, err := engine.ExecuteTool(ctx, sessionID, "admin_01", inspectLLMCall)
	if err != nil || llmStatusRes == "" {
		t.Fatalf("inspect_llm_status failed: %v", err)
	}

	rollbackResp, err := svc.RollbackBatch(ctx, "admin_01", subCard.BatchID)
	if err != nil || !rollbackResp.Success {
		t.Fatalf("rollback batch failed: %v", err)
	}
}

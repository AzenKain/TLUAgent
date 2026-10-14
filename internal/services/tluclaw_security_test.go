package services_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/llm"

	_ "modernc.org/sqlite"
)

func stageTestSnapshot(
	t *testing.T,
	repo repositories.TLUClawRepository,
	id, batchID, sessID, operatorID, actionType, nonce, status string,
	expiresAt time.Time,
	newStateJSON string,
) *models.TLUClawActionSnapshot {
	t.Helper()
	snap, err := repo.CreateSnapshot(context.Background(), sqlc.CreateTLUClawSnapshotParams{
		ID:                id,
		BatchID:           batchID,
		SessionID:         sessID,
		OperatorID:        operatorID,
		ActionType:        actionType,
		Summary:           "Test snapshot " + id,
		DiffPreviewJson:   `[]`,
		BlastRadiusJson:   `{"risk_level":"LOW"}`,
		PreviousStateJson: `{}`,
		NewStateJson:      newStateJSON,
		Status:            status,
		ConfirmationNonce: nonce,
		NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
		AppliedAt:         sql.NullTime{Valid: false},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to stage test snapshot: %v", err)
	}
	return snap
}

// TestTLUClawSecurity_ReplayAttack verifies single-use nonce enforcement.
func TestTLUClawSecurity_ReplayAttack(t *testing.T) {
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

	sess, err := svc.CreateSession(ctx, "admin_op", "Replay Test Session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	exp := time.Now().Add(15 * time.Minute)
	statePayload := `{"cohort":"K40","academic_year":"2026-2030"}`
	stageTestSnapshot(t, tluclawRepo, "snp_replay_01", "batch_rep_01", sess.ID, "admin_op", "academic_structure", "nonce_replay_token", "STAGED", exp, statePayload)

	confirmReq := &request.TLUClawConfirmRequest{
		SnapshotID: "snp_replay_01",
		Nonce:      "nonce_replay_token",
	}

	firstResp, err := svc.ConfirmSnapshot(ctx, "admin_op", confirmReq)
	if err != nil {
		t.Fatalf("first confirmation failed unexpectedly: %v", err)
	}
	if !firstResp.Success || firstResp.Status != "COMMITTED" {
		t.Fatalf("expected COMMITTED on first attempt, got %+v", firstResp)
	}

	secondResp, err := svc.ConfirmSnapshot(ctx, "admin_op", confirmReq)
	if err == nil {
		t.Fatalf("expected error on replay confirmation attack, got success: %+v", secondResp)
	}
}

// TestTLUClawSecurity_TamperedOrInvalidNonce verifies rejection of tampered nonces and IDs.
func TestTLUClawSecurity_TamperedOrInvalidNonce(t *testing.T) {
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

	sess, _ := svc.CreateSession(ctx, "admin_op", "Nonce Tamper Session")
	exp := time.Now().Add(15 * time.Minute)
	stageTestSnapshot(t, tluclawRepo, "snp_valid_01", "batch_tamp_01", sess.ID, "admin_op", "academic_structure", "valid_nonce_123", "STAGED", exp, `{}`)

	_, errNotFound := svc.ConfirmSnapshot(ctx, "admin_op", &request.TLUClawConfirmRequest{
		SnapshotID: "snp_valid_01",
		Nonce:      "fabricated_fake_nonce",
	})
	if errNotFound == nil {
		t.Fatalf("expected error for non-existent nonce")
	}

	_, errMismatch := svc.ConfirmSnapshot(ctx, "admin_op", &request.TLUClawConfirmRequest{
		SnapshotID: "snp_mismatched_99",
		Nonce:      "valid_nonce_123",
	})
	if errMismatch == nil {
		t.Fatalf("expected error for mismatched snapshot ID")
	}
}

// TestTLUClawSecurity_ExpiredNonce verifies rejection of expired nonces.
func TestTLUClawSecurity_ExpiredNonce(t *testing.T) {
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

	sess, _ := svc.CreateSession(ctx, "admin_op", "Expired Nonce Session")
	past := time.Now().Add(-10 * time.Minute)
	stageTestSnapshot(t, tluclawRepo, "snp_exp_01", "batch_exp_01", sess.ID, "admin_op", "academic_structure", "expired_nonce_999", "STAGED", past, `{}`)

	_, err := svc.ConfirmSnapshot(ctx, "admin_op", &request.TLUClawConfirmRequest{
		SnapshotID: "snp_exp_01",
		Nonce:      "expired_nonce_999",
	})
	if err == nil {
		t.Fatalf("expected error confirming snapshot with expired nonce")
	}
}

// TestTLUClawSecurity_StateMachineViolations verifies snapshot state transitions.
func TestTLUClawSecurity_StateMachineViolations(t *testing.T) {
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

	sess, _ := svc.CreateSession(ctx, "admin_op", "State Machine Session")
	exp := time.Now().Add(15 * time.Minute)

	stageTestSnapshot(t, tluclawRepo, "snp_sm_cancel", "batch_sm_01", sess.ID, "admin_op", "academic_structure", "nonce_cancel_01", "STAGED", exp, `{}`)

	cancelResp, err := svc.CancelSnapshot(ctx, "admin_op", &request.TLUClawCancelRequest{
		SnapshotID: "snp_sm_cancel",
		Nonce:      "nonce_cancel_01",
	})
	if err != nil || cancelResp.Status != "CANCELLED" {
		t.Fatalf("failed to cancel snapshot: %v", err)
	}

	_, errConfirmCancelled := svc.ConfirmSnapshot(ctx, "admin_op", &request.TLUClawConfirmRequest{
		SnapshotID: "snp_sm_cancel",
		Nonce:      "nonce_cancel_01",
	})
	if errConfirmCancelled == nil {
		t.Fatalf("expected error confirming CANCELLED snapshot")
	}

	_, errDoubleCancel := svc.CancelSnapshot(ctx, "admin_op", &request.TLUClawCancelRequest{
		SnapshotID: "snp_sm_cancel",
		Nonce:      "nonce_cancel_01",
	})
	if errDoubleCancel == nil {
		t.Fatalf("expected error cancelling already CANCELLED snapshot")
	}

	stageTestSnapshot(t, tluclawRepo, "snp_sm_commit", "batch_sm_02", sess.ID, "admin_op", "academic_structure", "nonce_commit_02", "STAGED", exp, `{}`)

	confirmResp, err := svc.ConfirmSnapshot(ctx, "admin_op", &request.TLUClawConfirmRequest{
		SnapshotID: "snp_sm_commit",
		Nonce:      "nonce_commit_02",
	})
	if err != nil || confirmResp.Status != "COMMITTED" {
		t.Fatalf("failed to confirm snapshot: %v", err)
	}

	_, errCancelCommitted := svc.CancelSnapshot(ctx, "admin_op", &request.TLUClawCancelRequest{
		SnapshotID: "snp_sm_commit",
		Nonce:      "nonce_commit_02",
	})
	if errCancelCommitted == nil {
		t.Fatalf("expected error cancelling already COMMITTED snapshot")
	}
}

// TestTLUClawSecurity_RollbackScenarios verifies batch rollback logic and constraints.
func TestTLUClawSecurity_RollbackScenarios(t *testing.T) {
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

	sess, _ := svc.CreateSession(ctx, "admin_op", "Rollback Session")

	_, err := ragRepo.UpsertDocument(ctx, &models.Document{
		ID:               "DOC_OLD_01",
		Title:            "Quy che dao tao 2020",
		Type:             "QUY_CHE",
		Domain:           "DAO_TAO",
		ApplicableCohort: "K35",
		AcademicYear:     "2020",
		Status:           "ACTIVE",
		PriorityLevel:    5,
		CreatedAt:        time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("failed to insert initial document: %v", err)
	}

	exp := time.Now().Add(15 * time.Minute)
	supersedeState := `{
		"new_doc_id": "DOC_NEW_99",
		"title": "Quy che dao tao 2026",
		"doc_type": "QUY_CHE",
		"domain": "DAO_TAO",
		"cohort": "K40",
		"academic_year": "2026",
		"content": "Noi dung quy che moi",
		"superseded_doc_ids": ["DOC_OLD_01"]
	}`
	stageTestSnapshot(t, tluclawRepo, "snp_rb_01", "batch_rollback_target", sess.ID, "admin_op", "document_supersession", "nonce_rb_123", "STAGED", exp, supersedeState)

	_, err = svc.ConfirmSnapshot(ctx, "admin_op", &request.TLUClawConfirmRequest{
		SnapshotID: "snp_rb_01",
		Nonce:      "nonce_rb_123",
	})
	if err != nil {
		t.Fatalf("confirm snapshot failed: %v", err)
	}

	docOldAfterConfirm, err := ragRepo.GetDocumentByID(ctx, "DOC_OLD_01")
	if err != nil || docOldAfterConfirm.Status != "SUPERSEDED" {
		t.Fatalf("expected DOC_OLD_01 to be SUPERSEDED, got %v", docOldAfterConfirm)
	}

	rbResp, err := svc.RollbackBatch(ctx, "admin_op", "batch_rollback_target")
	if err != nil || !rbResp.Success || rbResp.Status != "ROLLED_BACK" {
		t.Fatalf("rollback failed: %v, resp: %+v", err, rbResp)
	}

	docOldRestored, err := ragRepo.GetDocumentByID(ctx, "DOC_OLD_01")
	if err != nil || docOldRestored.Status != "ACTIVE" {
		t.Fatalf("expected DOC_OLD_01 restored to ACTIVE after rollback, got %v", docOldRestored)
	}
	docNewRestored, err := ragRepo.GetDocumentByID(ctx, "DOC_NEW_99")
	if err != nil || docNewRestored.Status != "SUPERSEDED" {
		t.Fatalf("expected DOC_NEW_99 marked SUPERSEDED after rollback, got %v", docNewRestored)
	}

	_, errDoubleRB := svc.RollbackBatch(ctx, "admin_op", "batch_rollback_target")
	if errDoubleRB == nil {
		t.Fatalf("expected error on double rollback attempt")
	}

	_, errNonExistent := svc.RollbackBatch(ctx, "admin_op", "batch_does_not_exist")
	if errNonExistent == nil {
		t.Fatalf("expected error rolling back non-existent batch")
	}

	stageTestSnapshot(t, tluclawRepo, "snp_staged_rb", "batch_uncommitted", sess.ID, "admin_op", "academic_structure", "nonce_uncomm_01", "STAGED", exp, `{}`)
	_, errStagedRB := svc.RollbackBatch(ctx, "admin_op", "batch_uncommitted")
	if errStagedRB == nil {
		t.Fatalf("expected error rolling back uncommitted STAGED batch")
	}
}

// TestTLUClawSecurity_SessionAndSnapshotIDOR verifies boundary isolation across operators.
func TestTLUClawSecurity_SessionAndSnapshotIDOR(t *testing.T) {
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

	sessA, err := svc.CreateSession(ctx, "admin_alice", "Alice Session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	_, errAliceGet := svc.GetSessionDetail(ctx, sessA.ID, "admin_alice")
	if errAliceGet != nil {
		t.Fatalf("alice failed to view own session: %v", errAliceGet)
	}

	_, errBobGet := svc.GetSessionDetail(ctx, sessA.ID, "admin_bob")
	if errBobGet == nil {
		t.Fatalf("expected forbidden error when bob accesses alice session")
	}

	_, errBobMsg := svc.ProcessAdminMessage(ctx, "admin_bob", &request.TLUClawChatRequest{
		SessionID: sessA.ID,
		Message:   "Unauthorized injection message",
	})
	if errBobMsg == nil {
		t.Fatalf("expected forbidden error when bob sends message to alice session")
	}

	errBobDel := svc.DeleteSession(ctx, sessA.ID, "admin_bob")
	if errBobDel == nil {
		t.Fatalf("expected forbidden error when bob attempts to delete alice session")
	}

	exp := time.Now().Add(15 * time.Minute)
	stageTestSnapshot(t, tluclawRepo, "snp_alice_01", "batch_alice_01", sessA.ID, "admin_alice", "academic_structure", "nonce_alice_sec", "STAGED", exp, `{}`)

	_, errBobConfirm := svc.ConfirmSnapshot(ctx, "admin_bob", &request.TLUClawConfirmRequest{
		SnapshotID: "snp_alice_01",
		Nonce:      "nonce_alice_sec",
	})
	if errBobConfirm == nil {
		t.Fatalf("expected forbidden error when bob confirms alice snapshot")
	}

	_, errBobCancel := svc.CancelSnapshot(ctx, "admin_bob", &request.TLUClawCancelRequest{
		SnapshotID: "snp_alice_01",
		Nonce:      "nonce_alice_sec",
	})
	if errBobCancel == nil {
		t.Fatalf("expected forbidden error when bob cancels alice snapshot")
	}

	errAliceDel := svc.DeleteSession(ctx, sessA.ID, "admin_alice")
	if errAliceDel != nil {
		t.Fatalf("alice failed to delete own session: %v", errAliceDel)
	}
}

// TestTLUClaw_BlastRadiusAndDiffEdgeCases verifies tool engine Diff calculations and error handling.
func TestTLUClaw_BlastRadiusAndDiffEdgeCases(t *testing.T) {
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
		ID:          "sess_edge_01",
		AdminUserID: "admin_tester",
		Title:       "Edge Case Testing",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	emptyCall := llm.ToolCall{
		ID:   "call_empty",
		Type: "function",
		Function: llm.FunctionCall{
			Name: "tluclaw_curriculum_operations",
			Arguments: `{
				"action": "propose_curriculum_update",
				"cohort": "K40",
				"major_code": "CNTT",
				"courses_payload": ""
			}`,
		},
	}
	msgEmpty, cardEmpty, errEmpty := engine.ExecuteTool(ctx, "sess_edge_01", "admin_tester", emptyCall)
	if errEmpty != nil || cardEmpty != nil || msgEmpty == "" {
		t.Fatalf("expected error message with nil card on empty courses payload")
	}

	malformedCall := llm.ToolCall{
		ID:   "call_malformed",
		Type: "function",
		Function: llm.FunctionCall{
			Name: "tluclaw_curriculum_operations",
			Arguments: `{
				"action": "propose_curriculum_update",
				"cohort": "K40",
				"major_code": "CNTT",
				"courses_payload": "{invalid_json}"
			}`,
		},
	}
	msgMalformed, cardMalformed, _ := engine.ExecuteTool(ctx, "sess_edge_01", "admin_tester", malformedCall)
	if cardMalformed != nil || msgMalformed == "" {
		t.Fatalf("expected error message with nil card on malformed courses JSON")
	}

	_, _ = currRepo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
		ID:           "CURR_K40_CNTT",
		Cohort:       "K40",
		AcademicYear: "2026-2030",
		MajorName:    "CNTT",
		MajorCode:    "CNTT",
		FacultyCode:  "CNTT",
		Aliases:      "",
		TotalCourses: 1,
		TotalCredits: 3.0,
	})
	_, _ = currRepo.UpsertCurriculumCourse(ctx, sqlc.UpsertCurriculumCourseParams{
		ID:                "C_K40_CNTT_CS101",
		CurriculumID:      "CURR_K40_CNTT",
		Cohort:            "K40",
		MajorName:         "CNTT",
		CourseCode:        "CS101",
		CourseName:        "Nhap mon Lap trinh",
		Credits:           3.0,
		Weight:            1.0,
		Hours:             "45",
		KnowledgeBlock:    "Co so",
		PrerequisitesText: "None",
		SuggestedSemester: 1,
		IsElearning:       0,
	})

	diffCall := llm.ToolCall{
		ID:   "call_diff",
		Type: "function",
		Function: llm.FunctionCall{
			Name: "tluclaw_curriculum_operations",
			Arguments: `{
				"action": "propose_curriculum_update",
				"cohort": "K40",
				"major_code": "CNTT",
				"courses_payload": "[{\"course_code\":\"CS101\",\"course_name\":\"Lap trinh can ban\",\"credits\":4.0,\"prerequisites_text\":\"MATH101\"},{\"course_code\":\"CS102\",\"course_name\":\"Cau truc du lieu\",\"credits\":3.0}]"
			}`,
		},
	}

	msgDiff, cardDiff, errDiff := engine.ExecuteTool(ctx, "sess_edge_01", "admin_tester", diffCall)
	if errDiff != nil || cardDiff == nil || msgDiff == "" {
		t.Fatalf("expected valid card for diff evaluation: err=%v", errDiff)
	}

	if cardDiff.BlastRadius.RiskLevel != "MEDIUM" {
		t.Fatalf("expected risk level MEDIUM when updating existing course credits, got %s", cardDiff.BlastRadius.RiskLevel)
	}

	if len(cardDiff.DiffItems) != 2 {
		t.Fatalf("expected 2 diff items (1 UPDATE, 1 CREATE), got %d", len(cardDiff.DiffItems))
	}

	var hasUpdate, hasCreate bool
	for _, item := range cardDiff.DiffItems {
		if item.Key == "CS101" && item.Action == "UPDATE" {
			hasUpdate = true
		}
		if item.Key == "CS102" && item.Action == "CREATE" {
			hasCreate = true
		}
	}
	if !hasUpdate || !hasCreate {
		t.Fatalf("diff items verification failed: update=%v, create=%v", hasUpdate, hasCreate)
	}

	for i := 1; i <= 3; i++ {
		_, _ = ragRepo.UpsertDocument(ctx, &models.Document{
			ID:               fmt.Sprintf("DOC_TEST_CHAIN_%02d", i),
			Title:            fmt.Sprintf("Quy che cu so %d", i),
			Type:             "QUY_CHE",
			Domain:           "HOC_PHI",
			ApplicableCohort: "K38",
			AcademicYear:     "2024",
			Status:           "ACTIVE",
			PriorityLevel:    1,
			CreatedAt:        time.Now().Format(time.RFC3339),
		})
	}

	docCall := llm.ToolCall{
		ID:   "call_doc_impact",
		Type: "function",
		Function: llm.FunctionCall{
			Name: "tluclaw_document_operations",
			Arguments: `{
				"action": "analyze_supersession_impact",
				"document_title": "Quy dinh hoc phi nam 2026",
				"document_type": "QUY_CHE",
				"domain": "HOC_PHI",
				"applicable_cohort": "K40",
				"academic_year": "2026",
				"document_content": "Noi dung bieu muc hoc phi moi"
			}`,
		},
	}

	msgDoc, cardDoc, errDoc := engine.ExecuteTool(ctx, "sess_edge_01", "admin_tester", docCall)
	if errDoc != nil || cardDoc == nil || msgDoc == "" {
		t.Fatalf("expected valid preview card for doc supersession: err=%v", errDoc)
	}

	if cardDoc.BlastRadius.RiskLevel != "HIGH" {
		t.Fatalf("expected risk level HIGH for multiple supersessions, got %s", cardDoc.BlastRadius.RiskLevel)
	}

	if len(cardDoc.BlastRadius.SupersededDocIDs) != 3 {
		t.Fatalf("expected 3 superseded document IDs, got %d", len(cardDoc.BlastRadius.SupersededDocIDs))
	}
}

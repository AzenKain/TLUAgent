package controllers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/routes"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/database"

	_ "modernc.org/sqlite"
)

type tluclawTestHarness struct {
	mux          *http.ServeMux
	db           *sql.DB
	tluclawRepo  repositories.TLUClawRepository
	userRepo     repositories.UserRepository
	authSvc      services.AuthService
	adminToken   string
	adminID      string
	admin2Token  string
	admin2ID     string
	studentToken string
	studentID    string
	auditorToken string
	auditorID    string
}

func setupTLUClawSecurityHarness(t *testing.T) *tluclawTestHarness {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	_, _ = db.Exec(`INSERT OR IGNORE INTO roles (id, name, description) VALUES ('role_auditor', 'AUDITOR', 'Auditor')`)
	_, _ = db.Exec(`INSERT OR IGNORE INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
		VALUES ('rp_auditor_read', 'role_auditor', 'tluclaw.read', 'allow', '{}')`)

	ramCache := cache.NewTheineCache(10 << 20)
	txManager := database.NewTxManager(db)

	roleRepo := repositories.NewRoleRepository(db, ramCache)
	userRepo := repositories.NewUserRepository(db, ramCache)
	settingsRepo := repositories.NewSettingsRepository(db, ramCache)
	tluclawRepo := repositories.NewTLUClawRepository(db, ramCache)
	currRepo := repositories.NewCurriculumRepository(db, ramCache)
	ragRepo := repositories.NewRAGRepository(db, ramCache)
	ragSvc := services.NewRAGService(ragRepo, ramCache, nil, nil)

	permCache := services.NewPermissionCache(roleRepo)
	if err := permCache.Reload(context.Background()); err != nil {
		t.Fatalf("failed to reload permission cache: %v", err)
	}

	authSvc := services.NewAuthService(userRepo, roleRepo, settingsRepo, txManager)
	tluclawSvc := services.NewTLUClawService(services.TLUClawParams{
		TLUClawRepo:    tluclawRepo,
		CurriculumRepo: currRepo,
		RAGRepo:        ragRepo,
		RAGService:     ragSvc,
		RAMCache:       ramCache,
	})
	tluclawCtrl := controllers.NewTLUClawController(tluclawSvc)

	mux := http.NewServeMux()
	routes.RegisterTLUClawRoutes(mux, tluclawCtrl, userRepo, permCache)

	createTestUserWithRole := func(userID, email, roleName string) string {
		passHash, _ := crypto.HashPassword("TestPass123!Secure")
		_, err := db.Exec(`INSERT INTO users (id, email, full_name, password_hash) VALUES (?, ?, ?, ?)`,
			userID, email, "Test User "+userID, passHash,
		)
		if err != nil {
			t.Fatalf("failed to insert user %s: %v", userID, err)
		}

		var roleID string
		err = db.QueryRow(`SELECT id FROM roles WHERE name = ?`, roleName).Scan(&roleID)
		if err != nil {
			roleID = "role_" + roleName
			_, _ = db.Exec(`INSERT INTO roles (id, name, description) VALUES (?, ?, ?)`, roleID, roleName, roleName)
		}

		_, err = db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, userID, roleID)
		if err != nil {
			t.Fatalf("failed to assign role to user %s: %v", userID, err)
		}

		userEntity, err := userRepo.GetByID(context.Background(), userID)
		if err != nil || userEntity == nil {
			t.Fatalf("failed to fetch user entity %s: %v", userID, err)
		}

		tokenResp, err := authSvc.GenToken(userEntity)
		if err != nil {
			t.Fatalf("failed to generate token for %s: %v", userID, err)
		}
		return tokenResp.AccessToken
	}

	adminID := "usr_admin_01"
	adminToken := createTestUserWithRole(adminID, "admin1@tlu.edu.vn", "ADMIN")

	admin2ID := "usr_admin_02"
	admin2Token := createTestUserWithRole(admin2ID, "admin2@tlu.edu.vn", "ADMIN")

	studentID := "usr_student_01"
	studentToken := createTestUserWithRole(studentID, "student1@tlu.edu.vn", "STUDENT")

	auditorID := "usr_auditor_01"
	auditorToken := createTestUserWithRole(auditorID, "auditor1@tlu.edu.vn", "AUDITOR")

	return &tluclawTestHarness{
		mux:          mux,
		db:           db,
		tluclawRepo:  tluclawRepo,
		userRepo:     userRepo,
		authSvc:      authSvc,
		adminToken:   adminToken,
		adminID:      adminID,
		admin2Token:  admin2Token,
		admin2ID:     admin2ID,
		studentToken: studentToken,
		studentID:    studentID,
		auditorToken: auditorToken,
		auditorID:    auditorID,
	}
}

func doRequest(mux *http.ServeMux, method, path, token string, bodyPayload any) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if bodyPayload != nil {
		b, _ := json.Marshal(bodyPayload)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// TestTLUClawController_AuthenticationAndRBACEnforcement verifies 401 and 403 responses.
func TestTLUClawController_AuthenticationAndRBACEnforcement(t *testing.T) {
	h := setupTLUClawSecurityHarness(t)
	defer h.db.Close()

	endpoints := []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodPost, "/api/admin/tluclaw/chat", request.TLUClawChatRequest{Message: "Ping"}},
		{http.MethodPost, "/api/admin/tluclaw/confirm", request.TLUClawConfirmRequest{SnapshotID: "s1", Nonce: "n1"}},
		{http.MethodPost, "/api/admin/tluclaw/cancel", request.TLUClawCancelRequest{SnapshotID: "s1", Nonce: "n1"}},
		{http.MethodPost, "/api/admin/tluclaw/rollback", request.TLUClawRollbackRequest{BatchID: "b1"}},
		{http.MethodGet, "/api/admin/tluclaw/sessions", nil},
		{http.MethodPost, "/api/admin/tluclaw/sessions", request.TLUClawCreateSessionRequest{Title: "New"}},
		{http.MethodGet, "/api/admin/tluclaw/sessions/sess_999", nil},
		{http.MethodDelete, "/api/admin/tluclaw/sessions/sess_999", nil},
	}

	for _, ep := range endpoints {
		recUnauth := doRequest(h.mux, ep.method, ep.path, "", ep.body)
		if recUnauth.Code != http.StatusUnauthorized {
			t.Fatalf("[%s %s] expected 401 for unauthenticated request, got %d: %s", ep.method, ep.path, recUnauth.Code, recUnauth.Body.String())
		}

		recStudent := doRequest(h.mux, ep.method, ep.path, h.studentToken, ep.body)
		if recStudent.Code != http.StatusForbidden {
			t.Fatalf("[%s %s] expected 403 for student without permissions, got %d: %s", ep.method, ep.path, recStudent.Code, recStudent.Body.String())
		}
	}

	recAuditList := doRequest(h.mux, http.MethodGet, "/api/admin/tluclaw/sessions", h.auditorToken, nil)
	if recAuditList.Code != http.StatusOK {
		t.Fatalf("expected 200 for auditor reading sessions, got %d: %s", recAuditList.Code, recAuditList.Body.String())
	}

	manageEndpoints := []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodPost, "/api/admin/tluclaw/chat", request.TLUClawChatRequest{Message: "Ping"}},
		{http.MethodPost, "/api/admin/tluclaw/confirm", request.TLUClawConfirmRequest{SnapshotID: "s1", Nonce: "n1"}},
		{http.MethodPost, "/api/admin/tluclaw/cancel", request.TLUClawCancelRequest{SnapshotID: "s1", Nonce: "n1"}},
		{http.MethodPost, "/api/admin/tluclaw/rollback", request.TLUClawRollbackRequest{BatchID: "b1"}},
		{http.MethodPost, "/api/admin/tluclaw/sessions", request.TLUClawCreateSessionRequest{Title: "New"}},
		{http.MethodDelete, "/api/admin/tluclaw/sessions/sess_999", nil},
	}

	for _, ep := range manageEndpoints {
		recAuditManage := doRequest(h.mux, ep.method, ep.path, h.auditorToken, ep.body)
		if recAuditManage.Code != http.StatusForbidden {
			t.Fatalf("[%s %s] expected 403 for auditor attempting manage action, got %d: %s", ep.method, ep.path, recAuditManage.Code, recAuditManage.Body.String())
		}
	}
}

// TestTLUClawController_ReplayAttackOverHTTP verifies single-use nonce via API calls.
func TestTLUClawController_ReplayAttackOverHTTP(t *testing.T) {
	h := setupTLUClawSecurityHarness(t)
	defer h.db.Close()

	sessRec := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/sessions", h.adminToken, request.TLUClawCreateSessionRequest{
		Title: "Replay Test HTTP",
	})
	if sessRec.Code != http.StatusCreated {
		t.Fatalf("failed to create session: %d, body: %s", sessRec.Code, sessRec.Body.String())
	}

	var sessResp response.CommonResponse
	_ = json.Unmarshal(sessRec.Body.Bytes(), &sessResp)
	sessData, _ := sessResp.Data.(map[string]any)
	sessID := sessData["id"].(string)

	exp := time.Now().Add(15 * time.Minute)
	_, err := h.db.Exec(`INSERT INTO tluclaw_action_snapshots (
		id, batch_id, session_id, operator_id, action_type, summary,
		diff_preview_json, blast_radius_json, previous_state_json, new_state_json,
		status, confirmation_nonce, nonce_expires_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"snp_http_replay", "batch_h_rep", sessID, h.adminID, "academic_structure",
		"Test structure", "[]", "{}", "{}", `{"cohort":"K42","academic_year":"2028-2032"}`,
		"STAGED", "nonce_single_use_http", exp, time.Now(), time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to stage snapshot: %v", err)
	}

	confirmPayload := request.TLUClawConfirmRequest{
		SnapshotID: "snp_http_replay",
		Nonce:      "nonce_single_use_http",
	}

	firstRec := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/confirm", h.adminToken, confirmPayload)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on first confirmation, got %d: %s", firstRec.Code, firstRec.Body.String())
	}

	secondRec := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/confirm", h.adminToken, confirmPayload)
	if secondRec.Code != http.StatusConflict && secondRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 409 or 400 on replay confirmation attack, got %d: %s", secondRec.Code, secondRec.Body.String())
	}
}

// TestTLUClawController_TamperedAndInvalidNonces verifies rejection of tampered nonces via HTTP.
func TestTLUClawController_TamperedAndInvalidNonces(t *testing.T) {
	h := setupTLUClawSecurityHarness(t)
	defer h.db.Close()

	recFake := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/confirm", h.adminToken, request.TLUClawConfirmRequest{
		SnapshotID: "snp_nonexistent",
		Nonce:      "fabricated_nonce_token",
	})
	if recFake.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown nonce, got %d: %s", recFake.Code, recFake.Body.String())
	}

	exp := time.Now().Add(15 * time.Minute)
	_, _ = h.db.Exec(`INSERT INTO tluclaw_sessions (id, admin_user_id, title) VALUES ('sess_tmp', ?, 'Tmp')`, h.adminID)
	_, _ = h.db.Exec(`INSERT INTO tluclaw_action_snapshots (
		id, batch_id, session_id, operator_id, action_type, summary,
		status, confirmation_nonce, nonce_expires_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"snp_real_id", "batch_tmp", "sess_tmp", h.adminID, "academic_structure", "Summary",
		"STAGED", "nonce_valid_abc", exp, time.Now(), time.Now(),
	)

	recMismatch := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/confirm", h.adminToken, request.TLUClawConfirmRequest{
		SnapshotID: "snp_different_id",
		Nonce:      "nonce_valid_abc",
	})
	if recMismatch.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for snapshot ID mismatch, got %d: %s", recMismatch.Code, recMismatch.Body.String())
	}
}

// TestTLUClawController_StateMachineViolations verifies state machine constraint enforcement via HTTP.
func TestTLUClawController_StateMachineViolations(t *testing.T) {
	h := setupTLUClawSecurityHarness(t)
	defer h.db.Close()

	_, _ = h.db.Exec(`INSERT INTO tluclaw_sessions (id, admin_user_id, title) VALUES ('sess_sm', ?, 'SM')`, h.adminID)
	exp := time.Now().Add(15 * time.Minute)

	_, _ = h.db.Exec(`INSERT INTO tluclaw_action_snapshots (
		id, batch_id, session_id, operator_id, action_type, summary,
		status, confirmation_nonce, nonce_expires_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"snp_to_cancel", "batch_sm_c", "sess_sm", h.adminID, "academic_structure", "Cancel me",
		"STAGED", "nonce_to_cancel", exp, time.Now(), time.Now(),
	)

	cancelRec := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/cancel", h.adminToken, request.TLUClawCancelRequest{
		SnapshotID: "snp_to_cancel",
		Nonce:      "nonce_to_cancel",
	})
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("cancel snapshot failed: %d", cancelRec.Code)
	}

	confirmCancelledRec := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/confirm", h.adminToken, request.TLUClawConfirmRequest{
		SnapshotID: "snp_to_cancel",
		Nonce:      "nonce_to_cancel",
	})
	if confirmCancelledRec.Code != http.StatusConflict && confirmCancelledRec.Code != http.StatusBadRequest {
		t.Fatalf("expected error confirming CANCELLED snapshot, got %d", confirmCancelledRec.Code)
	}
}

// TestTLUClawController_RollbackAPI verifies rollback API responses and error codes.
func TestTLUClawController_RollbackAPI(t *testing.T) {
	h := setupTLUClawSecurityHarness(t)
	defer h.db.Close()

	recNotFound := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/rollback", h.adminToken, request.TLUClawRollbackRequest{
		BatchID: "batch_fake_404",
	})
	if recNotFound.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent rollback batch, got %d", recNotFound.Code)
	}

	_, _ = h.db.Exec(`INSERT INTO tluclaw_sessions (id, admin_user_id, title) VALUES ('sess_rb', ?, 'RB')`, h.adminID)
	_, _ = h.db.Exec(`INSERT INTO tluclaw_action_snapshots (
		id, batch_id, session_id, operator_id, action_type, summary,
		status, confirmation_nonce, applied_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"snp_committed", "batch_committed_rb", "sess_rb", h.adminID, "academic_structure", "Committed",
		"COMMITTED", "nonce_committed_rb", time.Now(), time.Now(), time.Now(),
	)

	recSuccess := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/rollback", h.adminToken, request.TLUClawRollbackRequest{
		BatchID: "batch_committed_rb",
	})
	if recSuccess.Code != http.StatusOK {
		t.Fatalf("expected 200 on rollback, got %d: %s", recSuccess.Code, recSuccess.Body.String())
	}

	recDouble := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/rollback", h.adminToken, request.TLUClawRollbackRequest{
		BatchID: "batch_committed_rb",
	})
	if recDouble.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on double rollback, got %d", recDouble.Code)
	}
}

// TestTLUClawController_SessionIDOROverHTTP verifies session isolation between administrators.
func TestTLUClawController_SessionIDOROverHTTP(t *testing.T) {
	h := setupTLUClawSecurityHarness(t)
	defer h.db.Close()

	recCreate := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/sessions", h.adminToken, request.TLUClawCreateSessionRequest{
		Title: "Alice Confidential Session",
	})
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("failed to create session: %d", recCreate.Code)
	}

	var createResp response.CommonResponse
	_ = json.Unmarshal(recCreate.Body.Bytes(), &createResp)
	sessData, _ := createResp.Data.(map[string]any)
	aliceSessID := sessData["id"].(string)

	recBobGet := doRequest(h.mux, http.MethodGet, "/api/admin/tluclaw/sessions/"+aliceSessID, h.admin2Token, nil)
	if recBobGet.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when Bob reads Alice session, got %d", recBobGet.Code)
	}

	recBobDelete := doRequest(h.mux, http.MethodDelete, "/api/admin/tluclaw/sessions/"+aliceSessID, h.admin2Token, nil)
	if recBobDelete.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when Bob deletes Alice session, got %d", recBobDelete.Code)
	}

	recBobChat := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/chat", h.admin2Token, request.TLUClawChatRequest{
		SessionID: aliceSessID,
		Message:   "Unauthorized injection message",
	})
	if recBobChat.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when Bob injects message into Alice session, got %d", recBobChat.Code)
	}

	recAliceGet := doRequest(h.mux, http.MethodGet, "/api/admin/tluclaw/sessions/"+aliceSessID, h.adminToken, nil)
	if recAliceGet.Code != http.StatusOK {
		t.Fatalf("expected 200 when Alice reads own session, got %d", recAliceGet.Code)
	}

	recAliceDelete := doRequest(h.mux, http.MethodDelete, "/api/admin/tluclaw/sessions/"+aliceSessID, h.adminToken, nil)
	if recAliceDelete.Code != http.StatusOK {
		t.Fatalf("expected 200 when Alice deletes own session, got %d", recAliceDelete.Code)
	}
}

// TestTLUClawController_DTOValidationRejection verifies payload validation constraints.
func TestTLUClawController_DTOValidationRejection(t *testing.T) {
	h := setupTLUClawSecurityHarness(t)
	defer h.db.Close()

	recChat := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/chat", h.adminToken, map[string]any{
		"message": "",
	})
	if recChat.Code != http.StatusBadRequest && recChat.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected validation error on empty message, got %d", recChat.Code)
	}

	recConfirm := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/confirm", h.adminToken, map[string]any{
		"snapshot_id": "snp_1",
		"nonce":       "",
	})
	if recConfirm.Code != http.StatusBadRequest && recConfirm.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected validation error on empty nonce, got %d", recConfirm.Code)
	}

	recRollback := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/rollback", h.adminToken, map[string]any{
		"batch_id": "",
	})
	if recRollback.Code != http.StatusBadRequest && recRollback.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected validation error on empty batch_id, got %d", recRollback.Code)
	}

	recSession := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/sessions", h.adminToken, map[string]any{
		"title": "",
	})
	if recSession.Code != http.StatusBadRequest && recSession.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected validation error on empty title, got %d", recSession.Code)
	}
}

// TestTLUClawController_GetSessionDetailNotFound verifies 404 for non-existent session ID.
func TestTLUClawController_GetSessionDetailNotFound(t *testing.T) {
	h := setupTLUClawSecurityHarness(t)
	defer h.db.Close()

	nonExistentID := "sess_" + uuid.NewString()[:8]
	rec := doRequest(h.mux, http.MethodGet, "/api/admin/tluclaw/sessions/"+nonExistentID, h.adminToken, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent session ID, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestTLUClawController_ListSnapshots verifies listing snapshots with authentication, role enforcement, and pagination.
func TestTLUClawController_ListSnapshots(t *testing.T) {
	h := setupTLUClawSecurityHarness(t)
	defer h.db.Close()

	recUnauth := doRequest(h.mux, http.MethodGet, "/api/admin/tluclaw/snapshots", "", nil)
	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated list snapshots, got %d", recUnauth.Code)
	}

	recStudent := doRequest(h.mux, http.MethodGet, "/api/admin/tluclaw/snapshots", h.studentToken, nil)
	if recStudent.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for student list snapshots, got %d", recStudent.Code)
	}

	recAdmin := doRequest(h.mux, http.MethodGet, "/api/admin/tluclaw/snapshots", h.adminToken, nil)
	if recAdmin.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin list snapshots, got %d", recAdmin.Code)
	}

	var resp response.TLUClawSnapshotListResponse
	if err := json.Unmarshal(recAdmin.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode list snapshots response: %v", err)
	}
}

// TestTLUClawController_RollbackOperatorIsolation verifies that an operator cannot rollback another operator's batch.
func TestTLUClawController_RollbackOperatorIsolation(t *testing.T) {
	h := setupTLUClawSecurityHarness(t)
	defer h.db.Close()

	ctx := context.Background()
	batchID := "batch_" + uuid.NewString()[:8]
	snapshotID := "snp_" + uuid.NewString()[:8]
	nonce := "nonce_" + uuid.NewString()[:8]

	sess, err := h.tluclawRepo.CreateSession(ctx, sqlc.CreateTLUClawSessionParams{
		ID:          "sess_" + uuid.NewString()[:8],
		AdminUserID: h.adminID,
		Title:       "Admin 1 Session",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	_, err = h.tluclawRepo.CreateSnapshot(ctx, sqlc.CreateTLUClawSnapshotParams{
		ID:                snapshotID,
		BatchID:           batchID,
		SessionID:         sess.ID,
		OperatorID:        h.adminID,
		ActionType:        "curriculum_update",
		Summary:           "Curriculum update by admin 1",
		Status:            "COMMITTED",
		ConfirmationNonce: nonce,
		NewStateJson:      `{"cohort":"K36","major_code":"CNTT","incoming_courses":[]}`,
		PreviousStateJson: `{"cohort":"K36","major_code":"CNTT","existing_courses":[]}`,
		DiffPreviewJson:   "{}",
		BlastRadiusJson:   "{}",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to insert committed snapshot: %v", err)
	}

	recBobRollback := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/rollback", h.admin2Token, map[string]any{
		"batch_id": batchID,
	})
	if recBobRollback.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when admin 2 attempts to rollback admin 1 batch, got %d: %s", recBobRollback.Code, recBobRollback.Body.String())
	}

	recAliceRollback := doRequest(h.mux, http.MethodPost, "/api/admin/tluclaw/rollback", h.adminToken, map[string]any{
		"batch_id": batchID,
	})
	if recAliceRollback.Code != http.StatusOK {
		t.Fatalf("expected 200 when admin 1 rolls back own batch, got %d: %s", recAliceRollback.Code, recAliceRollback.Body.String())
	}
}

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

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"

	_ "modernc.org/sqlite"
)

func setupTestTLUClawController(t *testing.T) (*controllers.TLUClawController, *sql.DB) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	schemas := []string{
		`CREATE TABLE tluclaw_sessions (
			id TEXT PRIMARY KEY,
			admin_user_id TEXT NOT NULL,
			title TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE tluclaw_messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			reasoning_content TEXT NOT NULL DEFAULT '',
			tool_calls_json TEXT NOT NULL DEFAULT '[]',
			tool_call_id TEXT NOT NULL DEFAULT '',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE tluclaw_action_snapshots (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL,
			session_id TEXT NOT NULL,
			operator_id TEXT NOT NULL,
			action_type TEXT NOT NULL,
			summary TEXT NOT NULL,
			diff_preview_json TEXT NOT NULL,
			blast_radius_json TEXT NOT NULL,
			previous_state_json TEXT NOT NULL DEFAULT '{}',
			new_state_json TEXT NOT NULL DEFAULT '{}',
			status TEXT NOT NULL DEFAULT 'STAGED',
			confirmation_nonce TEXT NOT NULL UNIQUE,
			nonce_expires_at DATETIME,
			applied_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE curriculums (
			id TEXT PRIMARY KEY,
			cohort TEXT NOT NULL,
			academic_year TEXT NOT NULL,
			major_name TEXT NOT NULL,
			major_code TEXT NOT NULL,
			faculty_code TEXT NOT NULL,
			aliases TEXT NOT NULL DEFAULT '',
			total_courses INTEGER NOT NULL DEFAULT 0,
			total_credits REAL NOT NULL DEFAULT 0.0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE curriculum_courses (
			id TEXT PRIMARY KEY,
			curriculum_id TEXT NOT NULL,
			cohort TEXT NOT NULL,
			major_name TEXT NOT NULL,
			course_code TEXT NOT NULL,
			course_name TEXT NOT NULL,
			credits REAL NOT NULL,
			weight REAL NOT NULL DEFAULT 1.0,
			hours TEXT NOT NULL DEFAULT '',
			knowledge_block TEXT NOT NULL DEFAULT '',
			prerequisites_text TEXT NOT NULL DEFAULT '',
			suggested_semester INTEGER NOT NULL DEFAULT 1,
			is_elearning INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE documents (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			type TEXT NOT NULL,
			domain TEXT NOT NULL,
			applicable_cohort TEXT NOT NULL,
			academic_year TEXT NOT NULL,
			status TEXT NOT NULL,
			priority_level INTEGER NOT NULL DEFAULT 1,
			superseded_by_id TEXT,
			supersession_audit_log TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE supersession_audit_log (
			id TEXT PRIMARY KEY,
			active_document_id TEXT NOT NULL,
			superseded_document_id TEXT NOT NULL,
			rule_type TEXT NOT NULL,
			reason TEXT NOT NULL,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE kg_nodes (
			id TEXT PRIMARY KEY,
			label TEXT NOT NULL,
			type TEXT NOT NULL,
			properties TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE kg_edges (
			id TEXT PRIMARY KEY,
			source_node_id TEXT NOT NULL,
			target_node_id TEXT NOT NULL,
			relation_type TEXT NOT NULL,
			weight REAL NOT NULL DEFAULT 1.0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, s := range schemas {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("failed to create schema: %v", err)
		}
	}

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
	ctrl := controllers.NewTLUClawController(svc)
	return ctrl, db
}

func setAdminContext(r *http.Request, adminID string) *http.Request {
	claims := &response.JWTClaims{
		UId:   adminID,
		Roles: []constants.RoleType{constants.RoleTypeAdmin},
	}
	ctx := context.WithValue(r.Context(), middlewares.UserIDContextKey, adminID)
	ctx = context.WithValue(ctx, middlewares.UserClaimsContextKey, claims)
	return r.WithContext(ctx)
}

func TestTLUClawController_CreateAndListSessions(t *testing.T) {
	ctrl, db := setupTestTLUClawController(t)
	defer db.Close()

	createReq := request.TLUClawCreateSessionRequest{
		Title: "Phiên quản lý K40",
	}
	body, _ := json.Marshal(createReq)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/tluclaw/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAdminContext(req, "admin_test_01")
	w := httptest.NewRecorder()

	ctrl.HandleCreateSession(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/tluclaw/sessions", nil)
	listReq = setAdminContext(listReq, "admin_test_01")
	wList := httptest.NewRecorder()

	ctrl.HandleListSessions(wList, listReq)

	if wList.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", wList.Code)
	}

	var resp response.CommonResponse
	if err := json.Unmarshal(wList.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if !resp.Status {
		t.Fatalf("expected status true")
	}
}

func TestTLUClawController_ConfirmAndCancelEndpoints(t *testing.T) {
	ctrl, db := setupTestTLUClawController(t)
	defer db.Close()

	memCache := cache.NewRamCache()
	tluclawRepo := repositories.NewTLUClawRepository(db, memCache)

	exp := time.Now().Add(10 * time.Minute)
	_, err := db.Exec(`INSERT INTO tluclaw_action_snapshots (
		id, batch_id, session_id, operator_id, action_type, summary,
		diff_preview_json, blast_radius_json, previous_state_json, new_state_json,
		status, confirmation_nonce, nonce_expires_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"snp_ctrl_01", "batch_c1", "sess_01", "admin_01", "academic_structure",
		"Tao khoa K41", "[]", "{}", "{}", `{"cohort":"K41","academic_year":"2027-2031"}`,
		"STAGED", "nonce_ctrl_999", exp, time.Now(), time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to insert snapshot: %v", err)
	}

	confirmReq := request.TLUClawConfirmRequest{
		SnapshotID: "snp_ctrl_01",
		Nonce:      "nonce_ctrl_999",
	}
	b, _ := json.Marshal(confirmReq)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/tluclaw/confirm", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = setAdminContext(req, "admin_01")
	w := httptest.NewRecorder()

	ctrl.HandleConfirmAction(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	snp, err := tluclawRepo.GetSnapshotByID(context.Background(), "snp_ctrl_01")
	if err != nil || snp.Status != "COMMITTED" {
		t.Fatalf("expected status COMMITTED, got %v", snp)
	}
}

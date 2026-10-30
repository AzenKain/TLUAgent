package services_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func setupCommitTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:tluclaw_commit_test?mode=memory&cache=shared&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}
	for _, u := range []string{"admin_01", "stu_k40_a", "stu_k40_b", "stu_k41_c"} {
		if _, err := db.Exec(`INSERT OR IGNORE INTO users (id, email, full_name, password_hash) VALUES (?, ? || '@tlu.edu.vn', 'Test', 'hash')`, u, u); err != nil {
			t.Fatalf("failed to seed user: %v", err)
		}
	}
	for _, p := range [][2]string{{"stu_k40_a", "K40"}, {"stu_k40_b", "K40"}, {"stu_k41_c", "K41"}} {
		if _, err := db.Exec(`INSERT OR REPLACE INTO student_academic_profiles (user_id, major, cohort) VALUES (?, 'CNTT', ?)`, p[0], p[1]); err != nil {
			t.Fatalf("failed to seed profile: %v", err)
		}
	}
	return db
}

type commitFixture struct {
	db    *sql.DB
	svc   services.TLUClawService
	snap  *models.TLUClawActionSnapshot
	nonce string
}

func newCommitFixture(t *testing.T) *commitFixture {
	t.Helper()
	db := setupCommitTestDB(t)
	memCache := cache.NewRamCache()
	tluclawRepo := repositories.NewTLUClawRepository(db, memCache)
	currRepo := repositories.NewCurriculumRepository(db, memCache)
	ragRepo := repositories.NewRAGRepository(db, memCache)
	notifRepo := repositories.NewNotificationRepository(db, memCache)
	memoryRepo := repositories.NewMemoryRepository(db, memCache)

	svc := services.NewTLUClawService(services.TLUClawParams{
		TLUClawRepo:    tluclawRepo,
		CurriculumRepo: currRepo,
		RAGRepo:        ragRepo,
		RAGService:     services.NewRAGService(ragRepo, memCache, nil, nil),
		RAMCache:       memCache,
		NotifRepo:      notifRepo,
		MemoryRepo:     memoryRepo,
		TxManager:      database.NewTxManager(db),
	})

	sess, err := svc.CreateSession(context.Background(), "admin_01", "Commit test")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	nonce := "nonce_commit_test"
	exp := time.Now().Add(10 * time.Minute)
	snap, err := tluclawRepo.CreateSnapshot(context.Background(), sqlc.CreateTLUClawSnapshotParams{
		ID:              "snp_commit_01",
		BatchID:         "batch_commit_01",
		SessionID:       sess.ID,
		OperatorID:      "admin_01",
		ActionType:      "curriculum_update",
		Summary:         "Cập nhật CTĐT K40",
		DiffPreviewJson: `[]`,
		BlastRadiusJson: `{"affected_cohorts":["K40"],"risk_level":"LOW","impact_summary":"Cập nhật K40"}`,
		PreviousStateJson: `{"cohort":"K40","major_code":"CNTT","existing_courses":[]}`,
		NewStateJson: `{"cohort":"K40","major_code":"CNTT","incoming_courses":[
			{"course_code":"CS101","course_name":"Nhập môn CNTT","credits":3.0,"hours":"45","suggested_semester":1},
			{"course_code":"CS102","course_name":"Lập trình nâng cao","credits":3.0,"hours":"45","suggested_semester":2}]}`,
		Status:            "STAGED",
		ConfirmationNonce: nonce,
		NonceExpiresAt:    sql.NullTime{Time: exp, Valid: true},
		AppliedAt:         sql.NullTime{Valid: false},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	return &commitFixture{db: db, svc: svc, snap: snap, nonce: nonce}
}

func TestTLUClawCommitNotifiesAffectedStudents(t *testing.T) {
	f := newCommitFixture(t)
	defer f.db.Close()
	ctx := context.Background()

	resp, err := f.svc.ConfirmSnapshot(ctx, "admin_01", &request.TLUClawConfirmRequest{
		SnapshotID: f.snap.ID,
		Nonce:      f.nonce,
	})
	if err != nil {
		t.Fatalf("confirm snapshot: %v", err)
	}
	if resp.Status != "COMMITTED" {
		t.Fatalf("expected COMMITTED, got %s", resp.Status)
	}

	for _, userID := range []string{"stu_k40_a", "stu_k40_b"} {
		var count int
		if err := f.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND type = ?`,
			userID, "CURRICULUM_UPDATED").Scan(&count); err != nil {
			t.Fatalf("query notifications: %v", err)
		}
		if count == 0 {
			t.Fatalf("expected a CURRICULUM_UPDATED notification for %s", userID)
		}
	}

	var other int
	if err := f.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_notifications WHERE user_id = 'stu_k41_c'`).Scan(&other); err != nil {
		t.Fatalf("query notifications: %v", err)
	}
	if other != 0 {
		t.Fatalf("student outside affected cohorts must not be notified, got %d", other)
	}
}

func TestTLUClawCommitRollsBackOnFailure(t *testing.T) {
	f := newCommitFixture(t)
	defer f.db.Close()
	ctx := context.Background()

	if _, err := f.db.ExecContext(ctx, `CREATE TRIGGER fail_cs102 BEFORE INSERT ON curriculum_courses
		WHEN NEW.course_code = 'CS102' BEGIN SELECT RAISE(ABORT, 'simulated failure'); END;`); err != nil {
		t.Fatalf("failed to create failure trigger: %v", err)
	}

	if _, err := f.svc.ConfirmSnapshot(ctx, "admin_01", &request.TLUClawConfirmRequest{
		SnapshotID: f.snap.ID,
		Nonce:      f.nonce,
	}); err == nil {
		t.Fatalf("expected commit to fail when a course write fails")
	}

	var courses int
	if err := f.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM curriculum_courses`).Scan(&courses); err != nil {
		t.Fatalf("query courses: %v", err)
	}
	if courses != 0 {
		t.Fatalf("expected no courses to be written, got %d", courses)
	}

	var status string
	if err := f.db.QueryRowContext(ctx, `SELECT status FROM tluclaw_action_snapshots WHERE id = ?`, f.snap.ID).Scan(&status); err != nil {
		t.Fatalf("query snapshot: %v", err)
	}
	if status != "STAGED" {
		t.Fatalf("expected snapshot to stay STAGED, got %s", status)
	}

	var notifs int
	if err := f.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_notifications`).Scan(&notifs); err != nil {
		t.Fatalf("query notifications: %v", err)
	}
	if notifs != 0 {
		t.Fatalf("expected no notifications on rollback, got %d", notifs)
	}
}

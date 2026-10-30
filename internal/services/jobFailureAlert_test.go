package services_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/database"
	"tluagent-web/pkg/worker"

	_ "modernc.org/sqlite"
)

func TestJobService_RepeatedFailureAlertsAdmins(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO users (id, email, full_name, password_hash) VALUES ('admin_fail', 'a@tlu.edu.vn', 'Admin', 'h')`); err != nil {
		t.Fatalf("failed to seed admin: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES ('admin_fail', '01920000-0000-7000-8000-000000000001')`); err != nil {
		t.Fatalf("failed to assign admin role: %v", err)
	}

	ramCache := cache.NewRamCache()
	jobRepo := repositories.NewJobRepository(db, ramCache)
	notifRepo := repositories.NewNotificationRepository(db, ramCache)
	userRepo := repositories.NewUserRepository(db, ramCache)
	queue := worker.NewQueue(1)
	defer queue.Stop()

	svc := services.NewJobService(jobRepo, queue, nil, nil, nil)
	svc.SetFailureNotifier(notifRepo, userRepo)

	if _, err := jobRepo.CreateJob(ctx, "job_fail_1", constants.JobTypeCrawlNews, constants.JobStatusFailed, "{}"); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}
	if _, err := jobRepo.CreateJob(ctx, "job_fail_2", constants.JobTypeCrawlNews, constants.JobStatusPending, "{}"); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}
	if err := svc.Failed(ctx, worker.Job{ID: "job_fail_2", Type: constants.JobTypeCrawlNews}, errors.New("boom")); err != nil {
		t.Fatalf("failed lifecycle returned error: %v", err)
	}

	var alerts int
	if err := db.QueryRow(`SELECT COUNT(*) FROM user_notifications WHERE user_id = 'admin_fail' AND type = ?`, constants.NotifTypeJobFailure).Scan(&alerts); err != nil {
		t.Fatalf("query notifications: %v", err)
	}
	if alerts != 1 {
		t.Fatalf("expected 1 failure alert after 2 consecutive failures, got %d", alerts)
	}

	if _, err := jobRepo.CreateJob(ctx, "job_ok_1", constants.JobTypeCrawlNews, constants.JobStatusCompleted, "{}"); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}
	if _, err := jobRepo.CreateJob(ctx, "job_fail_3", constants.JobTypeCrawlNews, constants.JobStatusPending, "{}"); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}
	if err := svc.Failed(ctx, worker.Job{ID: "job_fail_3", Type: constants.JobTypeCrawlNews}, errors.New("boom")); err != nil {
		t.Fatalf("failed lifecycle returned error: %v", err)
	}

	if err := db.QueryRow(`SELECT COUNT(*) FROM user_notifications WHERE user_id = 'admin_fail' AND type = ?`, constants.NotifTypeJobFailure).Scan(&alerts); err != nil {
		t.Fatalf("query notifications: %v", err)
	}
	if alerts != 1 {
		t.Fatalf("expected no new alert when the previous run succeeded, got %d", alerts)
	}

	var kind string
	if err := db.QueryRow(`SELECT type FROM user_notifications WHERE user_id = 'admin_fail' LIMIT 1`).Scan(&kind); err != nil {
		t.Fatalf("query notification type: %v", err)
	}
	if kind != constants.NotifTypeJobFailure {
		t.Fatalf("unexpected notification type %q", kind)
	}
}

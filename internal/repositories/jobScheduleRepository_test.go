package repositories_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"

	_ "modernc.org/sqlite"
)

func newScheduleRepo(t *testing.T) (*sql.DB, repositories.JobScheduleRepository) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)&_time_format=datetime&_timezone=UTC")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}
	return db, repositories.NewJobScheduleRepository(db, cache.NewRamCache())
}

func createSchedule(t *testing.T, repo repositories.JobScheduleRepository, id string, nextRun time.Time) {
	t.Helper()
	if _, err := repo.Create(context.Background(), sqlc.CreateJobScheduleParams{
		ID:              id,
		Name:            "Test schedule " + id,
		TaskType:        "crawl_news",
		PayloadJson:     sql.NullString{String: "{}", Valid: true},
		IntervalMinutes: 60,
		Enabled:         1,
		NextRunAt:       nextRun,
	}); err != nil {
		t.Fatalf("failed to create schedule %s: %v", id, err)
	}
}

func TestJobScheduleStoresSortableTimestamps(t *testing.T) {
	db, repo := newScheduleRepo(t)
	createSchedule(t, repo, "sched-format", time.Now().Add(2*time.Hour))

	var raw string
	if err := db.QueryRow(`SELECT cast(next_run_at AS TEXT) FROM job_schedules WHERE id = 'sched-format'`).Scan(&raw); err != nil {
		t.Fatalf("failed to read stored timestamp: %v", err)
	}
	if strings.Contains(raw, " m=") {
		t.Fatalf("stored timestamp keeps a monotonic reading and breaks SQL comparisons: %q", raw)
	}
	if strings.Contains(raw, "T") || strings.Contains(raw, "Z") {
		t.Fatalf("expected sqlite datetime text format so SQL comparisons match datetime('now'), got %q", raw)
	}
	if len(raw) != len("2006-01-02 15:04:05") {
		t.Fatalf("unexpected datetime layout: %q", raw)
	}
}

func TestJobScheduleClaimHonoursDueTime(t *testing.T) {
	_, repo := newScheduleRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()

	createSchedule(t, repo, "sched-future", now.Add(2*time.Hour))
	createSchedule(t, repo, "sched-due", now.Add(-time.Minute))

	claimed, err := repo.Claim(ctx, "sched-future", "job-future", now, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("claim future schedule: %v", err)
	}
	if claimed {
		t.Fatal("schedule due in two hours must not be claimed")
	}

	claimed, err = repo.Claim(ctx, "sched-due", "job-due", now, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("claim due schedule: %v", err)
	}
	if !claimed {
		t.Fatal("overdue schedule must be claimed")
	}

	var lastJobID sql.NullString
	if err := readLastJobID(ctx, repo, "sched-due", &lastJobID); err != nil {
		t.Fatalf("failed to read schedule: %v", err)
	}
	if lastJobID.String != "job-due" {
		t.Fatalf("expected claimed job to be recorded, got %q", lastJobID.String)
	}
}

func readLastJobID(ctx context.Context, repo repositories.JobScheduleRepository, id string, out *sql.NullString) error {
	entity, err := repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if entity.LastJobID == nil {
		out.String = ""
		out.Valid = false
		return nil
	}
	out.String = *entity.LastJobID
	out.Valid = true
	return nil
}

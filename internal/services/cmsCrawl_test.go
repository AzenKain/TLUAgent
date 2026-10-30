package services_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/database"
	"tluagent-web/pkg/worker"

	_ "modernc.org/sqlite"
)

func newCMSCrawlFixture(t *testing.T) (*sql.DB, *worker.Queue, repositories.JobRepository) {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "crawl.db") + "?_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	ramCache := cache.NewRamCache()
	jobRepo := repositories.NewJobRepository(db, ramCache)
	ragRepo := repositories.NewRAGRepository(db, ramCache)
	ragSvc := services.NewRAGService(ragRepo, nil, nil, nil)

	queue := worker.NewQueue(1)
	_ = services.NewJobService(jobRepo, queue, nil, ragSvc, nil)
	queue.Start()

	t.Cleanup(func() {
		queue.Stop()
		_ = db.Close()
	})
	return db, queue, jobRepo
}

func runCrawlJob(t *testing.T, queue *worker.Queue, jobRepo repositories.JobRepository, jobID, jobType string) string {
	t.Helper()
	ctx := context.Background()
	if _, err := jobRepo.CreateJob(ctx, jobID, jobType, constants.JobStatusPending, "{}"); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}
	if err := queue.EnqueueExisting(ctx, worker.Job{ID: jobID, Type: jobType, Payload: "{}"}); err != nil {
		t.Fatalf("failed to enqueue job: %v", err)
	}

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		job, err := jobRepo.GetJob(ctx, jobID)
		if err != nil {
			t.Fatalf("failed to read job: %v", err)
		}
		if job.Status == constants.JobStatusCompleted || job.Status == constants.JobStatusFailed {
			return job.Status
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("job %s did not finish in time", jobID)
	return ""
}

func cmsPostsHandler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/posts/":
			_, _ = w.Write([]byte(`{"data": [{
				"id": "522",
				"title": "Ký kết hợp tác đào tạo quốc tế",
				"slug": "ky-ket-hop-tac-dao-tao-quoc-te",
				"schedule": "2026-10-28T04:39:43.139000Z",
				"content": {"type": "doc", "content": []}
			}], "pagination": {"total": 1, "skip": 0, "limit": 100, "has_more": false}}`))
		case "/api/v1/posts/522/":
			_, _ = w.Write([]byte(`{"data": {
				"id": "522",
				"title": "Ký kết hợp tác đào tạo quốc tế",
				"slug": "ky-ket-hop-tac-dao-tao-quoc-te",
				"schedule": "2026-10-28T04:39:43.139000Z",
				"content": {"type": "doc", "content": [
					{"type": "paragraph", "content": [{"type": "text", "text": "Ngày 27/10/2026, Trường Đại học Thăng Long ký kết hợp tác đào tạo với đối tác Nhật Bản về trao đổi sinh viên và công nhận tín chỉ giữa hai bên."}]}
				]}
			}}`))
		case "/api/v1/events/":
			_, _ = w.Write([]byte(`{"data": [], "pagination": {"total": 0, "skip": 0, "limit": 100, "has_more": false}}`))
		default:
			http.NotFound(w, r)
		}
	}
}

func TestJobService_CrawlNewsIngestsCMSItems(t *testing.T) {
	server := httptest.NewServer(cmsPostsHandler(t))
	defer server.Close()
	t.Setenv("CMS_API_BASE_URL", server.URL+"/api/v1")
	t.Setenv("CMS_SITE_URL", "https://cms.example.test")

	db, queue, jobRepo := newCMSCrawlFixture(t)

	if status := runCrawlJob(t, queue, jobRepo, "job_cms_news", constants.JobTypeCrawlNews); status != constants.JobStatusCompleted {
		t.Fatalf("expected crawl job to complete, got %s", status)
	}

	var (
		title, url, docType string
		chunks              int
	)
	if err := db.QueryRow(`
		SELECT title, file_path, type,
		       (SELECT COUNT(*) FROM document_chunks c WHERE c.document_id = d.id)
		FROM documents d WHERE id = 'DOC_TIN_TUC_522'`).Scan(&title, &url, &docType, &chunks); err != nil {
		t.Fatalf("expected crawled document to be stored: %v", err)
	}
	if docType != constants.DocTypeNews {
		t.Fatalf("expected news document type, got %s", docType)
	}
	wantURL := "https://cms.example.test/vi/tin-tuc-su-kien/tin-tuc/ky-ket-hop-tac-dao-tao-quoc-te"
	if url != wantURL {
		t.Fatalf("expected canonical CMS url %q, got %q", wantURL, url)
	}
	if chunks == 0 {
		t.Fatal("expected document chunks to be created")
	}

	var nodes int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kg_nodes WHERE id = 'doc:DOC_TIN_TUC_522'`).Scan(&nodes); err != nil {
		t.Fatalf("failed to count KG nodes: %v", err)
	}
	if nodes != 1 {
		t.Fatalf("expected knowledge graph node for crawled document, got %d", nodes)
	}

	if status := runCrawlJob(t, queue, jobRepo, "job_cms_news_again", constants.JobTypeCrawlNews); status != constants.JobStatusCompleted {
		t.Fatalf("expected second crawl job to complete, got %s", status)
	}
	var docs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM documents WHERE title = ?`, title).Scan(&docs); err != nil {
		t.Fatalf("failed to count documents: %v", err)
	}
	if docs != 1 {
		t.Fatalf("expected re-crawl to skip existing document, got %d copies", docs)
	}
}

func TestJobService_CrawlFailsWhenCMSReturnsNothing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": [], "pagination": {"total": 0, "skip": 0, "limit": 100, "has_more": false}}`))
	}))
	defer server.Close()
	t.Setenv("CMS_API_BASE_URL", server.URL+"/api/v1")
	t.Setenv("CMS_SITE_URL", "https://cms.example.test")

	db, queue, jobRepo := newCMSCrawlFixture(t)

	status := runCrawlJob(t, queue, jobRepo, "job_cms_empty", constants.JobTypeCrawlStaticPages)
	if status != constants.JobStatusFailed {
		t.Fatalf("expected empty CMS response to fail the job, got %s", status)
	}

	var msg sql.NullString
	if err := db.QueryRow(`SELECT error_msg FROM jobs WHERE id = 'job_cms_empty'`).Scan(&msg); err != nil {
		t.Fatalf("failed to read job error: %v", err)
	}
	if !msg.Valid || msg.String == "" {
		t.Fatal("expected the failure to be recorded with a reason")
	}
}

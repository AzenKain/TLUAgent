package crawler

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"

	"tluagent-web/pkg/netx"
)

func TestCrawler_DiscoverMaxPageAndAutoWait(t *testing.T) {
	netx.AllowPrivateIPsInTest = true
	defer func() { netx.AllowPrivateIPsInTest = false }()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
			<html>
				<body>
					<ul class="pager">
						<li><a href="/thong-bao?page=0">1</a></li>
						<li><a href="/thong-bao?page=1">2</a></li>
						<li><a href="/thong-bao?page=377">378</a></li>
					</ul>
				</body>
			</html>
		`))
	}))
	defer server.Close()

	c, err := NewCrawler(Config{
		BaseURL:          server.URL,
		TimeoutSec:       5,
		DelayMs:          10,
		MaxRetries:       3,
		RetryBaseDelayMs: 20,
	})
	if err != nil {
		t.Fatalf("failed to create crawler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	maxPage := c.DiscoverMaxPage(ctx, "thong-bao", "page")
	if maxPage != 377 {
		t.Errorf("expected maxPage 377, got %d", maxPage)
	}
	if callCount < 2 {
		t.Errorf("expected at least 2 calls due to retry on 429, got %d", callCount)
	}
}

func TestCrawler_CollisionProofDocID(t *testing.T) {
	url1 := "https://cms.example.test/thong-bao/hoc-phi-hoc-ky-1-nam-hoc-2025-2026-1234.html"
	url2 := "https://cms.example.test/thong-bao/hoc-phi-hoc-ky-2-nam-hoc-2025-2026-1234.html"

	nodeID1 := extractNodeID(url1)
	nodeID2 := extractNodeID(url2)
	hash1 := fmt.Sprintf("%x", sha256.Sum256([]byte(url1)))[:8]
	hash2 := fmt.Sprintf("%x", sha256.Sum256([]byte(url2)))[:8]

	docID1 := fmt.Sprintf("DOC_THONG_BAO_%s_%s", nodeID1, hash1)
	docID2 := fmt.Sprintf("DOC_THONG_BAO_%s_%s", nodeID2, hash2)

	if docID1 == docID2 {
		t.Errorf("expected collision-proof IDs to differ, got identical: %s", docID1)
	}
}

func TestCrawler_InMemoryStreaming(t *testing.T) {
	netx.AllowPrivateIPsInTest = true
	defer func() { netx.AllowPrivateIPsInTest = false }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if r.URL.Path == "/thong-bao" {
			if r.URL.Query().Get("page") != "" && r.URL.Query().Get("page") != "0" {
				_, _ = w.Write([]byte(`<html><body></body></html>`))
				return
			}
			_, _ = w.Write([]byte(`
				<html>
					<body>
						<a href="/thong-bao/thong-bao-hoc-phi-101.html">Thông báo học phí HK1</a>
					</body>
				</html>
			`))
			return
		}
		_, _ = w.Write([]byte(`
			<html>
				<body>
					<h1>Thông báo học phí HK1</h1>
					<p>Học phí kỳ 1 bắt đầu nộp từ ngày 01/09/2026.</p>
				</body>
			</html>
		`))
	}))
	defer server.Close()

	c, err := NewCrawler(Config{
		BaseURL:    server.URL,
		TimeoutSec: 5,
		DelayMs:    5,
	})
	if err != nil {
		t.Fatalf("failed to create crawler: %v", err)
	}

	var streamedDocs []*CrawledDocument
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = c.CrawlPagedCategory(ctx, "thong-bao", "THONG_BAO", 0, nil, func(ctx context.Context, doc *CrawledDocument) error {
		streamedDocs = append(streamedDocs, doc)
		return nil
	}, nil)

	if err != nil {
		t.Fatalf("crawl failed: %v", err)
	}
	if len(streamedDocs) != 1 {
		t.Fatalf("expected 1 streamed document, got %d", len(streamedDocs))
	}
	if streamedDocs[0].Title != "Thông báo học phí HK1" {
		t.Errorf("unexpected doc title: %s", streamedDocs[0].Title)
	}
}

func TestCrawler_ResolveBrowserIdentity(t *testing.T) {
	tests := []struct {
		name         string
		osName       string
		customUA     string
		expectedSub  string
		expectedPlat string
	}{
		{
			name:         "Linux default",
			osName:       "linux",
			customUA:     "",
			expectedSub:  "X11; Linux x86_64",
			expectedPlat: "\"Linux\"",
		},
		{
			name:         "Windows default",
			osName:       "windows",
			customUA:     "",
			expectedSub:  "Windows NT 10.0; Win64; x64",
			expectedPlat: "\"Windows\"",
		},
		{
			name:         "macOS default",
			osName:       "darwin",
			customUA:     "",
			expectedSub:  "Macintosh; Intel Mac OS X 10_15_7",
			expectedPlat: "\"macOS\"",
		},
		{
			name:         "Custom UA with Windows",
			osName:       "linux",
			customUA:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) CustomBrowser/1.0",
			expectedSub:  "CustomBrowser/1.0",
			expectedPlat: "\"Windows\"",
		},
		{
			name:         "Custom UA with Mac",
			osName:       "linux",
			customUA:     "Mozilla/5.0 (Macintosh; Intel Mac OS X) CustomBrowser/1.0",
			expectedSub:  "CustomBrowser/1.0",
			expectedPlat: "\"macOS\"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ua, platform := ResolveBrowserIdentity(tc.osName, tc.customUA)
			if tc.expectedSub != "" && !strings.Contains(ua, tc.expectedSub) {
				t.Errorf("expected UA to contain %q, got %q", tc.expectedSub, ua)
			}
			if platform != tc.expectedPlat {
				t.Errorf("expected platform %q, got %q", tc.expectedPlat, platform)
			}
		})
	}
}

func TestCrawler_ExtractTitle_SearchboxSuppression(t *testing.T) {
	htmlContent := `
		<!DOCTYPE html>
		<html>
		<head>
			<meta property="og:title" content="THONG BAO HOC PHI 2026 | Trường Đại học Thăng Long" />
			<title>THONG BAO HOC PHI 2026 | Trường Đại học Thăng Long</title>
		</head>
		<body>
			<div class="searchbox">
				<h1 class="searchbox__title">Tìm kiếm</h1>
			</div>
			<div class="new-detail__header">
				<h1 class="new-detail__title">THONG BAO HOC PHI 2026</h1>
			</div>
		</body>
		</html>
	`
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("failed to parse html: %v", err)
	}

	title := extractTitle(doc)
	if title != "THONG BAO HOC PHI 2026" {
		t.Errorf("expected authentic title, got %q", title)
	}
}

func TestCrawler_MaxTimeDaysFilter(t *testing.T) {
	netx.AllowPrivateIPsInTest = true
	defer func() { netx.AllowPrivateIPsInTest = false }()

	today := time.Now().Format("2006-01-02")
	oldDate := time.Now().AddDate(0, 0, -60).Format("2006-01-02")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		if strings.Contains(r.URL.Path, "node/recent") {
			_, _ = fmt.Fprintf(w, `<html><body><h1>Recent Notice</h1><p>Ngay: %s</p><div>Content recent</div></body></html>`, today)
			return
		}
		if strings.Contains(r.URL.Path, "node/old") {
			_, _ = fmt.Fprintf(w, `<html><body><h1>Old Notice</h1><p>Ngay: %s</p><div>Content old</div></body></html>`, oldDate)
			return
		}

		if r.URL.Path == "/thong-bao" {
			page := r.URL.Query().Get("page")
			if page == "0" {
				_, _ = w.Write([]byte(`
					<html><body>
						<a href="/node/recent">Recent Notice</a>
						<a href="/node/old">Old Notice</a>
						<a href="/thong-bao?page=1">2</a>
					</body></html>
				`))
				return
			}
			_, _ = w.Write([]byte(`<html><body><a href="/node/old">Old Notice</a></body></html>`))
			return
		}
	}))
	defer server.Close()

	c, err := NewCrawler(Config{
		BaseURL:     server.URL,
		DelayMs:     5,
		MaxTimeDays: 30,
	})
	if err != nil {
		t.Fatalf("failed to create crawler: %v", err)
	}

	var processed []string
	handler := func(ctx context.Context, doc *CrawledDocument) error {
		processed = append(processed, doc.Title)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = c.CrawlPagedCategory(ctx, "thong-bao", "notice", 1, nil, handler, nil)
	if err != nil {
		t.Fatalf("crawl failed: %v", err)
	}

	if len(processed) != 2 {
		t.Fatalf("expected 2 documents (boundary + recent), got %d", len(processed))
	}
	if !strings.Contains(processed[0], "Old") {
		t.Errorf("expected boundary document (< cutoff) processed first, got %s", processed[0])
	}
	if !strings.Contains(processed[1], "Recent") {
		t.Errorf("expected recent document processed last, got %s", processed[1])
	}
}

func TestCrawler_FetchListingItems_Deduplication(t *testing.T) {
	netx.AllowPrivateIPsInTest = true
	defer func() { netx.AllowPrivateIPsInTest = false }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`
			<html>
				<body>
					<div class="item">
						<a href="/node/123.html"><img src="thumb.jpg" alt="Thumb"/></a>
						<h3><a href="/node/123.html">Main Article Title For Node 123</a></h3>
						<span class="date">15/10/2026</span>
						<a href="/node/123.html">Xem them</a>
					</div>
					<div class="item">
						<h3><a href="/node/456.html">Second Article Title</a></h3>
					</div>
				</body>
			</html>
		`))
	}))
	defer server.Close()

	c, err := NewCrawler(Config{
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create crawler: %v", err)
	}

	items, err := c.fetchListingItems(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected exactly 2 deduplicated items, got %d", len(items))
	}
	if items[0].Title != "Main Article Title For Node 123" {
		t.Errorf("expected best non-generic title, got %q", items[0].Title)
	}
	if items[0].Date != "2026-10-15" {
		t.Errorf("expected date parsed from container, got %q", items[0].Date)
	}
}

package crawler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tluagent-web/pkg/crawler"
)

func newCMSTestServer(t *testing.T, handler http.HandlerFunc) *crawler.CMSClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := crawler.NewCMSClient(crawler.CMSConfig{
		BaseURL: server.URL + "/api/v1",
		SiteURL: "https://cms.example.test",
		Locale:  "vi",
	})
	if err != nil {
		t.Fatalf("failed to build CMS client: %v", err)
	}
	return client
}

func TestCMSClient_ListMapsMetadataAndPagination(t *testing.T) {
	client := newCMSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/announcements/" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("locale"); got != "vi" {
			t.Fatalf("unexpected locale %q", got)
		}
		if got := r.URL.Query().Get("skip"); got != "0" {
			t.Fatalf("expected first page to skip 0, got %q", got)
		}
		if got := r.URL.Query().Get("limit"); got == "" {
			t.Fatal("expected limit to be sent")
		}
		_, _ = w.Write([]byte(`{
			"data": [{
				"id": 2326,
				"title": "Thông báo đổi phòng học",
				"slug": "thong-bao-doi-phong-hoc",
				"date": "2026-10-29T09:45:10.283000Z",
				"content": {"type": "doc", "content": []},
				"main_doc": "https://drive.google.com/file/d/abc123/view"
			}],
			"pagination": {"total": 2324, "skip": 0, "limit": 100, "has_more": true}
		}`))
	})

	items, pagination, err := client.List(context.Background(), crawler.CMSCollectionAnnouncements, 0)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ID != "2326" {
		t.Fatalf("expected numeric id to be preserved, got %q", items[0].ID)
	}
	if items[0].PublishDate != "2026-10-29" {
		t.Fatalf("expected date part, got %q", items[0].PublishDate)
	}
	wantURL := "https://cms.example.test/vi/thong-bao/thong-bao-doi-phong-hoc"
	if items[0].URL != wantURL {
		t.Fatalf("expected %q, got %q", wantURL, items[0].URL)
	}
	if items[0].AttachmentURL == "" {
		t.Fatal("expected attachment url to be captured")
	}
	if pagination.Total != 2324 || !pagination.HasMore {
		t.Fatalf("unexpected pagination: %+v", pagination)
	}
}

func TestCMSClient_ListAcceptsBareArrayForPages(t *testing.T) {
	client := newCMSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/pages/" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{
			"id": "7",
			"name": "Tài liệu học tập",
			"code": "tai-lieu-hoc-tap",
			"page_data": {"content": [
				{"type": "TextBlock", "props": {"text": "Sinh viên tra cứu tài liệu học tập theo từng học phần tại đây."}},
				{"type": "LearningMaterialBanner", "props": {"image": {"src": "https://media.example.test/x.png", "alt": "banner"}}}
			]}
		}]`))
	})

	items, _, err := client.List(context.Background(), crawler.CMSCollectionPages, 0)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Title != "Tài liệu học tập" {
		t.Fatalf("unexpected title %q", items[0].Title)
	}
	if items[0].URL != "https://cms.example.test/vi/tai-lieu-hoc-tap" {
		t.Fatalf("unexpected url %q", items[0].URL)
	}
	if !strings.Contains(items[0].Body, "tra cứu tài liệu học tập") {
		t.Fatalf("expected page text body, got %q", items[0].Body)
	}
	if strings.Contains(items[0].Body, "media-cms") {
		t.Fatalf("expected media url to be skipped, got %q", items[0].Body)
	}
}

func TestCMSClient_DetailExtractsRichTextBody(t *testing.T) {
	client := newCMSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/posts/522/" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data": {
			"id": "522",
			"title": "Ký kết hợp tác",
			"slug": "ky-ket-hop-tac",
			"schedule": "2026-10-28T04:39:43.139000Z",
			"content": {"type": "doc", "content": [
				{"type": "heading", "content": [{"type": "text", "text": "Lễ ký kết"}]},
				{"type": "paragraph", "content": [{"type": "text", "text": "Ngày 27/10/2026, nhà trường đón tiếp đoàn đại biểu."}]},
				{"type": "bulletList", "content": [
					{"type": "listItem", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Hợp tác đào tạo"}]}]}
				]}
			]
		}}}`))
	})

	item, err := client.Detail(context.Background(), crawler.CMSCollectionPosts, "522")
	if err != nil {
		t.Fatalf("detail failed: %v", err)
	}
	if item.PublishDate != "2026-10-28" {
		t.Fatalf("unexpected publish date %q", item.PublishDate)
	}
	for _, want := range []string{"Lễ ký kết", "Ngày 27/10/2026", "Hợp tác đào tạo"} {
		if !strings.Contains(item.Body, want) {
			t.Fatalf("expected body to contain %q, got %q", want, item.Body)
		}
	}
	if item.URL != "https://cms.example.test/vi/tin-tuc-su-kien/tin-tuc/ky-ket-hop-tac" {
		t.Fatalf("unexpected url %q", item.URL)
	}
}

func TestCMSClient_ListFailsOnServerError(t *testing.T) {
	var attempts int
	client := newCMSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"detail":"boom"}`))
	})

	if _, _, err := client.List(context.Background(), crawler.CMSCollectionPosts, 0); err == nil {
		t.Fatal("expected list to fail on 500 responses")
	}
	if attempts < 2 {
		t.Fatalf("expected retries on 5xx, saw %d attempt(s)", attempts)
	}
}

func TestCMSDocIDAndCutoff(t *testing.T) {
	if got := crawler.CMSDocID("DOC_THONG_BAO_", "2326"); got != "DOC_THONG_BAO_2326" {
		t.Fatalf("unexpected doc id %q", got)
	}
	if got := crawler.CMSDocID("DOC_TIN_TUC_", "abc/def"); got != "DOC_TIN_TUC_abc-def" {
		t.Fatalf("unexpected sanitized doc id %q", got)
	}

	item := &crawler.CMSItem{PublishDate: "2026-10-20"}
	if crawler.CMSItemOlderThan(item, mustDate(t, "2026-10-25")) != true {
		t.Fatal("expected item to be older than cutoff")
	}
	if crawler.CMSItemOlderThan(item, mustDate(t, "2026-10-10")) != false {
		t.Fatal("expected item to be newer than cutoff")
	}
	if crawler.CMSItemOlderThan(&crawler.CMSItem{}, mustDate(t, "2026-10-10")) {
		t.Fatal("expected item without date to skip the cutoff")
	}
}

func TestCMSClient_PayloadKeepsHTMLInText(t *testing.T) {
	client := newCMSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		payload := map[string]any{
			"id":    "9",
			"title": "Tin có ký tự đặc biệt",
			"slug":  "tin-dac-biet",
			"content": map[string]any{"content": []any{
				map[string]any{"type": "paragraph", "content": []any{
					map[string]any{"type": "text", "text": "Nội dung <b>quan trọng</b> & cần giữ nguyên"},
				}},
			}},
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{payload}})
	})

	items, _, err := client.List(context.Background(), crawler.CMSCollectionPosts, 0)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !strings.Contains(items[0].Body, "Nội dung <b>quan trọng</b>") {
		t.Fatalf("expected raw text to survive decoding, got %q", items[0].Body)
	}
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("bad test date %q: %v", value, err)
	}
	return parsed
}

func TestCMSClient_PagesWithSkipOffset(t *testing.T) {
	var skips []string
	client := newCMSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		skip := r.URL.Query().Get("skip")
		skips = append(skips, skip)
		id := "1"
		if skip != "0" {
			id = "2"
		}
		_, _ = w.Write([]byte(`{"data": [{"id": "` + id + `", "title": "Tin ` + id + `", "slug": "tin-` + id + `", "date": "2026-10-29T00:00:00Z"}],
			"pagination": {"total": 2, "skip": ` + skip + `, "limit": 1, "has_more": ` + map[bool]string{true: "true", false: "false"}[skip == "0"] + `}}`))
	})

	first, pagination, err := client.List(context.Background(), crawler.CMSCollectionPosts, 0)
	if err != nil {
		t.Fatalf("first page failed: %v", err)
	}
	second, _, err := client.List(context.Background(), crawler.CMSCollectionPosts, client.PageSize())
	if err != nil {
		t.Fatalf("second page failed: %v", err)
	}
	if first[0].ID == second[0].ID {
		t.Fatalf("expected the offset to move past the first record, both returned %q", first[0].ID)
	}
	if !pagination.HasMore {
		t.Fatal("expected has_more on the first page")
	}
	if len(skips) != 2 || skips[1] != "100" {
		t.Fatalf("expected a skip=100 request for the second page, got %v", skips)
	}
}

func TestCMSClient_IgnoresPlaceholderDates(t *testing.T) {
	client := newCMSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data": [
			{"id": "1", "title": "Tin không có ngày", "slug": "tin-khong-ngay", "schedule": "1970-01-01T00:00:00Z"},
			{"id": "2", "title": "Tin có ngày", "slug": "tin-co-ngay", "schedule": "2026-10-20T00:00:00Z"}
		]}`))
	})

	items, _, err := client.List(context.Background(), crawler.CMSCollectionPosts, 0)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if items[0].PublishDate != "" {
		t.Fatalf("expected placeholder date to be dropped, got %q", items[0].PublishDate)
	}
	if items[1].PublishDate != "2026-10-20" {
		t.Fatalf("unexpected real date %q", items[1].PublishDate)
	}
	if crawler.CMSItemOlderThan(items[0], time.Now().AddDate(0, 0, -1)) {
		t.Fatal("items without a usable date must not be filtered by the cutoff")
	}
}

func TestCMSClient_RequiresConfiguredBaseURL(t *testing.T) {
	if _, err := crawler.NewCMSClient(crawler.CMSConfig{}); err == nil {
		t.Fatal("expected an error when CMS_API_BASE_URL is not configured")
	}
}

func TestCMSClient_DerivesSiteOriginFromBaseURL(t *testing.T) {
	client, err := crawler.NewCMSClient(crawler.CMSConfig{BaseURL: "https://cms.example.test/api/v1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := client.SiteURL(); got != "https://cms.example.test" {
		t.Fatalf("expected derived site origin, got %q", got)
	}
}

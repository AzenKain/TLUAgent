package crawler_test

import (
	"context"
	"os"
	"testing"

	"tluagent-web/pkg/crawler"
)

// TestCMSLiveCollections contacts the production CMS and only runs with CMS_LIVE_TEST=1.
func TestCMSLiveCollections(t *testing.T) {
	if os.Getenv("CMS_LIVE_TEST") != "1" {
		t.Skip("set CMS_LIVE_TEST=1 to contact the live CMS")
	}

	baseURL := os.Getenv("CMS_API_BASE_URL")
	if baseURL == "" {
		t.Skip("set CMS_API_BASE_URL to contact the live CMS")
	}
	client, err := crawler.NewCMSClient(crawler.CMSConfig{BaseURL: baseURL})
	if err != nil {
		t.Fatalf("failed to build CMS client: %v", err)
	}

	ctx := context.Background()
	for _, collection := range []string{
		crawler.CMSCollectionAnnouncements,
		crawler.CMSCollectionPosts,
		crawler.CMSCollectionEvents,
		crawler.CMSCollectionPages,
	} {
		items, pagination, err := client.List(ctx, collection, 0)
		if err != nil {
			t.Fatalf("%s list failed: %v", collection, err)
		}
		if len(items) == 0 {
			t.Fatalf("%s returned no items", collection)
		}
		t.Logf("%s: items=%d total=%d has_more=%v", collection, len(items), pagination.Total, pagination.HasMore)

		detail, err := client.Detail(ctx, collection, items[0].ID)
		if err != nil {
			t.Fatalf("%s detail failed: %v", collection, err)
		}
		if detail.Title == "" || detail.URL == "" {
			t.Fatalf("%s detail missing title or url: %+v", collection, detail)
		}
		t.Logf("  first: title=%q date=%s body=%d chars attachment=%q", detail.Title, detail.PublishDate, len(detail.Body), detail.AttachmentURL)
	}
}

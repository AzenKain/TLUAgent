package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultCMSLocale = "vi"

	cmsCollectionAnnouncements = "announcements"
	cmsCollectionPosts         = "posts"
	cmsCollectionEvents        = "events"
	cmsCollectionPages         = "pages"
)

// CMSCollections exposes the collection names the crawler can ingest.
const (
	CMSCollectionAnnouncements = cmsCollectionAnnouncements
	CMSCollectionPosts         = cmsCollectionPosts
	CMSCollectionEvents        = cmsCollectionEvents
	CMSCollectionPages         = cmsCollectionPages
)

// CMSConfig configures the headless CMS client.
type CMSConfig struct {
	BaseURL    string
	SiteURL    string
	Locale     string
	PageSize   int
	DelayMs    int
	MaxRetries int
	TimeoutSec int
	HTTPClient *http.Client
}

// CMSItem is a normalized CMS record ready for document ingestion.
type CMSItem struct {
	ID            string
	Title         string
	Slug          string
	URL           string
	PublishDate   string
	Category      string
	Body          string
	AttachmentURL string
}

// CMSPagination mirrors the pagination block returned by the CMS.
type CMSPagination struct {
	Total   int  `json:"total"`
	Skip    int  `json:"skip"`
	Limit   int  `json:"limit"`
	HasMore bool `json:"has_more"`
}

// CMSClient reads collections from the university headless CMS.
type CMSClient struct {
	cfg    CMSConfig
	client *http.Client
}

func NewCMSClient(cfg CMSConfig) (*CMSClient, error) {
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("CMS base url is not configured: set CMS_API_BASE_URL")
	}
	if cfg.SiteURL == "" {
		cfg.SiteURL = SiteOrigin(cfg.BaseURL)
	}
	if cfg.Locale == "" {
		cfg.Locale = DefaultCMSLocale
	}
	if cfg.PageSize <= 0 {
		cfg.PageSize = 100
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 30
	}
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || parsed.Host == "" {
		return nil, fmt.Errorf("invalid CMS base url %q: %w", cfg.BaseURL, err)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: time.Duration(cfg.TimeoutSec) * time.Second}
	}
	return &CMSClient{cfg: cfg, client: client}, nil
}

type cmsListEnvelope struct {
	Data       json.RawMessage `json:"data"`
	Pagination *CMSPagination  `json:"pagination"`
}

type cmsDetailEnvelope struct {
	Data json.RawMessage `json:"data"`
}

type cmsRawItem struct {
	ID              json.RawMessage `json:"id"`
	Title           string          `json:"title"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	Code            string          `json:"code"`
	Description     string          `json:"description"`
	Date            string          `json:"date"`
	Schedule        string          `json:"schedule"`
	CreatedAt       string          `json:"created_at"`
	CategoryName    string          `json:"category_name"`
	SubcategoryName string          `json:"subcategory_name"`
	MainDoc         string          `json:"main_doc"`
	Content         json.RawMessage `json:"content"`
	PageData        json.RawMessage `json:"page_data"`
}

func (c *CMSClient) endpoint(collection string) string {
	return strings.TrimRight(c.cfg.BaseURL, "/") + "/" + collection + "/"
}

// PageSize is the number of records requested per List call.
func (c *CMSClient) PageSize() int {
	return c.cfg.PageSize
}

// List returns one page of a collection, skipped by the given offset.
// The CMS ignores page/size query parameters and paginates with skip/limit only.
func (c *CMSClient) List(ctx context.Context, collection string, skip int) ([]*CMSItem, CMSPagination, error) {
	if skip < 0 {
		skip = 0
	}
	target := fmt.Sprintf("%s?locale=%s&skip=%d&limit=%d", c.endpoint(collection), url.QueryEscape(c.cfg.Locale), skip, c.cfg.PageSize)
	body, err := c.getJSON(ctx, target)
	if err != nil {
		return nil, CMSPagination{}, err
	}

	var envelope cmsListEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil || len(envelope.Data) == 0 {
		envelope.Data = json.RawMessage(body)
		envelope.Pagination = nil
	}

	pagination := CMSPagination{}
	if envelope.Pagination != nil {
		pagination = *envelope.Pagination
	}

	rawItems, err := decodeRawItems(envelope.Data)
	if err != nil {
		return nil, pagination, fmt.Errorf("decode %s items: %w", collection, err)
	}

	items := make([]*CMSItem, 0, len(rawItems))
	for _, raw := range rawItems {
		items = append(items, c.toItem(collection, raw))
	}
	return items, pagination, nil
}

// Detail fetches a single record by id, including its rich-text body.
func (c *CMSClient) Detail(ctx context.Context, collection, id string) (*CMSItem, error) {
	target := fmt.Sprintf("%s%s/?locale=%s", c.endpoint(collection), url.PathEscape(id), url.QueryEscape(c.cfg.Locale))
	body, err := c.getJSON(ctx, target)
	if err != nil {
		return nil, err
	}

	var envelope cmsDetailEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode %s detail %s: %w", collection, id, err)
	}
	raw := envelope.Data
	if len(raw) == 0 {
		raw = body
	}
	var item cmsRawItem
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, fmt.Errorf("decode %s detail %s: %w", collection, id, err)
	}
	return c.toItem(collection, &item), nil
}

func decodeRawItems(data json.RawMessage) ([]*cmsRawItem, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var items []*cmsRawItem
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, err
		}
		return items, nil
	}
	var single cmsRawItem
	if err := json.Unmarshal(data, &single); err != nil {
		return nil, err
	}
	return []*cmsRawItem{&single}, nil
}

func (c *CMSClient) toItem(collection string, raw *cmsRawItem) *CMSItem {
	if raw == nil {
		return &CMSItem{}
	}

	item := &CMSItem{
		ID:            rawID(raw.ID),
		Title:         firstNonEmpty(raw.Title, raw.Name),
		Slug:          firstNonEmpty(raw.Slug, raw.Code),
		PublishDate:   cmsDatePart(firstNonEmpty(raw.Date, raw.Schedule, raw.CreatedAt)),
		Category:      firstNonEmpty(raw.SubcategoryName, raw.CategoryName),
		AttachmentURL: strings.TrimSpace(raw.MainDoc),
	}

	switch collection {
	case cmsCollectionPages:
		item.Body = pageDataText(raw.PageData)
		item.URL = c.siteURL("/" + c.cfg.Locale + "/" + item.Slug)
	default:
		item.Body = blocksToText(raw.Content)
		if item.Body == "" {
			item.Body = strings.TrimSpace(raw.Description)
		}
		item.URL = c.siteURL(articlePath(collection, c.cfg.Locale, item.Slug))
	}

	if item.Title == "" {
		item.Title = item.Slug
	}
	return item
}

// SiteOrigin derives the public site origin from the CMS API base URL.
func SiteOrigin(baseURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

// SiteURL exposes the public site origin used to build canonical document URLs.
func (c *CMSClient) SiteURL() string {
	return c.cfg.SiteURL
}

func (c *CMSClient) siteURL(path string) string {
	return strings.TrimRight(c.cfg.SiteURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func articlePath(collection, locale, slug string) string {
	slug = strings.Trim(strings.TrimSpace(slug), "/")
	switch collection {
	case cmsCollectionAnnouncements:
		return "/" + locale + "/thong-bao/" + slug
	case cmsCollectionPosts:
		return "/" + locale + "/tin-tuc-su-kien/tin-tuc/" + slug
	case cmsCollectionEvents:
		return "/" + locale + "/tin-tuc-su-kien/su-kien/" + slug
	default:
		return "/" + locale + "/" + slug
	}
}

func cmsDatePart(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < 10 {
		return ""
	}
	parsed, err := time.Parse("2006-01-02", value[:10])
	if err != nil || parsed.Year() < 2000 {
		return ""
	}
	return parsed.Format("2006-01-02")
}

func rawID(raw json.RawMessage) string {
	trimmed := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	return trimmed
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func (c *CMSClient) getJSON(ctx context.Context, target string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
		if c.cfg.DelayMs > 0 && attempt == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(c.cfg.DelayMs) * time.Millisecond):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "TLUAgent-Crawler/1.0")

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := readAllLimited(resp.Body, 32*1024*1024)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, target)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, target)
		}
		return body, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("request failed for %s", target)
	}
	return nil, lastErr
}

func readAllLimited(body interface{ Read([]byte) (int, error) }, limit int64) ([]byte, error) {
	buf := make([]byte, 0, 32*1024)
	chunk := make([]byte, 32*1024)
	var total int64
	for {
		n, err := body.Read(chunk)
		if n > 0 {
			total += int64(n)
			if total > limit {
				return nil, fmt.Errorf("response exceeds %d bytes", limit)
			}
			buf = append(buf, chunk[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				return buf, nil
			}
			return buf, err
		}
	}
}

var cmsSkippedKeys = map[string]bool{
	"id": true, "src": true, "alt": true, "url": true, "href": true, "image": true,
	"banner": true, "banner_mobile": true, "color": true, "background": true,
	"classname": true, "class": true, "type": true, "slug": true, "code": true,
	"thumbnail_image": true, "registration_link": true, "author": true,
}

// blocksToText flattens a rich-text document (paragraph/list/heading blocks) into plain text.
func blocksToText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return ""
	}
	var sb strings.Builder
	writeBlocks(&sb, node)
	return strings.TrimSpace(collapseBlankLines(sb.String()))
}

func writeBlocks(sb *strings.Builder, node any) {
	switch v := node.(type) {
	case map[string]any:
		blockType, _ := v["type"].(string)
		if text, ok := v["text"].(string); ok && text != "" {
			sb.WriteString(text)
			sb.WriteString(" ")
		}
		if children, ok := v["content"]; ok {
			switch blockType {
			case "paragraph", "heading", "listItem", "blockquote", "codeBlock":
				writeBlocks(sb, children)
				sb.WriteString("\n")
			case "bulletList", "orderedList":
				writeBlocks(sb, children)
			default:
				writeBlocks(sb, children)
			}
			return
		}
		if blockType == "paragraph" || blockType == "heading" {
			sb.WriteString("\n")
		}
	case []any:
		for _, child := range v {
			writeBlocks(sb, child)
		}
	}
}

// pageDataText flattens the text fields of a page layout, skipping media and styling props.
func pageDataText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return ""
	}
	var sb strings.Builder
	writePageText(&sb, node)
	return strings.TrimSpace(collapseBlankLines(sb.String()))
}

func writePageText(sb *strings.Builder, node any) {
	switch v := node.(type) {
	case map[string]any:
		for key, child := range v {
			if cmsSkippedKeys[strings.ToLower(key)] {
				continue
			}
			if text, ok := child.(string); ok {
				trimmed := strings.TrimSpace(text)
				if len(trimmed) > 40 && !strings.HasPrefix(trimmed, "http") {
					sb.WriteString(trimmed)
					sb.WriteString("\n")
				}
				continue
			}
			writePageText(sb, child)
		}
	case []any:
		for _, child := range v {
			writePageText(sb, child)
		}
	}
}

func collapseBlankLines(text string) string {
	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	blank := 0
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if strings.TrimSpace(trimmed) == "" {
			blank++
			if blank > 1 {
				continue
			}
			kept = append(kept, "")
			continue
		}
		blank = 0
		kept = append(kept, trimmed)
	}
	return strings.Join(kept, "\n")
}

var cmsDriveIDRegex = regexp.MustCompile(`drive\.google\.com/file/d/([a-zA-Z0-9_-]+)`)
var cmsDriveConfirmRegex = regexp.MustCompile(`drive\.google\.com/uc\?export=download&amp;confirm=([a-zA-Z0-9_-]+)`)

// AttachmentText downloads a Drive preview link or a direct file link and returns its extracted text.
func (c *Crawler) AttachmentText(ctx context.Context, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}

	if m := cmsDriveIDRegex.FindStringSubmatch(ref); len(m) > 1 {
		driveID := m[1]
		if c.isDriveIDIgnored(driveID) {
			return ""
		}
		dlURL := fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", driveID)
		attBytes, _, err := c.doAttachmentRequestWithAutoWait(ctx, dlURL)
		if err != nil || len(attBytes) == 0 {
			return ""
		}
		if bytes.Contains(attBytes, []byte("drive.google.com/uc?export=download&amp;confirm=")) {
			if m := cmsDriveConfirmRegex.FindSubmatch(attBytes); len(m) > 1 {
				confirmURL := fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s&confirm=%s", driveID, string(m[1]))
				if confBytes, _, err := c.doAttachmentRequestWithAutoWait(ctx, confirmURL); err == nil && len(confBytes) > 0 {
					attBytes = confBytes
				}
			}
		}
		if bytes.Contains(attBytes, []byte("Google Drive - Can't download file")) ||
			bytes.Contains(attBytes, []byte("Sorry, the owner hasn't given you permission to download this file")) {
			return ""
		}
		text, err := c.extractAttachmentText(ctx, attBytes, "")
		if err != nil {
			return ""
		}
		return strings.TrimSpace(text)
	}

	if !strings.HasPrefix(ref, "http") {
		ref = strings.TrimRight(c.cfg.BaseURL, "/") + "/" + strings.TrimLeft(ref, "/")
	}
	attBytes, _, err := c.doAttachmentRequestWithAutoWait(ctx, ref)
	if err != nil || len(attBytes) == 0 {
		return ""
	}
	text, err := c.extractAttachmentText(ctx, attBytes, filepath.Base(ref))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(text)
}

// CMSItemOlderThan reports whether the item was published before the cutoff date.
func CMSItemOlderThan(item *CMSItem, cutoff time.Time) bool {
	if item == nil || item.PublishDate == "" {
		return false
	}
	parsed, err := time.Parse("2006-01-02", item.PublishDate)
	if err != nil {
		return false
	}
	return parsed.Before(cutoff)
}

func CMSDocID(prefix, id string) string {
	suffix := strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return r
		}
		return '-'
	}, id)
	if suffix == "" {
		suffix = strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return prefix + suffix
}

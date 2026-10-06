package crawler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

var (
	nodeIDRegex      = regexp.MustCompile(`-(\d+)\.html`)
	nodePathRegex    = regexp.MustCompile(`/node/(\d+)`)
	driveFileIDRegex = regexp.MustCompile(`/file/d/([a-zA-Z0-9_-]+)`)
)

type CrawlRecord struct {
	NodeID                string   `json:"node_id"`
	Type                  string   `json:"type"`
	URL                   string   `json:"url"`
	Title                 string   `json:"title"`
	PublishDate           string   `json:"publish_date"`
	SourceDepartmentRaw   string   `json:"source_department_raw"`
	HTMLPath              string   `json:"html_path"`
	ContentType           string   `json:"content_type"`
	HasPDF                bool     `json:"has_pdf"`
	DriveFileID           string   `json:"drive_file_id,omitempty"`
	PDFDownloadURLs       []string `json:"pdf_download_urls,omitempty"`
	LocalPDFPaths         []string `json:"local_pdf_paths,omitempty"`
}

type Config struct {
	BaseURL     string
	DataRawDir  string
	TimeoutSec  int
	MaxWorkers  int
}

type Crawler struct {
	cfg        Config
	client     *http.Client
	htmlDir    string
	pdfDir     string
	indexFile  string
	mu         sync.Mutex
	knownNodes map[string]bool
}

func NewCrawler(cfg Config) (*Crawler, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://thanglong.edu.vn"
	}
	if cfg.DataRawDir == "" {
		cfg.DataRawDir = "data/raw"
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 45
	}
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = 5
	}

	htmlDir := filepath.Join(cfg.DataRawDir, "html")
	pdfDir := filepath.Join(cfg.DataRawDir, "pdf")
	indexFile := filepath.Join(cfg.DataRawDir, "raw_index.jsonl")

	if err := os.MkdirAll(htmlDir, 0755); err != nil {
		return nil, fmt.Errorf("create html dir: %w", err)
	}
	if err := os.MkdirAll(pdfDir, 0755); err != nil {
		return nil, fmt.Errorf("create pdf dir: %w", err)
	}

	c := &Crawler{
		cfg:        cfg,
		client:     &http.Client{Timeout: time.Duration(cfg.TimeoutSec) * time.Second},
		htmlDir:    htmlDir,
		pdfDir:     pdfDir,
		indexFile:  indexFile,
		knownNodes: make(map[string]bool),
	}
	_ = c.loadKnownNodes()
	return c, nil
}

func (c *Crawler) loadKnownNodes() error {
	f, err := os.Open(c.indexFile)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec CrawlRecord
		if err := json.Unmarshal(scanner.Bytes(), &rec); err == nil && rec.NodeID != "" {
			c.knownNodes[rec.NodeID] = true
			if rec.URL != "" {
				c.knownNodes[rec.URL] = true
			}
		}
	}
	return scanner.Err()
}

func (c *Crawler) saveRecord(rec CrawlRecord) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	f, err := os.OpenFile(c.indexFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	c.knownNodes[rec.NodeID] = true
	c.knownNodes[rec.URL] = true
	return nil
}

func (c *Crawler) CrawlNotices(ctx context.Context, maxPages int, onProgress func(cur, total int64)) error {
	return c.crawlCategory(ctx, "thong-bao", "THONG_BAO", maxPages, onProgress)
}

func (c *Crawler) CrawlNews(ctx context.Context, maxPages int, onProgress func(cur, total int64)) error {
	return c.crawlCategory(ctx, "tin-tuc", "TIN_TUC", maxPages, onProgress)
}

func (c *Crawler) CrawlStaticPages(ctx context.Context, onProgress func(cur, total int64)) error {
	staticRoutes := []string{
		"/so-tay-sinh-vien",
		"/quy-che-dao-tao",
		"/chuong-trinh-dao-tao",
		"/tuyen-sinh",
		"/hoc-bong-va-ho-tro-tai-chinh",
	}

	total := int64(len(staticRoutes))
	for i, route := range staticRoutes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fullURL := strings.TrimRight(c.cfg.BaseURL, "/") + route
		nodeID := "static_" + strings.Trim(strings.ReplaceAll(route, "/", "_"), "_")

		c.mu.Lock()
		alreadyKnown := c.knownNodes[nodeID] || c.knownNodes[fullURL]
		c.mu.Unlock()

		if !alreadyKnown {
			_ = c.fetchAndStoreNode(ctx, fullURL, nodeID, "STATIC", "")
		}

		if onProgress != nil {
			onProgress(int64(i+1), total)
		}
	}
	return nil
}

type itemRef struct {
	URL   string
	Title string
	Date  string
}

func (c *Crawler) crawlCategory(ctx context.Context, slug, catType string, maxPages int, onProgress func(cur, total int64)) error {
	if maxPages <= 0 {
		maxPages = 500
	}

	var allItems []itemRef
	for page := 0; page < maxPages; page++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		pageURL := fmt.Sprintf("%s/%s?page=%d", strings.TrimRight(c.cfg.BaseURL, "/"), slug, page)
		items, hasNext, err := c.fetchListingPage(ctx, pageURL)
		if err != nil {
			break
		}
		allItems = append(allItems, items...)
		if !hasNext || len(items) == 0 {
			break
		}
	}

	total := int64(len(allItems))
	for i, item := range allItems {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		nodeID := extractNodeID(item.URL)
		if nodeID == "" {
			nodeID = fmt.Sprintf("%s_%d", strings.ToLower(catType), i)
		}

		c.mu.Lock()
		alreadyKnown := c.knownNodes[nodeID] || c.knownNodes[item.URL]
		c.mu.Unlock()

		if !alreadyKnown {
			_ = c.fetchAndStoreNode(ctx, item.URL, nodeID, catType, item.Date)
		}

		if onProgress != nil {
			onProgress(int64(i+1), total)
		}
	}

	return nil
}

func (c *Crawler) fetchListingPage(ctx context.Context, pageURL string) ([]itemRef, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", "TLUAgent/1.0 (Crawler; Educational Research)")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("status: %d", resp.StatusCode)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, false, err
	}

	var items []itemRef
	hasNext := false

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			var href string
			var relNext bool
			for _, a := range n.Attr {
				if a.Key == "href" {
					href = a.Val
				}
				if a.Key == "rel" && a.Val == "next" {
					relNext = true
				}
			}
			if relNext {
				hasNext = true
			}
			if href != "" && (strings.Contains(href, ".html") || strings.Contains(href, "/node/")) {
				text := getText(n)
				if strings.TrimSpace(text) != "" {
					fullURL := href
					if !strings.HasPrefix(fullURL, "http") {
						fullURL = strings.TrimRight(c.cfg.BaseURL, "/") + "/" + strings.TrimLeft(href, "/")
					}
					items = append(items, itemRef{
						URL:   fullURL,
						Title: strings.TrimSpace(text),
					})
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(doc)

	return items, hasNext, nil
}

func (c *Crawler) fetchAndStoreNode(ctx context.Context, itemURL, nodeID, catType, defaultDate string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, itemURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "TLUAgent/1.0 (Crawler; Educational Research)")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	htmlFileName := fmt.Sprintf("%s_%s.html", catType, nodeID)
	htmlRelPath := filepath.Join(c.cfg.DataRawDir, "html", htmlFileName)
	htmlAbsPath := filepath.Join(c.htmlDir, htmlFileName)
	_ = os.WriteFile(htmlAbsPath, bodyBytes, 0644)

	doc, err := html.Parse(strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}

	title := extractTitle(doc)
	publishDate := defaultDate
	if publishDate == "" {
		publishDate = extractDateFromHTML(string(bodyBytes))
	}

	rec := CrawlRecord{
		NodeID:      nodeID,
		Type:        catType,
		URL:         itemURL,
		Title:       title,
		PublishDate: publishDate,
		HTMLPath:    htmlRelPath,
		ContentType: "HTML_ARTICLE",
	}

	driveID := extractDriveFileID(string(bodyBytes))
	if driveID != "" {
		rec.DriveFileID = driveID
		rec.HasPDF = true
		rec.ContentType = "PDF_EMBEDDED"
		dlURL := fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", driveID)
		rec.PDFDownloadURLs = []string{dlURL}

		pdfFileName := fmt.Sprintf("%s.pdf", driveID)
		pdfRelPath := filepath.Join(c.cfg.DataRawDir, "pdf", pdfFileName)
		pdfAbsPath := filepath.Join(c.pdfDir, pdfFileName)

		if err := c.downloadFile(ctx, dlURL, pdfAbsPath); err == nil {
			rec.LocalPDFPaths = []string{pdfRelPath}
		}
	}

	return c.saveRecord(rec)
}

func (c *Crawler) downloadFile(ctx context.Context, dlURL, destPath string) error {
	if fi, err := os.Stat(destPath); err == nil && fi.Size() > 0 {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download status: %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractNodeID(rawURL string) string {
	m := nodeIDRegex.FindStringSubmatch(rawURL)
	if len(m) > 1 {
		return m[1]
	}
	m2 := nodePathRegex.FindStringSubmatch(rawURL)
	if len(m2) > 1 {
		return m2[1]
	}
	u, err := url.Parse(rawURL)
	if err == nil {
		seg := strings.Trim(u.Path, "/")
		if seg != "" {
			return strings.ReplaceAll(seg, "/", "_")
		}
	}
	return ""
}

func extractDriveFileID(htmlStr string) string {
	m := driveFileIDRegex.FindStringSubmatch(htmlStr)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func extractTitle(doc *html.Node) string {
	var title string
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "h1" || n.Data == "title") && title == "" {
			t := strings.TrimSpace(getText(n))
			if t != "" {
				title = t
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(doc)
	return title
}

func extractDateFromHTML(s string) string {
	dateRegex := regexp.MustCompile(`\b(\d{1,2}[./-]\d{1,2}[./-]\d{4})\b`)
	m := dateRegex.FindStringSubmatch(s)
	if len(m) > 1 {
		return m[1]
	}
	return time.Now().Format("2006-01-02")
}

func getText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(getText(c))
	}
	return sb.String()
}

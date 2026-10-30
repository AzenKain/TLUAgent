package crawler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"

	"tluagent-web/pkg/docparser"
	"tluagent-web/pkg/netx"
	"tluagent-web/pkg/pdf"
	"tluagent-web/pkg/pipeline"
)

var (
	nodeIDRegex         = regexp.MustCompile(`-(\d+)\.html`)
	nodePathRegex       = regexp.MustCompile(`/node/(\d+)`)
	iframeDriveRegex    = regexp.MustCompile(`(?i)<iframe[^>]+src=["'](?:https?:)?//drive\.google\.com/file/d/([a-zA-Z0-9_-]+)`)
	driveFileIDRegex    = regexp.MustCompile(`drive\.google\.com/file/d/([a-zA-Z0-9_-]+)`)
	pdfLinkRegex        = regexp.MustCompile(`(?i)href=["'](/sites/default/files/[^"']+\.(?:pdf|doc|docx|xls|xlsx))["']`)
	attachmentLinkRegex = pdfLinkRegex
	inlineImageRegex    = regexp.MustCompile(`(?i)<img[^>]+src=["'](/sites/default/files/inline-images/[^"']+\.(?:jpg|jpeg|png))["']`)
	pdfMagic            = []byte("%PDF-")
	oleMagic            = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
	zipMagic            = []byte{0x50, 0x4B, 0x03, 0x04}
)

// VisionOCRHandler defines an image OCR transcription callback.
type VisionOCRHandler func(ctx context.Context, images [][]byte) (string, error)

// CrawledDocument represents an in-memory document with cleaned markdown and metadata.
type CrawledDocument struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	PublishDate string `json:"publish_date"`
	Content     string `json:"content"`
}

// DocumentHandler streams parsed in-memory documents to subscribers.
type DocumentHandler func(ctx context.Context, doc *CrawledDocument) error

// SkipChecker checks whether a document ID, URL, or Title is already persisted.
type SkipChecker func(ctx context.Context, id, rawURL, title string) bool

// Config specifies crawling parameters and backoff timeouts.
type Config struct {
	BaseURL          string
	TimeoutSec       int
	DelayMs          int
	Concurrency      int
	MaxRetries       int
	RetryBaseDelayMs int
	MaxBackoffMs     int
	IgnoredDriveIDs  []string
	UserAgent        string
	MaxTimeDays      int
	VisionOCR        VisionOCRHandler
}

// Crawler manages resilient in-memory web scraping and asset processing.
type Crawler struct {
	cfg        Config
	client     *http.Client
	attClient  *http.Client
	mu         sync.Mutex
	knownNodes map[string]bool
	userAgent  string
	platform   string
}

// ResolveBrowserIdentity returns dynamic browser User-Agent and Sec-Ch-Ua-Platform based on OS.
func ResolveBrowserIdentity(targetOS, customUA string) (string, string) {
	if customUA != "" {
		switch {
		case strings.Contains(customUA, "Windows"):
			return customUA, "\"Windows\""
		case strings.Contains(customUA, "Macintosh") || strings.Contains(customUA, "Mac OS"):
			return customUA, "\"macOS\""
		case strings.Contains(customUA, "Linux"):
			return customUA, "\"Linux\""
		default:
			return customUA, "\"Linux\""
		}
	}

	if targetOS == "" {
		targetOS = runtime.GOOS
	}

	switch strings.ToLower(targetOS) {
	case "windows":
		return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/155.0.0.0 Safari/537.36 Edg/155.0.0.0", "\"Windows\""
	case "darwin":
		return "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/155.0.0.0 Safari/537.36 Edg/155.0.0.0", "\"macOS\""
	case "linux":
		return "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/155.0.0.0 Safari/537.36 Edg/155.0.0.0", "\"Linux\""
	default:
		return "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/155.0.0.0 Safari/537.36 Edg/155.0.0.0", "\"Linux\""
	}
}

// NewCrawler initializes an in-memory web crawler instance with human-browser headers.
func NewCrawler(cfg Config) (*Crawler, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("crawler base url is required")
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 15
	}
	if cfg.DelayMs <= 0 {
		cfg.DelayMs = 60
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 3
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 5
	}
	if cfg.RetryBaseDelayMs <= 0 {
		cfg.RetryBaseDelayMs = 500
	}
	if cfg.MaxBackoffMs <= 0 {
		cfg.MaxBackoffMs = 15000
	}

	ua, platform := ResolveBrowserIdentity(runtime.GOOS, cfg.UserAgent)
	cfg.UserAgent = ua

	attTimeout := time.Duration(cfg.TimeoutSec) * time.Second * 12
	if attTimeout < 180*time.Second {
		attTimeout = 180 * time.Second
	}

	return &Crawler{
		cfg:        cfg,
		client:     netx.NewSafeHTTPClient(time.Duration(cfg.TimeoutSec) * time.Second),
		attClient:  netx.NewSafeHTTPClient(attTimeout),
		knownNodes: make(map[string]bool),
		userAgent:  ua,
		platform:   platform,
	}, nil
}

func (c *Crawler) isAllowedHost(targetURL string) bool {
	parsedTarget, err := url.Parse(targetURL)
	if err != nil {
		return false
	}
	baseParsed, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsedTarget.Hostname(), baseParsed.Hostname())
}

func (c *Crawler) doRequestWithClient(ctx context.Context, client *http.Client, targetURL string, maxDownloadSize int64) ([]byte, int, error) {
	var lastErr error
	for attempt := 0; attempt < c.cfg.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			return nil, 0, err
		}

		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
		req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en-US;q=0.8,en;q=0.7")
		req.Header.Set("Sec-Ch-Ua", "\"Not A(Brand\";v=\"99\", \"Microsoft Edge\";v=\"155\", \"Chromium\";v=\"155\"")
		req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
		req.Header.Set("Sec-Ch-Ua-Platform", c.platform)
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		req.Header.Set("Sec-Fetch-User", "?1")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Referer", strings.TrimRight(c.cfg.BaseURL, "/")+"/")
		req.Header.Set("Cache-Control", "max-age=0")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			backoff := c.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(backoff):
			}
			continue
		}

		statusCode := resp.StatusCode
		if statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable || statusCode == http.StatusGatewayTimeout {
			retryAfterSec := parseRetryAfter(resp.Header.Get("Retry-After"))
			_ = resp.Body.Close()

			waitDuration := c.calculateBackoff(attempt)
			if retryAfterSec > 0 && retryAfterSec*1000 < c.cfg.MaxBackoffMs {
				waitDuration = time.Duration(retryAfterSec) * time.Second
			}

			select {
			case <-ctx.Done():
				return nil, statusCode, ctx.Err()
			case <-time.After(waitDuration):
			}
			continue
		}

		if resp.ContentLength > maxDownloadSize {
			_ = resp.Body.Close()
			return nil, statusCode, fmt.Errorf("file size %d exceeds limit of %d bytes", resp.ContentLength, maxDownloadSize)
		}

		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, maxDownloadSize+1))
		_ = resp.Body.Close()
		if int64(len(bodyBytes)) > maxDownloadSize {
			return nil, statusCode, fmt.Errorf("response exceeded limit of %d bytes", maxDownloadSize)
		}
		if readErr != nil {
			lastErr = readErr
			backoff := c.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(backoff):
			}
			continue
		}

		if statusCode != http.StatusOK {
			return nil, statusCode, fmt.Errorf("unexpected status code %d for %s", statusCode, targetURL)
		}

		return bodyBytes, statusCode, nil
	}

	return nil, 0, fmt.Errorf("exhausted %d retries for %s: %w", c.cfg.MaxRetries, targetURL, lastErr)
}

func (c *Crawler) doRequestWithAutoWait(ctx context.Context, targetURL string) ([]byte, int, error) {
	return c.doRequestWithClient(ctx, c.client, targetURL, 25*1024*1024)
}

func (c *Crawler) doAttachmentRequestWithAutoWait(ctx context.Context, targetURL string) ([]byte, int, error) {
	return c.doRequestWithClient(ctx, c.attClient, targetURL, 64*1024*1024)
}

func (c *Crawler) calculateBackoff(attempt int) time.Duration {
	baseMs := c.cfg.RetryBaseDelayMs
	shift := attempt
	if shift > 6 {
		shift = 6
	}
	expMs := baseMs * (1 << shift)
	if expMs > c.cfg.MaxBackoffMs {
		expMs = c.cfg.MaxBackoffMs
	}

	var b [4]byte
	_, _ = rand.Read(b[:])
	jitter := int(binary.BigEndian.Uint32(b[:]) % 250)

	totalMs := expMs + jitter
	return time.Duration(totalMs) * time.Millisecond
}

func parseRetryAfter(headerVal string) int {
	headerVal = strings.TrimSpace(headerVal)
	if headerVal == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(headerVal); err == nil && seconds > 0 {
		return seconds
	}
	return 0
}

// DiscoverMaxPage detects the maximum pagination number available on a listing route.
func (c *Crawler) DiscoverMaxPage(ctx context.Context, slug, pageParam string) int {
	if pageParam == "" {
		pageParam = "page"
	}
	page0URL := fmt.Sprintf("%s/%s?%s=0", strings.TrimRight(c.cfg.BaseURL, "/"), slug, pageParam)
	bodyBytes, _, err := c.doRequestWithAutoWait(ctx, page0URL)
	if err != nil || len(bodyBytes) == 0 {
		pageDirectURL := fmt.Sprintf("%s/%s", strings.TrimRight(c.cfg.BaseURL, "/"), slug)
		bodyBytes, _, err = c.doRequestWithAutoWait(ctx, pageDirectURL)
		if err != nil {
			return 0
		}
	}

	paramPattern := regexp.MustCompile(fmt.Sprintf(`%s=(\d+)`, regexp.QuoteMeta(pageParam)))
	matches := paramPattern.FindAllStringSubmatch(string(bodyBytes), -1)
	maxPage := 0
	for _, m := range matches {
		if len(m) > 1 {
			if val, err := strconv.Atoi(m[1]); err == nil && val > maxPage {
				maxPage = val
			}
		}
	}
	return maxPage
}

type itemRef struct {
	URL   string
	Title string
	Date  string
}

// CrawlPagedCategory crawls multi-page categories in RAM with automatic wait and streaming processing.
func (c *Crawler) CrawlPagedCategory(
	ctx context.Context,
	slug, catType string,
	maxPages int,
	skipChecker SkipChecker,
	handler DocumentHandler,
	onProgress func(cur, total int64),
) error {
	if c.cfg.MaxTimeDays > 0 {
		cutoffTime := time.Now().AddDate(0, 0, -c.cfg.MaxTimeDays)
		var collected []itemRef
		hitBoundary := false

		limitPages := maxPages
		if limitPages <= 0 {
			limitPages = 50
		}

		seenURLs := make(map[string]bool)
		for p := 0; p <= limitPages; p++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			pageURL := fmt.Sprintf("%s/%s?page=%d", strings.TrimRight(c.cfg.BaseURL, "/"), slug, p)
			items, err := c.fetchListingItems(ctx, pageURL)
			if err != nil || len(items) == 0 {
				if err != nil && len(collected) == 0 && p == 0 {
					return fmt.Errorf("failed to fetch initial listing page %s: %w", pageURL, err)
				}
				break
			}

			for _, item := range items {
				if seenURLs[item.URL] {
					continue
				}
				seenURLs[item.URL] = true
				if item.Date != "" {
					if t, err := time.Parse("2006-01-02", item.Date); err == nil && t.Before(cutoffTime) {
						collected = append(collected, item)
						hitBoundary = true
						break
					}
				}

				collected = append(collected, item)
			}

			if hitBoundary {
				break
			}
		}

		totalCount := int64(len(collected))
		var processedCount int64
		if onProgress != nil {
			onProgress(0, totalCount)
		}
		if totalCount == 0 {
			return nil
		}

		for i := len(collected) - 1; i >= 0; i-- {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			item := collected[i]
			nodeID := extractNodeID(item.URL)
			slugHash := fmt.Sprintf("%x", sha256.Sum256([]byte(item.URL)))[:8]
			docID := fmt.Sprintf("DOC_%s_%s_%s", strings.ToUpper(catType), nodeID, slugHash)

			c.mu.Lock()
			alreadySeen := c.knownNodes[docID] || c.knownNodes[item.URL]
			if !alreadySeen {
				c.knownNodes[docID] = true
				c.knownNodes[item.URL] = true
			}
			c.mu.Unlock()

			if alreadySeen {
				cur := atomic.AddInt64(&processedCount, 1)
				if onProgress != nil {
					onProgress(cur, totalCount)
				}
				continue
			}

			if skipChecker != nil && skipChecker(ctx, docID, item.URL, item.Title) {
				cur := atomic.AddInt64(&processedCount, 1)
				if onProgress != nil {
					onProgress(cur, totalCount)
				}
				continue
			}

			doc, err := c.fetchAndProcessNode(ctx, item.URL, docID, catType, item.Date, item.Title)
			if err == nil && doc != nil && handler != nil {
				_ = handler(ctx, doc)
			}

			cur := atomic.AddInt64(&processedCount, 1)
			if onProgress != nil {
				onProgress(cur, totalCount)
			}
			time.Sleep(time.Duration(c.cfg.DelayMs)*time.Millisecond + time.Duration(c.calculateJitter(40))*time.Millisecond)
		}

		if onProgress != nil {
			onProgress(processedCount, totalCount)
		}
		return nil
	}

	discoveredMax := c.DiscoverMaxPage(ctx, slug, "page")
	if maxPages <= 0 {
		maxPages = discoveredMax
	} else if discoveredMax > 0 && maxPages > discoveredMax {
		maxPages = discoveredMax
	}

	approxTotal := int64((maxPages + 1) * 9)
	var processedCount int64
	if onProgress != nil {
		onProgress(0, approxTotal)
	}

	pageIndices := make([]int, 0, maxPages+1)
	for p := maxPages; p >= 0; p-- {
		pageIndices = append(pageIndices, p)
	}

	totalPages := int64(len(pageIndices))
	consecutiveErrors := 0
	for pageIdx, p := range pageIndices {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		pageURL := fmt.Sprintf("%s/%s?page=%d", strings.TrimRight(c.cfg.BaseURL, "/"), slug, p)
		items, err := c.fetchListingItems(ctx, pageURL)
		if err != nil {
			consecutiveErrors++
			if consecutiveErrors >= 3 && processedCount == 0 {
				return fmt.Errorf("failed to connect to university portal after %d consecutive attempts: %w", consecutiveErrors, err)
			}
			time.Sleep(time.Duration(c.cfg.DelayMs*2) * time.Millisecond)
			continue
		}
		consecutiveErrors = 0

		itemsToProcess := make([]itemRef, len(items))
		for i := range items {
			itemsToProcess[i] = items[len(items)-1-i]
		}

		_ = c.processBatchConcurrently(ctx, itemsToProcess, catType, skipChecker, handler, func() {
			cur := atomic.AddInt64(&processedCount, 1)
			if onProgress != nil {
				tot := approxTotal
				if int64(pageIdx+1) < totalPages {
					estimatedTotal := (cur * totalPages) / int64(pageIdx+1)
					if estimatedTotal > tot {
						tot = estimatedTotal
					}
					if tot <= cur {
						tot = cur + 1
					}
				} else {
					if cur > tot {
						tot = cur
					}
				}
				onProgress(cur, tot)
			}
		})

		time.Sleep(time.Duration(c.cfg.DelayMs)*time.Millisecond + time.Duration(c.calculateJitter(80))*time.Millisecond)
	}

	if onProgress != nil {
		onProgress(processedCount, processedCount)
	}
	return nil
}

func (c *Crawler) processBatchConcurrently(
	ctx context.Context,
	items []itemRef,
	catType string,
	skipChecker SkipChecker,
	handler DocumentHandler,
	onItemProcessed func(),
) error {
	if len(items) == 0 {
		return nil
	}

	concurrency := c.cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 3
	}
	if concurrency > len(items) {
		concurrency = len(items)
	}

	taskChan := make(chan itemRef, len(items))
	for _, it := range items {
		taskChan <- it
	}
	close(taskChan)

	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range taskChan {
				select {
				case <-ctx.Done():
					errOnce.Do(func() { firstErr = ctx.Err() })
					return
				default:
				}

				nodeID := extractNodeID(item.URL)
				slugHash := fmt.Sprintf("%x", sha256.Sum256([]byte(item.URL)))[:8]
				docID := fmt.Sprintf("DOC_%s_%s_%s", strings.ToUpper(catType), nodeID, slugHash)

				c.mu.Lock()
				alreadySeen := c.knownNodes[docID] || c.knownNodes[item.URL]
				if !alreadySeen {
					c.knownNodes[docID] = true
					c.knownNodes[item.URL] = true
				}
				c.mu.Unlock()

				if alreadySeen {
					if onItemProcessed != nil {
						onItemProcessed()
					}
					continue
				}

				if skipChecker != nil && skipChecker(ctx, docID, item.URL, item.Title) {
					if onItemProcessed != nil {
						onItemProcessed()
					}
					continue
				}

				doc, err := c.fetchAndProcessNode(ctx, item.URL, docID, catType, item.Date, item.Title)
				if err == nil && doc != nil && handler != nil {
					_ = handler(ctx, doc)
				}

				if onItemProcessed != nil {
					onItemProcessed()
				}

				delay := time.Duration(c.cfg.DelayMs)*time.Millisecond + time.Duration(c.calculateJitter(80))*time.Millisecond
				select {
				case <-ctx.Done():
					return
				case <-time.After(delay):
				}
			}
		}()
	}

	wg.Wait()
	return firstErr
}

// CrawlRoutes processes an explicit list of routes in RAM without disk buffering.
func (c *Crawler) CrawlRoutes(
	ctx context.Context,
	routes []string,
	catType string,
	skipChecker SkipChecker,
	handler DocumentHandler,
	onProgress func(cur, total int64),
) error {
	total := int64(len(routes))
	if total == 0 {
		return nil
	}
	if onProgress != nil {
		onProgress(0, total)
	}

	items := make([]itemRef, len(routes))
	for i, route := range routes {
		fullURL := route
		if !strings.HasPrefix(fullURL, "http") {
			fullURL = strings.TrimRight(c.cfg.BaseURL, "/") + "/" + strings.TrimLeft(route, "/")
		}
		items[i] = itemRef{URL: fullURL}
	}

	var processedCount int64
	return c.processBatchConcurrently(ctx, items, catType, skipChecker, handler, func() {
		cur := atomic.AddInt64(&processedCount, 1)
		if onProgress != nil {
			onProgress(cur, total)
		}
	})
}

func (c *Crawler) peekItemDate(ctx context.Context, itemURL string) string {
	if !c.isAllowedHost(itemURL) {
		return ""
	}
	bodyBytes, _, err := c.doRequestWithAutoWait(ctx, itemURL)
	if err != nil {
		return ""
	}
	return extractDateFromHTML(string(bodyBytes))
}

func isInsideNavOrFooter(n *html.Node) bool {
	curr := n.Parent
	for curr != nil {
		if curr.Type == html.ElementNode {
			switch strings.ToLower(curr.Data) {
			case "header", "nav", "footer", "aside":
				return true
			}
			for _, a := range curr.Attr {
				if strings.EqualFold(a.Key, "class") {
					val := strings.ToLower(a.Val)
					if strings.Contains(val, "menu") ||
						strings.Contains(val, "footer") ||
						strings.Contains(val, "breadcrumb") ||
						strings.Contains(val, "pager") ||
						strings.Contains(val, "nav") ||
						strings.Contains(val, "sidebar") ||
						strings.Contains(val, "widget") {
						return true
					}
				}
			}
		}
		curr = curr.Parent
	}
	return false
}

func (c *Crawler) fetchListingItems(ctx context.Context, pageURL string) ([]itemRef, error) {
	bodyBytes, _, err := c.doRequestWithAutoWait(ctx, pageURL)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}

	var items []itemRef
	seenIndex := make(map[string]int)
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			if isInsideNavOrFooter(n) {
				return
			}
			var href string
			for _, a := range n.Attr {
				if a.Key == "href" {
					href = a.Val
					break
				}
			}
			if href != "" && (strings.Contains(href, ".html") || strings.Contains(href, "/node/")) {
				text := getText(n)
				cleanTitle := strings.TrimSpace(text)
				if cleanTitle != "" {
					fullURL := href
					if !strings.HasPrefix(fullURL, "http") {
						fullURL = strings.TrimRight(c.cfg.BaseURL, "/") + "/" + strings.TrimLeft(href, "/")
					}
					if c.isAllowedHost(fullURL) {
						itemDate := ""
						curr := n.Parent
						for depth := 0; depth < 3 && curr != nil; depth++ {
							itemDate = extractDateFromHTML(getText(curr))
							if itemDate != "" {
								break
							}
							curr = curr.Parent
						}
						if idx, exists := seenIndex[fullURL]; exists {
							existing := &items[idx]
							if existing.Date == "" && itemDate != "" {
								existing.Date = itemDate
							}
							if (isGenericTitle(existing.Title) && !isGenericTitle(cleanTitle)) ||
								(len(cleanTitle) > len(existing.Title) && !isGenericTitle(cleanTitle)) {
								existing.Title = cleanTitle
							}
						} else {
							seenIndex[fullURL] = len(items)
							items = append(items, itemRef{
								URL:   fullURL,
								Title: cleanTitle,
								Date:  itemDate,
							})
						}
					}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(doc)

	return items, nil
}

func (c *Crawler) fetchAndProcessNode(ctx context.Context, itemURL, docID, catType, defaultDate, defaultTitle string) (*CrawledDocument, error) {
	if !c.isAllowedHost(itemURL) {
		return nil, fmt.Errorf("crawler target outside allowed base domain: %s", itemURL)
	}

	lowerURL := strings.ToLower(itemURL)
	if strings.HasSuffix(lowerURL, ".pdf") || strings.HasSuffix(lowerURL, ".doc") || strings.HasSuffix(lowerURL, ".docx") || strings.HasSuffix(lowerURL, ".xlsx") || strings.HasSuffix(lowerURL, ".xls") {
		attBytes, _, err := c.doAttachmentRequestWithAutoWait(ctx, itemURL)
		if err != nil {
			return nil, err
		}
		attText, _ := c.extractAttachmentText(ctx, attBytes, filepath.Base(itemURL))
		title := strings.TrimSpace(defaultTitle)
		if isGenericTitle(title) {
			attFileName := filepath.Base(itemURL)
			title = strings.TrimSuffix(attFileName, filepath.Ext(attFileName))
		}
		if strings.TrimSpace(attText) == "" {
			attText = title
		}
		return &CrawledDocument{
			ID:          docID,
			Category:    catType,
			URL:         itemURL,
			Title:       title,
			PublishDate: defaultDate,
			Content:     attText,
		}, nil
	}

	bodyBytes, _, err := c.doRequestWithAutoWait(ctx, itemURL)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}

	title := extractTitle(doc)
	if isGenericTitle(title) {
		title = strings.TrimSpace(defaultTitle)
	}
	if isGenericTitle(title) {
		title = strings.TrimSpace(catType)
	}

	publishDate := defaultDate
	if publishDate == "" {
		publishDate = extractDateFromHTML(string(bodyBytes))
	}

	var sb strings.Builder
	md := pipeline.HTMLToMarkdown(string(bodyBytes))
	if strings.TrimSpace(md) != "" && !isGenericTitle(md) {
		sb.WriteString(md)
		sb.WriteString("\n\n")
	}

	driveID := c.extractDriveFileID(string(bodyBytes))
	if driveID != "" {
		dlURL := fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", driveID)
		if attBytes, _, err := c.doAttachmentRequestWithAutoWait(ctx, dlURL); err == nil && len(attBytes) > 0 {
			if bytes.Contains(attBytes, []byte("drive.google.com/uc?export=download&amp;confirm=")) {
				confirmRe := regexp.MustCompile(`drive\.google\.com/uc\?export=download&amp;confirm=([a-zA-Z0-9_-]+)`)
				if m := confirmRe.FindSubmatch(attBytes); len(m) > 1 {
					confirmURL := fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s&confirm=%s", driveID, string(m[1]))
					if confBytes, _, err := c.doAttachmentRequestWithAutoWait(ctx, confirmURL); err == nil && len(confBytes) > 0 {
						attBytes = confBytes
					}
				}
			}
			if !bytes.Contains(attBytes, []byte("Google Drive - Can't download file")) &&
				!bytes.Contains(attBytes, []byte("Sorry, the owner hasn't given you permission to download this file")) {
				if attText, err := c.extractAttachmentText(ctx, attBytes, ""); err == nil && strings.TrimSpace(attText) != "" {
					sb.WriteString(attText)
					sb.WriteString("\n\n")
				}
			}
		}
	}

	attMatches := attachmentLinkRegex.FindAllStringSubmatch(string(bodyBytes), -1)
	for _, am := range attMatches {
		if len(am) > 1 {
			relAtt := am[1]
			if isIgnoredAttachment(relAtt) {
				continue
			}
			fullAttURL := strings.TrimRight(c.cfg.BaseURL, "/") + "/" + strings.TrimLeft(relAtt, "/")
			if attBytes, _, err := c.doAttachmentRequestWithAutoWait(ctx, fullAttURL); err == nil && len(attBytes) > 0 {
				if attText, err := c.extractAttachmentText(ctx, attBytes, filepath.Base(relAtt)); err == nil && strings.TrimSpace(attText) != "" {
					sb.WriteString(attText)
					sb.WriteString("\n\n")
				}
			}
		}
	}

	fullContent := strings.TrimSpace(sb.String())
	if (fullContent == "" || isGenericTitle(fullContent)) && c.cfg.VisionOCR != nil {
		imgMatches := inlineImageRegex.FindAllStringSubmatch(string(bodyBytes), -1)
		for _, im := range imgMatches {
			if len(im) > 1 {
				fullImgURL := strings.TrimRight(c.cfg.BaseURL, "/") + "/" + strings.TrimLeft(im[1], "/")
				if imgBytes, _, err := c.doAttachmentRequestWithAutoWait(ctx, fullImgURL); err == nil && len(imgBytes) > 0 {
					if ocrText, err := c.cfg.VisionOCR(ctx, [][]byte{imgBytes}); err == nil && strings.TrimSpace(ocrText) != "" {
						fullContent = strings.TrimSpace(ocrText)
						break
					}
				}
			}
		}
	}

	if fullContent == "" || isGenericTitle(fullContent) {
		fullContent = title
	}

	return &CrawledDocument{
		ID:          docID,
		Category:    catType,
		URL:         itemURL,
		Title:       title,
		PublishDate: publishDate,
		Content:     fullContent,
	}, nil
}

func (c *Crawler) extractDriveFileID(htmlStr string) string {
	if m := iframeDriveRegex.FindStringSubmatch(htmlStr); len(m) > 1 {
		if !c.isDriveIDIgnored(m[1]) {
			return m[1]
		}
	}
	all := driveFileIDRegex.FindAllStringSubmatch(htmlStr, -1)
	for _, m := range all {
		if len(m) > 1 && !c.isDriveIDIgnored(m[1]) {
			return m[1]
		}
	}
	return ""
}

func (c *Crawler) isDriveIDIgnored(id string) bool {
	for _, ignored := range c.cfg.IgnoredDriveIDs {
		if ignored == id {
			return true
		}
	}
	return false
}

func isIgnoredAttachment(relPath string) bool {
	lower := strings.ToLower(relPath)
	return strings.Contains(lower, "gcn-kdcl") || strings.Contains(lower, "gcnkdcl")
}

func extractNodeID(urlStr string) string {
	if m := nodeIDRegex.FindStringSubmatch(urlStr); len(m) > 1 {
		return m[1]
	}
	if m := nodePathRegex.FindStringSubmatch(urlStr); len(m) > 1 {
		return m[1]
	}
	h := sha256.Sum256([]byte(urlStr))
	return fmt.Sprintf("%x", h)[:8]
}

func isGenericTitle(t string) bool {
	clean := strings.ToLower(strings.TrimSpace(t))
	return clean == "" || clean == "tìm kiếm" || clean == "search" || strings.HasPrefix(clean, "tìm kiếm")
}

func extractTitle(n *html.Node) string {
	var h1Title string
	var ogTitle string
	var pageTitle string

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch node.Data {
			case "meta":
				var prop, content string
				for _, a := range node.Attr {
					if strings.EqualFold(a.Key, "property") || strings.EqualFold(a.Key, "name") {
						prop = strings.ToLower(strings.TrimSpace(a.Val))
					} else if strings.EqualFold(a.Key, "content") {
						content = strings.TrimSpace(a.Val)
					}
				}
				if (prop == "og:title" || prop == "twitter:title") && !isGenericTitle(content) {
					clean := strings.TrimSuffix(content, " | Trường Đại học Thăng Long")
					clean = strings.TrimSuffix(clean, " - Trường Đại học Thăng Long")
					ogTitle = strings.TrimSpace(clean)
				}
			case "title":
				raw := strings.TrimSpace(getText(node))
				if !isGenericTitle(raw) {
					clean := strings.TrimSuffix(raw, " | Trường Đại học Thăng Long")
					clean = strings.TrimSuffix(clean, " - Trường Đại học Thăng Long")
					pageTitle = strings.TrimSpace(clean)
				}
			case "h1":
				var isSearchbox bool
				for _, a := range node.Attr {
					if strings.EqualFold(a.Key, "class") && strings.Contains(strings.ToLower(a.Val), "searchbox") {
						isSearchbox = true
						break
					}
				}
				if !isSearchbox && h1Title == "" {
					text := strings.TrimSpace(getText(node))
					if !isGenericTitle(text) {
						h1Title = text
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)

	if h1Title != "" {
		return h1Title
	}
	if ogTitle != "" {
		return ogTitle
	}
	return pageTitle
}

func (c *Crawler) calculateJitter(maxMs int) int {
	if maxMs <= 0 {
		return 0
	}
	var b [4]byte
	_, _ = rand.Read(b[:])
	return int(binary.BigEndian.Uint32(b[:]) % uint32(maxMs))
}

func extractDateFromHTML(raw string) string {
	dateRegexVN := regexp.MustCompile(`(\d{1,2})[/.-](\d{1,2})[/.-](\d{4})`)
	if m := dateRegexVN.FindStringSubmatch(raw); len(m) > 3 {
		day, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		year, _ := strconv.Atoi(m[3])
		if year >= 2000 && year <= 2099 && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		}
	}
	dateRegexISO := regexp.MustCompile(`(\d{4})-(\d{1,2})-(\d{1,2})`)
	if m := dateRegexISO.FindStringSubmatch(raw); len(m) > 3 {
		year, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		day, _ := strconv.Atoi(m[3])
		if year >= 2000 && year <= 2099 && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		}
	}
	return ""
}

func getText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return sb.String()
}

func (c *Crawler) extractAttachmentText(ctx context.Context, data []byte, filenameHint string) (string, error) {
	if len(data) == 0 {
		return "", nil
	}

	lowerHint := strings.ToLower(filenameHint)

	if bytes.HasPrefix(data, pdfMagic) || strings.HasSuffix(lowerHint, ".pdf") {
		text, errPdf := pipeline.ExtractPDFBytesPure(data)
		usable := errPdf == nil && len(strings.TrimSpace(text)) >= 150
		if usable && !pipeline.IsGarbledText(text) {
			return text, nil
		}
		if c.cfg.VisionOCR != nil {
			ocrText, errOCR := c.renderPDFAndOCR(ctx, data)
			if errOCR == nil && len(strings.TrimSpace(ocrText)) > 0 && (!usable || pipeline.TextGarbleRatio(ocrText) < pipeline.TextGarbleRatio(text)) {
				return ocrText, nil
			}
		}
		if errPdf == nil {
			return text, nil
		}
		return "", errPdf
	}

	if bytes.HasPrefix(data, oleMagic) || strings.HasSuffix(lowerHint, ".doc") || strings.HasSuffix(lowerHint, ".xls") {
		if oleText := extractOLETextPure(data); strings.TrimSpace(oleText) != "" {
			return oleText, nil
		}
	}

	if bytes.HasPrefix(data, zipMagic) || strings.HasSuffix(lowerHint, ".docx") || strings.HasSuffix(lowerHint, ".xlsx") {
		name := filenameHint
		if name == "" {
			name = "attachment.docx"
		}
		doc, err := docparser.ParseBytes(name, data)
		if err == nil && doc != nil && strings.TrimSpace(doc.TextContent) != "" {
			return strings.TrimSpace(doc.TextContent), nil
		}
	}

	isJPEG := len(data) > 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF
	isPNG := len(data) > 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})
	isWEBP := len(data) > 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"

	if isJPEG || isPNG || isWEBP || strings.HasSuffix(lowerHint, ".png") || strings.HasSuffix(lowerHint, ".jpg") || strings.HasSuffix(lowerHint, ".jpeg") || strings.HasSuffix(lowerHint, ".webp") {
		if c.cfg.VisionOCR != nil {
			ocrText, err := c.cfg.VisionOCR(ctx, [][]byte{data})
			if err == nil && strings.TrimSpace(ocrText) != "" {
				return strings.TrimSpace(ocrText), nil
			}
		}
	}

	if strings.HasSuffix(lowerHint, ".csv") || strings.HasSuffix(lowerHint, ".txt") {
		return string(data), nil
	}

	return "", fmt.Errorf("unsupported attachment format")
}

const ocrBatchSize = 1

func (c *Crawler) renderPDFAndOCR(ctx context.Context, data []byte) (string, error) {
	if c.cfg.VisionOCR == nil {
		return "", fmt.Errorf("vision OCR handler not configured")
	}

	allImages, err := pdf.ExtractImagesFromPDFBytes(data, 15)
	if err != nil || len(allImages) == 0 {
		return "", fmt.Errorf("no images extracted from pdf for ocr: %w", err)
	}

	var sb strings.Builder
	for i := 0; i < len(allImages); i += ocrBatchSize {
		end := min(i+ocrBatchSize, len(allImages))
		partText, errOCR := c.cfg.VisionOCR(ctx, allImages[i:end])
		if errOCR != nil {
			log.Warn().Err(errOCR).Int("from_page", i+1).Msg("Vision OCR failed for PDF pages")
			continue
		}
		if strings.TrimSpace(partText) != "" {
			sb.WriteString(strings.TrimSpace(partText))
			sb.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(sb.String()), nil
}

func extractOLETextPure(data []byte) string {
	var sb strings.Builder
	var cur []rune

	flush := func() {
		if len(cur) >= 4 {
			s := strings.TrimSpace(string(cur))
			if len(s) >= 4 {
				sb.WriteString(s)
				sb.WriteString("\n")
			}
		}
		cur = cur[:0]
	}

	for i := 0; i+1 < len(data); i += 2 {
		r := rune(uint16(data[i]) | uint16(data[i+1])<<8)
		if (r >= 32 && r < 127) || (r >= 0x00C0 && r <= 0x1EF9) || r == '\n' || r == '\t' {
			cur = append(cur, r)
		} else {
			flush()
		}
	}
	flush()

	if sb.Len() > 50 {
		return pipeline.StitchBrokenVietnameseLines(sb.String())
	}
	return ""
}

package pipeline

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
	"golang.org/x/net/html"
)

type CleanDocument struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Category    string   `json:"category"`
	PublishDate string   `json:"publish_date"`
	SourceURL   string   `json:"source_url"`
	Priority    int      `json:"priority"`
	Content     string   `json:"content"`
	HasPDF      bool     `json:"has_pdf"`
	PDFPaths    []string `json:"pdf_paths,omitempty"`
}

type CleanerConfig struct {
	DataRawDir   string
	DataCleanDir string
}

type Cleaner struct {
	cfg CleanerConfig
}

func NewCleaner(cfg CleanerConfig) *Cleaner {
	if cfg.DataRawDir == "" {
		cfg.DataRawDir = "data/raw"
	}
	if cfg.DataCleanDir == "" {
		cfg.DataCleanDir = "data/clean"
	}
	_ = os.MkdirAll(cfg.DataCleanDir, 0755)
	return &Cleaner{cfg: cfg}
}

func (c *Cleaner) ProcessAll(ctx context.Context, onProgress func(cur, total int64)) error {
	indexFile := filepath.Join(c.cfg.DataRawDir, "raw_index.jsonl")
	f, err := os.Open(indexFile)
	if err != nil {
		return err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			lines = append(lines, text)
		}
	}

	total := int64(len(lines))
	for i, line := range lines {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err == nil {
			_ = c.ProcessRecord(rec)
		}

		if onProgress != nil {
			onProgress(int64(i+1), total)
		}
	}

	return nil
}

func (c *Cleaner) ProcessRecord(rec map[string]any) error {
	nodeID, _ := rec["node_id"].(string)
	if nodeID == "" {
		return nil
	}
	recType, _ := rec["type"].(string)
	title, _ := rec["title"].(string)
	sourceURL, _ := rec["url"].(string)
	publishDateRaw, _ := rec["publish_date"].(string)
	htmlRelPath, _ := rec["html_path"].(string)

	publishDate := normalizeDate(publishDateRaw)

	priority := 50
	if strings.Contains(strings.ToLower(title), "học phí") {
		priority = 90
	} else if strings.Contains(strings.ToLower(title), "quy chế") || strings.Contains(strings.ToLower(title), "chuẩn đầu ra") {
		priority = 85
	} else if strings.Contains(strings.ToLower(title), "nghỉ học") || strings.Contains(strings.ToLower(title), "lịch thi") {
		priority = 75
	}

	var contentBuilder strings.Builder

	if htmlRelPath != "" {
		htmlBytes, err := os.ReadFile(htmlRelPath)
		if err == nil && len(htmlBytes) > 0 {
			md := HTMLToMarkdown(string(htmlBytes))
			if strings.TrimSpace(md) != "" {
				contentBuilder.WriteString(md)
				contentBuilder.WriteString("\n\n")
			}
		}
	}

	if localPDFs, ok := rec["local_pdf_paths"].([]any); ok {
		for _, p := range localPDFs {
			if pdfPath, ok := p.(string); ok && pdfPath != "" {
				pdfText, err := ExtractPDFTextPure(pdfPath)
				if err == nil && strings.TrimSpace(pdfText) != "" {
					contentBuilder.WriteString("### Nội dung tài liệu đính kèm (PDF):\n\n")
					contentBuilder.WriteString(pdfText)
					contentBuilder.WriteString("\n\n")
				}
			}
		}
	}

	fullContent := strings.TrimSpace(contentBuilder.String())
	if fullContent == "" {
		fullContent = title
	}

	docID := fmt.Sprintf("%s_%s", strings.ToUpper(recType), nodeID)
	doc := CleanDocument{
		ID:          docID,
		Title:       title,
		Category:    recType,
		PublishDate: publishDate,
		SourceURL:   sourceURL,
		Priority:    priority,
		Content:     fullContent,
	}

	return c.saveCleanDocument(doc)
}

func (c *Cleaner) saveCleanDocument(doc CleanDocument) error {
	outPath := filepath.Join(c.cfg.DataCleanDir, doc.ID+".md")

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", doc.ID))
	sb.WriteString(fmt.Sprintf("title: %q\n", doc.Title))
	sb.WriteString(fmt.Sprintf("category: %s\n", doc.Category))
	sb.WriteString(fmt.Sprintf("publish_date: %s\n", doc.PublishDate))
	sb.WriteString(fmt.Sprintf("source_url: %s\n", doc.SourceURL))
	sb.WriteString(fmt.Sprintf("priority: %d\n", doc.Priority))
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %s\n\n", doc.Title))
	sb.WriteString(doc.Content)
	sb.WriteString("\n")

	return os.WriteFile(outPath, []byte(sb.String()), 0644)
}

func ExtractPDFTextPure(pdfPath string) (string, error) {
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	plainReader, err := r.GetPlainText()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	_, err = buf.ReadFrom(plainReader)
	if err != nil {
		return "", err
	}

	return cleanExtractedText(buf.String()), nil
}

func HTMLToMarkdown(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}

	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "nav", "footer", "header", "aside":
				return
			case "h1":
				sb.WriteString("\n# ")
				sb.WriteString(cleanNodeText(n))
				sb.WriteString("\n\n")
				return
			case "h2":
				sb.WriteString("\n## ")
				sb.WriteString(cleanNodeText(n))
				sb.WriteString("\n\n")
				return
			case "h3":
				sb.WriteString("\n### ")
				sb.WriteString(cleanNodeText(n))
				sb.WriteString("\n\n")
				return
			case "p":
				pText := cleanNodeText(n)
				if pText != "" {
					sb.WriteString(pText)
					sb.WriteString("\n\n")
				}
				return
			case "li":
				sb.WriteString("- ")
				sb.WriteString(cleanNodeText(n))
				sb.WriteString("\n")
				return
			case "table":
				sb.WriteString(convertHTMLTableToMarkdown(n))
				sb.WriteString("\n\n")
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return cleanExtractedText(sb.String())
}

func convertHTMLTableToMarkdown(tableNode *html.Node) string {
	var rows [][]string

	var findRows func(*html.Node)
	findRows = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			var cells []string
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
					cells = append(cells, cleanNodeText(c))
				}
			}
			if len(cells) > 0 {
				rows = append(rows, cells)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findRows(c)
		}
	}
	findRows(tableNode)

	if len(rows) == 0 {
		return ""
	}

	maxCols := 0
	for _, r := range rows {
		if len(r) > maxCols {
			maxCols = len(r)
		}
	}

	var sb strings.Builder
	headers := rows[0]
	sb.WriteString("| ")
	for i := 0; i < maxCols; i++ {
		val := ""
		if i < len(headers) {
			val = headers[i]
		}
		sb.WriteString(val)
		sb.WriteString(" | ")
	}
	sb.WriteString("\n|")
	for i := 0; i < maxCols; i++ {
		sb.WriteString(" --- |")
	}
	sb.WriteString("\n")

	for _, r := range rows[1:] {
		sb.WriteString("| ")
		for i := 0; i < maxCols; i++ {
			val := ""
			if i < len(r) {
				val = r[i]
			}
			sb.WriteString(val)
			sb.WriteString(" | ")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func cleanNodeText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(sb.String()), " ")
}

func cleanExtractedText(s string) string {
	lines := strings.Split(s, "\n")
	var cleaned []string
	emptyCount := 0
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			emptyCount++
			if emptyCount <= 1 {
				cleaned = append(cleaned, "")
			}
		} else {
			emptyCount = 0
			cleaned = append(cleaned, trimmed)
		}
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func normalizeDate(dateStr string) string {
	dateStr = strings.TrimSpace(dateStr)
	parts := strings.Split(dateStr, ".")
	if len(parts) == 3 && len(parts[2]) == 4 {
		return fmt.Sprintf("%04s-%02s-%02s", parts[2], parts[1], parts[0])
	}
	partsSlash := strings.Split(dateStr, "/")
	if len(partsSlash) == 3 && len(partsSlash[2]) == 4 {
		return fmt.Sprintf("%04s-%02s-%02s", partsSlash[2], partsSlash[1], partsSlash[0])
	}
	if len(dateStr) == 10 && dateStr[4] == '-' && dateStr[7] == '-' {
		return dateStr
	}
	return time.Now().Format("2006-01-02")
}

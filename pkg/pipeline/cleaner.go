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
	"unicode"

	"golang.org/x/net/html"

	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/pdf"
)

// CleanDocument represents a sanitized markdown document ready for embedding.
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

// PriorityResolver determines a numerical ranking priority for a given category and title.
type PriorityResolver func(category, title string) int

// CleanerConfig specifies storage paths and ranking callbacks for document cleaning.
type CleanerConfig struct {
	DataRawDir       string
	DataCleanDir     string
	PriorityResolver PriorityResolver
}

// Cleaner coordinates HTML denoising, PDF text extraction, and Markdown normalization.
type Cleaner struct {
	cfg CleanerConfig
}

// NewCleaner initializes a document cleaner with resolved target directories.
func NewCleaner(cfg CleanerConfig) *Cleaner {
	if cfg.DataRawDir == "" {
		cfg.DataRawDir = constants.DefaultRawDir
	}
	if cfg.DataCleanDir == "" {
		cfg.DataCleanDir = constants.DefaultCleanDir
	}
	_ = os.MkdirAll(cfg.DataCleanDir, 0755)
	return &Cleaner{cfg: cfg}
}

// ProcessAll reads all entries from raw_index.jsonl and generates sanitized markdown documents.
func (c *Cleaner) ProcessAll(ctx context.Context, onProgress func(cur, total int64)) error {
	indexFile := filepath.Join(c.cfg.DataRawDir, filepath.Base(constants.DefaultRawIndex))
	f, err := os.Open(indexFile)
	if err != nil {
		if os.IsNotExist(err) {
			if onProgress != nil {
				onProgress(0, 0)
			}
			return nil
		}
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
	if err := scanner.Err(); err != nil {
		return err
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
			_ = c.ProcessRecord(ctx, rec)
		}

		if onProgress != nil {
			onProgress(int64(i+1), total)
		}
	}

	return nil
}

// ProcessRecord transforms a single raw crawler record into clean markdown.
func (c *Cleaner) ProcessRecord(ctx context.Context, rec map[string]any) error {
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
	if c.cfg.PriorityResolver != nil {
		priority = c.cfg.PriorityResolver(recType, title)
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
				pdfText, err := ExtractPDFTextPure(ctx, pdfPath)
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

// ExtractPDFTextPure extracts text from PDF documents using structural table detection or Poppler/pure Go fallback.
func ExtractPDFTextPure(ctx context.Context, pdfPath string) (extracted string, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("pdf text extractor recovered from panic: %v", rec)
			extracted = ""
		}
	}()

	data, errRead := os.ReadFile(pdfPath)
	if errRead == nil && len(data) > 0 {
		return ExtractPDFBytesPure(data)
	}

	f, r, errPdf := pdf.Open(pdfPath)
	if errPdf != nil {
		return "", errPdf
	}
	defer f.Close()

	text, errClean := r.ExtractCleanText()
	if errClean != nil || strings.TrimSpace(text) == "" {
		if plainReader, errPlain := r.GetPlainText(); errPlain == nil {
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(plainReader)
			text = buf.String()
		}
	}

	return cleanExtractedText(text), nil
}

// ExtractPDFBytesPure extracts text directly from in-memory PDF bytes using pure Go reader.
func ExtractPDFBytesPure(data []byte) (extracted string, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("pdf bytes extractor recovered from panic: %v", rec)
			extracted = ""
		}
	}()

	if structText, errStruct := ExtractPDFWithStructure(context.Background(), data); errStruct == nil && strings.TrimSpace(structText) != "" {
		return cleanExtractedText(structText), nil
	}

	r, errReader := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if errReader != nil {
		return "", errReader
	}

	text, errClean := r.ExtractCleanText()
	if errClean != nil || strings.TrimSpace(text) == "" {
		if plainReader, errPlain := r.GetPlainText(); errPlain == nil {
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(plainReader)
			text = buf.String()
		}
	}

	return cleanExtractedText(text), nil
}

// HTMLToMarkdown strips presentation noise and converts core article elements to Markdown.
func HTMLToMarkdown(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}

	targetNode := findContentContainer(doc)
	if targetNode == nil {
		targetNode = doc
	}

	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "nav", "footer", "header", "aside", "svg", "noscript", "iframe", "form", "button":
				return
			}

			for _, a := range n.Attr {
				if a.Key == "class" {
					lower := strings.ToLower(a.Val)
					if strings.Contains(lower, "searchbox") ||
						strings.Contains(lower, "menu-mobile") ||
						strings.Contains(lower, "breadcrumb-social") ||
						strings.Contains(lower, "new-detail__social") ||
						strings.Contains(lower, "breadcrumb-nav") ||
						strings.Contains(lower, "breadcrumb") ||
						strings.Contains(lower, "hidden") ||
						strings.Contains(lower, "visually-hidden") {
						return
					}
				}
			}

			switch n.Data {
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
			case "h3", "h4":
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
				liText := cleanNodeText(n)
				if liText != "" {
					sb.WriteString("- ")
					sb.WriteString(liText)
					sb.WriteString("\n")
				}
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
	walk(targetNode)

	return cleanExtractedText(sb.String())
}

func findContentContainer(root *html.Node) *html.Node {
	var specificNode *html.Node
	var pageContentNode *html.Node

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, a := range n.Attr {
				if a.Key == "class" {
					val := a.Val
					if strings.Contains(val, "new-detail__entry") ||
						strings.Contains(val, "field--name-field-article-content") ||
						strings.Contains(val, "field--name-body") ||
						strings.Contains(val, "content-detail") {
						if specificNode == nil {
							specificNode = n
						}
					}
					if strings.Contains(val, "page-content") || strings.Contains(val, "layout__content") {
						if pageContentNode == nil {
							pageContentNode = n
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(root)

	if specificNode != nil {
		return specificNode
	}
	if pageContentNode != nil {
		return pageContentNode
	}
	return nil
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

var vietnameseSyllableCodas = map[string]bool{
	"ng": true, "c": true, "p": true, "t": true, "m": true, "n": true,
	"nh": true, "ch": true, "i": true, "y": true, "u": true, "o": true,
	"a": true, "e": true,
	"NG": true, "C": true, "P": true, "T": true, "M": true, "N": true,
	"NH": true, "CH": true, "I": true, "Y": true, "U": true, "O": true,
	"A": true, "E": true,
}

// StitchBrokenVietnameseLines repairs fragmented text where vowels or syllables were split across newlines.
func StitchBrokenVietnameseLines(text string) string {
	paragraphs := strings.Split(text, "\n\n")
	var stitchedParagraphs []string

	for _, para := range paragraphs {
		rawLines := strings.Split(para, "\n")
		var lines []string
		for _, l := range rawLines {
			trimmed := strings.TrimSpace(l)
			if trimmed != "" {
				lines = append(lines, trimmed)
			}
		}
		if len(lines) == 0 {
			continue
		}

		var rebuilt []string
		for _, line := range lines {
			if len(rebuilt) == 0 {
				rebuilt = append(rebuilt, line)
				continue
			}

			prev := rebuilt[len(rebuilt)-1]
			prevRunes := []rune(prev)
			lineRunes := []rune(line)
			if len(prevRunes) == 0 || len(lineRunes) == 0 {
				rebuilt = append(rebuilt, line)
				continue
			}

			prevLast := prevRunes[len(prevRunes)-1]
			lineFirst := lineRunes[0]

			if strings.HasPrefix(line, "|") || strings.HasPrefix(prev, "|") || strings.HasPrefix(line, "#") || strings.HasPrefix(prev, "#") {
				rebuilt = append(rebuilt, line)
				continue
			}

			if len(lineRunes) == 1 && strings.ContainsRune(",.:;)%]}", lineFirst) {
				rebuilt[len(rebuilt)-1] = prev + line
				continue
			}

			if lineFirst == '/' && !strings.ContainsRune(".!?:", prevLast) {
				rebuilt[len(rebuilt)-1] = prev + " " + line
				continue
			}

			if lineFirst == '@' || prevLast == '@' || (strings.HasSuffix(prev, ".") && strings.Contains(line, "edu.vn")) {
				rebuilt[len(rebuilt)-1] = prev + line
				continue
			}

			if len(lineRunes) == 1 {
				if unicode.IsLetter(lineFirst) {
					if unicode.IsLetter(prevLast) {
						rebuilt[len(rebuilt)-1] = prev + line
						continue
					}
					if strings.ContainsRune("-–:(", prevLast) {
						rebuilt[len(rebuilt)-1] = prev + " " + line
						continue
					}
				} else if unicode.IsDigit(lineFirst) {
					if strings.ContainsRune("/-–", prevLast) {
						rebuilt[len(rebuilt)-1] = prev + line
						continue
					}
				}
				if (line == "h" || line == "H" || line == "g" || line == "p") && unicode.IsDigit(prevLast) {
					rebuilt[len(rebuilt)-1] = prev + line
					continue
				}
			}

			if strings.ContainsRune("-–/", prevLast) {
				rebuilt[len(rebuilt)-1] = prev + " " + line
				continue
			}

			if (unicode.IsDigit(prevLast) && lineFirst == '/') ||
				(prevLast == '/' && unicode.IsDigit(lineFirst)) {
				rebuilt[len(rebuilt)-1] = prev + line
				continue
			}

			tokens := strings.SplitN(line, " ", 2)
			firstTok := tokens[0]
			cleanTok := strings.TrimRight(firstTok, ".,:;!?)%]}")
			rest := ""
			if len(tokens) > 1 {
				rest = " " + tokens[1]
			}

			isLowerCoda := unicode.IsLower(prevLast) && unicode.IsLower(lineFirst) && vietnameseSyllableCodas[cleanTok]
			isUpperCoda := unicode.IsUpper(prevLast) && unicode.IsUpper(lineFirst) && vietnameseSyllableCodas[cleanTok]
			if (isLowerCoda || isUpperCoda) && !strings.ContainsRune(".!?:;,()[]", prevLast) {
				rebuilt[len(rebuilt)-1] = prev + firstTok + rest
				continue
			}

			if unicode.IsLetter(prevLast) && unicode.IsLower(lineFirst) && !strings.ContainsRune(".!?:", prevLast) {
				rebuilt[len(rebuilt)-1] = prev + " " + line
				continue
			}

			rebuilt = append(rebuilt, line)
		}

		stitchedParagraphs = append(stitchedParagraphs, strings.Join(rebuilt, "\n"))
	}

	return collapseDanglingParagraphs(strings.Join(stitchedParagraphs, "\n\n"))
}

func collapseDanglingParagraphs(text string) string {
	paras := strings.Split(text, "\n\n")
	var result []string
	for _, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if len(result) == 0 {
			result = append(result, p)
			continue
		}
		prev := result[len(result)-1]
		prevRunes := []rune(prev)
		pRunes := []rune(p)

		if len(pRunes) > 0 && unicode.IsLower(pRunes[0]) && !strings.ContainsRune(".!?:", prevRunes[len(prevRunes)-1]) {
			result[len(result)-1] = prev + " " + p
			continue
		}

		if len(prevRunes) > 0 && strings.ContainsRune("-–", prevRunes[len(prevRunes)-1]) {
			result[len(result)-1] = prev + " " + p
			continue
		}

		if len(prevRunes) <= 4 && len(pRunes) <= 6 && unicode.IsUpper(prevRunes[0]) && unicode.IsUpper(pRunes[0]) {
			result[len(result)-1] = prev + " " + p
			continue
		}

		result = append(result, p)
	}
	return strings.Join(result, "\n\n")
}

func cleanExtractedText(s string) string {
	blocks := strings.Split(s, "\n\n")
	var cleanedBlocks []string
	for _, block := range blocks {
		trimmed := strings.TrimSpace(block)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "|") || strings.HasPrefix(trimmed, "###") {
			cleanedBlocks = append(cleanedBlocks, trimmed)
		} else {
			cleanedBlocks = append(cleanedBlocks, StitchBrokenVietnameseLines(trimmed))
		}
	}
	s = strings.Join(cleanedBlocks, "\n\n")
	s = strings.ReplaceAll(s, "\x00", "")
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r) {
			b.WriteRune(r)
		}
	}
	lines := strings.Split(b.String(), "\n")
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

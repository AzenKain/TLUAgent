package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"

	"tluagent-web/pkg/pdf"
)

var (
	tableCodeRegex  = regexp.MustCompile(`^[A-Z]{2,4}\d{3,4}$`)
	tableSecRegex   = regexp.MustCompile(`^(I|II|III|IV|V|[A-Z]|\d+\.|\([a-z]\))\s*[-–]\s*`)
	pageFooterRegex = regexp.MustCompile(`(?i)^Page\s+\d+`)
	numDotRegex     = regexp.MustCompile(`(\d+)\s*\.\s*(\d+)`)
)

type textCell struct {
	Left float64
	Text string
}

func formatNumberSpacing(s string) string {
	s = numDotRegex.ReplaceAllString(s, "$1.$2")
	return numDotRegex.ReplaceAllString(s, "$1.$2")
}

func isNumericOrCurrency(s string) bool {
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isTableRow(cells []string) bool {
	if len(cells) < 2 {
		return false
	}
	hasCode := false
	hasNumeric := false
	for _, c := range cells {
		trimmed := strings.TrimSpace(c)
		if tableCodeRegex.MatchString(trimmed) {
			hasCode = true
		}
		if isNumericOrCurrency(trimmed) {
			hasNumeric = true
		}
	}
	if hasCode {
		return true
	}
	if hasNumeric && len(cells) >= 3 {
		return true
	}
	return false
}

func isSectionHeading(line string) bool {
	trimmed := strings.TrimSpace(line)
	return tableSecRegex.MatchString(trimmed)
}

func buildTableHeader(cells []string) string {
	cols := len(cells)
	if cols == 5 {
		return "| STT | Tên môn học / Học phần | Mã học phần | Số tín chỉ | Học phí được miễn giảm |\n|---|---|---|---|---|\n"
	}
	if cols == 6 {
		return "| Ngành / Khoa | Mã học phần | Tên học phần | Số tín chỉ | Mức miễn giảm 1 | Mức miễn giảm 2 |\n|---|---|---|---|---|---|\n"
	}
	var headers []string
	var seps []string
	for i := 1; i <= cols; i++ {
		headers = append(headers, fmt.Sprintf("Cột %d", i))
		seps = append(seps, "---")
	}
	return "| " + strings.Join(headers, " | ") + " |\n| " + strings.Join(seps, " | ") + " |\n"
}

// ExtractPDFWithStructure converts PDF bytes into structured markdown using pure Go geometry analysis.
func ExtractPDFWithStructure(ctx context.Context, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty pdf bytes")
	}

	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}

	num := r.NumPage()
	var pageOutputs []string
	for i := 1; i <= num; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		p := r.Page(i)
		parsed := parseSinglePDFPage(p)
		if strings.TrimSpace(parsed) != "" {
			pageOutputs = append(pageOutputs, parsed)
		}
	}

	if len(pageOutputs) == 0 {
		return "", fmt.Errorf("no content extracted from pdf")
	}

	return strings.Join(pageOutputs, "\n\n"), nil
}

// ExtractPDFFileWithStructure converts a PDF file into structured markdown using pure Go geometry analysis.
func ExtractPDFFileWithStructure(ctx context.Context, pdfPath string) (string, error) {
	data, err := os.ReadFile(pdfPath)
	if err != nil {
		return "", err
	}
	return ExtractPDFWithStructure(ctx, data)
}

func parseSinglePDFPage(p pdf.Page) string {
	content := p.Content()
	if len(content.Text) == 0 {
		clean, _ := p.ExtractCleanText()
		return clean
	}

	type rowGroup struct {
		Y     float64
		Texts []pdf.Text
	}
	var rowList []rowGroup
	for _, t := range content.Text {
		found := false
		for ri := range rowList {
			if math.Abs(rowList[ri].Y-t.Y) <= 3.0 {
				rowList[ri].Texts = append(rowList[ri].Texts, t)
				found = true
				break
			}
		}
		if !found {
			rowList = append(rowList, rowGroup{Y: t.Y, Texts: []pdf.Text{t}})
		}
	}

	sort.SliceStable(rowList, func(i, j int) bool {
		return rowList[i].Y > rowList[j].Y
	})

	var sb strings.Builder
	inTable := false

	for _, rg := range rowList {
		sort.SliceStable(rg.Texts, func(i, j int) bool {
			if math.Abs(rg.Texts[i].X-rg.Texts[j].X) > 1.0 {
				return rg.Texts[i].X < rg.Texts[j].X
			}
			return false
		})

		var cells []string
		var curCell strings.Builder
		var lastX float64 = -1
		var lastW float64 = 0

		for _, t := range rg.Texts {
			if lastX < 0 {
				curCell.WriteString(t.S)
				lastX = t.X
				lastW = t.W
				continue
			}

			diff := t.X - lastX
			if diff > 20.0 {
				cellStr := strings.TrimSpace(curCell.String())
				if cellStr != "" {
					cells = append(cells, formatNumberSpacing(cellStr))
				}
				curCell.Reset()
			} else if diff > lastW+2.5 && lastW > 0 && t.S != " " && !strings.HasSuffix(curCell.String(), " ") {
				curCell.WriteString(" ")
			}

			curCell.WriteString(t.S)
			lastX = t.X
			lastW = t.W
		}

		cellStr := strings.TrimSpace(curCell.String())
		if cellStr != "" {
			cells = append(cells, formatNumberSpacing(cellStr))
		}

		if len(cells) == 0 {
			continue
		}

		fullLine := strings.Join(cells, " ")
		if pageFooterRegex.MatchString(strings.TrimSpace(fullLine)) {
			continue
		}

		lowerLine := strings.ToLower(fullLine)
		if (strings.Contains(lowerLine, "stt") && strings.Contains(lowerLine, "môn")) ||
			(strings.Contains(lowerLine, "số tín") && strings.Contains(lowerLine, "học phí")) ||
			strings.TrimSpace(lowerLine) == "chỉ miễn giảm" {
			continue
		}

		if isSectionHeading(fullLine) {
			if inTable {
				inTable = false
				sb.WriteString("\n")
			}
			sb.WriteString("\n### " + strings.TrimSpace(fullLine) + "\n\n")
			continue
		}

		if isTableRow(cells) {
			if !inTable {
				inTable = true
				sb.WriteString(buildTableHeader(cells))
			}
			sb.WriteString("| " + strings.Join(cells, " | ") + " |\n")
			continue
		}

		if inTable {
			inTable = false
			sb.WriteString("\n")
		}

		sb.WriteString(fullLine + "\n")
	}

	return strings.TrimSpace(sb.String())
}

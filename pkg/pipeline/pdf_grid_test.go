package pipeline

import (
	"context"
	"os"
	"strings"
	"testing"

	"tluagent-web/pkg/pdf"
)

func vLine(x, y0, y1 float64) pdf.Rect {
	return pdf.Rect{Min: pdf.Point{X: x - 0.25, Y: y0}, Max: pdf.Point{X: x + 0.25, Y: y1}}
}

func hLine(y, x0, x1 float64) pdf.Rect {
	return pdf.Rect{Min: pdf.Point{X: x0, Y: y - 0.25}, Max: pdf.Point{X: x1, Y: y + 0.25}}
}

func runText(font string, x, y float64, s string) []pdf.Text {
	out := make([]pdf.Text, 0, len(s))
	for _, r := range s {
		out = append(out, pdf.Text{Font: font, FontSize: 11, X: x, Y: y, S: string(r)})
	}
	return out
}

func mergedHeaderTable() pdf.Content {
	rects := []pdf.Rect{
		vLine(50, 220, 300), vLine(100, 220, 300), vLine(200, 220, 300), vLine(300, 220, 280), vLine(400, 220, 300),
		hLine(300, 50, 400), hLine(280, 200, 400), hLine(260, 50, 400), hLine(240, 50, 400), hLine(220, 50, 400),
	}
	var texts []pdf.Text
	texts = append(texts, runText("Arial", 50, 320, "Intro line above the table")...)
	texts = append(texts, runText("Arial-BoldMT", 60, 288, "No")...)
	texts = append(texts, runText("Arial-BoldMT", 110, 288, "Item")...)
	texts = append(texts, runText("Arial-BoldMT", 250, 288, "Result")...)
	texts = append(texts, runText("Arial-BoldMT", 210, 268, "Basic")...)
	texts = append(texts, runText("Arial-BoldMT", 310, 268, "Advanced")...)
	texts = append(texts, runText("Arial", 60, 248, "1")...)
	texts = append(texts, runText("Arial", 110, 248, "TOEIC")...)
	texts = append(texts, runText("Arial", 210, 248, "450")...)
	texts = append(texts, runText("Arial", 310, 248, "550")...)
	texts = append(texts, runText("Arial", 60, 228, "2")...)
	texts = append(texts, runText("Arial", 110, 228, "IELTS")...)
	texts = append(texts, runText("Arial", 210, 228, "4.5")...)
	texts = append(texts, runText("Arial", 310, 228, "5.5")...)
	texts = append(texts, runText("Arial", 50, 200, "Footer line below the table")...)
	return pdf.Content{Text: texts, Rect: rects}
}

func TestExtractGridTables_MergedHeaderKeepsColumnMeaning(t *testing.T) {
	remaining, tables := extractGridTables(mergedHeaderTable())
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	want := strings.Join([]string{
		"| No | Item | Result – Basic | Result – Advanced |",
		"| --- | --- | --- | --- |",
		"| 1 | TOEIC | 450 | 550 |",
		"| 2 | IELTS | 4.5 | 5.5 |",
		"",
	}, "\n")
	if tables[0].markdown != want {
		t.Fatalf("unexpected markdown:\n%s\nwant:\n%s", tables[0].markdown, want)
	}
	var left strings.Builder
	for _, tx := range remaining {
		left.WriteString(tx.S)
	}
	if left.String() != "Intro line above the tableFooter line below the table" {
		t.Fatalf("text outside the table must stay in normal flow, got %q", left.String())
	}
}

func TestExtractGridTables_SingleBoxIsNotATable(t *testing.T) {
	content := pdf.Content{
		Rect: []pdf.Rect{vLine(50, 100, 200), vLine(300, 100, 200), hLine(100, 50, 300), hLine(200, 50, 300)},
		Text: runText("Arial", 60, 150, "Boxed note"),
	}
	remaining, tables := extractGridTables(content)
	if len(tables) != 0 || len(remaining) != len(content.Text) {
		t.Fatalf("a single bordered box must not become a table, tables=%d", len(tables))
	}
}

func TestExtractGridTables_NoBordersLeavesTextUntouched(t *testing.T) {
	content := pdf.Content{Text: runText("Arial", 60, 150, "Plain paragraph")}
	remaining, tables := extractGridTables(content)
	if len(tables) != 0 || len(remaining) != len(content.Text) {
		t.Fatalf("plain text must pass through unchanged")
	}
}

func TestExtractPDFWithStructure_LiveRuledTable(t *testing.T) {
	path := os.Getenv("TLU_TEST_RULED_TABLE_PDF")
	if path == "" {
		t.Skip("Skipping live test: missing TLU_TEST_RULED_TABLE_PDF")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pdf: %v", err)
	}
	out, err := ExtractPDFWithStructure(context.Background(), data)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	row := "| 4 | TOEIC (Quốc tế 4 kỹ năng) | ETS / Đơn vị được ủy quyền | Nghe-Đọc: ≥ 450 Nói-Viết: ≥ 200 | Nghe-Đọc: ≥ 550 Nói-Viết: ≥ 240 | - |"
	if !strings.Contains(out, row) {
		t.Fatalf("TOEIC row must keep each score in its program column, got:\n%s", out)
	}
}

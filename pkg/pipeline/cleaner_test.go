package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCleanerProcessAllMissingIndex(t *testing.T) {
	tempDir := t.TempDir()
	cleaner := NewCleaner(CleanerConfig{
		DataRawDir:   filepath.Join(tempDir, "raw"),
		DataCleanDir: filepath.Join(tempDir, "clean"),
	})

	var calledProgress bool
	err := cleaner.ProcessAll(context.Background(), func(cur, total int64) {
		calledProgress = true
	})
	if err != nil {
		t.Fatalf("expected nil error on missing index, got %v", err)
	}
	if !calledProgress {
		t.Fatal("expected progress callback to be invoked with 0, 0")
	}
}

func TestCleanerProcessAllValidIndex(t *testing.T) {
	tempDir := t.TempDir()
	rawDir := filepath.Join(tempDir, "raw")
	cleanDir := filepath.Join(tempDir, "clean")
	if err := os.MkdirAll(rawDir, 0755); err != nil {
		t.Fatal(err)
	}

	indexContent := `{"node_id":"test1","type":"TIN_TUC","url":"https://example.com","title":"Test Title"}` + "\n"
	if err := os.WriteFile(filepath.Join(rawDir, "raw_index.jsonl"), []byte(indexContent), 0644); err != nil {
		t.Fatal(err)
	}

	cleaner := NewCleaner(CleanerConfig{
		DataRawDir:   rawDir,
		DataCleanDir: cleanDir,
	})

	var lastCur, lastTotal int64
	err := cleaner.ProcessAll(context.Background(), func(cur, total int64) {
		lastCur = cur
		lastTotal = total
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lastCur != 1 || lastTotal != 1 {
		t.Fatalf("expected progress 1/1, got %d/%d", lastCur, lastTotal)
	}
}

func TestStitchBrokenVietnameseLines(t *testing.T) {
	input := `Trư
ờ
ng Đ
ạ
i h
ọ
c Thăng Long

Đư
ờ
ng Nghiêm Xuân Yêm, Phư
ờ
ng Đ
ị
nh Công
,

Email :
p.tckt
@thanglong.edu.vn

THÔNG BÁO

V/v THU H
Ọ
C PHÍ H
Ọ
C K
Ỳ

2

16.000.000 đồng
/ học kỳ`

	result := StitchBrokenVietnameseLines(input)
	if !strings.Contains(result, "Trường Đại học Thăng Long") {
		t.Fatalf("expected 'Trường Đại học Thăng Long' in result, got: %s", result)
	}
	if !strings.Contains(result, "Đường Nghiêm Xuân Yêm, Phường Định Công,") {
		t.Fatalf("expected 'Đường Nghiêm Xuân Yêm, Phường Định Công,' in result, got: %s", result)
	}
	if !strings.Contains(result, "p.tckt@thanglong.edu.vn") {
		t.Fatalf("expected 'p.tckt@thanglong.edu.vn' in result, got: %s", result)
	}
	if !strings.Contains(result, "16.000.000 đồng / học kỳ") && !strings.Contains(result, "16.000.000 đồng/ học kỳ") {
		t.Fatalf("expected '16.000.000 đồng / học kỳ' in result, got: %s", result)
	}
}

func TestCleanExtractedTextPreservesTables(t *testing.T) {
	input := "Lời mở đầu văn bản\n\n| Mã HP | Tên HP | Tín chỉ |\n|---|---|---|\n| FTO1201 | Cơ sở văn hóa | 3 |\n| BSA1201 | Quản trị học | 3 |\n\nKết thúc văn bản"
	cleaned := cleanExtractedText(input)
	if !strings.Contains(cleaned, "| FTO1201 | Cơ sở văn hóa | 3 |") {
		t.Fatalf("expected table row to be preserved, got: %s", cleaned)
	}
}

func TestExtractPDFFileWithStructureReal(t *testing.T) {
	pdfPath := "/tmp/test_22013.pdf"
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("skipping real pdf test: /tmp/test_22013.pdf not present")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	res, err := ExtractPDFFileWithStructure(ctx, pdfPath)
	if err != nil {
		t.Fatalf("failed to extract structured pdf: %v", err)
	}

	if !strings.Contains(res, "| FTO1201 |") {
		t.Fatalf("expected extracted text to contain markdown table row for FTO1201")
	}
}



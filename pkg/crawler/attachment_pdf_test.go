package crawler

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

func buildScanPDF(textLayer string) []byte {
	content := "BT /F1 10 Tf 40 700 Td (" + textLayer + ") Tj ET q 100 0 0 100 40 500 cm /Im1 Do Q"
	rgb := string(bytes.Repeat([]byte{0xff, 0x80, 0x10}, 4))
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> /XObject << /Im1 6 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width 2 /Height 2 /ColorSpace /DeviceRGB /BitsPerComponent 8 /Length %d >>\nstream\n%s\nendstream", len(rgb), rgb),
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects))
	for i, obj := range objects {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return buf.Bytes()
}

func newAttachmentTestCrawler(t *testing.T, ocr VisionOCRHandler) *Crawler {
	t.Helper()
	c, err := NewCrawler(Config{BaseURL: "https://example.com", VisionOCR: ocr})
	if err != nil {
		t.Fatalf("new crawler: %v", err)
	}
	return c
}

func TestExtractAttachmentText_GarbledTextLayerFallsBackToVisionOCR(t *testing.T) {
	garbled := strings.Repeat("Di~u 1 Pharn vi va dBi nrong ap dung chu\\247n d~u ra Ti~ng Anh t6t nghiep h9c ", 6)
	clean := strings.Repeat("\u0110i\u1ec1u 1. Ph\u1ea1m vi v\u00e0 \u0111\u1ed1i t\u01b0\u1ee3ng \u00e1p d\u1ee5ng chu\u1ea9n \u0111\u1ea7u ra Ti\u1ebfng Anh t\u1ed1t nghi\u1ec7p h\u1ecdc ", 6)
	calls := 0
	c := newAttachmentTestCrawler(t, func(ctx context.Context, images [][]byte) (string, error) {
		calls++
		return clean, nil
	})

	got, err := c.extractAttachmentText(context.Background(), buildScanPDF(garbled), "scan.pdf")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if calls != 1 || got != strings.TrimSpace(clean) {
		t.Fatalf("expected vision OCR result, calls=%d got=%q", calls, got[:min(len(got), 60)])
	}
}

func TestExtractAttachmentText_CleanTextLayerSkipsVisionOCR(t *testing.T) {
	clean := strings.Repeat("Dieu 1 Pham vi va doi tuong ap dung chuan dau ra Tieng Anh tot nghiep hoc ", 6)
	calls := 0
	c := newAttachmentTestCrawler(t, func(ctx context.Context, images [][]byte) (string, error) {
		calls++
		return "unused", nil
	})

	got, err := c.extractAttachmentText(context.Background(), buildScanPDF(clean), "born-digital.pdf")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if calls != 0 || !strings.Contains(got, "chuan dau ra") {
		t.Fatalf("clean text layer must be used directly, calls=%d got=%q", calls, got)
	}
}

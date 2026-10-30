package services

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSplitIntoChunks(t *testing.T) {
	longLine := strings.Repeat("quy che dao tao tin chi hoc phi mot tin chi ", 120)

	multiline := strings.Repeat("Điều 1. Phạm vi điều chỉnh\nSinh viên K38 ngành CNTT phải hoàn thành 130 tín chỉ.\n\n", 60)

	tests := []struct {
		name       string
		text       string
		maxLen     int
		overlap    int
		wantChunks int
	}{
		{name: "empty string", text: "", maxLen: 1200, overlap: 150, wantChunks: 0},
		{name: "whitespace only", text: "   \n\t ", maxLen: 1200, overlap: 150, wantChunks: 0},
		{name: "short text stays whole", text: "Học phí một tín chỉ là 500.000 đồng", maxLen: 1200, overlap: 150, wantChunks: 1},
		{name: "single overlong line", text: longLine, maxLen: 1200, overlap: 150, wantChunks: -1},
		{name: "vietnamese multiline", text: multiline, maxLen: 1200, overlap: 150, wantChunks: -1},
		{name: "overlap larger than max", text: longLine, maxLen: 400, overlap: 5000, wantChunks: -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			chunks := splitIntoChunks(tc.text, tc.maxLen, tc.overlap)
			if tc.wantChunks == 0 {
				if len(chunks) != 0 {
					t.Fatalf("expected no chunks, got %d", len(chunks))
				}
				return
			}
			if tc.wantChunks > 0 && len(chunks) != tc.wantChunks {
				t.Fatalf("expected %d chunk(s), got %d", tc.wantChunks, len(chunks))
			}

			effectiveOverlap := tc.overlap
			if limit := tc.maxLen / 4; effectiveOverlap > limit {
				effectiveOverlap = limit
			}

			for i, chunk := range chunks {
				if utf8.RuneCountInString(chunk) > tc.maxLen {
					t.Fatalf("chunk %d is %d runes, exceeds max %d", i, utf8.RuneCountInString(chunk), tc.maxLen)
				}
				if !utf8.ValidString(chunk) {
					t.Fatalf("chunk %d is not valid UTF-8", i)
				}
				if strings.TrimSpace(chunk) == "" {
					t.Fatalf("chunk %d is empty", i)
				}
				if i == 0 {
					continue
				}
				if effectiveOverlap > 0 {
					prevRunes := []rune(chunks[i-1])
					nextRunes := []rune(chunk)
					probe := effectiveOverlap / 4
					if probe > len(nextRunes) {
						probe = len(nextRunes)
					}
					if probe > 0 && !strings.Contains(string(prevRunes), string(nextRunes[:probe])) {
						t.Fatalf("chunk %d does not share the expected overlap with chunk %d", i, i-1)
					}
				}
			}
		})
	}
}

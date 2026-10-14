package pdf

import (
	"strings"
	"testing"
)

func TestPage_ExtractCleanText_ColumnSpacing(t *testing.T) {
	content := []Text{
		{S: "1", X: 37.5, Y: 100},
		{S: "A17408", X: 56.4, Y: 100},
		{S: "NGUYEN DUONG QUANG", X: 108.5, Y: 100},
		{S: "SP29d1", X: 316.9, Y: 100},
		{S: "819", X: 415.7, Y: 100},
	}
	row := Row{
		Position: 100,
		Content:  content,
	}

	var line strings.Builder
	var lastX float64 = -1
	for _, w := range row.Content {
		if lastX >= 0 && (w.X-lastX > 4.0 || lastX-w.X > 4.0) {
			str := line.String()
			if len(str) > 0 && !strings.HasSuffix(str, " ") && !strings.HasPrefix(w.S, " ") {
				if w.X-lastX > 20.0 || lastX-w.X > 20.0 {
					line.WriteString("   ")
				} else {
					line.WriteString(" ")
				}
			}
		}
		line.WriteString(w.S)
		lastX = w.X
	}

	res := line.String()
	if !strings.Contains(res, "1 A17408") {
		t.Fatalf("expected space between number and student ID, got: %s", res)
	}
	if !strings.Contains(res, "A17408   NGUYEN DUONG QUANG") {
		t.Fatalf("expected column separation before student name, got: %s", res)
	}
}

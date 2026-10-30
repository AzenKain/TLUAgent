package services

import (
	"strings"
	"testing"
)

func TestShouldReplaceContent_Decisions(t *testing.T) {
	garbled := strings.Repeat("Di~u 1 Pharn vi va dBi nrong chu§n d~u ra Ti~ng Anh t6t nghiep h9c ", 8)
	clean := strings.Repeat("Dieu 1 pham vi va doi tuong ap dung chuan dau ra tieng Anh tot nghiep hoc ", 8)
	flat := "TT Chung chi PET (B1) FCE (B2)\n" + clean
	table := "| TT | Chung chi | Basic |\n| --- | --- | --- |\n| 1 | IELTS | 4.5 |\n" + clean

	cases := []struct {
		name    string
		oldText string
		newText string
		force   bool
		want    bool
	}{
		{"corrupted layer replaced by clean text", garbled, clean, false, true},
		{"table recovered", flat, table, false, true},
		{"clean text kept when nothing improves", clean, clean, false, false},
		{"empty extraction never replaces", garbled, "  ", true, false},
		{"far shorter result is rejected", clean, "short", false, false},
		{"force overrides heuristics", clean, clean[:len(clean)*3/5], true, true},
		{"force still refuses a drastic shrink", clean, "short", true, false},
	}
	for _, tc := range cases {
		got, reason := shouldReplaceContent(tc.oldText, tc.newText, tc.force)
		if got != tc.want {
			t.Errorf("%s: got %v (%s), want %v", tc.name, got, reason, tc.want)
		}
	}
}

func TestCountTableRows_DeduplicatesOverlappingChunks(t *testing.T) {
	text := "| a | b |\n| --- | --- |\n| 1 | 2 |\nnote\n| 1 | 2 |\n"
	if got := countTableRows(text); got != 3 {
		t.Fatalf("expected 3 distinct table lines, got %d", got)
	}
}

func TestParseReextractPayload(t *testing.T) {
	p := parseReextractPayload(`{"doc_ids":["A","B"],"dry_run":true,"force":true,"all":false,"garble_threshold":0.05}`)
	if len(p.DocIDs) != 2 || !p.DryRun || !p.Force || p.Threshold != 0.05 {
		t.Fatalf("unexpected payload %+v", p)
	}
	if empty := parseReextractPayload("not json"); len(empty.DocIDs) != 0 || empty.DryRun {
		t.Fatalf("invalid payload must fall back to defaults, got %+v", empty)
	}
}

func TestMergeBodyAndAttachment(t *testing.T) {
	intro := "Short page intro that points readers to the attached decision."
	decision := strings.TrimSpace("Decision text. " + strings.Repeat("Article provisions of the attached decision. ", 12))

	if got := mergeBodyAndAttachment(intro, decision); !strings.Contains(got, intro) || !strings.Contains(got, "Article provisions") {
		t.Fatalf("a short intro and a real attachment must both be kept")
	}
	if got := mergeBodyAndAttachment(decision, decision); got != decision {
		t.Fatalf("an attachment identical to the body must not be duplicated")
	}
	if got := mergeBodyAndAttachment("", decision); got != decision {
		t.Fatalf("an empty body falls back to the attachment")
	}
	if got := mergeBodyAndAttachment(intro, "   "); got != intro {
		t.Fatalf("an empty attachment leaves the body untouched")
	}
	if got := mergeBodyAndAttachment(decision+" Extra closing remarks from the page.", decision); strings.Count(got, "Article provisions") != 12 {
		t.Fatalf("an attachment already embedded in the body must not be appended again")
	}
}

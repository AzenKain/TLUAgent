package services

import (
	"strings"
	"testing"
)

func TestStructuralArticleDiff_ReportsChangedAddedAndRemovedArticles(t *testing.T) {
	oldText := "\u0110i\u1ec1u 1. Scope\nApplies to all students.\n\u0110i\u1ec1u 2. Fee\nThe fee is 450 points.\n\u0110i\u1ec1u 3. Old only\nLegacy clause text here."
	newText := "\u0110i\u1ec1u 1. Scope\nApplies to all students.\n\u0110i\u1ec1u 2. Fee\nThe fee is 550 points.\n\u0110i\u1ec1u 4. New only\nBrand new clause text."

	out := structuralArticleDiff(oldText, newText, "")
	for _, want := range []string{"\u0110i\u1ec1u 1: IDENTICAL", "\u0110i\u1ec1u 2: CHANGED", "removed: [450]", "added: [550]", "\u0110i\u1ec1u 4: ONLY IN INCOMING DOCUMENT", "\u0110i\u1ec1u 3: ONLY IN EARLIER DOCUMENT"} {
		if !strings.Contains(out, want) {
			t.Fatalf("diff must contain %q, got:\n%s", want, out)
		}
	}
	only := structuralArticleDiff(oldText, newText, "\u0111i\u1ec1u 2")
	if strings.Contains(only, "\u0110i\u1ec1u 4") || !strings.Contains(only, "\u0110i\u1ec1u 2") {
		t.Fatalf("the article filter must narrow the report, got:\n%s", only)
	}
}

func TestStructuralArticleDiff_FallsBackToSentencesWithoutArticles(t *testing.T) {
	out := structuralArticleDiff("Students must pay the fee before the deadline.", "Students must pay the fee after the deadline.", "")
	if !strings.Contains(out, "REMOVED:") || !strings.Contains(out, "ADDED:") {
		t.Fatalf("a document without articles needs a sentence-level diff, got %q", out)
	}
}

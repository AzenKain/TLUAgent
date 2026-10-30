package services

import (
	"strings"
	"testing"

	"tluagent-web/internal/models"
)

const replacementSample = "Điều 3: Quyết định này thay thế Quyết định số 17081612/QĐ-ĐHTL ngày 16/8/2017 của Hiệu trưởng. Điều 4: Quyết định có hiệu lực kể từ ngày ký."

func TestExtractReplacementClauses_ReadsCitedDecisionAndDate(t *testing.T) {
	clauses := extractReplacementClauses(replacementSample)
	if len(clauses) != 1 {
		t.Fatalf("expected one clause, got %+v", clauses)
	}
	if clauses[0].Number != "17081612/QĐ-ĐHTL" || clauses[0].Date != "2017-08-16" {
		t.Fatalf("unexpected clause %+v", clauses[0])
	}
	if got := extractReplacementClauses("An ordinary sentence with no replacement language."); len(got) != 0 {
		t.Fatalf("plain text must yield no clause, got %+v", got)
	}
}

func TestMatchClausesToCandidates_ByNumberThenByDate(t *testing.T) {
	clauses := extractReplacementClauses(replacementSample)
	byDate := &models.Document{ID: "DOC_2017", PublishDate: "2017-08-16"}
	other := &models.Document{ID: "DOC_OTHER", PublishDate: "2020-01-01"}
	texts := map[string]string{"DOC_2017": "Số: 178/16/QĐ-ĐHTL", "DOC_OTHER": "Số: 9/QĐ"}
	textOf := func(id string) string { return texts[id] }

	matches := matchClausesToCandidates(clauses, []*models.Document{byDate, other}, textOf)
	if len(matches) != 1 || matches[0].Candidate.ID != "DOC_2017" || matches[0].By != "decision date" {
		t.Fatalf("a scan whose number was misread must still match by date, got %+v", matches)
	}

	texts["DOC_OTHER"] = "Số: 17081612/QĐ-ĐHTL"
	matches = matchClausesToCandidates(clauses, []*models.Document{byDate, other}, textOf)
	if len(matches) != 1 || matches[0].Candidate.ID != "DOC_OTHER" || matches[0].By != "decision number" {
		t.Fatalf("an exact decision number wins over the date fallback, got %+v", matches)
	}
}

func TestFormatEvidence(t *testing.T) {
	clauses := extractReplacementClauses(replacementSample)
	out := formatEvidence(clauses, []clauseMatch{{Clause: clauses[0], Candidate: &models.Document{ID: "DOC_2017"}, By: "decision date"}})
	if !strings.Contains(out, "DOC_2017") || !strings.Contains(out, "2017-08-16") {
		t.Fatalf("evidence must name the matched candidate and date, got %q", out)
	}
	if formatEvidence(nil, nil) != "" {
		t.Fatalf("no clauses means no evidence block")
	}
}

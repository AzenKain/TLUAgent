package services

import (
	"fmt"
	"regexp"
	"strings"

	"tluagent-web/internal/models"
)

const (
	evidenceSentenceRunes = 220
	decisionHeadRunes     = 1500
	maxEvidenceClauses    = 6
)

var (
	replacementClauseRegex    = regexp.MustCompile(`(?is)(?:thay th\x{1ebf}|b\x{00e3}i b\x{1ecf}|h\x{1ebf}t hi\x{1ec7}u l\x{1ef1}c)[^\n.;]{0,160}?s\x{1ed1}\s+([0-9][0-9A-Za-z./\-\x{0110}\x{00c0}-\x{1ef9}]*)(?:[^\n;]{0,60}?ng\x{00e0}y\s+(\d{1,2})[/\-](\d{1,2})[/\-](\d{4}))?`)
	headerDecisionNumberRegex = regexp.MustCompile(`(?i)s\x{1ed1}:?\s*([0-9][0-9A-Za-z./\-\x{0110}\x{00c0}-\x{1ef9}]*)`)
)

type replacementClause struct {
	Sentence string
	Number   string
	Date     string
}

func normalizeDecisionNumber(raw string) string {
	return strings.ToUpper(strings.Join(strings.Fields(raw), ""))
}

func isoDate(day, month, year string) string {
	if day == "" || month == "" || year == "" {
		return ""
	}
	return fmt.Sprintf("%s-%s-%s", year, month, day)
}

func padDatePart(v string) string {
	if len(v) == 1 {
		return "0" + v
	}
	return v
}

// extractReplacementClauses finds sentences where a document states that it replaces or repeals another numbered decision.
func extractReplacementClauses(text string) []replacementClause {
	var out []replacementClause
	for _, m := range replacementClauseRegex.FindAllStringSubmatchIndex(text, -1) {
		sentence := strings.Join(strings.Fields(text[m[0]:m[1]]), " ")
		if runes := []rune(sentence); len(runes) > evidenceSentenceRunes {
			sentence = string(runes[:evidenceSentenceRunes]) + "..."
		}
		clause := replacementClause{Sentence: sentence, Number: normalizeDecisionNumber(text[m[2]:m[3]])}
		if m[4] >= 0 {
			clause.Date = isoDate(padDatePart(text[m[4]:m[5]]), padDatePart(text[m[6]:m[7]]), text[m[8]:m[9]])
		}
		out = append(out, clause)
		if len(out) >= maxEvidenceClauses {
			break
		}
	}
	return out
}

func decisionNumbersOf(text string) map[string]bool {
	head := text
	if runes := []rune(head); len(runes) > decisionHeadRunes {
		head = string(runes[:decisionHeadRunes])
	}
	numbers := map[string]bool{}
	for _, m := range headerDecisionNumberRegex.FindAllStringSubmatch(head, -1) {
		numbers[normalizeDecisionNumber(m[1])] = true
	}
	return numbers
}

type clauseMatch struct {
	Clause    replacementClause
	Candidate *models.Document
	By        string
}

// matchClausesToCandidates links each replacement clause to the earlier document it cites, by decision number or by decision date.
func matchClausesToCandidates(clauses []replacementClause, candidates []*models.Document, textOf func(id string) string) []clauseMatch {
	var matches []clauseMatch
	for _, clause := range clauses {
		matched := false
		for _, cand := range candidates {
			if decisionNumbersOf(textOf(cand.ID))[clause.Number] || normalizeDecisionNumber(cand.DocCode) == clause.Number {
				matches = append(matches, clauseMatch{Clause: clause, Candidate: cand, By: "decision number"})
				matched = true
			}
		}
		if matched || clause.Date == "" {
			continue
		}
		for _, cand := range candidates {
			if cand.PublishDate == clause.Date {
				matches = append(matches, clauseMatch{Clause: clause, Candidate: cand, By: "decision date"})
			}
		}
	}
	return matches
}

func formatEvidence(clauses []replacementClause, matches []clauseMatch) string {
	if len(clauses) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n\nEXPLICIT REPLACEMENT CLAUSES FOUND IN THE INCOMING DOCUMENT (verified text, treat as strong evidence):\n")
	for _, c := range clauses {
		fmt.Fprintf(&sb, "- \"%s\" (cited decision %s", c.Sentence, c.Number)
		if c.Date != "" {
			sb.WriteString(", dated ")
			sb.WriteString(c.Date)
		}
		sb.WriteString(")\n")
	}
	for _, m := range matches {
		fmt.Fprintf(&sb, "  => matches candidate %s by %s\n", m.Candidate.ID, m.By)
	}
	sb.WriteString("A clause only proves what it names: still check cohort scope before FULL_SUPERSEDE.\n")
	return sb.String()
}

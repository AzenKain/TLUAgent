package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tluagent-web/internal/models"
	"tluagent-web/pkg/llm"
)

const (
	verifyExcerptRunes = 3500
	verifyEffort       = "high"
	maxPromptCands     = 5
)

const verifySystemPrompt = `You are a meticulous legal auditor of university regulations. A first-pass agent proposed that the INCOMING document fully supersedes the EARLIER document. Decide independently using only the evidence given.
Rules: FULL_SUPERSEDE requires that the incoming document contains substantive provisions covering the same subject AND applies to every cohort the earlier document applies to. If it covers only some cohorts or only some articles, answer PARTIAL_AMEND. If the evidence does not show replacement, answer NONE.
Answer ONLY with JSON: {"verdict":"FULL_SUPERSEDE|PARTIAL_AMEND|NONE","reason":"one or two sentences quoting the decisive text"}`

func (e *RegulatoryDissectionEngine) documentText(ctx context.Context, docID string) string {
	if e.ragRepo == nil {
		return ""
	}
	chunks, err := e.ragRepo.GetChunksByDocumentID(ctx, docID)
	if err != nil {
		return ""
	}
	parts := make([]string, 0, len(chunks))
	for _, ch := range chunks {
		parts = append(parts, ch.Content)
	}
	return strings.Join(parts, "\n")
}

func headRunes(text string, n int) string {
	if runes := []rune(text); len(runes) > n {
		return string(runes[:n])
	}
	return text
}

func parseVerdict(content string) (SupersessionAction, string, bool) {
	start, end := strings.Index(content, "{"), strings.LastIndex(content, "}")
	if start < 0 || end <= start {
		return "", "", false
	}
	var out struct {
		Verdict string `json:"verdict"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &out); err != nil {
		return "", "", false
	}
	switch action := SupersessionAction(strings.ToUpper(strings.TrimSpace(out.Verdict))); action {
	case ActionFullSupersede, ActionPartialAmend, ActionNone:
		return action, strings.TrimSpace(out.Reason), true
	}
	return "", "", false
}

// verifyFullSupersession asks the audit model for an independent high-effort second opinion on a proposed full supersession.
func (e *RegulatoryDissectionEngine) verifyFullSupersession(ctx context.Context, req *DissectionRequest, target *models.Document) (SupersessionAction, string, bool) {
	if e.mgr == nil || target == nil {
		return "", "", false
	}
	targetText := e.documentText(ctx, target.ID)
	prompt := fmt.Sprintf(`INCOMING: %s | %s | published %s | cohort %s
%s

EARLIER: %s | %s | published %s | cohort %s
%s

ARTICLE-LEVEL DIFF (earlier vs incoming):
%s`,
		req.NewDocID, req.Title, req.PublishDate, req.ApplicableCohort, headRunes(req.Content, verifyExcerptRunes),
		target.ID, target.Title, target.PublishDate, target.ApplicableCohort, headRunes(targetText, verifyExcerptRunes),
		structuralArticleDiff(targetText, req.Content, ""))

	enabled := true
	resp, err := e.mgr.ChatWithChain(ctx, llm.ChainTypeAudit, &llm.ChatRequest{
		Messages:        []llm.Message{llm.SystemMessage(verifySystemPrompt), llm.UserMessage(prompt)},
		ThinkingEnabled: &enabled,
		ReasoningEffort: verifyEffort,
	})
	if err != nil || resp == nil {
		return "", "", false
	}
	return parseVerdict(resp.Message.Content)
}

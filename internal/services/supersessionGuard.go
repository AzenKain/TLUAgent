package services

import (
	"context"
	"strings"

	"tluagent-web/pkg/constants"
)

// isTitleOnlyContent reports whether stored text carries nothing beyond the document title.
func isTitleOnlyContent(content, title string) bool {
	text := documentTitleKey(content)
	return text == "" || text == documentTitleKey(title)
}

func parseCohortSet(cohorts string) map[string]bool {
	cohorts = strings.TrimSpace(cohorts)
	if cohorts == "" || strings.EqualFold(cohorts, constants.DocCohortAll) {
		return nil
	}
	set := map[string]bool{}
	for _, c := range strings.Split(cohorts, ",") {
		if c = strings.ToUpper(strings.TrimSpace(c)); c != "" {
			set[c] = true
		}
	}
	return set
}

// cohortCovers reports whether a document scoped to newCohort can stand in for every cohort of oldCohort.
func cohortCovers(newCohort, oldCohort string) bool {
	newSet := parseCohortSet(newCohort)
	if newSet == nil {
		return true
	}
	oldSet := parseCohortSet(oldCohort)
	if oldSet == nil {
		return false
	}
	for c := range oldSet {
		if !newSet[c] {
			return false
		}
	}
	return true
}

// reviewSupersessionAction downgrades proposals that the available evidence cannot justify.
func (e *RegulatoryDissectionEngine) reviewSupersessionAction(ctx context.Context, req *DissectionRequest, action SupersessionAction, targetDocID string) (SupersessionAction, string) {
	if action != ActionFullSupersede && action != ActionPartialAmend {
		return action, ""
	}
	if isTitleOnlyContent(req.Content, req.Title) {
		return ActionNone, "Ignored: the incoming document has no content beyond its title, so it cannot replace or amend another document."
	}
	if action != ActionFullSupersede || e.ragRepo == nil || targetDocID == "" {
		return action, ""
	}
	target, err := e.ragRepo.GetDocumentByID(ctx, targetDocID)
	if err != nil || target == nil {
		return action, ""
	}
	targetChunks, _ := e.ragRepo.GetChunksByDocumentID(ctx, targetDocID)
	parts := make([]string, 0, len(targetChunks))
	for _, ch := range targetChunks {
		parts = append(parts, ch.Content)
	}
	if isTitleOnlyContent(strings.Join(parts, "\n"), target.Title) {
		return action, ""
	}
	if !cohortCovers(req.ApplicableCohort, target.ApplicableCohort) {
		return ActionPartialAmend, "Downgraded to partial amendment: the incoming document applies to " + req.ApplicableCohort + " but the target applies to " + target.ApplicableCohort + "."
	}
	return action, ""
}

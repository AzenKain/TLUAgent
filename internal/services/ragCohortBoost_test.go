package services

import (
	"testing"

	"tluagent-web/internal/models"
)

func TestScoreChunk_PrefersDocumentNamingStudentCohort(t *testing.T) {
	general := &models.DocumentChunk{ID: "general", DocumentID: "d1", Content: "graduation rank penalty", ApplicableCohort: "ALL", PriorityLevel: 50}
	specific := &models.DocumentChunk{ID: "specific", DocumentID: "d2", Content: "graduation rank penalty", ApplicableCohort: "K38,K39", PriorityLevel: 50}
	tokens := []string{"graduation", "rank", "penalty"}

	base := scoreChunk(specific, nil, nil, tokens, 0, "")
	boosted := scoreChunk(specific, nil, nil, tokens, 0, "k39")
	plain := scoreChunk(general, nil, nil, tokens, 0, "K39")
	if base == nil || boosted == nil || plain == nil {
		t.Fatal("all chunks must score above the minimum threshold")
	}
	if boosted.FinalScore <= plain.FinalScore || boosted.FinalScore <= base.FinalScore {
		t.Fatalf("cohort-specific chunk must outrank the generic one for its cohort: boosted=%v plain=%v base=%v", boosted.FinalScore, plain.FinalScore, base.FinalScore)
	}
	if other := scoreChunk(specific, nil, nil, tokens, 0, "K35"); other.FinalScore != base.FinalScore {
		t.Fatal("a different cohort must not receive the boost")
	}
}

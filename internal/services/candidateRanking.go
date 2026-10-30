package services

import (
	"context"
	"math"
	"sort"

	"tluagent-web/internal/models"
)

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func (e *RegulatoryDissectionEngine) leadEmbedding(ctx context.Context, docID string) []float32 {
	if e.ragRepo == nil {
		return nil
	}
	chunks, err := e.ragRepo.GetChunksByDocumentID(ctx, docID)
	if err != nil || len(chunks) == 0 {
		return nil
	}
	return chunks[0].Embedding
}

// rankCandidates keeps the candidates closest to the incoming document using the stored chunk embeddings, always retaining pinned ones.
func (e *RegulatoryDissectionEngine) rankCandidates(ctx context.Context, newDocID string, candidates []*models.Document, pinned map[string]bool) []*models.Document {
	if len(candidates) <= maxPromptCands {
		return candidates
	}
	incoming := e.leadEmbedding(ctx, newDocID)
	if len(incoming) == 0 {
		return candidates
	}
	type scored struct {
		doc   *models.Document
		score float64
	}
	ranked := make([]scored, 0, len(candidates))
	for _, c := range candidates {
		score := cosineSimilarity(incoming, e.leadEmbedding(ctx, c.ID))
		if pinned[c.ID] {
			score = math.MaxFloat64
		}
		ranked = append(ranked, scored{doc: c, score: score})
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	out := make([]*models.Document, 0, maxPromptCands)
	for i := 0; i < len(ranked) && i < maxPromptCands; i++ {
		out = append(out, ranked[i].doc)
	}
	return out
}

package services

import (
	"context"
	"fmt"

	"tluagent-web/internal/models"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/llm"
)

const (
	contentChunkSize    = 1200
	contentChunkOverlap = 150
	embeddingBatchSize  = 32
)

// buildContentChunks splits text into overlapping chunks and attaches embeddings when an embedder is available.
func buildContentChunks(ctx context.Context, mgr *llm.Manager, doc *models.Document, content, status string) []*models.DocumentChunk {
	chunkTexts := splitIntoChunks(content, contentChunkSize, contentChunkOverlap)
	var embeddings [][]float32
	if mgr != nil && len(chunkTexts) > 0 {
		for i := 0; i < len(chunkTexts); i += embeddingBatchSize {
			end := min(i+embeddingBatchSize, len(chunkTexts))
			batch, err := mgr.ExecuteEmbedding(ctx, chunkTexts[i:end])
			if err != nil {
				break
			}
			embeddings = append(embeddings, batch...)
		}
	}

	chunks := make([]*models.DocumentChunk, 0, len(chunkTexts))
	for idx, text := range chunkTexts {
		var emb []float32
		if idx < len(embeddings) {
			emb = embeddings[idx]
		}
		chunks = append(chunks, &models.DocumentChunk{
			ID:               fmt.Sprintf(constants.PrefixChunk+"%s_%d", doc.ID, idx),
			DocumentID:       doc.ID,
			ChunkIndex:       idx,
			ChunkType:        constants.ChunkTypeContent,
			Content:          text,
			Status:           status,
			Domain:           doc.Domain,
			ApplicableCohort: doc.ApplicableCohort,
			AcademicYear:     doc.AcademicYear,
			PriorityLevel:    doc.PriorityLevel,
			Embedding:        emb,
		})
	}
	return chunks
}

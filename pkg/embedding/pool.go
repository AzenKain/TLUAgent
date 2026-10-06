package embedding

import "math"

// MeanPool averages hidden states over attended positions using the attention mask.
func MeanPool(hidden []float32, seqLen, dim int, attentionMask []int64) []float32 {
	pooled := make([]float32, dim)
	var maskSum float32
	for _, m := range attentionMask {
		maskSum += float32(m)
	}
	if maskSum == 0 {
		maskSum = 1e-9
	}
	for i := 0; i < seqLen; i++ {
		maskVal := float32(attentionMask[i])
		if maskVal == 0 {
			continue
		}
		offset := i * dim
		for j := 0; j < dim; j++ {
			pooled[j] += hidden[offset+j] * maskVal
		}
	}
	for j := range pooled {
		pooled[j] /= maskSum
	}
	return pooled
}

// CLSPool extracts the hidden state of the first token from the sequence output.
func CLSPool(hidden []float32, seqLen, dim int) []float32 {
	out := make([]float32, dim)
	if seqLen == 0 {
		return out
	}
	copy(out, hidden[:dim])
	return out
}

// L2Normalize scales a vector to unit length in place.
func L2Normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	norm := float32(math.Sqrt(sum))
	if norm == 0 {
		return
	}
	for i := range v {
		v[i] /= norm
	}
}

// CosineSimilarity computes the cosine similarity between two equal-length vectors.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		x := float64(a[i])
		y := float64(b[i])
		dot += x * y
		normA += x * x
		normB += y * y
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

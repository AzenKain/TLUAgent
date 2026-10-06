package embedding

import (
	"math"
	"testing"
)

func TestMeanPool_WithMask(t *testing.T) {
	hidden := []float32{1, 2, 3, 4, 5, 6, 7, 8}
	mask := []int64{1, 1, 0}
	pooled := MeanPool(hidden, 3, 2, mask)
	expected := []float32{2, 3}
	for i := range expected {
		if math.Abs(float64(pooled[i]-expected[i])) > 1e-6 {
			t.Fatalf("expected %v, got %v", expected, pooled)
		}
	}
}

func TestMeanPool_ZeroMaskGuard(t *testing.T) {
	hidden := []float32{1, 2, 3, 4}
	pooled := MeanPool(hidden, 2, 2, []int64{0, 0})
	if math.IsNaN(float64(pooled[0])) || math.IsInf(float64(pooled[0]), 0) {
		t.Fatalf("zero mask must not produce NaN or Inf: %v", pooled)
	}
}

func TestCLSPool(t *testing.T) {
	hidden := []float32{1, 2, 3, 4, 5, 6}
	pooled := CLSPool(hidden, 3, 2)
	if pooled[0] != 1 || pooled[1] != 2 {
		t.Fatalf("expected first token vector, got %v", pooled)
	}
}

func TestL2Normalize(t *testing.T) {
	vec := []float32{3, 4}
	L2Normalize(vec)
	if math.Abs(float64(vec[0])-0.6) > 1e-6 || math.Abs(float64(vec[1])-0.8) > 1e-6 {
		t.Fatalf("expected unit vector, got %v", vec)
	}
}

func TestL2Normalize_ZeroVector(t *testing.T) {
	vec := []float32{0, 0}
	L2Normalize(vec)
	if math.IsNaN(float64(vec[0])) || math.IsNaN(float64(vec[1])) {
		t.Fatalf("zero vector must stay zero, got %v", vec)
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float32{3, 4}
	if got := CosineSimilarity(a, a); math.Abs(got-1) > 1e-9 {
		t.Fatalf("self cosine must be 1, got %f", got)
	}
	if got := CosineSimilarity(a, []float32{4, -3}); math.Abs(got) > 1e-9 {
		t.Fatalf("orthogonal vectors must have 0 similarity, got %f", got)
	}
	if got := CosineSimilarity([]float32{1}, []float32{1, 2}); got != 0 {
		t.Fatalf("mismatched lengths must return 0, got %f", got)
	}
}

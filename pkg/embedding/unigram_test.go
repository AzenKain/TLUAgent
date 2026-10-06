package embedding

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnigramTokenizer_Accuracy(t *testing.T) {
	tokPath := filepath.Join("..", "..", "..", "scratch", "onnx", "tokenizer.json")
	if _, err := os.Stat(tokPath); err != nil {
		tokPath = filepath.Join("..", "..", "data", "onnx", "paraphrase-multilingual-minilm-l12-v2", "tokenizer.json")
		if _, err := os.Stat(tokPath); err != nil {
			t.Skipf("Skipping TestUnigramTokenizer_Accuracy: %s not found", tokPath)
		}
	}

	tokenizer, err := LoadUnigramTokenizer(tokPath)
	require.NoError(t, err)
	require.NotNil(t, tokenizer)

	testCases := []struct {
		text        string
		expectedIDs []int64
	}{
		{
			text:        "chuẩn đầu ra tiếng anh k35",
			expectedIDs: []int64{0, 19254, 2494, 673, 9457, 3616, 472, 5843, 2},
		},
		{
			text:        "Sinh viên K38 xin nghỉ học tạm thời bảo lưu kết quả ở đâu?",
			expectedIDs: []int64{0, 69729, 4603, 341, 10991, 21928, 45897, 2546, 107310, 4194, 6122, 22891, 6301, 6573, 2059, 16922, 32, 2},
		},
		{
			text:        "Điểm thi lại môn học tối đa được mấy điểm theo quy chế?",
			expectedIDs: []int64{0, 137228, 6117, 1917, 37496, 2546, 21785, 18233, 912, 34794, 6924, 3790, 8317, 12240, 32, 2},
		},
		{
			text:        "Học phí một tín chỉ năm học 2026 2027 là bao nhiêu tiền?",
			expectedIDs: []int64{0, 38635, 12495, 889, 28244, 2524, 2933, 2546, 387, 4046, 387, 3768, 580, 8609, 60649, 8708, 32, 2},
		},
	}

	for _, tc := range testCases {
		ids := tokenizer.TokenIDs(tc.text, 256)
		assert.Equal(t, tc.expectedIDs, ids, "token IDs for %q must match HuggingFace exactly", tc.text)
	}
}

func TestValidateModel_UnigramModel(t *testing.T) {
	dataOnnxDir := filepath.Join("..", "..", "data", "onnx")
	modelDir := filepath.Join(dataOnnxDir, "bge-m3")
	if _, err := os.Stat(filepath.Join(modelDir, "model.onnx")); err != nil {
		t.Skip("Skipping TestValidateModel_UnigramModel: model.onnx not found")
	}

	res := ValidateModel(t.Context(), dataOnnxDir, "bge-m3")
	require.True(t, res.OK, "model validation failed: %s", res.Error)
	assert.Equal(t, 1024, res.Dim)
	assert.Greater(t, res.LatencyMs, 0.0)

	rec, err := VerifyValidationIntegrity(modelDir)
	require.NoError(t, err)
	require.True(t, rec.OK)
}

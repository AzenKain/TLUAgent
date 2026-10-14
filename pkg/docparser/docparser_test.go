package docparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocParser_CSVAndCourseDetection(t *testing.T) {
	csvContent := `STT,Mã học phần,Tên học phần,Số tín chỉ,Điều kiện tiên quyết,Khối kiến thức,Học kỳ
1,CSE101,Tin học đại cương,3,None,Cơ sở,1
2,CSE102,Lập trình nâng cao,3.5,CSE101,Cơ sở ngành,2
3,MAT101,Giải tích 1,4,,Đại cương,1
`

	parsed, err := ParseBytes("curriculum.csv", []byte(csvContent))
	require.NoError(t, err)
	assert.Equal(t, ".csv", parsed.FileType)
	assert.Len(t, parsed.Sheets["Sheet1"], 4)
	assert.Len(t, parsed.Courses, 3)

	c1 := parsed.Courses[0]
	assert.Equal(t, "CSE101", c1.CourseCode)
	assert.Equal(t, "Tin học đại cương", c1.CourseName)
	assert.Equal(t, 3.0, c1.Credits)
	assert.Equal(t, "None", c1.PrerequisitesText)
	assert.Equal(t, "Cơ sở", c1.KnowledgeBlock)
	assert.Equal(t, int64(1), c1.SuggestedSemester)

	c2 := parsed.Courses[1]
	assert.Equal(t, "CSE102", c2.CourseCode)
	assert.Equal(t, 3.5, c2.Credits)
	assert.Equal(t, "CSE101", c2.PrerequisitesText)

	c3 := parsed.Courses[2]
	assert.Equal(t, "MAT101", c3.CourseCode)
	assert.Equal(t, 4.0, c3.Credits)
}

func TestDocParser_CreditValueParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"3", 3.0},
		{"3.5", 3.5},
		{"3,5", 3.5},
		{"3(2,1)", 3.0},
		{"4 (3+1)", 4.0},
		{"", 3.0},
		{"invalid", 3.0},
	}

	for _, tt := range tests {
		actual := parseCreditValue(tt.input)
		assert.Equal(t, tt.expected, actual, "failed for input: %s", tt.input)
	}
}

func TestDocParser_PlainTextAndMarkdown(t *testing.T) {
	content := "# Quy chế đào tạo 2026\nĐiều 1: Áp dụng từ K36."
	parsed, err := ParseBytes("quy_che.md", []byte(content))
	require.NoError(t, err)
	assert.Equal(t, ".md", parsed.FileType)
	assert.True(t, strings.Contains(parsed.TextContent, "Quy chế đào tạo 2026"))
	assert.Empty(t, parsed.Courses)
}

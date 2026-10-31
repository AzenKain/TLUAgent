package services

import "testing"

func TestIsGeneralEnglishOnlyCountsGeneralEducation(t *testing.T) {
	cases := []struct {
		code, block string
		want        bool
	}{
		{"GE101", generalEducationBlock, true},
		{"ENG1111", generalEducationBlock, true},
		{"ENG1401", "Kiến thức ngành", false},
		{"IS222", generalEducationBlock, false},
	}
	for _, c := range cases {
		if got := isGeneralEnglish(c.code, c.block); got != c.want {
			t.Errorf("isGeneralEnglish(%q, %q) = %v, want %v", c.code, c.block, got, c.want)
		}
	}
}

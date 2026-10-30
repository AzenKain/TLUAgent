package services

import (
	"strings"
	"testing"
)

func TestExtractScopeCohortFromBody(t *testing.T) {
	scope := "Đối tượng: sinh viên K36, K37 (trừ ngành X)."
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"explicit scope line", "Notice\n- " + scope + "\nFee: 900.000", "K36,K37"},
		{"no scope line", "Notice about sinh vien K36 and K37 without a scope line", ""},
		{"long document is ignored", "Notice\n" + scope + "\n" + strings.Repeat("filler text ", 400), ""},
		{"too many cohorts is not a narrow scope", "Đối tượng: K24, K25, K26, K27, K28, K29, K30, K31", ""},
	}
	for _, tc := range cases {
		if got := extractScopeCohortFromBody(tc.content); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

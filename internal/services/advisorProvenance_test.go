package services

import (
	"context"
	"testing"

	"tluagent-web/pkg/llm"
)

func TestCohortStatedByStudent(t *testing.T) {
	cases := []struct {
		text, cohort string
		want         bool
	}{
		{"Em K35 ngành CNTT", "K35", true},
		{"em khóa 35 nè", "K35", true},
		{"Em Khoá K35", "K35", true},
		{"Lớp TT35CL01 bảng điểm", "K35", false},
		{"mã sinh viên A44519", "K35", false},
		{"em K350", "K35", false},
		{"Em K36", "K35", false},
	}
	for _, c := range cases {
		ctx := withStudentText(context.Background(), []llm.Message{{Role: llm.RoleUser, Content: c.text}})
		if got := cohortStatedByStudent(ctx, c.cohort); got != c.want {
			t.Fatalf("text %q cohort %s: got %v want %v", c.text, c.cohort, got, c.want)
		}
	}
	if cohortStatedByStudent(context.Background(), "K35") {
		t.Fatal("no student text means nothing was stated")
	}
}

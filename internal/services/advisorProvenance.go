package services

import (
	"context"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"tluagent-web/pkg/llm"
)

type studentTextKey struct{}

var cohortNumberRegex = regexp.MustCompile(`\d+`)

func withStudentText(ctx context.Context, messages []llm.Message) context.Context {
	var sb strings.Builder
	for _, m := range messages {
		if m.Role != llm.RoleUser {
			continue
		}
		sb.WriteString(m.Content)
		sb.WriteString("\n")
		for _, p := range m.Parts {
			if p.Type == llm.ContentPartText {
				sb.WriteString(p.Text)
				sb.WriteString("\n")
			}
		}
	}
	return context.WithValue(ctx, studentTextKey{}, sb.String())
}

func foldVietnamese(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "đ", "d"), "Đ", "d")
	var sb strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		if !unicode.Is(unicode.Mn, r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func cohortStatedByStudent(ctx context.Context, cohort string) bool {
	digits := cohortNumberRegex.FindString(cohort)
	if digits == "" {
		return false
	}
	text, _ := ctx.Value(studentTextKey{}).(string)
	folded := foldVietnamese(text)
	pattern := regexp.MustCompile(`(^|[^a-z0-9])(k|khoa\s*k?|nj|na|nk)\s*` + regexp.QuoteMeta(digits) + `([^0-9]|$)`)
	return pattern.MatchString(folded)
}

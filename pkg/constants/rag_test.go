package constants

import "testing"

func TestDefaultValidityTier(t *testing.T) {
	official := []string{DocTypeRegulation, DocTypeRule, DocTypeDecision, DocTypeGuideline, DocTypeHandbook, DocTypeCurriculum, DocTypePlan, DocTypeScheme}
	for _, docType := range official {
		if got := DefaultValidityTier(docType); got != ValidityTierOfficial {
			t.Fatalf("expected %s to map to %q, got %q", docType, ValidityTierOfficial, got)
		}
	}
	if got := DefaultValidityTier(DocTypeNotice); got != ValidityTierSemesterDynamic {
		t.Fatalf("expected notices to map to %q, got %q", ValidityTierSemesterDynamic, got)
	}
	for _, docType := range []string{DocTypeNews, DocTypeStatic} {
		if got := DefaultValidityTier(docType); got != ValidityTierCurrentOperational {
			t.Fatalf("expected %s to map to %q, got %q", docType, ValidityTierCurrentOperational, got)
		}
	}
	if got := DefaultValidityTier(""); got != "" {
		t.Fatalf("expected an empty doc type to stay unassigned, got %q", got)
	}
}

func TestDeriveEffectiveFromAcademicYear(t *testing.T) {
	cases := map[string]string{
		"2011-2012": "2011-09-01",
		"2025-2026": "2025-09-01",
		"2025":      "2025-09-01",
		"":          "",
		"HK1":       "",
		"abcd-2026": "",
	}
	for input, want := range cases {
		if got := DeriveEffectiveFromAcademicYear(input); got != want {
			t.Fatalf("DeriveEffectiveFromAcademicYear(%q) = %q, want %q", input, got, want)
		}
	}
}

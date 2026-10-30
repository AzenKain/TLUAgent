package defaults

import (
	"strings"
	"testing"
)

func TestDiscoverDefaultSkills(t *testing.T) {
	skills, err := DiscoverDefaultSkills()
	if err != nil {
		t.Fatalf("unexpected error discovering default skills: %v", err)
	}

	if len(skills) < 18 {
		t.Fatalf("expected at least 18 skills, got %d", len(skills))
	}

	foundIDs := make(map[string]bool)
	for _, s := range skills {
		foundIDs[s.ID] = true
		if s.Name == "" {
			t.Errorf("skill %s missing name", s.ID)
		}
		if s.Description == "" {
			t.Errorf("skill %s missing description", s.ID)
		}
		if s.Priority <= 0 {
			t.Errorf("skill %s priority must be > 0, got %d", s.ID, s.Priority)
		}
	}

	expectedIDs := []string{
		SkillSearchRegulationsID,
		SkillCurriculumAdvisoryID,
		SkillStudyRoadmapOptimizationID,
		SkillAcademicStandingID,
		SkillTuitionCalcID,
		SkillCourseRegID,
		SkillInquiryEscalationID,
		SkillTLUClawCurriculumOpsID,
		SkillTLUClawDocSupersessionOpsID,
		SkillTLUClawRollbackAuditOpsID,
		SkillTLUClawSubstitutionOpsID,
		SkillTLUClawStudentAcademicOpsID,
		SkillTLUClawUserOpsID,
		SkillTLUClawNotificationOpsID,
		SkillTLUClawJobOpsID,
		SkillTLUClawInquiryOpsID,
		SkillTLUClawSystemOpsID,
		SkillTLUClawAdminMemoryOpsID,
	}

	for _, id := range expectedIDs {
		if !foundIDs[id] {
			t.Errorf("expected skill %s was not discovered", id)
		}
	}
}

func TestReadDefaultFileExpandsConfiguredEndpoints(t *testing.T) {
	t.Setenv("TUITION_PORTAL_URL", "https://portal.example.test/pay/tenant")
	t.Setenv("CMS_SITE_URL", "https://site.example.test")

	content, err := ReadDefaultFile("skills/tuition_calculation.md")
	if err != nil {
		t.Fatalf("unexpected error reading default skill: %v", err)
	}
	if !strings.Contains(content, "https://portal.example.test/pay/tenant") {
		t.Fatal("expected the tuition portal env value to be substituted into the default skill")
	}
	if strings.Contains(content, "{{tuition_portal_url}}") || strings.Contains(content, "{{university_site_url}}") {
		t.Fatal("expected every placeholder to be substituted")
	}
}

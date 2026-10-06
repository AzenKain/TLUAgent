package memory

import (
	"strings"
	"testing"
	"time"
)

func TestFormatStudentPromptContext(t *testing.T) {
	if FormatStudentPromptContext(nil) != "" {
		t.Errorf("expected empty string for nil profile")
	}

	profile := &StudentAcademicProfile{
		StudentID:        "A34567",
		FullName:         "Nguyen Van A",
		Major:            "Computer Science",
		CohortYear:       34,
		AcademicStanding: StandingNormal,
		CompletedCredits: 95,
		CumulativeGPA:    3.25,
		Preferences: map[string]string{
			"schedule": "evening",
		},
	}

	formatted := FormatStudentPromptContext(profile)
	if !strings.Contains(formatted, "A34567") {
		t.Errorf("expected student ID in formatted output")
	}
	if !strings.Contains(formatted, "Computer Science") {
		t.Errorf("expected major in formatted output")
	}
	if !strings.Contains(formatted, "evening") {
		t.Errorf("expected preferences in formatted output")
	}
}

func TestFormatLearnedKnowledgePromptContext(t *testing.T) {
	if FormatLearnedKnowledgePromptContext(nil) != "" {
		t.Errorf("expected empty string for nil items")
	}

	items := []LearnedKnowledgeItem{
		{
			ID:             "qa-1",
			Topic:          "Graduation Condition",
			Question:       "Can I graduate with TOEIC 450?",
			VerifiedAnswer: "No, TLU requires minimum TOEIC 500 or B1 equivalent for graduation.",
			AnsweredBy:     "Academic Affairs Dept",
			Department:     "Dao Tao",
			CreatedAt:      time.Now(),
			UsageCount:     5,
		},
	}

	formatted := FormatLearnedKnowledgePromptContext(items)
	if !strings.Contains(formatted, "TOEIC 500") {
		t.Errorf("expected verified answer in formatted output")
	}
	if !strings.Contains(formatted, "Graduation Condition") {
		t.Errorf("expected topic in formatted output")
	}
}

func TestValidateEscalationParams(t *testing.T) {
	if err := ValidateEscalationParams(nil); err == nil {
		t.Fatalf("expected error for nil params")
	}

	invalidParams := &EscalationInquiryParams{
		Summary:         "",
		DetailedInquiry: "Details",
	}
	if err := ValidateEscalationParams(invalidParams); err == nil {
		t.Fatalf("expected error for empty summary")
	}

	validParams := &EscalationInquiryParams{
		Summary:         "Tuition payment deadline issue",
		DetailedInquiry: "Student was unable to pay tuition due to banking maintenance",
	}
	if err := ValidateEscalationParams(validParams); err != nil {
		t.Fatalf("unexpected error for valid params: %v", err)
	}
	if validParams.Category != CategoryGeneral {
		t.Errorf("expected default category CategoryGeneral, got %v", validParams.Category)
	}
	if validParams.SuggestedUrgency != UrgencyNormal {
		t.Errorf("expected default urgency UrgencyNormal, got %v", validParams.SuggestedUrgency)
	}

	if GetEscalationToolName() != "escalate_inquiry" {
		t.Errorf("unexpected tool name: %s", GetEscalationToolName())
	}
}

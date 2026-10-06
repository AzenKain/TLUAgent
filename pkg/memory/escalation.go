package memory

import (
	"fmt"
	"strings"
)

type InquiryCategory string

const (
	CategoryTuition            InquiryCategory = "TUITION"
	CategoryExamSchedule       InquiryCategory = "EXAM_SCHEDULE"
	CategoryCourseRegistration InquiryCategory = "COURSE_REGISTRATION"
	CategoryGraduation         InquiryCategory = "GRADUATION"
	CategoryGradeAppeal        InquiryCategory = "GRADE_APPEAL"
	CategoryGeneral            InquiryCategory = "GENERAL_INQUIRY"
)

type UrgencyLevel string

const (
	UrgencyLow    UrgencyLevel = "LOW"
	UrgencyNormal UrgencyLevel = "NORMAL"
	UrgencyHigh   UrgencyLevel = "HIGH"
)

type EscalationInquiryParams struct {
	Category         InquiryCategory `json:"category"`
	Department       string          `json:"department"`
	Summary          string          `json:"summary"`
	DetailedInquiry  string          `json:"detailed_inquiry"`
	SuggestedUrgency UrgencyLevel    `json:"suggested_urgency"`
}

// ValidateEscalationParams ensures all required escalation fields are present and safe.
func ValidateEscalationParams(p *EscalationInquiryParams) error {
	if p == nil {
		return fmt.Errorf("escalation parameters cannot be nil")
	}
	if strings.TrimSpace(p.Summary) == "" {
		return fmt.Errorf("escalation summary cannot be empty")
	}
	if strings.TrimSpace(p.DetailedInquiry) == "" {
		return fmt.Errorf("detailed inquiry cannot be empty")
	}
	if p.Category == "" {
		p.Category = CategoryGeneral
	}
	if p.SuggestedUrgency == "" {
		p.SuggestedUrgency = UrgencyNormal
	}
	return nil
}

// GetEscalationToolName returns the exact function call name for inquiry escalation.
func GetEscalationToolName() string {
	return "escalate_inquiry"
}

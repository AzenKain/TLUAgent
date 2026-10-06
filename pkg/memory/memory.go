package memory

import (
	"fmt"
	"strings"
	"time"
)

type AcademicStanding string

const (
	StandingNormal    AcademicStanding = "NORMAL"
	StandingWarning   AcademicStanding = "ACADEMIC_WARNING"
	StandingProbation AcademicStanding = "PROBATION"
)

type StudentAcademicProfile struct {
	UserID           string           `json:"user_id"`
	StudentID        string           `json:"student_id"`
	FullName         string           `json:"full_name"`
	Major            string           `json:"major"`
	CohortYear       int              `json:"cohort_year"`
	AcademicStanding AcademicStanding `json:"academic_standing"`
	CompletedCredits int              `json:"completed_credits"`
	CumulativeGPA    float64          `json:"cumulative_gpa"`
	Preferences      map[string]string `json:"preferences,omitempty"`
}

type LearnedKnowledgeItem struct {
	ID             string    `json:"id"`
	Topic          string    `json:"topic"`
	Question       string    `json:"question"`
	VerifiedAnswer string    `json:"verified_answer"`
	AnsweredBy     string    `json:"answered_by"`
	Department     string    `json:"department"`
	CreatedAt      time.Time `json:"created_at"`
	UsageCount     int       `json:"usage_count"`
}

// FormatStudentPromptContext transforms student profile into a system prompt injection snippet.
func FormatStudentPromptContext(p *StudentAcademicProfile) string {
	if p == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("[STUDENT ACADEMIC PROFILE]\n")
	if p.StudentID != "" {
		fmt.Fprintf(&sb, "- Student ID: %s\n", p.StudentID)
	}
	if p.FullName != "" {
		fmt.Fprintf(&sb, "- Full Name: %s\n", p.FullName)
	}
	if p.Major != "" {
		fmt.Fprintf(&sb, "- Major: %s\n", p.Major)
	}
	if p.CohortYear > 0 {
		fmt.Fprintf(&sb, "- Cohort Year: K%d\n", p.CohortYear)
	}
	if p.CompletedCredits > 0 {
		fmt.Fprintf(&sb, "- Completed Credits: %d\n", p.CompletedCredits)
	}
	if p.CumulativeGPA > 0 {
		fmt.Fprintf(&sb, "- Cumulative GPA: %.2f\n", p.CumulativeGPA)
	}
	if p.AcademicStanding != "" {
		fmt.Fprintf(&sb, "- Academic Standing: %s\n", p.AcademicStanding)
	}
	if len(p.Preferences) > 0 {
		for k, v := range p.Preferences {
			fmt.Fprintf(&sb, "- Preference [%s]: %s\n", k, v)
		}
	}
	return sb.String()
}

// FormatLearnedKnowledgePromptContext builds system prompt context from teacher-verified QAs.
func FormatLearnedKnowledgePromptContext(items []LearnedKnowledgeItem) string {
	if len(items) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("[VERIFIED ACADEMIC KNOWLEDGE BASE (TEACHER-APPROVED)]\n")
	for i, item := range items {
		sb.WriteString(fmt.Sprintf("[%d] Topic: %s (%s)\n", i+1, item.Topic, item.Department))
		sb.WriteString(fmt.Sprintf("    Q: %s\n", item.Question))
		sb.WriteString(fmt.Sprintf("    A: %s\n", item.VerifiedAnswer))
	}
	return sb.String()
}

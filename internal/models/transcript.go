package models

import (
	"time"

	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
)

// StudentCourseGradeRecord represents a persistent student course grade in the database.
type StudentCourseGradeRecord struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	CourseCode  string    `json:"course_code"`
	CourseName  string    `json:"course_name"`
	Credits     float64   `json:"credits"`
	Score10     float64   `json:"score_10"`
	Score4      float64   `json:"score_4"`
	GradeLetter string    `json:"grade_letter"`
	IsPassed    bool      `json:"is_passed"`
	IsRetake    bool      `json:"is_retake"`
	TermTaken   string    `json:"term_taken"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FromSqlc transforms a generated sqlc StudentCourseGrade model into StudentCourseGradeRecord.
func (r *StudentCourseGradeRecord) FromSqlc(row sqlc.StudentCourseGrade) *StudentCourseGradeRecord {
	return &StudentCourseGradeRecord{
		ID:          row.ID,
		UserID:      row.UserID,
		CourseCode:  row.CourseCode,
		CourseName:  row.CourseName,
		Credits:     row.Credits,
		Score10:     row.Score10,
		Score4:      row.Score4,
		GradeLetter: row.GradeLetter,
		IsPassed:    row.IsPassed != 0,
		IsRetake:    row.IsRetake != 0,
		TermTaken:   row.TermTaken,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

// ToResponse converts a StudentCourseGradeRecord to response DTO.
func (r *StudentCourseGradeRecord) ToResponse() *response.StudentCourseGradeItem {
	if r == nil {
		return nil
	}
	return &response.StudentCourseGradeItem{
		ID:          r.ID,
		UserID:      r.UserID,
		CourseCode:  r.CourseCode,
		CourseName:  r.CourseName,
		Credits:     r.Credits,
		Score10:     r.Score10,
		Score4:      r.Score4,
		GradeLetter: r.GradeLetter,
		IsPassed:    r.IsPassed,
		IsRetake:    r.IsRetake,
		TermTaken:   r.TermTaken,
	}
}

// TranscriptSummary aggregates student GPA, total credits, and all course grades.
type TranscriptSummary struct {
	TotalCreditsCompleted float64                     `json:"total_credits_completed"`
	TotalCreditsAttempted float64                     `json:"total_credits_attempted"`
	CumulativeGPA10       float64                     `json:"cumulative_gpa_10"`
	CumulativeGPA4        float64                     `json:"cumulative_gpa_4"`
	TotalCourses          int                         `json:"total_courses"`
	PassedCoursesCount    int                         `json:"passed_courses_count"`
	FailedCoursesCount    int                         `json:"failed_courses_count"`
	Grades                []*StudentCourseGradeRecord `json:"grades"`
}

// ToResponse converts TranscriptSummary to response DTO.
func (s *TranscriptSummary) ToResponse() *response.TranscriptSummaryResponse {
	if s == nil {
		return nil
	}
	items := make([]*response.StudentCourseGradeItem, len(s.Grades))
	for i, g := range s.Grades {
		items[i] = g.ToResponse()
	}
	return &response.TranscriptSummaryResponse{
		TotalCreditsCompleted: s.TotalCreditsCompleted,
		TotalCreditsAttempted: s.TotalCreditsAttempted,
		CumulativeGPA10:       s.CumulativeGPA10,
		CumulativeGPA4:        s.CumulativeGPA4,
		TotalCourses:          s.TotalCourses,
		PassedCoursesCount:    s.PassedCoursesCount,
		FailedCoursesCount:    s.FailedCoursesCount,
		Grades:                items,
	}
}

// TranscriptDiffItem records grade differences between incoming transcript data and database state.
type TranscriptDiffItem struct {
	CourseCode     string  `json:"course_code"`
	CourseName     string  `json:"course_name"`
	Credits        float64 `json:"credits"`
	OldScore10     float64 `json:"old_score_10"`
	NewScore10     float64 `json:"new_score_10"`
	OldGradeLetter string  `json:"old_grade_letter"`
	NewGradeLetter string  `json:"new_grade_letter"`
	OldIsPassed    bool    `json:"old_is_passed"`
	NewIsPassed    bool    `json:"new_is_passed"`
	ChangeType     string  `json:"change_type"`
}

// ToResponse converts TranscriptDiffItem to response DTO.
func (d *TranscriptDiffItem) ToResponse() *response.TranscriptDiffItemResponse {
	if d == nil {
		return nil
	}
	return &response.TranscriptDiffItemResponse{
		CourseCode:     d.CourseCode,
		CourseName:     d.CourseName,
		Credits:        d.Credits,
		OldScore10:     d.OldScore10,
		NewScore10:     d.NewScore10,
		OldGradeLetter: d.OldGradeLetter,
		NewGradeLetter: d.NewGradeLetter,
		OldIsPassed:    d.OldIsPassed,
		NewIsPassed:    d.NewIsPassed,
		ChangeType:     d.ChangeType,
	}
}

// TranscriptDiffResult contains detailed diff breakdown and human-readable confirmation summaries.
type TranscriptDiffResult struct {
	HasChanges            bool                  `json:"has_changes"`
	NewCoursesCount       int                   `json:"new_courses_count"`
	UpgradedCoursesCount  int                   `json:"upgraded_courses_count"`
	UnchangedCoursesCount int                   `json:"unchanged_courses_count"`
	Items                 []*TranscriptDiffItem `json:"items"`
	ConfirmationPrompt    string                `json:"confirmation_prompt"`
}

// ToResponse converts TranscriptDiffResult to response DTO.
func (r *TranscriptDiffResult) ToResponse() *response.TranscriptDiffResultResponse {
	if r == nil {
		return nil
	}
	items := make([]*response.TranscriptDiffItemResponse, len(r.Items))
	for i, item := range r.Items {
		items[i] = item.ToResponse()
	}
	return &response.TranscriptDiffResultResponse{
		HasChanges:            r.HasChanges,
		NewCoursesCount:       r.NewCoursesCount,
		UpgradedCoursesCount:  r.UpgradedCoursesCount,
		UnchangedCoursesCount: r.UnchangedCoursesCount,
		Items:                 items,
		ConfirmationPrompt:    r.ConfirmationPrompt,
	}
}

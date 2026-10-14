package response

// StudentCourseGradeItem describes a course grade in transcript responses.
type StudentCourseGradeItem struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	CourseCode  string  `json:"course_code"`
	CourseName  string  `json:"course_name"`
	Credits     float64 `json:"credits"`
	Score10     float64 `json:"score_10"`
	Score4      float64 `json:"score_4"`
	GradeLetter string  `json:"grade_letter"`
	IsPassed    bool    `json:"is_passed"`
	IsRetake    bool    `json:"is_retake"`
	TermTaken   string  `json:"term_taken"`
}

// TranscriptDiffItemResponse records grade differences between incoming transcript data and database state.
type TranscriptDiffItemResponse struct {
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

// TranscriptDiffResultResponse contains diff metrics and confirmation text.
type TranscriptDiffResultResponse struct {
	HasChanges            bool                          `json:"has_changes"`
	NewCoursesCount       int                           `json:"new_courses_count"`
	UpgradedCoursesCount  int                           `json:"upgraded_courses_count"`
	UnchangedCoursesCount int                           `json:"unchanged_courses_count"`
	Items                 []*TranscriptDiffItemResponse `json:"items"`
	ConfirmationPrompt    string                        `json:"confirmation_prompt"`
}

// TranscriptSummaryResponse aggregates student GPA, total credits, and all course grades.
type TranscriptSummaryResponse struct {
	TotalCreditsCompleted float64                   `json:"total_credits_completed"`
	TotalCreditsAttempted float64                   `json:"total_credits_attempted"`
	CumulativeGPA10       float64                   `json:"cumulative_gpa_10"`
	CumulativeGPA4        float64                   `json:"cumulative_gpa_4"`
	TotalCourses          int                       `json:"total_courses"`
	PassedCoursesCount    int                       `json:"passed_courses_count"`
	FailedCoursesCount    int                       `json:"failed_courses_count"`
	Grades                []*StudentCourseGradeItem `json:"grades"`
}

// ExtractTranscriptImagesResponse wraps extracted grades and comparison diff against existing database state.
type ExtractTranscriptImagesResponse struct {
	Extracted []*StudentCourseGradeItem      `json:"extracted"`
	Diff      *TranscriptDiffResultResponse `json:"diff"`
}

// BatchSaveTranscriptResponse returns the updated summary after committing grades.
type BatchSaveTranscriptResponse struct {
	Summary *TranscriptSummaryResponse `json:"summary"`
	Message string                     `json:"message"`
}

package request

// StudentCourseGradeInput represents an individual grade record submitted in a batch.
type StudentCourseGradeInput struct {
	ID          string  `json:"id,omitempty" validate:"omitempty,max=128"`
	CourseCode  string  `json:"course_code" validate:"required,min=1,max=50"`
	CourseName  string  `json:"course_name" validate:"omitempty,max=200"`
	Credits     float64 `json:"credits" validate:"gte=0,lte=50"`
	Score10     float64 `json:"score_10" validate:"gte=0,lte=10"`
	Score4      float64 `json:"score_4" validate:"gte=0,lte=4"`
	GradeLetter string  `json:"grade_letter" validate:"omitempty,max=10"`
	IsPassed    bool    `json:"is_passed"`
	IsRetake    bool    `json:"is_retake"`
	TermTaken   string  `json:"term_taken" validate:"omitempty,max=50"`
}

// ExtractTranscriptImagesRequest represents the payload containing base64 transcript images.
type ExtractTranscriptImagesRequest struct {
	Images []string `json:"images" validate:"required,min=1,max=10,dive,required"`
}

// BatchSaveTranscriptRequest represents the payload to commit extracted transcript grades.
type BatchSaveTranscriptRequest struct {
	Grades []StudentCourseGradeInput `json:"grades" validate:"required,min=1,dive"`
}

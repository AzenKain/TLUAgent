package request

type UpsertStudentProfileDto struct {
	Major            string  `json:"major" validate:"omitempty,max=100"`
	Cohort           string  `json:"cohort" validate:"omitempty,max=50"`
	AcademicStanding string  `json:"academic_standing" validate:"omitempty,max=50"`
	CompletedCredits int     `json:"completed_credits" validate:"omitempty,min=0,max=500"`
	CumulativeGPA    float64 `json:"cumulative_gpa" validate:"omitempty,min=0,max=4"`
	TargetGPA        float64 `json:"target_gpa" validate:"omitempty,min=0,max=4"`
	AdvisorNotes     string  `json:"advisor_notes" validate:"omitempty,max=2000"`
}

type SaveUserMemoryDto struct {
	Category    string  `json:"category" validate:"required,min=1,max=100"`
	MemoryKey   string  `json:"memory_key" validate:"required,min=1,max=200"`
	MemoryValue string  `json:"memory_value" validate:"required,min=1,max=2000"`
	Confidence  float64 `json:"confidence" validate:"omitempty,min=0,max=1"`
	Source      string  `json:"source" validate:"omitempty,max=100"`
}

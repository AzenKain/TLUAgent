package response

import "time"

// StudentProfileResponse is the API representation of a student academic profile.
type StudentProfileResponse struct {
	UserID           string    `json:"user_id"`
	Major            string    `json:"major"`
	Cohort           string    `json:"cohort"`
	AcademicStanding string    `json:"academic_standing"`
	CompletedCredits int       `json:"completed_credits"`
	CumulativeGPA    float64   `json:"cumulative_gpa"`
	TargetGPA        float64   `json:"target_gpa"`
	AdvisorNotes     string    `json:"advisor_notes"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// UserMemoryItemResponse is the API representation of a retained student memory fact.
type UserMemoryItemResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Category    string    `json:"category"`
	MemoryKey   string    `json:"memory_key"`
	MemoryValue string    `json:"memory_value"`
	Confidence  float64   `json:"confidence"`
	Source      string    `json:"source"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// StudentProfileBundleResponse pairs a student profile with its retained memory facts.
type StudentProfileBundleResponse struct {
	Profile  *StudentProfileResponse   `json:"profile"`
	Memories []*UserMemoryItemResponse `json:"memories"`
}

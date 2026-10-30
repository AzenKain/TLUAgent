package response

// ScheduledCourseResponse mirrors the serialized shape of a scheduled course.
type ScheduledCourseResponse struct {
	CourseCode           string   `json:"course_code"`
	CourseName           string   `json:"course_name"`
	Credits              float64  `json:"credits"`
	KnowledgeBlock       string   `json:"knowledge_block"`
	IsElearning          bool     `json:"is_elearning"`
	IsRetake             bool     `json:"is_retake"`
	PriorityReason       string   `json:"priority_reason"`
	PrerequisitesMet     []string `json:"prerequisites_met"`
	SubstitutedFrom      string   `json:"substituted_from,omitempty"`
	AvailableSubstitutes []string `json:"available_substitutes,omitempty"`
}

// ScheduledSemesterResponse mirrors the serialized shape of a scheduled semester.
type ScheduledSemesterResponse struct {
	SemesterNumber int                        `json:"semester_number"`
	IsSummer       bool                       `json:"is_summer"`
	TotalCredits   float64                    `json:"total_credits"`
	Courses        []*ScheduledCourseResponse `json:"courses"`
	PaceFeedback   string                     `json:"pace_feedback"`
}

// FreeElectiveWaterfallResponse mirrors the serialized shape of the free elective allocation.
type FreeElectiveWaterfallResponse struct {
	TargetCredits     float64  `json:"target_credits"`
	EarnedCredits     float64  `json:"earned_credits"`
	CrossMajorCourses []string `json:"cross_major_courses"`
	ExtraLanguages    []string `json:"extra_languages"`
	IsFulfilled       bool     `json:"is_fulfilled"`
}

// DegreeDowngradeRiskWarningResponse mirrors the serialized shape of the degree downgrade risk warning.
type DegreeDowngradeRiskWarningResponse struct {
	TotalRetakeCredits float64 `json:"total_retake_credits"`
	MaxAllowedCredits  float64 `json:"max_allowed_credits"`
	IsAtRisk           bool    `json:"is_at_risk"`
	WarningMessage     string  `json:"warning_message"`
	Note               string  `json:"note,omitempty"`
}

// RoadmapOptimizationResultResponse mirrors the serialized shape of the optimized multi-semester plan.
type RoadmapOptimizationResultResponse struct {
	Cohort                 string                             `json:"cohort"`
	MajorName              string                             `json:"major_name"`
	AccumulatedCredits     float64                            `json:"accumulated_credits"`
	RemainingCredits       float64                            `json:"remaining_credits"`
	TotalCurriculumCredits float64                            `json:"total_curriculum_credits"`
	CurrentSemester        int                                `json:"current_semester"`
	TargetGraduationTerm   int                                `json:"target_graduation_term"`
	Semesters              []*ScheduledSemesterResponse       `json:"semesters"`
	CriticalPathCourses    []string                           `json:"critical_path_courses"`
	FreeElectives          FreeElectiveWaterfallResponse      `json:"free_electives"`
	DowngradeRisk          DegreeDowngradeRiskWarningResponse `json:"downgrade_risk"`
	Recommendations        []string                           `json:"recommendations"`
}

// PersonalityQuizResultResponse mirrors the serialized shape of the diagnostic quiz evaluation.
type PersonalityQuizResultResponse struct {
	TargetCreditsPerSemester  float64            `json:"target_credits_per_semester"`
	RecommendedGraduationTerm int                `json:"recommended_graduation_term"`
	CompetencyRadar           map[string]float64 `json:"competency_radar"`
	DeliveryBias              string             `json:"delivery_bias"`
	SummaryNotes              string             `json:"summary_notes"`
}

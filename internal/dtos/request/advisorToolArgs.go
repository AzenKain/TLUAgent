package request

// AdvisorSearchRegulationsArgs defines typed parameters for regulation search tool calls.
type AdvisorSearchRegulationsArgs struct {
	Query  string `json:"query"`
	Cohort string `json:"cohort,omitempty"`
	Domain string `json:"domain,omitempty"`
}

// AdvisorLookupCurriculumArgs defines typed parameters for curriculum lookup tool calls.
type AdvisorLookupCurriculumArgs struct {
	Cohort string `json:"cohort,omitempty"`
	Major  string `json:"major,omitempty"`
}

// AdvisorLookupSubstitutionsArgs defines typed parameters for course substitutions lookup tool calls.
type AdvisorLookupSubstitutionsArgs struct {
	CourseCode string `json:"course_code"`
	Cohort     string `json:"cohort,omitempty"`
	Major      string `json:"major,omitempty"`
}

// AdvisorLookupPrerequisitesArgs defines typed parameters for course prerequisites lookup tool calls.
type AdvisorLookupPrerequisitesArgs struct {
	CourseCode string `json:"course_code"`
	Cohort     string `json:"cohort,omitempty"`
}

// AdvisorOptimizeRoadmapArgs defines typed parameters for roadmap optimization tool calls.
type AdvisorOptimizeRoadmapArgs struct {
	CurrentSemester    int    `json:"current_semester,omitempty"`
	TargetSemesters    int    `json:"target_semesters,omitempty"`
	Cohort             string `json:"cohort,omitempty"`
	Major              string `json:"major,omitempty"`
	PacePreference     string `json:"pace_preference,omitempty"`
	SummerWillingness  bool   `json:"summer_willingness,omitempty"`
	PrimaryStrength    string `json:"primary_strength,omitempty"`
	DeliveryPreference string `json:"delivery_preference,omitempty"`
}

// AdvisorLookupTuitionArgs defines typed parameters for tuition fee lookup tool calls.
type AdvisorLookupTuitionArgs struct {
	StudentID string `json:"student_id,omitempty"`
}

// AdvisorUpdatePreferencesArgs defines typed parameters for student preference updates.
type AdvisorUpdatePreferencesArgs struct {
	TargetSemesters    int    `json:"target_semesters,omitempty"`
	PacePreference     string `json:"pace_preference,omitempty"`
	SummerWillingness  bool   `json:"summer_willingness,omitempty"`
	WorkSchedule       string `json:"work_schedule,omitempty"`
	FocusTrack         string `json:"focus_track,omitempty"`
	CustomNotes        string `json:"custom_notes,omitempty"`
	PrimaryStrength    string `json:"primary_strength,omitempty"`
	DeliveryPreference string `json:"delivery_preference,omitempty"`
}

// AdvisorSyncTranscriptGradesArgs defines typed parameters for transcript grade synchronization.
type AdvisorSyncTranscriptGradesArgs struct {
	Grades []AdvisorTranscriptGradeItem `json:"grades"`
}

// AdvisorTranscriptGradeItem defines an individual course grade entry for transcript synchronization.
type AdvisorTranscriptGradeItem struct {
	CourseCode  string  `json:"course_code"`
	CourseName  string  `json:"course_name"`
	Credits     float64 `json:"credits"`
	Score10     float64 `json:"score_10"`
	Score4      float64 `json:"score_4"`
	GradeLetter string  `json:"grade_letter"`
	IsPassed    bool    `json:"is_passed"`
	IsRetake    bool    `json:"is_retake"`
	TermTaken   string  `json:"term_taken,omitempty"`
}

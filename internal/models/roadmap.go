package models

import (
	"tluagent-web/internal/dtos/response"
)

// StudentCourseGrade describes an academic grade record for a completed or attempted course.
type StudentCourseGrade struct {
	CourseCode    string  `json:"course_code" validate:"required,min=1,max=50"`
	CourseName    string  `json:"course_name,omitempty" validate:"omitempty,max=200"`
	Credits       float64 `json:"credits,omitempty" validate:"omitempty,gte=0,lte=50"`
	Grade         string  `json:"grade" validate:"omitempty,max=10"`
	Score         float64 `json:"score" validate:"omitempty,gte=0,lte=10"`
	SemesterTaken int     `json:"semester_taken,omitempty" validate:"omitempty,gte=0,lte=20"`
	IsRetake      bool    `json:"is_retake"`
}

// PersonalityQuizAnswers contains answers to diagnostic questions.
type PersonalityQuizAnswers struct {
	PacePreference     string `json:"pace_preference" validate:"required,max=50"`
	PrimaryStrength    string `json:"primary_strength" validate:"omitempty,max=50"`
	DeliveryPreference string `json:"delivery_preference" validate:"omitempty,max=50"`
	SummerWillingness  bool   `json:"summer_willingness"`
}

// PersonalityQuizResult contains evaluated diagnostic scores and pacing recommendations.
type PersonalityQuizResult struct {
	TargetCreditsPerSemester  float64            `json:"target_credits_per_semester"`
	RecommendedGraduationTerm int                `json:"recommended_graduation_term"`
	CompetencyRadar           map[string]float64 `json:"competency_radar"`
	DeliveryBias              string             `json:"delivery_bias"`
	SummaryNotes              string             `json:"summary_notes"`
}

// RoadmapOptimizationRequest encapsulates all parameters needed to optimize a study roadmap.
type RoadmapOptimizationRequest struct {
	Cohort               string                  `json:"cohort" validate:"required,min=1,max=50"`
	MajorName            string                  `json:"major_name" validate:"required,min=1,max=100"`
	CurrentSemester      int                     `json:"current_semester" validate:"gte=1,lte=20"`
	TargetGraduationTerm int                     `json:"target_graduation_term" validate:"gte=1,lte=20"`
	Transcript           []StudentCourseGrade    `json:"transcript" validate:"omitempty,dive"`
	QuizAnswers          *PersonalityQuizAnswers `json:"quiz_answers,omitempty" validate:"omitempty"`
}

// ScheduledCourse represents a planned course inside a future semester.
type ScheduledCourse struct {
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

// ToResponse converts a ScheduledCourse domain model to a response DTO.
func (sc *ScheduledCourse) ToResponse() *response.ScheduledCourseResponse {
	if sc == nil {
		return nil
	}
	return &response.ScheduledCourseResponse{
		CourseCode:           sc.CourseCode,
		CourseName:           sc.CourseName,
		Credits:              sc.Credits,
		KnowledgeBlock:       sc.KnowledgeBlock,
		IsElearning:          sc.IsElearning,
		IsRetake:             sc.IsRetake,
		PriorityReason:       sc.PriorityReason,
		PrerequisitesMet:     sc.PrerequisitesMet,
		SubstitutedFrom:      sc.SubstitutedFrom,
		AvailableSubstitutes: sc.AvailableSubstitutes,
	}
}

// ScheduledCoursesToResponse converts a batch of ScheduledCourse to response DTOs.
func ScheduledCoursesToResponse(items []*ScheduledCourse) []*response.ScheduledCourseResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.ScheduledCourseResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

// ScheduledSemester aggregates scheduled courses for a term.
type ScheduledSemester struct {
	SemesterNumber int                `json:"semester_number"`
	IsSummer       bool               `json:"is_summer"`
	TotalCredits   float64            `json:"total_credits"`
	Courses        []*ScheduledCourse `json:"courses"`
	PaceFeedback   string             `json:"pace_feedback"`
}

// ToResponse converts a ScheduledSemester domain model to a response DTO.
func (ss *ScheduledSemester) ToResponse() *response.ScheduledSemesterResponse {
	if ss == nil {
		return nil
	}
	return &response.ScheduledSemesterResponse{
		SemesterNumber: ss.SemesterNumber,
		IsSummer:       ss.IsSummer,
		TotalCredits:   ss.TotalCredits,
		Courses:        ScheduledCoursesToResponse(ss.Courses),
		PaceFeedback:   ss.PaceFeedback,
	}
}

// ScheduledSemestersToResponse converts a batch of ScheduledSemester to response DTOs.
func ScheduledSemestersToResponse(items []*ScheduledSemester) []*response.ScheduledSemesterResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.ScheduledSemesterResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

// FreeElectiveWaterfall tracks the dynamic allocation of 5 elective credits.
type FreeElectiveWaterfall struct {
	TargetCredits     float64  `json:"target_credits"`
	EarnedCredits     float64  `json:"earned_credits"`
	CrossMajorCourses []string `json:"cross_major_courses"`
	ExtraLanguages    []string `json:"extra_languages"`
	ExcessElectives   []string `json:"excess_electives"`
	IsFulfilled       bool     `json:"is_fulfilled"`
}

// ToResponse converts a FreeElectiveWaterfall domain model to a response DTO.
func (few *FreeElectiveWaterfall) ToResponse() *response.FreeElectiveWaterfallResponse {
	if few == nil {
		return nil
	}
	return &response.FreeElectiveWaterfallResponse{
		TargetCredits:     few.TargetCredits,
		EarnedCredits:     few.EarnedCredits,
		CrossMajorCourses: few.CrossMajorCourses,
		ExtraLanguages:    few.ExtraLanguages,
		ExcessElectives:   few.ExcessElectives,
		IsFulfilled:       few.IsFulfilled,
	}
}

// DegreeDowngradeRiskWarning evaluates risk of academic honors downgrade.
type DegreeDowngradeRiskWarning struct {
	TotalRetakeCredits float64 `json:"total_retake_credits"`
	MaxAllowedCredits  float64 `json:"max_allowed_credits"`
	IsAtRisk           bool    `json:"is_at_risk"`
	WarningMessage     string  `json:"warning_message"`
}

// ToResponse converts a DegreeDowngradeRiskWarning domain model to a response DTO.
func (ddrw *DegreeDowngradeRiskWarning) ToResponse() *response.DegreeDowngradeRiskWarningResponse {
	if ddrw == nil {
		return nil
	}
	return &response.DegreeDowngradeRiskWarningResponse{
		TotalRetakeCredits: ddrw.TotalRetakeCredits,
		MaxAllowedCredits:  ddrw.MaxAllowedCredits,
		IsAtRisk:           ddrw.IsAtRisk,
		WarningMessage:     ddrw.WarningMessage,
	}
}

// RoadmapOptimizationResult holds the fully scheduled multi-semester plan.
type RoadmapOptimizationResult struct {
	Cohort                 string                     `json:"cohort"`
	MajorName              string                     `json:"major_name"`
	AccumulatedCredits     float64                    `json:"accumulated_credits"`
	RemainingCredits       float64                    `json:"remaining_credits"`
	TotalCurriculumCredits float64                    `json:"total_curriculum_credits"`
	CurrentSemester        int                        `json:"current_semester"`
	TargetGraduationTerm   int                        `json:"target_graduation_term"`
	Semesters              []*ScheduledSemester       `json:"semesters"`
	CriticalPathCourses    []string                   `json:"critical_path_courses"`
	FreeElectives          FreeElectiveWaterfall      `json:"free_electives"`
	DowngradeRisk          DegreeDowngradeRiskWarning `json:"downgrade_risk"`
	Recommendations        []string                   `json:"recommendations"`
}

// ToResponse converts a RoadmapOptimizationResult domain model to a response DTO.
func (ror *RoadmapOptimizationResult) ToResponse() *response.RoadmapOptimizationResultResponse {
	if ror == nil {
		return nil
	}
	return &response.RoadmapOptimizationResultResponse{
		Cohort:                 ror.Cohort,
		MajorName:              ror.MajorName,
		AccumulatedCredits:     ror.AccumulatedCredits,
		RemainingCredits:       ror.RemainingCredits,
		TotalCurriculumCredits: ror.TotalCurriculumCredits,
		CurrentSemester:        ror.CurrentSemester,
		TargetGraduationTerm:   ror.TargetGraduationTerm,
		Semesters:              ScheduledSemestersToResponse(ror.Semesters),
		CriticalPathCourses:    ror.CriticalPathCourses,
		FreeElectives:          *ror.FreeElectives.ToResponse(),
		DowngradeRisk:          *ror.DowngradeRisk.ToResponse(),
		Recommendations:        ror.Recommendations,
	}
}

// ToResponse converts a PersonalityQuizResult domain model to a response DTO.
func (pqr *PersonalityQuizResult) ToResponse() *response.PersonalityQuizResultResponse {
	if pqr == nil {
		return nil
	}
	return &response.PersonalityQuizResultResponse{
		TargetCreditsPerSemester:  pqr.TargetCreditsPerSemester,
		RecommendedGraduationTerm: pqr.RecommendedGraduationTerm,
		CompetencyRadar:           pqr.CompetencyRadar,
		DeliveryBias:              pqr.DeliveryBias,
		SummaryNotes:              pqr.SummaryNotes,
	}
}

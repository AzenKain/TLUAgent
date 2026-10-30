package models

import (
	"strings"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
)

// Curriculum represents an academic cohort major curriculum.
type Curriculum struct {
	ID           string  `json:"id"`
	Cohort       string  `json:"cohort"`
	AcademicYear string  `json:"academic_year"`
	MajorName    string  `json:"major_name"`
	MajorCode    string  `json:"major_code"`
	FacultyCode  string  `json:"faculty_code"`
	Aliases      string  `json:"aliases"`
	TotalCourses int     `json:"total_courses"`
	TotalCredits float64 `json:"total_credits"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// FromSqlc converts a sqlc Curriculum row to Curriculum model.
func (c *Curriculum) FromSqlc(row sqlc.Curriculum) *Curriculum {
	c.ID = row.ID
	c.Cohort = row.Cohort
	c.AcademicYear = row.AcademicYear
	c.MajorName = row.MajorName
	c.MajorCode = row.MajorCode
	c.FacultyCode = row.FacultyCode
	c.Aliases = row.Aliases
	c.TotalCourses = int(row.TotalCourses)
	c.TotalCredits = row.TotalCredits
	c.CreatedAt = row.CreatedAt
	c.UpdatedAt = row.UpdatedAt
	return c
}

// ToResponse converts a Curriculum domain model to response DTO.
func (c *Curriculum) ToResponse() *response.CurriculumResponse {
	if c == nil {
		return nil
	}
	return &response.CurriculumResponse{
		ID:           c.ID,
		Cohort:       c.Cohort,
		AcademicYear: c.AcademicYear,
		MajorName:    c.MajorName,
		MajorCode:    c.MajorCode,
		FacultyCode:  c.FacultyCode,
		Aliases:      c.Aliases,
		TotalCourses: c.TotalCourses,
		TotalCredits: c.TotalCredits,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

// CurriculumCourse represents an individual course within a curriculum.
type CurriculumCourse struct {
	ID                string   `json:"id"`
	CurriculumID      string   `json:"curriculum_id"`
	Cohort            string   `json:"cohort"`
	MajorName         string   `json:"major_name"`
	CourseCode        string   `json:"course_code"`
	CourseName        string   `json:"course_name"`
	Credits           float64  `json:"credits"`
	Weight            float64  `json:"weight"`
	Hours             string   `json:"hours"`
	KnowledgeBlock    string   `json:"knowledge_block"`
	PrerequisitesText string   `json:"prerequisites_text"`
	PrerequisitesList []string `json:"prerequisites_list,omitempty"`
	SuggestedSemester int      `json:"suggested_semester"`
	IsElearning       bool     `json:"is_elearning"`
	Substitutions     []string `json:"substitutions,omitempty"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
}

// FromSqlc converts a sqlc CurriculumCourse row to CurriculumCourse model.
func (cc *CurriculumCourse) FromSqlc(row sqlc.CurriculumCourse) *CurriculumCourse {
	cc.ID = row.ID
	cc.CurriculumID = row.CurriculumID
	cc.Cohort = row.Cohort
	cc.MajorName = row.MajorName
	cc.CourseCode = row.CourseCode
	cc.CourseName = row.CourseName
	cc.Credits = row.Credits
	cc.Weight = row.Weight
	cc.Hours = row.Hours
	cc.KnowledgeBlock = row.KnowledgeBlock
	cc.PrerequisitesText = row.PrerequisitesText
	cc.SuggestedSemester = int(row.SuggestedSemester)
	cc.IsElearning = row.IsElearning == 1
	cc.CreatedAt = row.CreatedAt
	cc.UpdatedAt = row.UpdatedAt
	return cc
}

// ToResponse converts a CurriculumCourse domain model to response DTO.
func (cc *CurriculumCourse) ToResponse() *response.CurriculumCourseResponse {
	if cc == nil {
		return nil
	}
	return &response.CurriculumCourseResponse{
		ID:                cc.ID,
		CurriculumID:      cc.CurriculumID,
		Cohort:            cc.Cohort,
		MajorName:         cc.MajorName,
		CourseCode:        cc.CourseCode,
		CourseName:        cc.CourseName,
		Credits:           cc.Credits,
		Weight:            cc.Weight,
		Hours:             cc.Hours,
		KnowledgeBlock:    cc.KnowledgeBlock,
		PrerequisitesText: cc.PrerequisitesText,
		PrerequisitesList: cc.PrerequisitesList,
		SuggestedSemester: cc.SuggestedSemester,
		IsElearning:       cc.IsElearning,
		Substitutions:     cc.Substitutions,
		CreatedAt:         cc.CreatedAt,
		UpdatedAt:         cc.UpdatedAt,
	}
}

// CurriculumCoursesToResponse converts a batch of CurriculumCourse to response DTOs.
func CurriculumCoursesToResponse(items []*CurriculumCourse) []*response.CurriculumCourseResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.CurriculumCourseResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

// CoursePrerequisite represents a direct dependency edge between two courses.
type CoursePrerequisite struct {
	ID                     string  `json:"id"`
	CurriculumCourseID     string  `json:"curriculum_course_id"`
	Cohort                 string  `json:"cohort"`
	MajorName              string  `json:"major_name"`
	CourseCode             string  `json:"course_code"`
	PrerequisiteCourseCode string  `json:"prerequisite_course_code"`
	MinGrade               float64 `json:"min_grade"`
	ConditionType          string  `json:"condition_type"`
	RawExpression          string  `json:"raw_expression"`
	CreatedAt              string  `json:"created_at"`
}

// FromSqlc converts a sqlc CoursePrerequisite row to CoursePrerequisite model.
func (cp *CoursePrerequisite) FromSqlc(row sqlc.CoursePrerequisite) *CoursePrerequisite {
	cp.ID = row.ID
	cp.CurriculumCourseID = row.CurriculumCourseID
	cp.Cohort = row.Cohort
	cp.MajorName = row.MajorName
	cp.CourseCode = row.CourseCode
	cp.PrerequisiteCourseCode = row.PrerequisiteCourseCode
	cp.MinGrade = row.MinGrade
	cp.ConditionType = row.ConditionType
	cp.RawExpression = row.RawExpression
	cp.CreatedAt = row.CreatedAt
	return cp
}

// ToResponse converts a CoursePrerequisite domain model to response DTO.
func (cp *CoursePrerequisite) ToResponse() *response.CoursePrerequisiteResponse {
	if cp == nil {
		return nil
	}
	return &response.CoursePrerequisiteResponse{
		ID:                     cp.ID,
		CurriculumCourseID:     cp.CurriculumCourseID,
		Cohort:                 cp.Cohort,
		MajorName:              cp.MajorName,
		CourseCode:             cp.CourseCode,
		PrerequisiteCourseCode: cp.PrerequisiteCourseCode,
		MinGrade:               cp.MinGrade,
		ConditionType:          cp.ConditionType,
		RawExpression:          cp.RawExpression,
		CreatedAt:              cp.CreatedAt,
	}
}

// CoursePrerequisitesToResponse converts a batch of CoursePrerequisite to response DTOs.
func CoursePrerequisitesToResponse(items []*CoursePrerequisite) []*response.CoursePrerequisiteResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.CoursePrerequisiteResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

// CourseSubstitution represents a validated course substitution rule.
type CourseSubstitution struct {
	ID                    string  `json:"id"`
	OriginalCourseCode    string  `json:"original_course_code"`
	OriginalCourseName    string  `json:"original_course_name"`
	ReplacementCourseCode string  `json:"replacement_course_code"`
	ReplacementCourseName string  `json:"replacement_course_name"`
	ScopeCohort           string  `json:"scope_cohort"`
	ScopeMajor            string  `json:"scope_major"`
	MinCohort             int     `json:"min_cohort"`
	MaxCohort             int     `json:"max_cohort"`
	IsBidirectional       bool    `json:"is_bidirectional"`
	IsTransitive          bool    `json:"is_transitive"`
	CreditsOriginal       float64 `json:"credits_original"`
	CreditsReplacement    float64 `json:"credits_replacement"`
	RuleType              string  `json:"rule_type"`
	RuleNote              string  `json:"rule_note"`
	SourceReference       string  `json:"source_reference"`
	CreatedAt             string  `json:"created_at"`
}

// FromSqlc converts a sqlc CourseSubstitution row to CourseSubstitution model.
func (cs *CourseSubstitution) FromSqlc(row sqlc.CourseSubstitution) *CourseSubstitution {
	cs.ID = row.ID
	cs.OriginalCourseCode = row.OriginalCourseCode
	cs.OriginalCourseName = row.OriginalCourseName
	cs.ReplacementCourseCode = row.ReplacementCourseCode
	cs.ReplacementCourseName = row.ReplacementCourseName
	cs.ScopeCohort = row.ScopeCohort
	cs.ScopeMajor = row.ScopeMajor
	cs.MinCohort = int(row.MinCohort)
	cs.MaxCohort = int(row.MaxCohort)
	cs.IsBidirectional = row.IsBidirectional == 1
	cs.IsTransitive = row.IsTransitive == 1
	cs.CreditsOriginal = row.CreditsOriginal
	cs.CreditsReplacement = row.CreditsReplacement
	cs.RuleType = row.RuleType
	cs.RuleNote = row.RuleNote
	cs.SourceReference = row.SourceReference
	cs.CreatedAt = row.CreatedAt
	return cs
}

// ToResponse converts a CourseSubstitution domain model to response DTO.
func (cs *CourseSubstitution) ToResponse() *response.CourseSubstitutionResponse {
	if cs == nil {
		return nil
	}
	return &response.CourseSubstitutionResponse{
		ID:                    cs.ID,
		OriginalCourseCode:    cs.OriginalCourseCode,
		OriginalCourseName:    cs.OriginalCourseName,
		ReplacementCourseCode: cs.ReplacementCourseCode,
		ReplacementCourseName: cs.ReplacementCourseName,
		ScopeCohort:           cs.ScopeCohort,
		ScopeMajor:            cs.ScopeMajor,
		MinCohort:             cs.MinCohort,
		MaxCohort:             cs.MaxCohort,
		IsBidirectional:       cs.IsBidirectional,
		IsTransitive:          cs.IsTransitive,
		CreditsOriginal:       cs.CreditsOriginal,
		CreditsReplacement:    cs.CreditsReplacement,
		RuleType:              cs.RuleType,
		RuleNote:              cs.RuleNote,
		SourceReference:       cs.SourceReference,
		CreatedAt:             cs.CreatedAt,
	}
}

// CourseSubstitutionsToResponse converts a batch of CourseSubstitution to response DTOs.
func CourseSubstitutionsToResponse(items []*CourseSubstitution) []*response.CourseSubstitutionResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.CourseSubstitutionResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

// CourseGroupSubstitution represents a group/wildcard elective substitution rule.
type CourseGroupSubstitution struct {
	ID                         string `json:"id"`
	TargetDescription          string `json:"target_description"`
	ReplacementRuleDescription string `json:"replacement_rule_description"`
	ScopeCohort                string `json:"scope_cohort"`
	ScopeMajor                 string `json:"scope_major"`
	MinCohort                  int    `json:"min_cohort"`
	MaxCohort                  int    `json:"max_cohort"`
	AllowedPrefixes            string `json:"allowed_prefixes"`
	SourceReference            string `json:"source_reference"`
	CreatedAt                  string `json:"created_at"`
}

// FromSqlc converts a sqlc CourseGroupSubstitution row to CourseGroupSubstitution model.
func (cgs *CourseGroupSubstitution) FromSqlc(row sqlc.CourseGroupSubstitution) *CourseGroupSubstitution {
	cgs.ID = row.ID
	cgs.TargetDescription = row.TargetDescription
	cgs.ReplacementRuleDescription = row.ReplacementRuleDescription
	cgs.ScopeCohort = row.ScopeCohort
	cgs.ScopeMajor = row.ScopeMajor
	cgs.MinCohort = int(row.MinCohort)
	cgs.MaxCohort = int(row.MaxCohort)
	cgs.AllowedPrefixes = row.AllowedPrefixes
	cgs.SourceReference = row.SourceReference
	cgs.CreatedAt = row.CreatedAt
	return cgs
}

// ToResponse converts a CourseGroupSubstitution domain model to response DTO.
func (cgs *CourseGroupSubstitution) ToResponse() *response.CourseGroupSubstitutionResponse {
	if cgs == nil {
		return nil
	}
	return &response.CourseGroupSubstitutionResponse{
		ID:                         cgs.ID,
		TargetDescription:          cgs.TargetDescription,
		ReplacementRuleDescription: cgs.ReplacementRuleDescription,
		ScopeCohort:                cgs.ScopeCohort,
		ScopeMajor:                 cgs.ScopeMajor,
		MinCohort:                  cgs.MinCohort,
		MaxCohort:                  cgs.MaxCohort,
		AllowedPrefixes:            cgs.AllowedPrefixes,
		SourceReference:            cgs.SourceReference,
		CreatedAt:                  cgs.CreatedAt,
	}
}

// CourseGroupSubstitutionsToResponse converts a batch of CourseGroupSubstitution to response DTOs.
func CourseGroupSubstitutionsToResponse(items []*CourseGroupSubstitution) []*response.CourseGroupSubstitutionResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.CourseGroupSubstitutionResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

// SemesterGroup aggregates curriculum courses by semester.
type SemesterGroup struct {
	Semester     int                 `json:"semester"`
	TotalCredits float64             `json:"total_credits"`
	CourseCount  int                 `json:"course_count"`
	Courses      []*CurriculumCourse `json:"courses"`
}

// ToResponse converts a SemesterGroup domain model to response DTO.
func (sg *SemesterGroup) ToResponse() *response.SemesterGroupResponse {
	if sg == nil {
		return nil
	}
	return &response.SemesterGroupResponse{
		Semester:     sg.Semester,
		TotalCredits: sg.TotalCredits,
		CourseCount:  sg.CourseCount,
		Courses:      CurriculumCoursesToResponse(sg.Courses),
	}
}

// SemesterGroupsToResponse converts a batch of SemesterGroup to response DTOs.
func SemesterGroupsToResponse(items []*SemesterGroup) []*response.SemesterGroupResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.SemesterGroupResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

// KnowledgeBlockDistribution reports credit and course statistics for a knowledge block.
type KnowledgeBlockDistribution struct {
	Name           string  `json:"name"`
	KnowledgeBlock string  `json:"knowledge_block,omitempty"`
	CourseCount    int     `json:"course_count"`
	TotalCredits   float64 `json:"total_credits"`
	Percentage     float64 `json:"percentage"`
}

// ToResponse converts a KnowledgeBlockDistribution domain model to response DTO.
func (kbd *KnowledgeBlockDistribution) ToResponse() *response.KnowledgeBlockDistributionResponse {
	if kbd == nil {
		return nil
	}
	return &response.KnowledgeBlockDistributionResponse{
		Name:           kbd.Name,
		KnowledgeBlock: kbd.KnowledgeBlock,
		CourseCount:    kbd.CourseCount,
		TotalCredits:   kbd.TotalCredits,
		Percentage:     kbd.Percentage,
	}
}

// KnowledgeBlockDistributionsToResponse converts a batch of KnowledgeBlockDistribution to response DTOs.
func KnowledgeBlockDistributionsToResponse(items []*KnowledgeBlockDistribution) []*response.KnowledgeBlockDistributionResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.KnowledgeBlockDistributionResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

// PrerequisiteEdge describes a directed dependency between courses.
type PrerequisiteEdge struct {
	From               string  `json:"from"`
	To                 string  `json:"to"`
	FromCourseCode     string  `json:"from_course_code"`
	ToCourseCode       string  `json:"to_course_code"`
	FromCourseName     string  `json:"from_course_name,omitempty"`
	ToCourseName       string  `json:"to_course_name,omitempty"`
	FromCredits        float64 `json:"from_credits,omitempty"`
	ToCredits          float64 `json:"to_credits,omitempty"`
	FromKnowledgeBlock string  `json:"from_knowledge_block,omitempty"`
	ToKnowledgeBlock   string  `json:"to_knowledge_block,omitempty"`
	MinGrade           float64 `json:"min_grade"`
	ConditionType      string  `json:"condition_type"`
}

// ToResponse converts a PrerequisiteEdge domain model to response DTO.
func (pe *PrerequisiteEdge) ToResponse() *response.PrerequisiteEdgeResponse {
	if pe == nil {
		return nil
	}
	return &response.PrerequisiteEdgeResponse{
		From:               pe.From,
		To:                 pe.To,
		FromCourseCode:     pe.FromCourseCode,
		ToCourseCode:       pe.ToCourseCode,
		FromCourseName:     pe.FromCourseName,
		ToCourseName:       pe.ToCourseName,
		FromCredits:        pe.FromCredits,
		ToCredits:          pe.ToCredits,
		FromKnowledgeBlock: pe.FromKnowledgeBlock,
		ToKnowledgeBlock:   pe.ToKnowledgeBlock,
		MinGrade:           pe.MinGrade,
		ConditionType:      pe.ConditionType,
	}
}

// PrerequisiteEdgesToResponse converts a batch of PrerequisiteEdge to response DTOs.
func PrerequisiteEdgesToResponse(items []*PrerequisiteEdge) []*response.PrerequisiteEdgeResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.PrerequisiteEdgeResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

// CurriculumVisualizationData encapsulates visual roadmap data for an academic program.
type CurriculumVisualizationData struct {
	Curriculum        *Curriculum                   `json:"curriculum"`
	TotalCourses      int                           `json:"total_courses"`
	TotalCredits      float64                       `json:"total_credits"`
	Courses           []*CurriculumCourse           `json:"courses,omitempty"`
	Semesters         []*SemesterGroup              `json:"semesters,omitempty"`
	KnowledgeBlocks   []*KnowledgeBlockDistribution `json:"knowledge_blocks"`
	PrerequisiteEdges []*PrerequisiteEdge           `json:"prerequisite_edges"`
}

// ToResponse converts a CurriculumVisualizationData domain model to response DTO.
func (cvd *CurriculumVisualizationData) ToResponse() *response.CurriculumVisualizationDataResponse {
	if cvd == nil {
		return nil
	}
	return &response.CurriculumVisualizationDataResponse{
		Curriculum:        cvd.Curriculum.ToResponse(),
		TotalCourses:      cvd.TotalCourses,
		TotalCredits:      cvd.TotalCredits,
		Courses:           CurriculumCoursesToResponse(cvd.Courses),
		Semesters:         SemesterGroupsToResponse(cvd.Semesters),
		KnowledgeBlocks:   KnowledgeBlockDistributionsToResponse(cvd.KnowledgeBlocks),
		PrerequisiteEdges: PrerequisiteEdgesToResponse(cvd.PrerequisiteEdges),
	}
}

// StudentRoadmapCourse extends CurriculumCourse with student grade and eligibility status.
type StudentRoadmapCourse struct {
	CurriculumCourse
	Status         string   `json:"status"`
	Score10        float64  `json:"score_10,omitempty"`
	Score4         float64  `json:"score_4,omitempty"`
	GradeLetter    string   `json:"grade_letter,omitempty"`
	TermTaken      string   `json:"term_taken,omitempty"`
	MissingPrereqs []string `json:"missing_prereqs,omitempty"`
	IsCriticalPath bool     `json:"is_critical_path"`
}

// ToResponse converts a StudentRoadmapCourse domain model to response DTO.
func (src *StudentRoadmapCourse) ToResponse() *response.StudentRoadmapCourseResponse {
	if src == nil {
		return nil
	}
	return &response.StudentRoadmapCourseResponse{
		CurriculumCourseResponse: *src.CurriculumCourse.ToResponse(),
		Status:                   src.Status,
		Score10:                  src.Score10,
		Score4:                   src.Score4,
		GradeLetter:              src.GradeLetter,
		TermTaken:                src.TermTaken,
		MissingPrereqs:           src.MissingPrereqs,
		IsCriticalPath:           src.IsCriticalPath,
	}
}

// StudentRoadmapCoursesToResponse converts a batch of StudentRoadmapCourse to response DTOs.
func StudentRoadmapCoursesToResponse(items []*StudentRoadmapCourse) []*response.StudentRoadmapCourseResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.StudentRoadmapCourseResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

// StudentSemesterGroup groups personal roadmap courses by semester.
type StudentSemesterGroup struct {
	Semester     int                     `json:"semester"`
	TotalCredits float64                 `json:"total_credits"`
	CourseCount  int                     `json:"course_count"`
	Courses      []*StudentRoadmapCourse `json:"courses"`
}

// ToResponse converts a StudentSemesterGroup domain model to response DTO.
func (ssg *StudentSemesterGroup) ToResponse() *response.StudentSemesterGroupResponse {
	if ssg == nil {
		return nil
	}
	return &response.StudentSemesterGroupResponse{
		Semester:     ssg.Semester,
		TotalCredits: ssg.TotalCredits,
		CourseCount:  ssg.CourseCount,
		Courses:      StudentRoadmapCoursesToResponse(ssg.Courses),
	}
}

// StudentSemesterGroupsToResponse converts a batch of StudentSemesterGroup to response DTOs.
func StudentSemesterGroupsToResponse(items []*StudentSemesterGroup) []*response.StudentSemesterGroupResponse {
	if items == nil {
		return nil
	}
	out := make([]*response.StudentSemesterGroupResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, item.ToResponse())
	}
	return out
}

// StudentRoadmapSummary reports comprehensive student academic progress metrics.
type StudentRoadmapSummary struct {
	Cohort                 string  `json:"cohort"`
	MajorName              string  `json:"major_name"`
	TotalCurriculumCredits float64 `json:"total_curriculum_credits"`
	TotalCreditsCompleted  float64 `json:"total_credits_completed"`
	TotalCreditsRemaining  float64 `json:"total_credits_remaining"`
	ProgressPercentage     float64 `json:"progress_percentage"`
	CumulativeGPA10        float64 `json:"cumulative_gpa_10"`
	CumulativeGPA4         float64 `json:"cumulative_gpa_4"`
	TotalCourses           int     `json:"total_courses"`
	PassedCoursesCount     int     `json:"passed_courses_count"`
	FailedCoursesCount     int     `json:"failed_courses_count"`
}

// ToResponse converts a StudentRoadmapSummary domain model to response DTO.
func (srs *StudentRoadmapSummary) ToResponse() *response.StudentRoadmapSummaryResponse {
	if srs == nil {
		return nil
	}
	return &response.StudentRoadmapSummaryResponse{
		Cohort:                 srs.Cohort,
		MajorName:              srs.MajorName,
		TotalCurriculumCredits: srs.TotalCurriculumCredits,
		TotalCreditsCompleted:  srs.TotalCreditsCompleted,
		TotalCreditsRemaining:  srs.TotalCreditsRemaining,
		ProgressPercentage:     srs.ProgressPercentage,
		CumulativeGPA10:        srs.CumulativeGPA10,
		CumulativeGPA4:         srs.CumulativeGPA4,
		TotalCourses:           srs.TotalCourses,
		PassedCoursesCount:     srs.PassedCoursesCount,
		FailedCoursesCount:     srs.FailedCoursesCount,
	}
}

// StudentPersonalRoadmapData encapsulates personal roadmap and progress visualization data.
type StudentPersonalRoadmapData struct {
	Summary           *StudentRoadmapSummary        `json:"summary"`
	Semesters         []*StudentSemesterGroup       `json:"semesters"`
	KnowledgeBlocks   []*KnowledgeBlockDistribution `json:"knowledge_blocks"`
	PrerequisiteEdges []*PrerequisiteEdge           `json:"prerequisite_edges"`
	OptimizationPlan  *RoadmapOptimizationResult    `json:"optimization_plan,omitempty"`
}

// ToResponse converts a StudentPersonalRoadmapData domain model to response DTO.
func (sprd *StudentPersonalRoadmapData) ToResponse() *response.StudentPersonalRoadmapDataResponse {
	if sprd == nil {
		return nil
	}
	return &response.StudentPersonalRoadmapDataResponse{
		Summary:           sprd.Summary.ToResponse(),
		Semesters:         StudentSemesterGroupsToResponse(sprd.Semesters),
		KnowledgeBlocks:   KnowledgeBlockDistributionsToResponse(sprd.KnowledgeBlocks),
		PrerequisiteEdges: PrerequisiteEdgesToResponse(sprd.PrerequisiteEdges),
		OptimizationPlan:  sprd.OptimizationPlan.ToResponse(),
	}
}

const (
	RequirementTotalMin     = "TOTAL_MIN"
	RequirementChooseOne    = "CHOOSE_ONE"
	RequirementElectivePool = "ELECTIVE_POOL"
	RequirementFreeElective = "FREE_ELECTIVE"
	RequirementNonCredit    = "NON_CREDIT"
)

// RequirementRule describes one graduation requirement of a curriculum beyond the plain course list.
type RequirementRule struct {
	ID          string  `json:"id"`
	Cohort      string  `json:"cohort"`
	MajorName   string  `json:"major_name"`
	RuleType    string  `json:"rule_type"`
	GroupKey    string  `json:"group_key"`
	MinCredits  float64 `json:"min_credits"`
	CourseCodes string  `json:"course_codes"`
	Note        string  `json:"note"`
	Source      string  `json:"source"`
}

// CodeSets splits CourseCodes into alternatives separated by "|", each holding comma-separated course codes.
func (r *RequirementRule) CodeSets() [][]string {
	var sets [][]string
	for alt := range strings.SplitSeq(r.CourseCodes, "|") {
		var codes []string
		for _, c := range strings.Split(alt, ",") {
			if c = strings.ToUpper(strings.TrimSpace(c)); c != "" {
				codes = append(codes, c)
			}
		}
		if len(codes) > 0 {
			sets = append(sets, codes)
		}
	}
	return sets
}

// MatchesCode reports whether a course code equals the pattern or starts with the pattern's prefix before a trailing "*".
func MatchesCode(pattern, code string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(code, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == code
}

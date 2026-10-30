package request

// CreateCourseRequest defines payload for adding a new curriculum course.
type CreateCourseRequest struct {
	Cohort            string  `json:"cohort" validate:"required,min=1,max=50"`
	MajorName         string  `json:"major_name" validate:"required,min=1,max=100"`
	CourseCode        string  `json:"course_code" validate:"required,min=1,max=50"`
	CourseName        string  `json:"course_name" validate:"required,min=1,max=200"`
	Credits           float64 `json:"credits" validate:"gte=0,lte=50"`
	Weight            float64 `json:"weight" validate:"omitempty,gte=0,lte=10"`
	Hours             string  `json:"hours" validate:"omitempty,max=50"`
	KnowledgeBlock    string  `json:"knowledge_block" validate:"omitempty,max=100"`
	PrerequisitesText string  `json:"prerequisites_text" validate:"omitempty,max=500"`
	SuggestedSemester int     `json:"suggested_semester" validate:"omitempty,gte=0,lte=20"`
	IsElearning       bool    `json:"is_elearning"`
}

// UpdateCourseRequest defines payload for modifying an existing curriculum course.
type UpdateCourseRequest struct {
	CourseName        string  `json:"course_name" validate:"required,min=1,max=200"`
	Credits           float64 `json:"credits" validate:"gte=0,lte=50"`
	Weight            float64 `json:"weight" validate:"omitempty,gte=0,lte=10"`
	Hours             string  `json:"hours" validate:"omitempty,max=50"`
	KnowledgeBlock    string  `json:"knowledge_block" validate:"omitempty,max=100"`
	PrerequisitesText string  `json:"prerequisites_text" validate:"omitempty,max=500"`
	SuggestedSemester int     `json:"suggested_semester" validate:"omitempty,gte=0,lte=20"`
	IsElearning       bool    `json:"is_elearning"`
}

// CreateSubstitutionRequest defines payload for establishing a course substitution rule.
type CreateSubstitutionRequest struct {
	OriginalCourseCode    string  `json:"original_course_code" validate:"required,min=1,max=50"`
	OriginalCourseName    string  `json:"original_course_name" validate:"omitempty,max=200"`
	ReplacementCourseCode string  `json:"replacement_course_code" validate:"required,min=1,max=50"`
	ReplacementCourseName string  `json:"replacement_course_name" validate:"omitempty,max=200"`
	ScopeCohort           string  `json:"scope_cohort" validate:"omitempty,max=50"`
	ScopeMajor            string  `json:"scope_major" validate:"omitempty,max=100"`
	IsBidirectional       bool    `json:"is_bidirectional"`
	IsTransitive          bool    `json:"is_transitive"`
	MinCohort             int     `json:"min_cohort" validate:"omitempty,gte=0,lte=999"`
	MaxCohort             int     `json:"max_cohort" validate:"omitempty,gte=0,lte=999"`
	CreditsOriginal       float64 `json:"credits_original" validate:"omitempty,gte=0,lte=50"`
	CreditsReplacement    float64 `json:"credits_replacement" validate:"omitempty,gte=0,lte=50"`
	RuleType              string  `json:"rule_type" validate:"omitempty,max=50"`
	RuleNote              string  `json:"rule_note" validate:"omitempty,max=500"`
	SourceReference       string  `json:"source_reference" validate:"omitempty,max=200"`
}

// UpdateSubstitutionRequest defines payload for editing a course substitution rule.
type UpdateSubstitutionRequest struct {
	OriginalCourseCode    string  `json:"original_course_code" validate:"required,min=1,max=50"`
	OriginalCourseName    string  `json:"original_course_name" validate:"omitempty,max=200"`
	ReplacementCourseCode string  `json:"replacement_course_code" validate:"required,min=1,max=50"`
	ReplacementCourseName string  `json:"replacement_course_name" validate:"omitempty,max=200"`
	ScopeCohort           string  `json:"scope_cohort" validate:"omitempty,max=50"`
	ScopeMajor            string  `json:"scope_major" validate:"omitempty,max=100"`
	MinCohort             int     `json:"min_cohort" validate:"omitempty,gte=0,lte=999"`
	MaxCohort             int     `json:"max_cohort" validate:"omitempty,gte=0,lte=999"`
	IsBidirectional       bool    `json:"is_bidirectional"`
	IsTransitive          bool    `json:"is_transitive"`
	CreditsOriginal       float64 `json:"credits_original" validate:"omitempty,gte=0,lte=50"`
	CreditsReplacement    float64 `json:"credits_replacement" validate:"omitempty,gte=0,lte=50"`
	RuleType              string  `json:"rule_type" validate:"omitempty,max=50"`
	RuleNote              string  `json:"rule_note" validate:"omitempty,max=500"`
	SourceReference       string  `json:"source_reference" validate:"omitempty,max=200"`
}

// ListMajorsQueryDto defines query parameters for filtering majors.
type ListMajorsQueryDto struct {
	Cohort string `query:"cohort" validate:"omitempty,max=50"`
}

// GetVisualizationQueryDto defines query parameters for fetching visualization graphs.
type GetVisualizationQueryDto struct {
	Cohort string `query:"cohort" validate:"required,min=1,max=50"`
	Major  string `query:"major" validate:"required,min=1,max=100"`
}

// ListCoursesQueryDto defines query parameters for filtering and paginating courses.
type ListCoursesQueryDto struct {
	Cohort         string `query:"cohort" validate:"omitempty,max=50"`
	Major          string `query:"major" validate:"omitempty,max=100"`
	KnowledgeBlock string `query:"knowledge_block" validate:"omitempty,max=100"`
	Keyword        string `query:"keyword" validate:"omitempty,max=200"`
	Semester       int    `query:"semester" validate:"omitempty,gte=0,lte=20"`
	IsElearning    string `query:"is_elearning" validate:"omitempty,max=10"`
	Page           int    `query:"page" validate:"omitempty,gte=0"`
	Limit          int    `query:"limit" validate:"omitempty,gte=0,lte=500"`
}

// ListSubstitutionsQueryDto defines query parameters for filtering and paginating course substitutions.
type ListSubstitutionsQueryDto struct {
	Cohort  string `query:"cohort" validate:"omitempty,max=50"`
	Major   string `query:"major" validate:"omitempty,max=100"`
	Keyword string `query:"keyword" validate:"omitempty,max=200"`
	Page    int    `query:"page" validate:"omitempty,gte=0"`
	Limit   int    `query:"limit" validate:"omitempty,gte=0,lte=500"`
}

// RequirementRuleInput is one graduation requirement rule supplied by an administrator.
type RequirementRuleInput struct {
	RuleType    string  `json:"rule_type" validate:"required,oneof=TOTAL_MIN CHOOSE_ONE ELECTIVE_POOL FREE_ELECTIVE NON_CREDIT"`
	GroupKey    string  `json:"group_key" validate:"omitempty,max=100"`
	MinCredits  float64 `json:"min_credits" validate:"gte=0,lte=500"`
	CourseCodes string  `json:"course_codes" validate:"omitempty,max=2000"`
	Note        string  `json:"note" validate:"omitempty,max=500"`
	Source      string  `json:"source" validate:"omitempty,max=300"`
}

// ReplaceRequirementRulesRequest replaces every requirement rule of one curriculum.
type ReplaceRequirementRulesRequest struct {
	Cohort    string                 `json:"cohort" validate:"required,max=50"`
	MajorName string                 `json:"major_name" validate:"required,max=100"`
	Rules     []RequirementRuleInput `json:"rules" validate:"max=50,dive"`
}

// CurriculumScopeQueryDto selects one curriculum by cohort and major in query parameters.
type CurriculumScopeQueryDto struct {
	Cohort string `query:"cohort" validate:"omitempty,max=50"`
	Major  string `query:"major" validate:"omitempty,max=100"`
}

// RequiredCurriculumScopeQueryDto selects one curriculum and rejects requests that omit the cohort or the major.
type RequiredCurriculumScopeQueryDto struct {
	Cohort string `query:"cohort" validate:"required,max=50"`
	Major  string `query:"major" validate:"required,max=100"`
}

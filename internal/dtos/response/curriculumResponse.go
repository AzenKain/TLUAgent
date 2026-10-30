package response

// CurriculumResponse mirrors the serialized shape of an academic cohort major curriculum.
type CurriculumResponse struct {
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

// CurriculumCourseResponse mirrors the serialized shape of a course within a curriculum.
type CurriculumCourseResponse struct {
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

// CoursePrerequisiteResponse mirrors the serialized shape of a direct dependency edge between two courses.
type CoursePrerequisiteResponse struct {
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

// CourseSubstitutionResponse mirrors the serialized shape of a validated course substitution rule.
type CourseSubstitutionResponse struct {
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

// CourseGroupSubstitutionResponse mirrors the serialized shape of a group substitution rule.
type CourseGroupSubstitutionResponse struct {
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

// SemesterGroupResponse mirrors the serialized shape of curriculum courses aggregated by semester.
type SemesterGroupResponse struct {
	Semester     int                         `json:"semester"`
	TotalCredits float64                     `json:"total_credits"`
	CourseCount  int                         `json:"course_count"`
	Courses      []*CurriculumCourseResponse `json:"courses"`
}

// KnowledgeBlockDistributionResponse mirrors the serialized shape of a knowledge block credit distribution.
type KnowledgeBlockDistributionResponse struct {
	Name           string  `json:"name"`
	KnowledgeBlock string  `json:"knowledge_block,omitempty"`
	CourseCount    int     `json:"course_count"`
	TotalCredits   float64 `json:"total_credits"`
	Percentage     float64 `json:"percentage"`
}

// PrerequisiteEdgeResponse mirrors the serialized shape of a directed dependency between courses.
type PrerequisiteEdgeResponse struct {
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

// CurriculumVisualizationDataResponse mirrors the serialized shape of visual roadmap data for an academic program.
type CurriculumVisualizationDataResponse struct {
	Curriculum        *CurriculumResponse                   `json:"curriculum"`
	TotalCourses      int                                   `json:"total_courses"`
	TotalCredits      float64                               `json:"total_credits"`
	Courses           []*CurriculumCourseResponse           `json:"courses,omitempty"`
	Semesters         []*SemesterGroupResponse              `json:"semesters,omitempty"`
	KnowledgeBlocks   []*KnowledgeBlockDistributionResponse `json:"knowledge_blocks"`
	PrerequisiteEdges []*PrerequisiteEdgeResponse           `json:"prerequisite_edges"`
}

// StudentRoadmapCourseResponse mirrors the serialized shape of a curriculum course enriched with student status.
type StudentRoadmapCourseResponse struct {
	CurriculumCourseResponse
	Status         string   `json:"status"`
	Score10        float64  `json:"score_10,omitempty"`
	Score4         float64  `json:"score_4,omitempty"`
	GradeLetter    string   `json:"grade_letter,omitempty"`
	TermTaken      string   `json:"term_taken,omitempty"`
	MissingPrereqs []string `json:"missing_prereqs,omitempty"`
	IsCriticalPath bool     `json:"is_critical_path"`
}

// StudentSemesterGroupResponse mirrors the serialized shape of personal roadmap courses grouped by semester.
type StudentSemesterGroupResponse struct {
	Semester     int                             `json:"semester"`
	TotalCredits float64                         `json:"total_credits"`
	CourseCount  int                             `json:"course_count"`
	Courses      []*StudentRoadmapCourseResponse `json:"courses"`
}

// StudentRoadmapSummaryResponse mirrors the serialized shape of student academic progress metrics.
type StudentRoadmapSummaryResponse struct {
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

// StudentPersonalRoadmapDataResponse mirrors the serialized shape of personal roadmap and progress visualization data.
type StudentPersonalRoadmapDataResponse struct {
	Summary           *StudentRoadmapSummaryResponse        `json:"summary"`
	Semesters         []*StudentSemesterGroupResponse       `json:"semesters"`
	KnowledgeBlocks   []*KnowledgeBlockDistributionResponse `json:"knowledge_blocks"`
	PrerequisiteEdges []*PrerequisiteEdgeResponse           `json:"prerequisite_edges"`
	OptimizationPlan  *RoadmapOptimizationResultResponse    `json:"optimization_plan,omitempty"`
}

// CurriculumStatsResponse reports aggregate counts across the curriculum catalog.
type CurriculumStatsResponse struct {
	Curriculums   int64 `json:"curriculums"`
	Courses       int64 `json:"courses"`
	Substitutions int64 `json:"substitutions"`
	Prerequisites int64 `json:"prerequisites"`
}

// CurriculumCourseListResponse is a paginated envelope of curriculum courses.
type CurriculumCourseListResponse struct {
	Items []*CurriculumCourseResponse `json:"items"`
	Total int64                       `json:"total"`
	Page  int                         `json:"page"`
	Limit int                         `json:"limit"`
}

// CourseSubstitutionListResponse is a paginated envelope of course substitution rules.
type CourseSubstitutionListResponse struct {
	Items []*CourseSubstitutionResponse `json:"items"`
	Total int64                         `json:"total"`
	Page  int                           `json:"page"`
	Limit int                           `json:"limit"`
}

// MajorResponse identifies an academic major available within a curriculum.
type MajorResponse struct {
	MajorName string `json:"major_name"`
	MajorCode string `json:"major_code"`
}

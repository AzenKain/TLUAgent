package request

// TLUClawCurriculumOpArgs defines typed parameters for curriculum operations tool calls.
type TLUClawCurriculumOpArgs struct {
	Action            string   `json:"action"`
	Cohort            string   `json:"cohort,omitempty"`
	AcademicYear      string   `json:"academic_year,omitempty"`
	MajorCode         string   `json:"major_code,omitempty"`
	MajorName         string   `json:"major_name,omitempty"`
	MajorNameVi       string   `json:"major_name_vi,omitempty"`
	FacultyCode       string   `json:"faculty_code,omitempty"`
	Aliases           []string `json:"aliases,omitempty"`
	CourseCode        string   `json:"course_code,omitempty"`
	CourseName        string   `json:"course_name,omitempty"`
	Credits           float64  `json:"credits,omitempty"`
	Hours             string   `json:"hours,omitempty"`
	KnowledgeBlock    string   `json:"knowledge_block,omitempty"`
	PrerequisitesText string   `json:"prerequisites_text,omitempty"`
	SuggestedSemester int64    `json:"suggested_semester,omitempty"`
	IsElearning       int64    `json:"is_elearning,omitempty"`
	Keyword           string   `json:"keyword,omitempty"`
	CoursesPayload    string   `json:"courses_payload,omitempty"`
	Page              int      `json:"page,omitempty"`
	Limit             int      `json:"limit,omitempty"`
}

// TLUClawSubstitutionOpArgs defines typed parameters for substitution operations tool calls.
type TLUClawSubstitutionOpArgs struct {
	Action                 string  `json:"action"`
	CourseCode             string  `json:"course_code,omitempty"`
	Cohort                 string  `json:"cohort,omitempty"`
	Major                  string  `json:"major,omitempty"`
	MajorName              string  `json:"major_name,omitempty"`
	Keyword                string  `json:"keyword,omitempty"`
	OriginalCourseCode     string  `json:"original_course_code,omitempty"`
	OriginalCourseName     string  `json:"original_course_name,omitempty"`
	ReplacementCourseCode  string  `json:"replacement_course_code,omitempty"`
	ReplacementCourseName  string  `json:"replacement_course_name,omitempty"`
	ScopeCohort            string  `json:"scope_cohort,omitempty"`
	ScopeMajor             string  `json:"scope_major,omitempty"`
	MinCohort              int64   `json:"min_cohort,omitempty"`
	MaxCohort              int64   `json:"max_cohort,omitempty"`
	PrerequisiteCourseCode string  `json:"prerequisite_course_code,omitempty"`
	MinGrade               string  `json:"min_grade,omitempty"`
	ConditionType          string  `json:"condition_type,omitempty"`
	RuleNote               string  `json:"rule_note,omitempty"`
	IsBidirectional        bool    `json:"is_bidirectional,omitempty"`
	CreditsOriginal        float64 `json:"credits_original,omitempty"`
	CreditsReplacement     float64 `json:"credits_replacement,omitempty"`
	Page                   int     `json:"page,omitempty"`
	Limit                  int     `json:"limit,omitempty"`
}

// TLUClawStudentAcademicOpArgs defines typed parameters for student academic operations tool calls.
type TLUClawStudentAcademicOpArgs struct {
	Action               string                    `json:"action"`
	StudentID            string                    `json:"student_id,omitempty"`
	StudentCode          string                    `json:"student_code,omitempty"`
	UserID               string                    `json:"user_id,omitempty"`
	Email                string                    `json:"email,omitempty"`
	AdvisorNote          string                    `json:"advisor_note,omitempty"`
	Grades               []StudentCourseGradeInput `json:"grades,omitempty"`
	Cohort               string                    `json:"cohort,omitempty"`
	Major                string                    `json:"major,omitempty"`
	MajorName            string                    `json:"major_name,omitempty"`
	AcademicStanding     string                    `json:"academic_standing,omitempty"`
	CurrentSemester      int                       `json:"current_semester,omitempty"`
	TargetGraduationTerm int                       `json:"target_graduation_term,omitempty"`
	Page                 int                       `json:"page,omitempty"`
	Limit                int                       `json:"limit,omitempty"`
}

// TLUClawUserOpArgs defines typed parameters for user administration tool calls.
type TLUClawUserOpArgs struct {
	Action      string `json:"action"`
	UserID      string `json:"user_id,omitempty"`
	StudentCode string `json:"student_code,omitempty"`
	Email       string `json:"email,omitempty"`
	RoleID      string `json:"role_id,omitempty"`
	RoleName    string `json:"role_name,omitempty"`
	RoleAction  string `json:"role_action,omitempty"`
	SearchText  string `json:"search_text,omitempty"`
	Keyword     string `json:"keyword,omitempty"`
	Status      string `json:"status,omitempty"`
	Reason      string `json:"reason,omitempty"`
	Page        int    `json:"page,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

// TLUClawInquiryOpArgs defines typed parameters for academic inquiry tool calls.
type TLUClawInquiryOpArgs struct {
	Action    string `json:"action"`
	InquiryID string `json:"inquiry_id,omitempty"`
	Search    string `json:"search,omitempty"`
	Reply     string `json:"reply,omitempty"`
	Answer    string `json:"answer,omitempty"`
	Status    string `json:"status,omitempty"`
	ChunkID   string `json:"chunk_id,omitempty"`
	Page      int    `json:"page,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

// TLUClawSystemOpArgs defines typed parameters for system diagnostic tool calls.
type TLUClawSystemOpArgs struct {
	Action     string `json:"action"`
	ProviderID string `json:"provider_id,omitempty"`
	ModelID    string `json:"model_id,omitempty"`
	ModelType  string `json:"model_type,omitempty"`
	Scope      string `json:"scope,omitempty"`
}

// TLUClawDocumentOpArgs defines typed parameters for document and supersession tool calls.
type TLUClawDocumentOpArgs struct {
	Action           string `json:"action"`
	DocumentID       string `json:"document_id,omitempty"`
	Domain           string `json:"domain,omitempty"`
	ApplicableCohort string `json:"applicable_cohort,omitempty"`
	DocumentTitle    string `json:"document_title,omitempty"`
	DocumentType     string `json:"document_type,omitempty"`
	AcademicYear     string `json:"academic_year,omitempty"`
	DocumentContent  string `json:"document_content,omitempty"`
	Status           string `json:"status,omitempty"`
	Reason           string `json:"reason,omitempty"`
	Query            string `json:"query,omitempty"`
	NodeID           string `json:"node_id,omitempty"`
	Page             int    `json:"page,omitempty"`
	Limit            int    `json:"limit,omitempty"`
}

// TLUClawAuditOpArgs defines typed parameters for audit and rollback tool calls.
type TLUClawAuditOpArgs struct {
	Action     string `json:"action"`
	SnapshotID string `json:"snapshot_id,omitempty"`
	BatchID    string `json:"batch_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Page       int    `json:"page,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// TLUClawMemoryOpArgs defines typed parameters for administrative memory tool calls.
type TLUClawMemoryOpArgs struct {
	Action      string `json:"action"`
	Key         string `json:"key,omitempty"`
	Value       string `json:"value,omitempty"`
	Category    string `json:"category,omitempty"`
	MemoryKey   string `json:"memory_key,omitempty"`
	MemoryValue string `json:"memory_value,omitempty"`
	TargetKey   string `json:"target_key,omitempty"`
	Page        int    `json:"page,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

// TLUClawNotificationOpArgs defines typed parameters for notification tool calls.
type TLUClawNotificationOpArgs struct {
	Action           string `json:"action"`
	RecipientUserID  string `json:"recipient_user_id,omitempty"`
	UserID           string `json:"user_id,omitempty"`
	Title            string `json:"title,omitempty"`
	Content          string `json:"content,omitempty"`
	TargetAudience   string `json:"target_audience,omitempty"`
	TargetType       string `json:"target_type,omitempty"`
	StudentCode      string `json:"student_code,omitempty"`
	Cohort           string `json:"cohort,omitempty"`
	Type             string `json:"type,omitempty"`
	NotificationType string `json:"notification_type,omitempty"`
	Page             int    `json:"page,omitempty"`
	Limit            int    `json:"limit,omitempty"`
}

// TLUClawJobOpArgs defines typed parameters for async job tool calls.
type TLUClawJobOpArgs struct {
	Action   string `json:"action"`
	JobID    string `json:"job_id,omitempty"`
	TaskType string `json:"task_type,omitempty"`
	Payload  string `json:"payload,omitempty"`
	Page     int    `json:"page,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// IncomingCourseRecord defines an incoming course record within a curriculum update.
type IncomingCourseRecord struct {
	CourseCode        string  `json:"course_code"`
	CourseName        string  `json:"course_name"`
	Credits           float64 `json:"credits"`
	Weight            float64 `json:"weight"`
	Hours             string  `json:"hours"`
	KnowledgeBlock    string  `json:"knowledge_block"`
	PrerequisitesText string  `json:"prerequisites_text"`
	SuggestedSemester int64   `json:"suggested_semester"`
	IsElearning       int64   `json:"is_elearning"`
}

// TLUClawAcademicStructureSnapshotState encapsulates state payload for cohort and major snapshots.
type TLUClawAcademicStructureSnapshotState struct {
	Cohort       string   `json:"cohort,omitempty"`
	AcademicYear string   `json:"academic_year,omitempty"`
	MajorCode    string   `json:"major_code,omitempty"`
	MajorNameVi  string   `json:"major_name_vi,omitempty"`
	FacultyCode  string   `json:"faculty_code,omitempty"`
	Aliases      []string `json:"aliases,omitempty"`
	IsUpdate     bool     `json:"is_update,omitempty"`
}

// TLUClawCurriculumSnapshotState encapsulates state payload for curriculum course updates.
type TLUClawCurriculumSnapshotState struct {
	Cohort          string                 `json:"cohort"`
	MajorCode       string                 `json:"major_code"`
	IncomingCourses []IncomingCourseRecord `json:"incoming_courses"`
}

// TLUClawDocumentSnapshotState encapsulates state payload for document ingestion.
type TLUClawDocumentSnapshotState struct {
	NewDocID         string   `json:"new_doc_id"`
	Title            string   `json:"title"`
	DocType          string   `json:"doc_type"`
	Domain           string   `json:"domain"`
	Cohort           string   `json:"cohort"`
	AcademicYear     string   `json:"academic_year"`
	Content          string   `json:"content"`
	SupersededDocIDs []string `json:"superseded_doc_ids"`
	PlanJSON         string   `json:"plan_json,omitempty"`
}

// TLUClawSubstitutionSnapshotState encapsulates state payload for substitution snapshots.
type TLUClawSubstitutionSnapshotState struct {
	ID                    string  `json:"id"`
	OriginalCourseCode    string  `json:"original_course_code"`
	OriginalCourseName    string  `json:"original_course_name"`
	ReplacementCourseCode string  `json:"replacement_course_code"`
	ReplacementCourseName string  `json:"replacement_course_name"`
	MinCohort             int64   `json:"min_cohort"`
	MaxCohort             int64   `json:"max_cohort"`
	ScopeMajor            string  `json:"scope_major"`
	IsBidirectional       int64   `json:"is_bidirectional"`
	CreditsOriginal       float64 `json:"credits_original"`
	CreditsReplacement    float64 `json:"credits_replacement"`
	RuleNote              string  `json:"rule_note"`
}

// TLUClawPrerequisiteSnapshotState encapsulates state payload for prerequisite snapshots.
type TLUClawPrerequisiteSnapshotState struct {
	ID                     string `json:"id"`
	CurriculumCourseID     string `json:"curriculum_course_id"`
	Cohort                 string `json:"cohort"`
	MajorName              string `json:"major_name"`
	CourseCode             string `json:"course_code"`
	PrerequisiteCourseCode string `json:"prerequisite_course_code"`
	MinGrade               string `json:"min_grade"`
	ConditionType          string `json:"condition_type"`
	RawExpression          string `json:"raw_expression"`
}

// TLUClawAdvisorNoteSnapshotState encapsulates state payload for advisor note updates.
type TLUClawAdvisorNoteSnapshotState struct {
	UserID      string `json:"user_id"`
	StudentCode string `json:"student_code"`
	AdvisorNote string `json:"advisor_note"`
}

// TLUClawUserStatusSnapshotState encapsulates state payload for user status changes.
type TLUClawUserStatusSnapshotState struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// TLUClawUserRoleChangeSnapshotState encapsulates state payload for user role assignments.
type TLUClawUserRoleChangeSnapshotState struct {
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	RoleAction string `json:"role_action"`
	RoleID     string `json:"role_id"`
	RoleName   string `json:"role_name"`
}

// TLUClawInquiryReplySnapshotState encapsulates state payload for inquiry responses.
type TLUClawInquiryReplySnapshotState struct {
	InquiryID string `json:"inquiry_id"`
	Reply     string `json:"reply"`
	ChunkID   string `json:"chunk_id"`
}

// TLUClawLLMConfigSnapshotState encapsulates state payload for LLM default model changes.
type TLUClawLLMConfigSnapshotState struct {
	ModelID   string `json:"model_id"`
	ModelType string `json:"model_type"`
}

// TLUClawDocumentStatusSnapshotState encapsulates state payload for document status changes.
type TLUClawDocumentStatusSnapshotState struct {
	DocumentID string `json:"document_id"`
	Status     string `json:"status"`
	Reason     string `json:"reason"`
}

// TLUClawNotificationItem represents a dispatched notification payload.
type TLUClawNotificationItem struct {
	ID      string `json:"id"`
	UserID  string `json:"user_id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

// TLUClawNotificationDispatchSnapshotState encapsulates state payload for notification dispatches.
type TLUClawNotificationDispatchSnapshotState struct {
	Notifications []TLUClawNotificationItem `json:"notifications"`
}

// TLUClawJobTriggerSnapshotState encapsulates state payload for manual background job triggers.
type TLUClawJobTriggerSnapshotState struct {
	TaskType string `json:"task_type"`
	Payload  string `json:"payload"`
}

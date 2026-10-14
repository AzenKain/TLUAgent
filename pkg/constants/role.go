package constants

type RoleType string

const (
	RoleTypeAdmin   RoleType = "ADMIN"
	RoleTypeAdvisor RoleType = "ADVISOR"
	RoleTypeStudent RoleType = "STUDENT"
	RoleTypeGuest   RoleType = "GUEST"
	RoleTypeBanned  RoleType = "BANNED"
)

func (r RoleType) String() string {
	return string(r)
}

const (
	// Regulation permissions
	PermRegulationRead  = "regulation.read"
	PermRegulationWrite = "regulation.write"

	// Curriculum & Degree Audit permissions
	PermCurriculumRead   = "curriculum.read"
	PermCurriculumManage = "curriculum.manage"

	// Transcript & Academic Course Grades permissions
	PermTranscriptRead   = "transcript.read"
	PermTranscriptManage = "transcript.manage"

	// Crawler permissions
	PermCrawlerTrigger = "crawler.trigger"

	// Audit & Logs permissions
	PermAuditRead = "audit.read"

	// Chat & Advisory permissions
	PermChatAsk     = "chat.ask"
	PermChatPersist = "chat.persist"
	PermChatManage  = "chat.manage"

	// Administrative & System permissions
	PermAdminAccess   = "admin.access"
	PermUserManage    = "user.manage"
	PermRoleManage    = "role.manage"
	PermSettingManage = "setting.manage"
	PermLLMManage     = "llm.manage"
	PermJobRead       = "job.read"
	PermJobManage     = "job.manage"

	// RAG & Knowledge Graph permissions
	PermRAGRead   = "rag.read"
	PermRAGManage = "rag.manage"

	// Inquiry & Self-Evolution permissions
	PermInquiryRead   = "inquiry.read"
	PermInquiryCreate = "inquiry.create"
	PermInquiryAnswer = "inquiry.answer"
	PermInquiryManage = "inquiry.manage"

	// Admin TLUClaw permissions
	PermTLUClawRead   = "tluclaw.read"
	PermTLUClawManage = "tluclaw.manage"
)

type PermissionInfo struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

type PermissionCategory struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Permissions []PermissionInfo `json:"permissions"`
}

func GetDefaultPermissionsForRole(role RoleType) []string {
	switch role {
	case RoleTypeGuest:
		return []string{
			PermRegulationRead,
			PermChatAsk,
			PermRAGRead,
		}
	case RoleTypeStudent:
		return []string{
			PermRegulationRead,
			PermCurriculumRead,
			PermTranscriptRead,
			PermTranscriptManage,
			PermChatAsk,
			PermChatPersist,
			PermRAGRead,
			PermInquiryRead,
			PermInquiryCreate,
		}
	case RoleTypeAdvisor:
		return []string{
			PermRegulationRead,
			PermRegulationWrite,
			PermCurriculumRead,
			PermCurriculumManage,
			PermTranscriptRead,
			PermTranscriptManage,
			PermChatAsk,
			PermChatPersist,
			PermChatManage,
			PermAuditRead,
			PermRAGRead,
			PermInquiryRead,
			PermInquiryCreate,
			PermInquiryAnswer,
		}
	case RoleTypeAdmin:
		return []string{
			PermRegulationRead,
			PermRegulationWrite,
			PermCurriculumRead,
			PermCurriculumManage,
			PermTranscriptRead,
			PermTranscriptManage,
			PermCrawlerTrigger,
			PermAuditRead,
			PermChatAsk,
			PermChatPersist,
			PermChatManage,
			PermAdminAccess,
			PermUserManage,
			PermRoleManage,
			PermSettingManage,
			PermLLMManage,
			PermRAGRead,
			PermRAGManage,
			PermInquiryRead,
			PermInquiryCreate,
			PermInquiryAnswer,
			PermInquiryManage,
			PermTLUClawRead,
			PermTLUClawManage,
		}
	default:
		return []string{}
	}
}

package constants

import "strings"

const (
	DocStatusActive           = "ACTIVE"
	DocStatusSuperseded       = "SUPERSEDED"
	DocStatusDraft            = "DRAFT"
	DocStatusArchived         = "ARCHIVED"
	DocStatusPartiallyAmended = "PARTIALLY_AMENDED"

	DocTypeNotice     = "THONG_BAO"
	DocTypeNews       = "TIN_TUC"
	DocTypeRegulation = "QUY_CHE"
	DocTypeRule       = "QUY_DINH"
	DocTypeDecision   = "QUYET_DINH"
	DocTypeScheme     = "DE_AN"
	DocTypePlan       = "KE_HOACH"
	DocTypeGuideline  = "HUONG_DAN"
	DocTypeCurriculum = "CTDT"
	DocTypeHandbook   = "SO_TAY"
	DocTypeStatic     = "STATIC"
	DocTypeAdvisoryQA = "ADVISORY_QA"

	DomainTraining   = "DAO_TAO"
	DomainAcademic   = "HOC_VU"
	DomainRegulation = "QUY_CHE"
	DomainTuition    = "HOC_PHI"
	DomainAdmission  = "TUYEN_SINH"
	DomainBenchmark  = "CHUAN_DAU_RA"
	DomainAdvising   = "CO_VAN_HOC_TAP"
	DomainNews       = "TIN_TUC"
	DomainGeneral    = "CHUNG"
	DomainAdvisoryQA = "ADVISORY_QA"

	ValidityTierEmergency          = "EMERGENCY_OVERRIDE"
	ValidityTierCurrentOperational = "CURRENT_OPERATIONAL"
	ValidityTierCorePermanent      = "CORE_PERMANENT"
	ValidityTierHistoricalArchive  = "HISTORICAL_ARCHIVE"
	ValidityTierSemesterDynamic    = "SEMESTER_DYNAMIC"
	ValidityTierOfficial           = "OFFICIAL"

	KGNodeTypeDocument = "DOCUMENT"
	KGNodeTypeTopic    = "TOPIC"
	KGNodeTypeCohort   = "COHORT"
	KGNodeTypeRule     = "RULE"
	KGNodeTypeEntity   = "ENTITY"

	KGRelationSupersedes      = "SUPERSEDES"
	KGRelationCoversTopic     = "COVERS_TOPIC"
	KGRelationAppliesToCohort = "APPLIES_TO_COHORT"
	KGRelationAmends          = "AMENDS"
	KGRelationPartiallyAmends = "PARTIALLY_AMENDS"
	KGRelationReferences      = "REFERENCES"
	KGRelationBelongsTo       = "BELONGS_TO"

	ChunkTypeContent   = "CONTENT_CHUNK"
	ChunkTypeRule      = "RULE_CHUNK"
	ChunkTypeTeacherQA = "TEACHER_QA"

	RuleTypeAdminOverride       = "ADMIN_OVERRIDE"
	RuleTypeDocumentReplacement = "DOCUMENT_REPLACEMENT"
	RuleTypePartialAmendment    = "PARTIAL_AMENDMENT"
	RuleTypeTeacherQAExpired    = "TEACHER_QA_EXPIRED"

	ActionNone            = "NONE"
	ActionFullSupersede   = "FULL_SUPERSEDE"
	ActionPartialAmend    = "PARTIAL_AMEND"
	ActionCoexistByCohort = "COEXIST_BY_COHORT"

	RiskLevelLow    = "LOW"
	RiskLevelMedium = "MEDIUM"
	RiskLevelHigh   = "HIGH"

	InquiryStatusPending  = "PENDING"
	InquiryStatusAnswered = "ANSWERED"
	InquiryStatusExpired  = "EXPIRED"

	NotifTypeInquiryAnswered    = "INQUIRY_ANSWERED"
	NotifTypeInquiryExpired     = "INQUIRY_EXPIRED"
	NotifTypeSupersededProposed = "SUPERSEDED_PROPOSED"
	NotifTypeCurriculumUpdated  = "CURRICULUM_UPDATED"
	NotifTypeJobFailure         = "JOB_FAILURE_ALERT"

	DocCohortAll = "ALL"

	DocIDTeacherQA   = "doc_teacher_qa"
	DocCodeTeacherQA = "QA-ADVISORY-TLU"

	PrefixDocNode     = "doc:"
	PrefixTopicNode   = "topic:"
	PrefixCohortNode  = "cohort:"
	PrefixChunk       = "chk_"
	PrefixRule        = "rule_"
	PrefixQAChunk     = "chk_qa_"
	PrefixEdgeTopic   = "edge:topic:"
	PrefixEdgeCohort  = "edge:cohort:"
	PrefixEdgeSuper   = "edge:super:"
	PrefixEdgeAmend   = "edge:amend:"
	PrefixEdgeSup     = "edge_sup_"
	PrefixDocInternal = "DOC_NB_"
	PrefixDocPublic   = "DOC_"

	CachePrefixRAGSearch     = "rag_search:*"
	CachePrefixRAGCandidates = "rag_candidates*"
)

// MinRetrievableCharCount keeps stub documents (title-only notices) out of retrieval results.
const MinRetrievableCharCount = 100

var defaultValidityTiers = map[string]string{
	DocTypeRegulation: ValidityTierOfficial,
	DocTypeRule:       ValidityTierOfficial,
	DocTypeDecision:   ValidityTierOfficial,
	DocTypeGuideline:  ValidityTierOfficial,
	DocTypeHandbook:   ValidityTierOfficial,
	DocTypeCurriculum: ValidityTierOfficial,
	DocTypePlan:       ValidityTierOfficial,
	DocTypeScheme:     ValidityTierOfficial,
	DocTypeNotice:     ValidityTierSemesterDynamic,
	DocTypeNews:       ValidityTierCurrentOperational,
	DocTypeStatic:     ValidityTierCurrentOperational,
}

func DefaultValidityTier(docType string) string {
	return defaultValidityTiers[docType]
}

func DeriveEffectiveFromAcademicYear(academicYear string) string {
	year := strings.TrimSpace(academicYear)
	if len(year) < 4 {
		return ""
	}
	for _, r := range year[:4] {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return year[:4] + "-09-01"
}

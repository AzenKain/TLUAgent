package request

// RAGSearchRequest encapsulates HTTP request parameters for hybrid search.
type RAGSearchRequest struct {
	Query         string  `json:"query" validate:"required,min=1,max=4000"`
	StudentCohort string  `json:"student_cohort,omitempty" validate:"omitempty,max=50"`
	Domain        string  `json:"domain,omitempty" validate:"omitempty,max=100"`
	TopK          int     `json:"top_k,omitempty" validate:"omitempty,min=1,max=50"`
	MinScore      float32 `json:"min_score,omitempty" validate:"omitempty,gte=0,lte=1"`
}

// UpdateDocumentStatusRequest encapsulates input for document status modifications.
type UpdateDocumentStatusRequest struct {
	Status         string `json:"status" validate:"required,oneof=ACTIVE SUPERSEDED DEPRECATED ARCHIVED"`
	SupersededByID string `json:"superseded_by_id,omitempty" validate:"omitempty,max=128"`
	Reason         string `json:"reason,omitempty" validate:"omitempty,max=1000"`
}

// UpdateDocumentCohortRequest encapsulates the cohort scope an administrator sets on a document.
type UpdateDocumentCohortRequest struct {
	ApplicableCohort string `json:"applicable_cohort" validate:"required,max=200"`
}

// RevertSupersessionRequest identifies the audited pair an administrator wants to undo.
type RevertSupersessionRequest struct {
	ActiveDocID     string `json:"active_doc_id" validate:"required,max=128"`
	SupersededDocID string `json:"superseded_doc_id" validate:"required,max=128"`
}

// KGEdgeRequest encapsulates input for creating or updating a knowledge graph relationship.
type KGEdgeRequest struct {
	ID           string  `json:"id,omitempty" validate:"omitempty,max=128"`
	SourceNodeID string  `json:"source_node_id" validate:"required,max=256"`
	TargetNodeID string  `json:"target_node_id" validate:"required,max=256"`
	RelationType string  `json:"relation_type" validate:"required,max=100"`
	Weight       float64 `json:"weight,omitempty" validate:"omitempty,gte=0,lte=100"`
	Status       string  `json:"status,omitempty" validate:"omitempty,max=50"`
}

// ListDocumentsDto encapsulates query parameters for document filtering.
type ListDocumentsDto struct {
	PaginationDto
	Domain string `json:"domain,omitempty" query:"domain" validate:"omitempty,max=100"`
	Status string `json:"status,omitempty" query:"status" validate:"omitempty,max=50"`
	Cohort string `json:"cohort,omitempty" query:"cohort" validate:"omitempty,max=50"`
	Search string `json:"search,omitempty" query:"search" validate:"omitempty,max=200"`
}

// ListAuditLogsDto encapsulates query parameters for supersession audit logs.
type ListAuditLogsDto struct {
	PaginationDto
}

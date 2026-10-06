package request

// RAGSearchRequest encapsulates HTTP request parameters for hybrid search.
type RAGSearchRequest struct {
	Query         string  `json:"query" validate:"required,min=1,max=4000"`
	StudentCohort string  `json:"student_cohort,omitempty"`
	Domain        string  `json:"domain,omitempty"`
	TopK          int     `json:"top_k,omitempty" validate:"omitempty,min=1,max=50"`
	MinScore      float32 `json:"min_score,omitempty"`
}

// UpdateDocumentStatusRequest encapsulates input for document status modifications.
type UpdateDocumentStatusRequest struct {
	Status         string `json:"status" validate:"required,oneof=ACTIVE SUPERSEDED DEPRECATED ARCHIVED"`
	SupersededByID string `json:"superseded_by_id,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

// KGEdgeRequest encapsulates input for creating or updating a knowledge graph relationship.
type KGEdgeRequest struct {
	ID           string  `json:"id,omitempty"`
	SourceNodeID string  `json:"source_node_id" validate:"required"`
	TargetNodeID string  `json:"target_node_id" validate:"required"`
	RelationType string  `json:"relation_type" validate:"required"`
	Weight       float64 `json:"weight,omitempty"`
	Status       string  `json:"status,omitempty"`
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

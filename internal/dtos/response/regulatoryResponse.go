package response

// RegulatorySearchResultResponse encapsulates documents and chunks found during FTS regulation search.
type RegulatorySearchResultResponse struct {
	Documents []*DocumentResponse      `json:"documents"`
	Chunks    []*DocumentChunkResponse `json:"chunks"`
}

// RegulatoryInspectDocumentArticlesResponse encapsulates article breakdowns or chunks of an inspected document.
type RegulatoryInspectDocumentArticlesResponse struct {
	DocumentID string                   `json:"document_id"`
	Rules      []*DocumentRuleResponse  `json:"rules,omitempty"`
	Chunks     []*DocumentChunkResponse `json:"chunks,omitempty"`
}

// RegulatorySearchKnowledgeGraphResponse encapsulates matched Knowledge Graph edges.
type RegulatorySearchKnowledgeGraphResponse struct {
	MatchedEdges []*KGEdgeResponse `json:"matched_edges"`
}

// RegulatoryInspectCohortApplicabilityResponse encapsulates cohort comparison between two documents.
type RegulatoryInspectCohortApplicabilityResponse struct {
	Doc1       *DocumentResponse `json:"doc_1"`
	Doc2       *DocumentResponse `json:"doc_2"`
	IsDisjoint bool              `json:"is_disjoint"`
}

// RegulatoryProposeStatusResponse represents status acknowledgement of a supersession proposal.
type RegulatoryProposeStatusResponse struct {
	Status string `json:"status"`
}

// AuditInquiryOutdatednessResponse represents the structured LLM audit outcome for advisory inquiries.
type AuditInquiryOutdatednessResponse struct {
	IsOutdated bool   `json:"is_outdated"`
	Reason     string `json:"reason"`
}

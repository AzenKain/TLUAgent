package response

// DocumentResponse mirrors the serialized shape of a knowledge base document.
type DocumentResponse struct {
	ID               string `json:"id"`
	DocCode          string `json:"doc_code"`
	Type             string `json:"type"`
	Title            string `json:"title"`
	Domain           string `json:"domain"`
	ValidityTier     string `json:"validity_tier"`
	Status           string `json:"status"`
	SupersededByID   string `json:"superseded_by_id,omitempty"`
	DepartmentCode   string `json:"department_code"`
	PublishDate      string `json:"publish_date"`
	EffectiveFrom    string `json:"effective_from,omitempty"`
	EffectiveTo      string `json:"effective_to,omitempty"`
	AcademicYear     string `json:"academic_year"`
	Semester         string `json:"semester"`
	ApplicableCohort string `json:"applicable_cohort"`
	PriorityLevel    int    `json:"priority_level"`
	FilePath         string `json:"file_path"`
	CharCount        int    `json:"char_count"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// DocumentRuleResponse mirrors the serialized shape of a structured document rule.
type DocumentRuleResponse struct {
	ID             string `json:"id"`
	DocumentID     string `json:"document_id"`
	RuleIndex      int    `json:"rule_index"`
	Chapter        string `json:"chapter,omitempty"`
	ArticleNumber  string `json:"article_number,omitempty"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	Status         string `json:"status"`
	SupersededByID string `json:"superseded_by_id,omitempty"`
	CreatedAt      string `json:"created_at"`
}

// DocumentChunkResponse mirrors the serialized shape of a searchable document chunk.
type DocumentChunkResponse struct {
	ID               string    `json:"id"`
	DocumentID       string    `json:"document_id"`
	RuleID           string    `json:"rule_id,omitempty"`
	ChunkIndex       int       `json:"chunk_index"`
	ChunkType        string    `json:"chunk_type"`
	Content          string    `json:"content"`
	Status           string    `json:"status"`
	Domain           string    `json:"domain"`
	ApplicableCohort string    `json:"applicable_cohort"`
	AcademicYear     string    `json:"academic_year"`
	PriorityLevel    int       `json:"priority_level"`
	Embedding        []float32 `json:"embedding,omitempty"`
	CreatedAt        string    `json:"created_at"`
}

// KGNodeResponse mirrors the serialized shape of a knowledge graph entity.
type KGNodeResponse struct {
	ID        string `json:"id"`
	NodeType  string `json:"node_type"`
	Name      string `json:"name"`
	Code      string `json:"code,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// KGEdgeResponse mirrors the serialized shape of a knowledge graph relationship.
type KGEdgeResponse struct {
	ID           string  `json:"id"`
	SourceNodeID string  `json:"source_node_id"`
	TargetNodeID string  `json:"target_node_id"`
	RelationType string  `json:"relation_type"`
	Weight       float64 `json:"weight"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"created_at"`
}

// SupersessionAuditLogResponse mirrors the serialized shape of a document supersession audit record.
type SupersessionAuditLogResponse struct {
	ID              int64  `json:"id"`
	ActiveDocID     string `json:"active_doc_id"`
	SupersededDocID string `json:"superseded_doc_id"`
	RuleType        string `json:"rule_type"`
	Reason          string `json:"reason"`
	CreatedAt       string `json:"created_at"`
}

// DocumentDetailResponse mirrors the serialized shape of a document with its chunks and relationships.
type DocumentDetailResponse struct {
	Document     *DocumentResponse               `json:"document"`
	SupersededBy *DocumentResponse               `json:"superseded_by,omitempty"`
	Supersedes   []*DocumentResponse             `json:"supersedes,omitempty"`
	Chunks       []*DocumentChunkResponse        `json:"chunks"`
	AuditLogs    []*SupersessionAuditLogResponse `json:"audit_logs"`
	FullContent  string                          `json:"full_content"`
}

// KnowledgeGraphDataResponse mirrors the serialized shape of the graph nodes and edges.
type KnowledgeGraphDataResponse struct {
	Nodes []*KGNodeResponse `json:"nodes"`
	Edges []*KGEdgeResponse `json:"edges"`
}

// KnowledgeStatsResponse mirrors the serialized shape of the RAG knowledge graph metrics.
type KnowledgeStatsResponse struct {
	TotalDocuments            int `json:"total_documents"`
	ActiveDocuments           int `json:"active_documents"`
	SupersededDocuments       int `json:"superseded_documents"`
	PartiallyAmendedDocuments int `json:"partially_amended_documents"`
	ArchivedDocuments         int `json:"archived_documents"`
	TotalChunks               int `json:"total_chunks"`
	TotalNodes                int `json:"total_nodes"`
	TotalEdges                int `json:"total_edges"`
}

// RAGSearchResultItemResponse mirrors the serialized shape of a ranked chunk search result.
type RAGSearchResultItemResponse struct {
	ChunkID          string  `json:"chunk_id"`
	DocumentID       string  `json:"document_id"`
	DocTitle         string  `json:"doc_title"`
	DocType          string  `json:"doc_type"`
	Domain           string  `json:"domain"`
	ApplicableCohort string  `json:"applicable_cohort"`
	Content          string  `json:"content"`
	VectorSim        float32 `json:"vector_sim"`
	BM25Score        float32 `json:"bm25_score"`
	PriorityLevel    int     `json:"priority_level"`
	FinalScore       float32 `json:"final_score"`
}

// DocumentListResponse is a paginated envelope of knowledge documents.
type DocumentListResponse struct {
	Documents []*DocumentResponse `json:"documents"`
	Total     int                 `json:"total"`
	Page      int                 `json:"page"`
	Limit     int                 `json:"limit"`
}

// AuditLogListResponse is a paginated envelope of document supersession audit logs.
type AuditLogListResponse struct {
	Logs  []*SupersessionAuditLogResponse `json:"logs"`
	Total int                             `json:"total"`
	Page  int                             `json:"page"`
	Limit int                             `json:"limit"`
}

// IngestSyncResponse reports how many documents were ingested during an index sync.
type IngestSyncResponse struct {
	IngestedDocuments int `json:"ingested_documents"`
}

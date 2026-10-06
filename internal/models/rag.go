package models

import (
	"database/sql"
	"encoding/binary"
	"math"
	"time"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/pkg/convert"
)

// Document represents an official institutional document in the knowledge base.
type Document struct {
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

// FromSqlc converts a sqlc Document row to a Document model pointer.
func (d *Document) FromSqlc(row sqlc.Document) *Document {
	d.ID = row.ID
	d.DocCode = convert.NullStringToString(row.DocCode)
	d.Type = row.Type
	d.Title = row.Title
	d.Domain = row.Domain
	d.ValidityTier = row.ValidityTier
	d.Status = row.Status
	d.SupersededByID = convert.NullStringToString(row.SupersededByID)
	d.DepartmentCode = row.DepartmentCode
	d.PublishDate = row.PublishDate
	d.EffectiveFrom = convert.NullStringToString(row.EffectiveFrom)
	d.EffectiveTo = convert.NullStringToString(row.EffectiveTo)
	d.AcademicYear = row.AcademicYear
	d.Semester = row.Semester
	d.ApplicableCohort = row.ApplicableCohort
	d.PriorityLevel = int(row.PriorityLevel)
	d.FilePath = row.FilePath
	d.CharCount = int(row.CharCount)
	d.CreatedAt = row.CreatedAt.Format(time.RFC3339)
	d.UpdatedAt = row.UpdatedAt.Format(time.RFC3339)
	return d
}

// ToSqlcUpsertParams converts a Document model to sqlc UpsertDocumentParams.
func (d *Document) ToSqlcUpsertParams() sqlc.UpsertDocumentParams {
	now := time.Now().UTC()
	return sqlc.UpsertDocumentParams{
		ID:               d.ID,
		DocCode:          sql.NullString{String: d.DocCode, Valid: d.DocCode != ""},
		Type:             d.Type,
		Title:            d.Title,
		Domain:           d.Domain,
		ValidityTier:     d.ValidityTier,
		Status:           d.Status,
		SupersededByID:   sql.NullString{String: d.SupersededByID, Valid: d.SupersededByID != ""},
		DepartmentCode:   d.DepartmentCode,
		PublishDate:      d.PublishDate,
		EffectiveFrom:    sql.NullString{String: d.EffectiveFrom, Valid: d.EffectiveFrom != ""},
		EffectiveTo:      sql.NullString{String: d.EffectiveTo, Valid: d.EffectiveTo != ""},
		AcademicYear:     d.AcademicYear,
		Semester:         d.Semester,
		ApplicableCohort: d.ApplicableCohort,
		PriorityLevel:    int64(d.PriorityLevel),
		FilePath:         d.FilePath,
		CharCount:        int64(d.CharCount),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// DocumentRule represents a structured section or article inside a document.
type DocumentRule struct {
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

// DocumentChunk represents a searchable text fragment with dense vector embedding.
type DocumentChunk struct {
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

// FromSqlc converts a sqlc DocumentChunk row to a DocumentChunk model pointer.
func (c *DocumentChunk) FromSqlc(row sqlc.DocumentChunk) *DocumentChunk {
	c.ID = row.ID
	c.DocumentID = row.DocumentID
	c.RuleID = convert.NullStringToString(row.RuleID)
	c.ChunkIndex = int(row.ChunkIndex)
	c.ChunkType = row.ChunkType
	c.Content = row.Content
	c.Status = row.Status
	c.Domain = row.Domain
	c.ApplicableCohort = row.ApplicableCohort
	c.AcademicYear = row.AcademicYear
	c.PriorityLevel = int(row.PriorityLevel)
	c.CreatedAt = row.CreatedAt.Format(time.RFC3339)
	if len(row.Embedding) > 0 {
		c.Embedding = DecodeBlobVector(row.Embedding)
	}
	return c
}

// FromCandidateRow converts a sqlc QueryActiveChunkCandidatesRow to a DocumentChunk model pointer.
func (c *DocumentChunk) FromCandidateRow(row sqlc.QueryActiveChunkCandidatesRow) *DocumentChunk {
	c.ID = row.ID
	c.DocumentID = row.DocumentID
	c.RuleID = convert.NullStringToString(row.RuleID)
	c.ChunkIndex = int(row.ChunkIndex)
	c.ChunkType = row.ChunkType
	c.Content = row.Content
	c.Status = row.Status
	c.Domain = row.Domain
	c.ApplicableCohort = row.ApplicableCohort
	c.AcademicYear = row.AcademicYear
	c.PriorityLevel = int(row.PriorityLevel)
	c.CreatedAt = row.CreatedAt.Format(time.RFC3339)
	if len(row.Embedding) > 0 {
		c.Embedding = DecodeBlobVector(row.Embedding)
	}
	return c
}

// ToSqlcUpsertParams converts a DocumentChunk model to sqlc UpsertDocumentChunkParams.
func (c *DocumentChunk) ToSqlcUpsertParams() sqlc.UpsertDocumentChunkParams {
	now := time.Now().UTC()
	var blob []byte
	if len(c.Embedding) > 0 {
		blob = EncodeVectorBlob(c.Embedding)
	}
	return sqlc.UpsertDocumentChunkParams{
		ID:               c.ID,
		DocumentID:       c.DocumentID,
		RuleID:           sql.NullString{String: c.RuleID, Valid: c.RuleID != ""},
		ChunkIndex:       int64(c.ChunkIndex),
		ChunkType:        c.ChunkType,
		Content:          c.Content,
		Status:           c.Status,
		Domain:           c.Domain,
		ApplicableCohort: c.ApplicableCohort,
		AcademicYear:     c.AcademicYear,
		PriorityLevel:    int64(c.PriorityLevel),
		Embedding:        blob,
		CreatedAt:        now,
	}
}

// KGNode represents an entity in the institutional knowledge graph.
type KGNode struct {
	ID        string `json:"id"`
	NodeType  string `json:"node_type"`
	Name      string `json:"name"`
	Code      string `json:"code,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// FromSqlc converts a sqlc KgNode row to a KGNode model pointer.
func (n *KGNode) FromSqlc(row sqlc.KgNode) *KGNode {
	n.ID = row.ID
	n.NodeType = row.NodeType
	n.Name = row.Name
	n.Code = convert.NullStringToString(row.Code)
	n.Status = row.Status
	n.CreatedAt = row.CreatedAt.Format(time.RFC3339)
	return n
}

// KGEdge represents a directional relationship between two graph nodes.
type KGEdge struct {
	ID           string  `json:"id"`
	SourceNodeID string  `json:"source_node_id"`
	TargetNodeID string  `json:"target_node_id"`
	RelationType string  `json:"relation_type"`
	Weight       float64 `json:"weight"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"created_at"`
}

// FromSqlc converts a sqlc KgEdge row to a KGEdge model pointer.
func (e *KGEdge) FromSqlc(row sqlc.KgEdge) *KGEdge {
	e.ID = row.ID
	e.SourceNodeID = row.SourceNodeID
	e.TargetNodeID = row.TargetNodeID
	e.RelationType = row.RelationType
	if row.Weight.Valid {
		e.Weight = row.Weight.Float64
	} else {
		e.Weight = 1.0
	}
	e.Status = row.Status
	e.CreatedAt = row.CreatedAt.Format(time.RFC3339)
	return e
}

// SupersessionAuditLog tracks automated or manual document invalidation events.
type SupersessionAuditLog struct {
	ID              int64  `json:"id"`
	ActiveDocID     string `json:"active_doc_id"`
	SupersededDocID string `json:"superseded_doc_id"`
	RuleType        string `json:"rule_type"`
	Reason          string `json:"reason"`
	CreatedAt       string `json:"created_at"`
}

// FromSqlc converts a sqlc SupersessionAuditLog row to a SupersessionAuditLog model pointer.
func (l *SupersessionAuditLog) FromSqlc(row sqlc.SupersessionAuditLog) *SupersessionAuditLog {
	l.ID = row.ID
	l.ActiveDocID = row.ActiveDocID
	l.SupersededDocID = row.SupersededDocID
	l.RuleType = row.RuleType
	l.Reason = row.Reason
	l.CreatedAt = row.CreatedAt.Format(time.RFC3339)
	return l
}

// DocumentDetail represents comprehensive metadata, chunks, and relationships of a document.
type DocumentDetail struct {
	Document     *Document               `json:"document"`
	SupersededBy *Document               `json:"superseded_by,omitempty"`
	Supersedes   []*Document             `json:"supersedes,omitempty"`
	Chunks       []*DocumentChunk        `json:"chunks"`
	AuditLogs    []*SupersessionAuditLog `json:"audit_logs"`
}

// KnowledgeGraphData encapsulates nodes and edges for graph visualization.
type KnowledgeGraphData struct {
	Nodes []*KGNode `json:"nodes"`
	Edges []*KGEdge `json:"edges"`
}

// KnowledgeStats encapsulates high-level metrics for the RAG knowledge graph.
type KnowledgeStats struct {
	TotalDocuments      int `json:"total_documents"`
	ActiveDocuments     int `json:"active_documents"`
	SupersededDocuments int `json:"superseded_documents"`
	TotalChunks         int `json:"total_chunks"`
	TotalNodes          int `json:"total_nodes"`
	TotalEdges          int `json:"total_edges"`
}

// RAGSearchParams encapsulates query criteria for hybrid retrieval.
type RAGSearchParams struct {
	Query         string  `json:"query"`
	StudentCohort string  `json:"student_cohort"`
	Domain        string  `json:"domain"`
	AcademicYear  string  `json:"academic_year"`
	Semester      string  `json:"semester"`
	TopK          int     `json:"top_k"`
	MinScore      float32 `json:"min_score"`
}

// RAGSearchResultItem represents a ranked chunk result with individual scoring signals.
type RAGSearchResultItem struct {
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

// EncodeVectorBlob serializes a float32 slice into IEEE 754 little-endian binary bytes.
func EncodeVectorBlob(vec []float32) []byte {
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

// DecodeBlobVector deserializes IEEE 754 little-endian binary bytes into a float32 slice.
func DecodeBlobVector(blob []byte) []float32 {
	n := len(blob) / 4
	vec := make([]float32, n)
	for i := 0; i < n; i++ {
		bits := binary.LittleEndian.Uint32(blob[i*4:])
		vec[i] = math.Float32frombits(bits)
	}
	return vec
}

// CosineSimilarity calculates the cosine similarity between two float32 vectors.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA <= 0 || normB <= 0 {
		return 0
	}
	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

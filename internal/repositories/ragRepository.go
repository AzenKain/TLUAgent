package repositories

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
)

// RAGRepository defines database persistence and graph operations for documents, chunks, and knowledge entities.
type RAGRepository interface {
	GetDocumentByID(ctx context.Context, id string) (*models.Document, error)
	GetDocumentsByIDs(ctx context.Context, ids []string) ([]*models.Document, error)
	ListDocuments(ctx context.Context, dto *request.ListDocumentsDto) ([]*models.Document, int, error)
	UpsertDocument(ctx context.Context, doc *models.Document) (*models.Document, error)
	UpdateDocumentStatus(ctx context.Context, id, status, supersededByID string) error
	ListSupersededDocuments(ctx context.Context, docID string) ([]*models.Document, error)

	GetChunkByID(ctx context.Context, id string) (*models.DocumentChunk, error)
	GetChunksByIDs(ctx context.Context, ids []string) ([]*models.DocumentChunk, error)
	GetChunksByDocumentID(ctx context.Context, docID string) ([]*models.DocumentChunk, error)
	UpsertChunk(ctx context.Context, chunk *models.DocumentChunk) (*models.DocumentChunk, error)
	UpsertChunksBatch(ctx context.Context, chunks []*models.DocumentChunk) error
	UpdateChunkStatus(ctx context.Context, id, status string) error
	QueryActiveCandidates(ctx context.Context, cohort, domain string) ([]*models.DocumentChunk, error)

	GetKGNodeByID(ctx context.Context, id string) (*models.KGNode, error)
	ListAllKGNodes(ctx context.Context) ([]*models.KGNode, error)
	UpsertKGNode(ctx context.Context, node *models.KGNode) (*models.KGNode, error)

	GetKGEdgeByID(ctx context.Context, id string) (*models.KGEdge, error)
	ListAllKGEdges(ctx context.Context) ([]*models.KGEdge, error)
	UpsertKGEdge(ctx context.Context, edge *models.KGEdge) (*models.KGEdge, error)
	DeleteKGEdge(ctx context.Context, id string) error
	FindSupersededDocIDs(ctx context.Context) (map[string]bool, error)

	RecordSupersessionLog(ctx context.Context, activeID, supersededID, ruleType, reason string) error
	ListAuditLogsByDocID(ctx context.Context, docID string) ([]*models.SupersessionAuditLog, error)
	ListAllAuditLogs(ctx context.Context, limit, offset int) ([]*models.SupersessionAuditLog, int, error)

	CountStats(ctx context.Context) (docs int, chunks int, nodes int, edges int, err error)
	GetKnowledgeStats(ctx context.Context) (*models.KnowledgeStats, error)

	InvalidateDocumentCache(ctx context.Context, id string)
	InvalidateChunkCache(ctx context.Context, id string)
	WithTx(tx *sql.Tx) RAGRepository
}

type ragRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

// NewRAGRepository creates a new RAG repository instance backed by sqlc and RAM cache.
func NewRAGRepository(db sqlc.DBTX, c cache.Cache) RAGRepository {
	return &ragRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *ragRepository) WithTx(tx *sql.Tx) RAGRepository {
	return &ragRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sfg:  &singleflight.Group{},
	}
}

func (r *ragRepository) GetDocumentByID(ctx context.Context, id string) (*models.Document, error) {
	key := cache.BuildKey("rag_doc", "id", id)
	if r.c != nil && !r.inTx {
		var doc models.Document
		if err := r.c.Get(ctx, key, &doc); err == nil {
			return &doc, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetDocumentByID(ctx, id)
		if err != nil {
			return nil, err
		}
		docPtr := (&models.Document{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, docPtr, constants.NormalCacheDuration)
		}
		return docPtr, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.Document), nil
}

func (r *ragRepository) GetDocumentsByIDs(ctx context.Context, ids []string) ([]*models.Document, error) {
	if len(ids) == 0 {
		return []*models.Document{}, nil
	}

	resultMap := make(map[string]*models.Document, len(ids))
	var missingIDs []string

	for _, id := range ids {
		key := cache.BuildKey("rag_doc", "id", id)
		if r.c != nil && !r.inTx {
			var doc models.Document
			if err := r.c.Get(ctx, key, &doc); err == nil {
				resultMap[id] = &doc
				continue
			}
		}
		missingIDs = append(missingIDs, id)
	}

	if len(missingIDs) > 0 {
		rows, err := r.q.GetDocumentsByIDs(ctx, missingIDs)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			docPtr := (&models.Document{}).FromSqlc(row)
			resultMap[docPtr.ID] = docPtr
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, cache.BuildKey("rag_doc", "id", docPtr.ID), docPtr, constants.NormalCacheDuration)
			}
		}
	}

	out := make([]*models.Document, 0, len(ids))
	for _, id := range ids {
		if doc, ok := resultMap[id]; ok {
			out = append(out, doc)
		}
	}
	return out, nil
}

func (r *ragRepository) ListDocuments(ctx context.Context, dto *request.ListDocumentsDto) ([]*models.Document, int, error) {
	limit := 20
	offset := 0
	var domainArg any = nil
	var statusArg any = nil
	var cohortArg any = nil
	var searchArg any = nil

	if dto != nil {
		limit = dto.GetLimit(20)
		offset = dto.GetOffset(20)
		if dto.Domain != "" {
			domainArg = dto.Domain
		}
		if dto.Status != "" {
			statusArg = dto.Status
		}
		if dto.Cohort != "" {
			cohortArg = dto.Cohort
		}
		if strings.TrimSpace(dto.Search) != "" {
			searchArg = strings.TrimSpace(dto.Search)
		}
	}

	total, err := r.q.CountDocuments(ctx, sqlc.CountDocumentsParams{
		Domain: domainArg,
		Status: statusArg,
		Cohort: cohortArg,
		Search: searchArg,
	})
	if err != nil {
		return nil, 0, err
	}

	ids, err := r.q.SearchDocumentIDs(ctx, sqlc.SearchDocumentIDsParams{
		Domain: domainArg,
		Status: statusArg,
		Cohort: cohortArg,
		Search: searchArg,
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	docs, err := r.GetDocumentsByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	return docs, int(total), nil
}

func (r *ragRepository) UpsertDocument(ctx context.Context, doc *models.Document) (*models.Document, error) {
	params := doc.ToSqlcUpsertParams()
	row, err := r.q.UpsertDocument(ctx, params)
	if err != nil {
		return nil, err
	}
	r.InvalidateDocumentCache(ctx, doc.ID)
	return (&models.Document{}).FromSqlc(row), nil
}

func (r *ragRepository) UpdateDocumentStatus(ctx context.Context, id, status, supersededByID string) error {
	now := time.Now().UTC()
	var supArg sql.NullString
	if supersededByID != "" {
		supArg = sql.NullString{String: supersededByID, Valid: true}
	}
	err := r.q.UpdateDocumentStatus(ctx, sqlc.UpdateDocumentStatusParams{
		Status:         status,
		SupersededByID: supArg,
		UpdatedAt:      now,
		ID:             id,
	})
	if err == nil {
		r.InvalidateDocumentCache(ctx, id)
		if supersededByID != "" {
			r.InvalidateDocumentCache(ctx, supersededByID)
		}
	}
	return err
}

func (r *ragRepository) ListSupersededDocuments(ctx context.Context, docID string) ([]*models.Document, error) {
	key := cache.BuildKey("rag_superseded_docs", "doc_id", docID)
	if r.c != nil && !r.inTx {
		var docs []*models.Document
		if err := r.c.Get(ctx, key, &docs); err == nil {
			return docs, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.ListSupersededDocuments(ctx, sql.NullString{String: docID, Valid: true})
		if err != nil {
			return nil, err
		}
		docs := make([]*models.Document, 0, len(rows))
		for _, row := range rows {
			docs = append(docs, (&models.Document{}).FromSqlc(row))
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, docs, constants.NormalCacheDuration)
		}
		return docs, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.Document), nil
}

func (r *ragRepository) GetChunkByID(ctx context.Context, id string) (*models.DocumentChunk, error) {
	key := cache.BuildKey("rag_chunk", "id", id)
	if r.c != nil && !r.inTx {
		var chunk models.DocumentChunk
		if err := r.c.Get(ctx, key, &chunk); err == nil {
			return &chunk, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetChunkByID(ctx, id)
		if err != nil {
			return nil, err
		}
		chunkPtr := (&models.DocumentChunk{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, chunkPtr, constants.NormalCacheDuration)
		}
		return chunkPtr, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.DocumentChunk), nil
}

func (r *ragRepository) GetChunksByIDs(ctx context.Context, ids []string) ([]*models.DocumentChunk, error) {
	if len(ids) == 0 {
		return []*models.DocumentChunk{}, nil
	}

	resultMap := make(map[string]*models.DocumentChunk, len(ids))
	var missingIDs []string

	for _, id := range ids {
		key := cache.BuildKey("rag_chunk", "id", id)
		if r.c != nil && !r.inTx {
			var chunk models.DocumentChunk
			if err := r.c.Get(ctx, key, &chunk); err == nil {
				resultMap[id] = &chunk
				continue
			}
		}
		missingIDs = append(missingIDs, id)
	}

	if len(missingIDs) > 0 {
		rows, err := r.q.GetChunksByIDs(ctx, missingIDs)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			chunkPtr := (&models.DocumentChunk{}).FromSqlc(row)
			resultMap[chunkPtr.ID] = chunkPtr
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, cache.BuildKey("rag_chunk", "id", chunkPtr.ID), chunkPtr, constants.NormalCacheDuration)
			}
		}
	}

	out := make([]*models.DocumentChunk, 0, len(ids))
	for _, id := range ids {
		if chunk, ok := resultMap[id]; ok {
			out = append(out, chunk)
		}
	}
	return out, nil
}

func (r *ragRepository) GetChunksByDocumentID(ctx context.Context, docID string) ([]*models.DocumentChunk, error) {
	key := cache.BuildKey("rag_chunks_by_doc", "doc_id", docID)
	if r.c != nil && !r.inTx {
		var chunks []*models.DocumentChunk
		if err := r.c.Get(ctx, key, &chunks); err == nil {
			return chunks, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.ListChunksByDocumentID(ctx, docID)
		if err != nil {
			return nil, err
		}
		chunks := make([]*models.DocumentChunk, 0, len(rows))
		for _, row := range rows {
			chunks = append(chunks, (&models.DocumentChunk{}).FromSqlc(row))
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, chunks, constants.NormalCacheDuration)
		}
		return chunks, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.DocumentChunk), nil
}

func (r *ragRepository) UpsertChunk(ctx context.Context, chunk *models.DocumentChunk) (*models.DocumentChunk, error) {
	params := chunk.ToSqlcUpsertParams()
	row, err := r.q.UpsertDocumentChunk(ctx, params)
	if err != nil {
		return nil, err
	}
	r.InvalidateChunkCache(ctx, chunk.ID)
	return (&models.DocumentChunk{}).FromSqlc(row), nil
}

func (r *ragRepository) UpsertChunksBatch(ctx context.Context, chunks []*models.DocumentChunk) error {
	for _, ch := range chunks {
		params := ch.ToSqlcUpsertParams()
		if _, err := r.q.UpsertDocumentChunk(ctx, params); err != nil {
			return err
		}
		r.InvalidateChunkCache(ctx, ch.ID)
	}
	return nil
}

func (r *ragRepository) UpdateChunkStatus(ctx context.Context, id, status string) error {
	if err := r.q.UpdateChunkStatus(ctx, sqlc.UpdateChunkStatusParams{
		Status: status,
		ID:     id,
	}); err != nil {
		return err
	}
	r.InvalidateChunkCache(ctx, id)
	return nil
}

func (r *ragRepository) QueryActiveCandidates(ctx context.Context, cohort, domain string) ([]*models.DocumentChunk, error) {
	key := cache.BuildKey("rag_candidates", "cohort", cohort, "domain", domain)
	if r.c != nil && !r.inTx {
		var chunks []*models.DocumentChunk
		if err := r.c.Get(ctx, key, &chunks); err == nil {
			return chunks, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		var cohortArg sql.NullString
		if strings.TrimSpace(cohort) != "" {
			cohortArg = sql.NullString{String: strings.TrimSpace(cohort), Valid: true}
		}
		var domainArg any = nil
		if strings.TrimSpace(domain) != "" {
			domainArg = strings.TrimSpace(domain)
		}

		rows, err := r.q.QueryActiveChunkCandidates(ctx, sqlc.QueryActiveChunkCandidatesParams{
			Cohort: cohortArg,
			Domain: domainArg,
		})
		if err != nil {
			return nil, err
		}

		chunks := make([]*models.DocumentChunk, 0, len(rows))
		for _, row := range rows {
			chunks = append(chunks, (&models.DocumentChunk{}).FromCandidateRow(row))
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, chunks, constants.NormalCacheDuration)
		}
		return chunks, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.DocumentChunk), nil
}

func (r *ragRepository) GetKGNodeByID(ctx context.Context, id string) (*models.KGNode, error) {
	row, err := r.q.GetKGNodeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return (&models.KGNode{}).FromSqlc(row), nil
}

func (r *ragRepository) ListAllKGNodes(ctx context.Context) ([]*models.KGNode, error) {
	rows, err := r.q.ListAllKGNodes(ctx)
	if err != nil {
		return nil, err
	}
	nodes := make([]*models.KGNode, 0, len(rows))
	for _, row := range rows {
		nodes = append(nodes, (&models.KGNode{}).FromSqlc(row))
	}
	return nodes, nil
}

func (r *ragRepository) UpsertKGNode(ctx context.Context, node *models.KGNode) (*models.KGNode, error) {
	now := time.Now().UTC()
	var codeArg sql.NullString
	if node.Code != "" {
		codeArg = sql.NullString{String: node.Code, Valid: true}
	}
	row, err := r.q.UpsertKGNode(ctx, sqlc.UpsertKGNodeParams{
		ID:        node.ID,
		NodeType:  node.NodeType,
		Name:      node.Name,
		Code:      codeArg,
		Status:    node.Status,
		CreatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	return (&models.KGNode{}).FromSqlc(row), nil
}

func (r *ragRepository) GetKGEdgeByID(ctx context.Context, id string) (*models.KGEdge, error) {
	row, err := r.q.GetKGEdgeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return (&models.KGEdge{}).FromSqlc(row), nil
}

func (r *ragRepository) ListAllKGEdges(ctx context.Context) ([]*models.KGEdge, error) {
	rows, err := r.q.ListAllKGEdges(ctx)
	if err != nil {
		return nil, err
	}
	edges := make([]*models.KGEdge, 0, len(rows))
	for _, row := range rows {
		edges = append(edges, (&models.KGEdge{}).FromSqlc(row))
	}
	return edges, nil
}

func (r *ragRepository) UpsertKGEdge(ctx context.Context, edge *models.KGEdge) (*models.KGEdge, error) {
	now := time.Now().UTC()
	var weightArg sql.NullFloat64
	if edge.Weight > 0 {
		weightArg = sql.NullFloat64{Float64: edge.Weight, Valid: true}
	} else {
		weightArg = sql.NullFloat64{Float64: 1.0, Valid: true}
	}
	row, err := r.q.UpsertKGEdge(ctx, sqlc.UpsertKGEdgeParams{
		ID:           edge.ID,
		SourceNodeID: edge.SourceNodeID,
		TargetNodeID: edge.TargetNodeID,
		RelationType: edge.RelationType,
		Weight:       weightArg,
		Status:       edge.Status,
		CreatedAt:    now,
	})
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("rag_graph", "superseded_chain"))
	}
	return (&models.KGEdge{}).FromSqlc(row), nil
}

func (r *ragRepository) DeleteKGEdge(ctx context.Context, id string) error {
	rows, err := r.q.DeleteKGEdge(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("rag_graph", "superseded_chain"))
	}
	return nil
}

func (r *ragRepository) FindSupersededDocIDs(ctx context.Context) (map[string]bool, error) {
	key := cache.BuildKey("rag_graph", "superseded_chain")
	if r.c != nil && !r.inTx {
		var cached map[string]bool
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rawIDs, err := r.q.FindSupersededDocIDs(ctx)
		if err != nil {
			return nil, err
		}
		result := make(map[string]bool, len(rawIDs))
		for _, rawID := range rawIDs {
			cleanID := strings.TrimPrefix(rawID, "doc:")
			result[cleanID] = true
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(map[string]bool), nil
}

func (r *ragRepository) RecordSupersessionLog(ctx context.Context, activeID, supersededID, ruleType, reason string) error {
	now := time.Now().UTC()
	_, err := r.q.RecordSupersessionAudit(ctx, sqlc.RecordSupersessionAuditParams{
		ActiveDocID:     activeID,
		SupersededDocID: supersededID,
		RuleType:        ruleType,
		Reason:          reason,
		CreatedAt:       now,
	})
	return err
}

func (r *ragRepository) ListAuditLogsByDocID(ctx context.Context, docID string) ([]*models.SupersessionAuditLog, error) {
	rows, err := r.q.ListAuditLogsByDocID(ctx, sqlc.ListAuditLogsByDocIDParams{
		ActiveDocID:     docID,
		SupersededDocID: docID,
	})
	if err != nil {
		return nil, err
	}
	logs := make([]*models.SupersessionAuditLog, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, (&models.SupersessionAuditLog{}).FromSqlc(row))
	}
	return logs, nil
}

func (r *ragRepository) ListAllAuditLogs(ctx context.Context, limit, offset int) ([]*models.SupersessionAuditLog, int, error) {
	total, err := r.q.CountAllAuditLogs(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := r.q.ListAllAuditLogs(ctx, sqlc.ListAllAuditLogsParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, 0, err
	}
	logs := make([]*models.SupersessionAuditLog, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, (&models.SupersessionAuditLog{}).FromSqlc(row))
	}
	return logs, int(total), nil
}

func (r *ragRepository) CountStats(ctx context.Context) (docs int, chunks int, nodes int, edges int, err error) {
	st, err := r.GetKnowledgeStats(ctx)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return st.TotalDocuments, st.TotalChunks, st.TotalNodes, st.TotalEdges, nil
}

func (r *ragRepository) GetKnowledgeStats(ctx context.Context) (*models.KnowledgeStats, error) {
	key := cache.BuildKey("rag_stats", "knowledge")
	if r.c != nil && !r.inTx {
		var stats models.KnowledgeStats
		if err := r.c.Get(ctx, key, &stats); err == nil {
			return &stats, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		totalDocs, err := r.q.CountDocumentsTotal(ctx)
		if err != nil {
			return nil, err
		}
		activeDocs, err := r.q.CountActiveDocuments(ctx)
		if err != nil {
			return nil, err
		}
		supersededDocs, err := r.q.CountSupersededDocuments(ctx)
		if err != nil {
			return nil, err
		}
		totalChunks, err := r.q.CountChunksTotal(ctx)
		if err != nil {
			return nil, err
		}
		totalNodes, err := r.q.CountNodesTotal(ctx)
		if err != nil {
			return nil, err
		}
		totalEdges, err := r.q.CountEdgesTotal(ctx)
		if err != nil {
			return nil, err
		}
		st := &models.KnowledgeStats{
			TotalDocuments:      int(totalDocs),
			ActiveDocuments:     int(activeDocs),
			SupersededDocuments: int(supersededDocs),
			TotalChunks:         int(totalChunks),
			TotalNodes:          int(totalNodes),
			TotalEdges:          int(totalEdges),
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, st, constants.ListCacheDuration)
		}
		return st, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.KnowledgeStats), nil
}

func (r *ragRepository) InvalidateDocumentCache(ctx context.Context, id string) {
	if r.c == nil {
		return
	}
	_ = r.c.Del(ctx, cache.BuildKey("rag_doc", "id", id))
	_ = r.c.DelByPattern(ctx, "rag_candidates*")
	_ = r.c.DelByPattern(ctx, "rag_superseded_docs*")
	_ = r.c.DelByPattern(ctx, "rag_chunks_by_doc*")
	_ = r.c.DelByPattern(ctx, "rag_stats*")
}

func (r *ragRepository) InvalidateChunkCache(ctx context.Context, id string) {
	if r.c == nil {
		return
	}
	_ = r.c.Del(ctx, cache.BuildKey("rag_chunk", "id", id))
	_ = r.c.DelByPattern(ctx, "rag_candidates*")
	_ = r.c.DelByPattern(ctx, "rag_chunks_by_doc*")
	_ = r.c.DelByPattern(ctx, "rag_stats*")
}

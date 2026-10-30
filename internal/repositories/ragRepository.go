package repositories

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
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
	ListDocumentIDsMissingGraphNode(ctx context.Context, limit int64) ([]string, error)
	ListSupersessionPairsMissingEdge(ctx context.Context, limit int64) ([]*models.SupersessionPair, error)
	ListAuditLogPairsMissingEdge(ctx context.Context, limit int64) ([]*models.SupersessionPair, error)
	GetDocumentByID(ctx context.Context, id string) (*models.Document, error)
	GetDocumentByURL(ctx context.Context, rawURL string) (*models.Document, error)
	DocumentExistsByURL(ctx context.Context, rawURL string) (bool, error)
	FindDocumentByIdentifier(ctx context.Context, identifier string) (*models.Document, error)
	GetDocumentsByIDs(ctx context.Context, ids []string) ([]*models.Document, error)
	ListDocuments(ctx context.Context, dto *request.ListDocumentsDto) ([]*models.Document, int, error)
	ListAllRegulationsChronological(ctx context.Context) ([]*models.Document, error)
	ListAllDocumentIdentifiers(ctx context.Context) ([]sqlc.ListAllDocumentIdentifiersRow, error)
	ListActiveDocumentsForDedupe(ctx context.Context) ([]sqlc.ListActiveDocumentsForDedupeRow, error)
	UpsertDocument(ctx context.Context, doc *models.Document) (*models.Document, error)
	UpdateDocumentStatus(ctx context.Context, id, status, supersededByID string) error
	ListSupersededDocuments(ctx context.Context, docID string) ([]*models.Document, error)

	UpsertRule(ctx context.Context, rule *models.DocumentRule) (*models.DocumentRule, error)
	GetRuleByID(ctx context.Context, id string) (*models.DocumentRule, error)
	ListRulesByDocumentID(ctx context.Context, docID string) ([]*models.DocumentRule, error)
	ListActiveRulesByDocumentID(ctx context.Context, docID string) ([]*models.DocumentRule, error)
	UpdateRuleStatus(ctx context.Context, id, status, supersededByID string) error
	UpdateChunksByRuleIDStatus(ctx context.Context, ruleID, status string) error
	UpdateChunksByDocIDStatus(ctx context.Context, docID, status string) error
	DeleteChunksByDocumentID(ctx context.Context, docID string) error
	DeleteRulesByDocumentID(ctx context.Context, docID string) error
	InvalidateRuleCache(ctx context.Context, id string)

	GetChunkByID(ctx context.Context, id string) (*models.DocumentChunk, error)
	GetChunksByIDs(ctx context.Context, ids []string) ([]*models.DocumentChunk, error)
	GetChunksByDocumentID(ctx context.Context, docID string) ([]*models.DocumentChunk, error)
	UpsertChunk(ctx context.Context, chunk *models.DocumentChunk) (*models.DocumentChunk, error)
	UpsertChunksBatch(ctx context.Context, chunks []*models.DocumentChunk) error
	UpdateChunkStatus(ctx context.Context, id, status string) error
	QueryActiveCandidates(ctx context.Context, cohort, domain string) ([]*models.DocumentChunk, error)
	SearchActiveChunkIDsFTS(ctx context.Context, query, cohort, domain string, limit int) ([]string, error)

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
	AuditLogPairExists(ctx context.Context, activeID, supersededID string) (bool, error)
	DocumentHasAuditLog(ctx context.Context, activeID string) (bool, error)

	CreateAuditRun(ctx context.Context, id, cursorDocID string, totalDocs int64) (*models.SupersessionAuditRun, error)
	GetRunningAuditRun(ctx context.Context) (*models.SupersessionAuditRun, error)
	GetLatestAuditRun(ctx context.Context) (*models.SupersessionAuditRun, error)
	UpdateAuditRunProgress(ctx context.Context, id, cursorDocID string, processed, totalDocs int64) error
	CompleteAuditRun(ctx context.Context, id string) error
	FailAuditRun(ctx context.Context, id, lastError string) error

	CountStats(ctx context.Context) (docs int, chunks int, nodes int, edges int, err error)
	GetKnowledgeStats(ctx context.Context) (*models.KnowledgeStats, error)

	WipeCorpus(ctx context.Context) error
	ResetRegulationSupersessions(ctx context.Context) error
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

func (r *ragRepository) GetDocumentByURL(ctx context.Context, rawURL string) (*models.Document, error) {
	clean := strings.TrimSpace(rawURL)
	if clean == "" {
		return nil, sql.ErrNoRows
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(clean)))[:8]
	key := cache.BuildKey("rag_doc", "url", hash)
	if r.c != nil && !r.inTx {
		var doc models.Document
		if err := r.c.Get(ctx, key, &doc); err == nil {
			return &doc, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetDocumentByURL(ctx, sqlc.GetDocumentByURLParams{
			FilePath: clean,
			UrlHash:  sql.NullString{String: hash, Valid: true},
		})
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

func (r *ragRepository) DocumentExistsByURL(ctx context.Context, rawURL string) (bool, error) {
	clean := strings.TrimSpace(rawURL)
	if clean == "" {
		return false, nil
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(clean)))[:8]
	key := cache.BuildKey("rag_doc_exists", "url", hash)
	if r.c != nil && !r.inTx {
		var exists bool
		if err := r.c.Get(ctx, key, &exists); err == nil {
			return exists, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		exists, err := r.q.DocumentExistsByURL(ctx, sqlc.DocumentExistsByURLParams{
			FilePath: clean,
			UrlHash:  sql.NullString{String: hash, Valid: true},
		})
		if err != nil {
			return false, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, exists, constants.NormalCacheDuration)
		}
		return exists, nil
	})
	if err != nil {
		return false, err
	}
	return v.(bool), nil
}

func (r *ragRepository) FindDocumentByIdentifier(ctx context.Context, identifier string) (*models.Document, error) {
	clean := strings.TrimSpace(identifier)
	if clean == "" {
		return nil, sql.ErrNoRows
	}
	key := cache.BuildKey("rag_doc", "ident", clean)
	if r.c != nil && !r.inTx {
		var doc models.Document
		if err := r.c.Get(ctx, key, &doc); err == nil {
			return &doc, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.FindDocumentByIdentifier(ctx, clean)
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
		sfgKey := "rag_docs:get_by_ids:" + strings.Join(missingIDs, ",")
		v, err, _ := r.sfg.Do(sfgKey, func() (any, error) {
			rows, err := r.q.GetDocumentsByIDs(ctx, missingIDs)
			if err != nil {
				return nil, err
			}
			fetched := make([]*models.Document, 0, len(rows))
			for _, row := range rows {
				docPtr := (&models.Document{}).FromSqlc(row)
				if r.c != nil && !r.inTx {
					_ = r.c.Set(ctx, cache.BuildKey("rag_doc", "id", docPtr.ID), docPtr, constants.NormalCacheDuration)
				}
				fetched = append(fetched, docPtr)
			}
			return fetched, nil
		})
		if err != nil {
			return nil, err
		}
		for _, doc := range v.([]*models.Document) {
			resultMap[doc.ID] = doc
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

	countKey := cache.BuildKey("rag_docs_count", "d", domainArg, "st", statusArg, "c", cohortArg, "s", searchArg)
	var total int64
	if r.c != nil && !r.inTx {
		_ = r.c.Get(ctx, countKey, &total)
	}
	if total == 0 {
		v, err, _ := r.sfg.Do(countKey, func() (any, error) {
			cnt, err := r.q.CountDocuments(ctx, sqlc.CountDocumentsParams{
				Domain: domainArg,
				Status: statusArg,
				Cohort: cohortArg,
				Search: searchArg,
			})
			if err != nil {
				return int64(0), err
			}
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, countKey, cnt, constants.NormalCacheDuration)
			}
			return cnt, nil
		})
		if err != nil {
			return nil, 0, err
		}
		total = v.(int64)
	}

	idsKey := cache.BuildKey("rag_docs_ids", "d", domainArg, "st", statusArg, "c", cohortArg, "s", searchArg, "l", limit, "o", offset)
	var ids []string
	if r.c != nil && !r.inTx {
		_ = r.c.Get(ctx, idsKey, &ids)
	}
	if len(ids) == 0 && total > 0 {
		v, err, _ := r.sfg.Do(idsKey, func() (any, error) {
			foundIDs, err := r.q.SearchDocumentIDs(ctx, sqlc.SearchDocumentIDsParams{
				Domain: domainArg,
				Status: statusArg,
				Cohort: cohortArg,
				Search: searchArg,
				Limit:  int64(limit),
				Offset: int64(offset),
			})
			if err != nil {
				return nil, err
			}
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, idsKey, foundIDs, constants.NormalCacheDuration)
			}
			return foundIDs, nil
		})
		if err != nil {
			return nil, 0, err
		}
		ids = v.([]string)
	}

	docs, err := r.GetDocumentsByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	return docs, int(total), nil
}

// ListAllRegulationsChronological retrieves all non-news regulations ordered chronologically from oldest to newest.
func (r *ragRepository) ListAllRegulationsChronological(ctx context.Context) ([]*models.Document, error) {
	key := cache.BuildKey("rag_regulations_chronological")
	if r.c != nil && !r.inTx {
		var cachedIDs []string
		if err := r.c.Get(ctx, key, &cachedIDs); err == nil && len(cachedIDs) > 0 {
			return r.GetDocumentsByIDs(ctx, cachedIDs)
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		ids, err := r.q.ListAllRegulationIDsChronological(ctx)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.NormalCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	ids := v.([]string)
	return r.GetDocumentsByIDs(ctx, ids)
}

// ListActiveDocumentsForDedupe returns the metadata needed to collapse duplicate titles.
func (r *ragRepository) ListActiveDocumentsForDedupe(ctx context.Context) ([]sqlc.ListActiveDocumentsForDedupeRow, error) {
	return r.q.ListActiveDocumentsForDedupe(ctx)
}

// ListAllDocumentIdentifiers retrieves lightweight identifier metadata (id, title, file_path) for all documents.
func (r *ragRepository) ListAllDocumentIdentifiers(ctx context.Context) ([]sqlc.ListAllDocumentIdentifiersRow, error) {
	key := cache.BuildKey("rag_all_document_identifiers")
	if r.c != nil && !r.inTx {
		var cached []sqlc.ListAllDocumentIdentifiersRow
		if err := r.c.Get(ctx, key, &cached); err == nil && len(cached) > 0 {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.ListAllDocumentIdentifiers(ctx)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, rows, constants.NormalCacheDuration)
		}
		return rows, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]sqlc.ListAllDocumentIdentifiersRow), nil
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
		sfgKey := "rag_chunks:get_by_ids:" + strings.Join(missingIDs, ",")
		v, err, _ := r.sfg.Do(sfgKey, func() (any, error) {
			rows, err := r.q.GetChunksByIDs(ctx, missingIDs)
			if err != nil {
				return nil, err
			}
			fetched := make([]*models.DocumentChunk, 0, len(rows))
			for _, row := range rows {
				chunkPtr := (&models.DocumentChunk{}).FromSqlc(row)
				if r.c != nil && !r.inTx {
					_ = r.c.Set(ctx, cache.BuildKey("rag_chunk", "id", chunkPtr.ID), chunkPtr, constants.NormalCacheDuration)
				}
				fetched = append(fetched, chunkPtr)
			}
			return fetched, nil
		})
		if err != nil {
			return nil, err
		}
		for _, chunk := range v.([]*models.DocumentChunk) {
			resultMap[chunk.ID] = chunk
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
		if obj, ok := r.c.GetObject(key); ok && obj != nil {
			return obj.([]*models.DocumentChunk), nil
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
			MinCharCount: constants.MinRetrievableCharCount,
			Cohort:       cohortArg,
			Domain:       domainArg,
		})
		if err != nil {
			return nil, err
		}

		chunks := make([]*models.DocumentChunk, 0, len(rows))
		for _, row := range rows {
			chunks = append(chunks, (&models.DocumentChunk{}).FromCandidateRow(row))
		}
		if r.c != nil && !r.inTx {
			_ = r.c.SetObject(ctx, key, chunks, constants.NormalCacheDuration)
		}
		return chunks, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.DocumentChunk), nil
}

func (r *ragRepository) SearchActiveChunkIDsFTS(ctx context.Context, query, cohort, domain string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 50
	}
	key := cache.BuildKey("rag_fts_ids", "q", query, "cohort", cohort, "domain", domain, "limit", limit)
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return ids, nil
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

		ids, err := r.q.SearchActiveChunkIDsFTS(ctx, sqlc.SearchActiveChunkIDsFTSParams{
			Query:        query,
			MinCharCount: constants.MinRetrievableCharCount,
			Cohort:       cohortArg,
			Domain:       domainArg,
			Limit:        int64(limit),
		})
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.NormalCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}

func (r *ragRepository) GetKGNodeByID(ctx context.Context, id string) (*models.KGNode, error) {
	key := cache.BuildKey("rag_kg_node", "id", id)
	if r.c != nil && !r.inTx {
		var node models.KGNode
		if err := r.c.Get(ctx, key, &node); err == nil {
			return &node, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetKGNodeByID(ctx, id)
		if err != nil {
			return nil, err
		}
		nodePtr := (&models.KGNode{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, nodePtr, constants.NormalCacheDuration)
		}
		return nodePtr, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.KGNode), nil
}

func (r *ragRepository) ListAllKGNodes(ctx context.Context) ([]*models.KGNode, error) {
	key := cache.BuildKey("rag_kg_node", "all")
	if r.c != nil && !r.inTx {
		if obj, ok := r.c.GetObject(key); ok && obj != nil {
			return obj.([]*models.KGNode), nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.ListAllKGNodes(ctx)
		if err != nil {
			return nil, err
		}
		nodes := make([]*models.KGNode, 0, len(rows))
		for _, row := range rows {
			nodes = append(nodes, (&models.KGNode{}).FromSqlc(row))
		}
		if r.c != nil && !r.inTx {
			_ = r.c.SetObject(ctx, key, nodes, constants.NormalCacheDuration)
		}
		return nodes, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.KGNode), nil
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
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("rag_kg_node", "id", node.ID), cache.BuildKey("rag_kg_node", "all"))
		_ = r.c.DelByPattern(ctx, "rag_stats*")
	}
	return (&models.KGNode{}).FromSqlc(row), nil
}

func (r *ragRepository) GetKGEdgeByID(ctx context.Context, id string) (*models.KGEdge, error) {
	key := cache.BuildKey("rag_kg_edge", "id", id)
	if r.c != nil && !r.inTx {
		var edge models.KGEdge
		if err := r.c.Get(ctx, key, &edge); err == nil {
			return &edge, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetKGEdgeByID(ctx, id)
		if err != nil {
			return nil, err
		}
		edgePtr := (&models.KGEdge{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, edgePtr, constants.NormalCacheDuration)
		}
		return edgePtr, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.KGEdge), nil
}

func (r *ragRepository) ListAllKGEdges(ctx context.Context) ([]*models.KGEdge, error) {
	key := cache.BuildKey("rag_kg_edge", "all")
	if r.c != nil && !r.inTx {
		if obj, ok := r.c.GetObject(key); ok && obj != nil {
			return obj.([]*models.KGEdge), nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.ListAllKGEdges(ctx)
		if err != nil {
			return nil, err
		}
		edges := make([]*models.KGEdge, 0, len(rows))
		for _, row := range rows {
			edges = append(edges, (&models.KGEdge{}).FromSqlc(row))
		}
		if r.c != nil && !r.inTx {
			_ = r.c.SetObject(ctx, key, edges, constants.NormalCacheDuration)
		}
		return edges, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.KGEdge), nil
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
		_ = r.c.Del(ctx, cache.BuildKey("rag_kg_edge", "id", edge.ID), cache.BuildKey("rag_kg_edge", "all"), cache.BuildKey("rag_graph", "superseded_chain"))
		_ = r.c.DelByPattern(ctx, "rag_stats*")
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
		_ = r.c.Del(ctx, cache.BuildKey("rag_kg_edge", "id", id), cache.BuildKey("rag_kg_edge", "all"), cache.BuildKey("rag_graph", "superseded_chain"))
		_ = r.c.DelByPattern(ctx, "rag_stats*")
	}
	return nil
}

func (r *ragRepository) FindSupersededDocIDs(ctx context.Context) (map[string]bool, error) {
	key := cache.BuildKey("rag_graph", "superseded_chain")
	if r.c != nil && !r.inTx {
		if obj, ok := r.c.GetObject(key); ok && obj != nil {
			return obj.(map[string]bool), nil
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
			_ = r.c.SetObject(ctx, key, result, constants.NormalCacheDuration)
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
	if err == nil && r.c != nil {
		_ = r.c.DelByPattern(ctx, "rag_audit_logs*")
	}
	return err
}

func (r *ragRepository) ListAuditLogsByDocID(ctx context.Context, docID string) ([]*models.SupersessionAuditLog, error) {
	key := cache.BuildKey("rag_audit_logs", "doc_id", docID)
	if r.c != nil && !r.inTx {
		var cached []*models.SupersessionAuditLog
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
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
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, logs, constants.NormalCacheDuration)
		}
		return logs, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.SupersessionAuditLog), nil
}

func (r *ragRepository) ListAllAuditLogs(ctx context.Context, limit, offset int) ([]*models.SupersessionAuditLog, int, error) {
	type pagedAuditLogs struct {
		Logs  []*models.SupersessionAuditLog
		Total int
	}
	key := cache.BuildKey("rag_audit_logs", "all", "l", limit, "o", offset)
	if r.c != nil && !r.inTx {
		var cached pagedAuditLogs
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached.Logs, cached.Total, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		total, err := r.q.CountAllAuditLogs(ctx)
		if err != nil {
			return nil, err
		}
		rows, err := r.q.ListAllAuditLogs(ctx, sqlc.ListAllAuditLogsParams{
			Limit:  int64(limit),
			Offset: int64(offset),
		})
		if err != nil {
			return nil, err
		}
		logs := make([]*models.SupersessionAuditLog, 0, len(rows))
		for _, row := range rows {
			logs = append(logs, (&models.SupersessionAuditLog{}).FromSqlc(row))
		}
		res := pagedAuditLogs{Logs: logs, Total: int(total)}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, 0, err
	}
	res := v.(pagedAuditLogs)
	return res.Logs, res.Total, nil
}

func (r *ragRepository) AuditLogPairExists(ctx context.Context, activeID, supersededID string) (bool, error) {
	n, err := r.q.CountAuditLogPairs(ctx, sqlc.CountAuditLogPairsParams{
		ActiveDocID:     activeID,
		SupersededDocID: supersededID,
	})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *ragRepository) DocumentHasAuditLog(ctx context.Context, activeID string) (bool, error) {
	n, err := r.q.CountAuditLogsByActiveDoc(ctx, activeID)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *ragRepository) CreateAuditRun(ctx context.Context, id, cursorDocID string, totalDocs int64) (*models.SupersessionAuditRun, error) {
	row, err := r.q.CreateSupersessionAuditRun(ctx, sqlc.CreateSupersessionAuditRunParams{
		ID:          id,
		CursorDocID: cursorDocID,
		TotalDocs:   totalDocs,
		Processed:   0,
	})
	if err != nil {
		return nil, err
	}
	return (&models.SupersessionAuditRun{}).FromSqlc(row), nil
}

func (r *ragRepository) GetRunningAuditRun(ctx context.Context) (*models.SupersessionAuditRun, error) {
	row, err := r.q.GetRunningSupersessionAuditRun(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return (&models.SupersessionAuditRun{}).FromSqlc(row), nil
}

func (r *ragRepository) GetLatestAuditRun(ctx context.Context) (*models.SupersessionAuditRun, error) {
	row, err := r.q.GetLatestSupersessionAuditRun(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return (&models.SupersessionAuditRun{}).FromSqlc(row), nil
}

func (r *ragRepository) UpdateAuditRunProgress(ctx context.Context, id, cursorDocID string, processed, totalDocs int64) error {
	return r.q.UpdateSupersessionAuditRunProgress(ctx, sqlc.UpdateSupersessionAuditRunProgressParams{
		ID:          id,
		CursorDocID: cursorDocID,
		Processed:   processed,
		TotalDocs:   totalDocs,
	})
}

func (r *ragRepository) CompleteAuditRun(ctx context.Context, id string) error {
	return r.q.CompleteSupersessionAuditRun(ctx, id)
}

func (r *ragRepository) FailAuditRun(ctx context.Context, id, lastError string) error {
	return r.q.FailSupersessionAuditRun(ctx, sqlc.FailSupersessionAuditRunParams{
		ID:        id,
		LastError: lastError,
	})
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
		partiallyAmendedDocs, err := r.q.CountPartiallyAmendedDocuments(ctx)
		if err != nil {
			return nil, err
		}
		archivedDocs, err := r.q.CountArchivedDocuments(ctx)
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
			TotalDocuments:            int(totalDocs),
			ActiveDocuments:           int(activeDocs),
			SupersededDocuments:       int(supersededDocs),
			PartiallyAmendedDocuments: int(partiallyAmendedDocs),
			ArchivedDocuments:         int(archivedDocs),
			TotalChunks:               int(totalChunks),
			TotalNodes:                int(totalNodes),
			TotalEdges:                int(totalEdges),
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
	_ = r.c.DelByPattern(ctx, "rag_doc:url*")
	_ = r.c.DelByPattern(ctx, "rag_doc_exists:url*")
	_ = r.c.DelByPattern(ctx, "rag_doc:ident*")
	_ = r.c.DelByPattern(ctx, "rag_docs_count*")
	_ = r.c.DelByPattern(ctx, "rag_docs_ids*")
	_ = r.c.DelByPattern(ctx, "rag_candidates*")
	_ = r.c.DelByPattern(ctx, "rag_fts_ids*")
	_ = r.c.DelByPattern(ctx, "rag_superseded_docs*")
	_ = r.c.DelByPattern(ctx, "rag_chunks_by_doc*")
	_ = r.c.DelByPattern(ctx, "rag_rules_by_doc*")
	_ = r.c.DelByPattern(ctx, "rag_active_rules_by_doc*")
	_ = r.c.DelByPattern(ctx, "rag_regulations_chronological*")
	_ = r.c.DelByPattern(ctx, "rag_all_document_identifiers*")
	_ = r.c.DelByPattern(ctx, "rag_stats*")
}

func (r *ragRepository) InvalidateChunkCache(ctx context.Context, id string) {
	if r.c == nil {
		return
	}
	_ = r.c.Del(ctx, cache.BuildKey("rag_chunk", "id", id))
	_ = r.c.DelByPattern(ctx, "rag_docs_count*")
	_ = r.c.DelByPattern(ctx, "rag_docs_ids*")
	_ = r.c.DelByPattern(ctx, "rag_candidates*")
	_ = r.c.DelByPattern(ctx, "rag_fts_ids*")
	_ = r.c.DelByPattern(ctx, "rag_chunks_by_doc*")
	_ = r.c.DelByPattern(ctx, "rag_stats*")
}

func (r *ragRepository) UpsertRule(ctx context.Context, rule *models.DocumentRule) (*models.DocumentRule, error) {
	chapter := sql.NullString{String: rule.Chapter, Valid: rule.Chapter != ""}
	artNum := sql.NullString{String: rule.ArticleNumber, Valid: rule.ArticleNumber != ""}
	superID := sql.NullString{String: rule.SupersededByID, Valid: rule.SupersededByID != ""}
	createdAt := time.Now()
	if rule.CreatedAt != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", rule.CreatedAt); err == nil {
			createdAt = t
		}
	}

	row, err := r.q.UpsertDocumentRule(ctx, sqlc.UpsertDocumentRuleParams{
		ID:             rule.ID,
		DocumentID:     rule.DocumentID,
		RuleIndex:      int64(rule.RuleIndex),
		Chapter:        chapter,
		ArticleNumber:  artNum,
		Title:          rule.Title,
		Content:        rule.Content,
		Status:         rule.Status,
		SupersededByID: superID,
		CreatedAt:      createdAt,
	})
	if err != nil {
		return nil, err
	}

	res := (&models.DocumentRule{}).FromSqlc(row)
	r.InvalidateRuleCache(ctx, rule.ID)
	r.InvalidateDocumentCache(ctx, rule.DocumentID)
	return res, nil
}

func (r *ragRepository) GetRuleByID(ctx context.Context, id string) (*models.DocumentRule, error) {
	key := cache.BuildKey("rag_rule", "id", id)
	if r.c != nil && !r.inTx {
		var rule models.DocumentRule
		if err := r.c.Get(ctx, key, &rule); err == nil {
			return &rule, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetDocumentRuleByID(ctx, id)
		if err != nil {
			return nil, err
		}
		rulePtr := (&models.DocumentRule{}).FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, rulePtr, constants.NormalCacheDuration)
		}
		return rulePtr, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.DocumentRule), nil
}

func (r *ragRepository) ListRulesByDocumentID(ctx context.Context, docID string) ([]*models.DocumentRule, error) {
	key := cache.BuildKey("rag_rules_by_doc", docID)
	if r.c != nil && !r.inTx {
		var cached []*models.DocumentRule
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.ListRulesByDocumentID(ctx, docID)
		if err != nil {
			return nil, err
		}
		res := make([]*models.DocumentRule, len(rows))
		for i, row := range rows {
			res[i] = (&models.DocumentRule{}).FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.DocumentRule), nil
}

func (r *ragRepository) ListActiveRulesByDocumentID(ctx context.Context, docID string) ([]*models.DocumentRule, error) {
	key := cache.BuildKey("rag_active_rules_by_doc", docID)
	if r.c != nil && !r.inTx {
		var cached []*models.DocumentRule
		if err := r.c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.ListActiveRulesByDocumentID(ctx, docID)
		if err != nil {
			return nil, err
		}
		res := make([]*models.DocumentRule, len(rows))
		for i, row := range rows {
			res[i] = (&models.DocumentRule{}).FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.DocumentRule), nil
}

func (r *ragRepository) UpdateRuleStatus(ctx context.Context, id, status, supersededByID string) error {
	superID := sql.NullString{String: supersededByID, Valid: supersededByID != ""}
	err := r.q.UpdateRuleStatus(ctx, sqlc.UpdateRuleStatusParams{
		Status:         status,
		SupersededByID: superID,
		ID:             id,
	})
	if err == nil {
		r.InvalidateRuleCache(ctx, id)
	}
	return err
}

func (r *ragRepository) UpdateChunksByRuleIDStatus(ctx context.Context, ruleID, status string) error {
	return r.q.UpdateDocumentChunksByRuleIDStatus(ctx, sqlc.UpdateDocumentChunksByRuleIDStatusParams{
		Status: status,
		RuleID: sql.NullString{String: ruleID, Valid: ruleID != ""},
	})
}

func (r *ragRepository) UpdateChunksByDocIDStatus(ctx context.Context, docID, status string) error {
	return r.q.UpdateDocumentChunksByDocIDStatus(ctx, sqlc.UpdateDocumentChunksByDocIDStatusParams{
		Status:     status,
		DocumentID: docID,
	})
}

// DeleteChunksByDocumentID removes all chunks associated with a document ID.
func (r *ragRepository) DeleteChunksByDocumentID(ctx context.Context, docID string) error {
	if err := r.q.DeleteChunksByDocumentID(ctx, docID); err != nil {
		return err
	}
	if r.c != nil && !r.inTx {
		_ = r.c.DelByPattern(ctx, "rag_chunk:*")
		_ = r.c.DelByPattern(ctx, "rag_candidates*")
	}
	return nil
}

// DeleteRulesByDocumentID removes all rules associated with a document ID.
func (r *ragRepository) DeleteRulesByDocumentID(ctx context.Context, docID string) error {
	err := r.q.DeleteRulesByDocumentID(ctx, docID)
	if err == nil && r.c != nil && !r.inTx {
		_ = r.c.DelByPattern(ctx, "rag_rule:*")
		_ = r.c.DelByPattern(ctx, "rag_rules_by_doc*")
		_ = r.c.DelByPattern(ctx, "rag_active_rules_by_doc*")
		_ = r.c.DelByPattern(ctx, "rag_stats*")
	}
	return err
}

func (r *ragRepository) InvalidateRuleCache(ctx context.Context, id string) {
	if r.c == nil {
		return
	}
	if id != "" {
		_ = r.c.Del(ctx, cache.BuildKey("rag_rule", "id", id))
	}
	_ = r.c.DelByPattern(ctx, "rag_rules_by_doc*")
	_ = r.c.DelByPattern(ctx, "rag_active_rules_by_doc*")
	_ = r.c.DelByPattern(ctx, "rag_stats*")
}

func (r *ragRepository) WipeCorpus(ctx context.Context) error {
	if err := r.q.ClearAllDocumentChunks(ctx); err != nil {
		return err
	}
	if err := r.q.ClearAllDocumentRules(ctx); err != nil {
		return err
	}
	if err := r.q.ClearSupersessionAuditLogs(ctx); err != nil {
		return err
	}
	if err := r.q.ClearAllKGEdges(ctx); err != nil {
		return err
	}
	if err := r.q.ClearAllKGNodes(ctx); err != nil {
		return err
	}
	if err := r.q.ClearAllDocuments(ctx); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.DelByPattern(ctx, "rag_*")
	}
	return nil
}

// ResetRegulationSupersessions clears supersession state of the documents covered by the audit,
// leaving announcements and news already reconciled by other repair passes untouched.
func (r *ragRepository) ResetRegulationSupersessions(ctx context.Context) error {
	if err := r.q.ResetRegulationDocumentSupersessions(ctx); err != nil {
		return err
	}
	if err := r.q.ResetRegulationRuleSupersessions(ctx); err != nil {
		return err
	}
	if err := r.q.ResetRegulationChunkSupersessions(ctx); err != nil {
		return err
	}
	if err := r.q.ClearRegulationSupersessionEdges(ctx); err != nil {
		return err
	}
	if err := r.q.ClearRegulationSupersessionAuditLogs(ctx); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.DelByPattern(ctx, "rag_*")
	}
	return nil
}

func (r *ragRepository) ListDocumentIDsMissingGraphNode(ctx context.Context, limit int64) ([]string, error) {
	return r.q.ListDocumentIDsMissingGraphNode(ctx, limit)
}

func (r *ragRepository) ListSupersessionPairsMissingEdge(ctx context.Context, limit int64) ([]*models.SupersessionPair, error) {
	rows, err := r.q.ListSupersessionPairsMissingEdge(ctx, limit)
	if err != nil {
		return nil, err
	}
	pairs := make([]*models.SupersessionPair, len(rows))
	for i, row := range rows {
		pairs[i] = &models.SupersessionPair{
			ActiveDocID:     row.ActiveDocID.String,
			SupersededDocID: row.SupersededDocID,
			RelationType:    row.RelationType,
		}
	}
	return pairs, nil
}

func (r *ragRepository) ListAuditLogPairsMissingEdge(ctx context.Context, limit int64) ([]*models.SupersessionPair, error) {
	rows, err := r.q.ListAuditLogPairsMissingEdge(ctx, limit)
	if err != nil {
		return nil, err
	}
	pairs := make([]*models.SupersessionPair, len(rows))
	for i, row := range rows {
		pairs[i] = &models.SupersessionPair{
			ActiveDocID:     row.ActiveDocID,
			SupersededDocID: row.SupersededDocID,
			RelationType:    row.RelationType,
		}
	}
	return pairs, nil
}

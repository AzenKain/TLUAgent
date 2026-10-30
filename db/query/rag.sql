-- name: UpsertDocument :one
INSERT INTO documents (
    id, doc_code, type, title, domain, validity_tier, status, superseded_by_id,
    department_code, publish_date, effective_from, effective_to, academic_year,
    semester, applicable_cohort, priority_level, file_path, char_count, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    doc_code = excluded.doc_code,
    type = excluded.type,
    title = excluded.title,
    domain = excluded.domain,
    validity_tier = excluded.validity_tier,
    status = excluded.status,
    superseded_by_id = excluded.superseded_by_id,
    department_code = excluded.department_code,
    publish_date = excluded.publish_date,
    effective_from = excluded.effective_from,
    effective_to = excluded.effective_to,
    academic_year = excluded.academic_year,
    semester = excluded.semester,
    applicable_cohort = excluded.applicable_cohort,
    priority_level = excluded.priority_level,
    file_path = excluded.file_path,
    char_count = excluded.char_count,
    updated_at = excluded.updated_at
RETURNING *;

-- name: GetDocumentByID :one
SELECT * FROM documents WHERE id = ? LIMIT 1;

-- name: GetDocumentByURL :one
SELECT * FROM documents 
WHERE file_path = sqlc.arg('file_path') 
   OR (sqlc.arg('url_hash') != '' AND id LIKE ('%' || sqlc.arg('url_hash')))
LIMIT 1;

-- name: DocumentExistsByURL :one
SELECT EXISTS(
    SELECT 1 FROM documents 
    WHERE file_path = sqlc.arg('file_path') 
       OR (sqlc.arg('url_hash') != '' AND id LIKE ('%' || sqlc.arg('url_hash')))
);

-- name: FindDocumentByIdentifier :one
SELECT * FROM documents 
WHERE id = ?1 
   OR doc_code = ?1 
   OR title = ?1
   OR title LIKE ('%' || ?1 || '%')
ORDER BY 
   CASE 
     WHEN id = ?1 THEN 1 
     WHEN doc_code = ?1 THEN 2 
     WHEN title = ?1 THEN 3 
     ELSE 4 
   END ASC
LIMIT 1;

-- name: GetDocumentsByIDs :many
SELECT * FROM documents WHERE id IN (sqlc.slice('ids'));

-- name: SearchDocumentIDs :many
SELECT id FROM documents
WHERE (sqlc.narg('domain') IS NULL OR domain = sqlc.narg('domain'))
  AND (sqlc.narg('status') IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('cohort') IS NULL OR applicable_cohort = 'ALL' OR (',' || REPLACE(applicable_cohort, ' ', '') || ',') LIKE ('%,' || sqlc.narg('cohort') || ',%'))
  AND (sqlc.narg('search') IS NULL OR (
      title LIKE ('%' || sqlc.narg('search') || '%') OR 
      doc_code LIKE ('%' || sqlc.narg('search') || '%')
  ))
ORDER BY priority_level DESC, publish_date DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountDocuments :one
SELECT COUNT(*) FROM documents
WHERE (sqlc.narg('domain') IS NULL OR domain = sqlc.narg('domain'))
  AND (sqlc.narg('status') IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('cohort') IS NULL OR applicable_cohort = 'ALL' OR (',' || REPLACE(applicable_cohort, ' ', '') || ',') LIKE ('%,' || sqlc.narg('cohort') || ',%'))
  AND (sqlc.narg('search') IS NULL OR (
      title LIKE ('%' || sqlc.narg('search') || '%') OR 
      doc_code LIKE ('%' || sqlc.narg('search') || '%')
  ));

-- name: ListSupersededDocuments :many
SELECT * FROM documents
WHERE superseded_by_id = ?
ORDER BY publish_date DESC;

-- name: UpdateDocumentStatus :exec
UPDATE documents
SET status = ?, superseded_by_id = ?, updated_at = ?
WHERE id = ?;

-- name: UpsertDocumentChunk :one
INSERT INTO document_chunks (
    id, document_id, rule_id, chunk_index, chunk_type, content, status,
    domain, applicable_cohort, academic_year, priority_level, embedding, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    content = excluded.content,
    status = excluded.status,
    domain = excluded.domain,
    applicable_cohort = excluded.applicable_cohort,
    academic_year = excluded.academic_year,
    priority_level = excluded.priority_level,
    embedding = excluded.embedding
RETURNING *;

-- name: GetChunkByID :one
SELECT * FROM document_chunks WHERE id = ? LIMIT 1;

-- name: GetChunksByIDs :many
SELECT * FROM document_chunks WHERE id IN (sqlc.slice('ids'));

-- name: ListChunksByDocumentID :many
SELECT * FROM document_chunks
WHERE document_id = ?
ORDER BY chunk_index ASC;

-- name: QueryActiveChunkCandidates :many
SELECT c.id, c.document_id, c.rule_id, c.chunk_index, c.chunk_type, c.content, c.status,
       c.domain, c.applicable_cohort, c.academic_year, c.priority_level, c.embedding, c.created_at,
       d.title AS doc_title, d.type AS doc_type
FROM document_chunks c
JOIN documents d ON c.document_id = d.id
WHERE c.status = 'ACTIVE'
  AND d.status = 'ACTIVE'
  AND (d.char_count = 0 OR d.char_count >= sqlc.arg('min_char_count'))
  AND (sqlc.narg('cohort') IS NULL OR sqlc.narg('cohort') = '' OR c.applicable_cohort = 'ALL' OR (',' || REPLACE(c.applicable_cohort, ' ', '') || ',') LIKE ('%,' || sqlc.narg('cohort') || ',%'))
  AND (sqlc.narg('domain') IS NULL OR sqlc.narg('domain') = '' OR c.domain = sqlc.narg('domain'));

-- name: UpsertKGNode :one
INSERT INTO kg_nodes (id, node_type, name, code, status, created_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    node_type = excluded.node_type,
    name = excluded.name,
    code = excluded.code,
    status = excluded.status
RETURNING *;

-- name: GetKGNodeByID :one
SELECT * FROM kg_nodes WHERE id = ? LIMIT 1;

-- name: ListAllKGNodes :many
SELECT * FROM kg_nodes
ORDER BY node_type ASC, name ASC;

-- name: UpsertKGEdge :one
INSERT INTO kg_edges (id, source_node_id, target_node_id, relation_type, weight, status, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    relation_type = excluded.relation_type,
    weight = excluded.weight,
    status = excluded.status
RETURNING *;

-- name: GetKGEdgeByID :one
SELECT * FROM kg_edges WHERE id = ? LIMIT 1;

-- name: ListAllKGEdges :many
SELECT * FROM kg_edges
ORDER BY created_at DESC;

-- name: DeleteKGEdge :execrows
DELETE FROM kg_edges
WHERE id = ?;

-- name: FindSupersededDocIDs :many
WITH RECURSIVE superseding_chain AS (
    SELECT source_node_id AS active_doc, target_node_id AS superseded_doc
    FROM kg_edges
    WHERE relation_type = 'SUPERSEDES' AND status = 'ACTIVE'
    UNION ALL
    SELECT sc.active_doc, ke.target_node_id
    FROM superseding_chain sc
    JOIN kg_edges ke ON sc.superseded_doc = ke.source_node_id
    WHERE ke.relation_type = 'SUPERSEDES' AND ke.status = 'ACTIVE'
)
SELECT superseded_doc FROM superseding_chain;

-- name: RecordSupersessionAudit :one
INSERT INTO supersession_audit_log (active_doc_id, superseded_doc_id, rule_type, reason, created_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteAuditLogPair :exec
DELETE FROM supersession_audit_log
WHERE active_doc_id = ? AND superseded_doc_id = ?;

-- name: ListAuditLogsByDocID :many
SELECT id, active_doc_id, superseded_doc_id, rule_type, reason, created_at
FROM supersession_audit_log
WHERE active_doc_id = ? OR superseded_doc_id = ?
ORDER BY created_at DESC;

-- name: ListAllAuditLogs :many
SELECT id, active_doc_id, superseded_doc_id, rule_type, reason, created_at
FROM supersession_audit_log
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountAllAuditLogs :one
SELECT COUNT(*) FROM supersession_audit_log;

-- name: CountDocumentsTotal :one
SELECT COUNT(*) FROM documents;

-- name: CountActiveDocuments :one
SELECT COUNT(*) FROM documents WHERE status = 'ACTIVE';

-- name: CountSupersededDocuments :one
SELECT COUNT(*) FROM documents WHERE status = 'SUPERSEDED';

-- name: CountPartiallyAmendedDocuments :one
SELECT COUNT(*) FROM documents WHERE status = 'PARTIALLY_AMENDED';

-- name: CountArchivedDocuments :one
SELECT COUNT(*) FROM documents WHERE status = 'ARCHIVED';

-- name: CountChunksTotal :one
SELECT COUNT(*) FROM document_chunks;

-- name: CountNodesTotal :one
SELECT COUNT(*) FROM kg_nodes;

-- name: CountEdgesTotal :one
SELECT COUNT(*) FROM kg_edges;

-- name: UpdateChunkStatus :exec
UPDATE document_chunks
SET status = ?
WHERE id = ?;

-- name: SearchActiveChunkIDsFTS :many
SELECT f.chunk_id
FROM document_chunks_fts f
JOIN document_chunks c ON c.id = f.chunk_id
JOIN documents d ON d.id = c.document_id
WHERE f.content MATCH sqlc.arg('query')
  AND f.status = 'ACTIVE'
  AND d.status = 'ACTIVE'
  AND (d.char_count = 0 OR d.char_count >= sqlc.arg('min_char_count'))
  AND (sqlc.narg('domain') IS NULL OR sqlc.narg('domain') = '' OR f.domain = sqlc.narg('domain'))
  AND (sqlc.narg('cohort') IS NULL OR sqlc.narg('cohort') = '' OR f.applicable_cohort = 'ALL' OR (',' || REPLACE(f.applicable_cohort, ' ', '') || ',') LIKE ('%,' || sqlc.narg('cohort') || ',%'))
ORDER BY rank
LIMIT sqlc.arg('limit');

-- name: UpsertDocumentRule :one
INSERT INTO document_rules (
    id, document_id, rule_index, chapter, article_number, title, content, status, superseded_by_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    rule_index = excluded.rule_index,
    chapter = excluded.chapter,
    article_number = excluded.article_number,
    title = excluded.title,
    content = excluded.content,
    status = excluded.status,
    superseded_by_id = excluded.superseded_by_id
RETURNING *;

-- name: GetDocumentRuleByID :one
SELECT * FROM document_rules WHERE id = ? LIMIT 1;

-- name: ListRulesByDocumentID :many
SELECT * FROM document_rules
WHERE document_id = ?
ORDER BY rule_index ASC;

-- name: ListActiveRulesByDocumentID :many
SELECT * FROM document_rules
WHERE document_id = ? AND status = 'ACTIVE'
ORDER BY rule_index ASC;

-- name: UpdateRuleStatus :exec
UPDATE document_rules
SET status = ?, superseded_by_id = ?
WHERE id = ?;

-- name: UpdateDocumentChunksByRuleIDStatus :exec
UPDATE document_chunks
SET status = ?
WHERE rule_id = ?;

-- name: UpdateDocumentChunksByDocIDStatus :exec
UPDATE document_chunks
SET status = ?
WHERE document_id = ?;

-- name: UpdateChunksCohortByDocumentID :exec
UPDATE document_chunks
SET applicable_cohort = ?
WHERE document_id = ?;

-- name: DeleteChunksByDocumentID :exec
DELETE FROM document_chunks
WHERE document_id = ?;

-- name: DeleteRulesByDocumentID :exec
DELETE FROM document_rules
WHERE document_id = ?;

-- name: ClearAllDocumentChunks :exec
DELETE FROM document_chunks;

-- name: ClearAllDocumentRules :exec
DELETE FROM document_rules;

-- name: ClearSupersessionAuditLogs :exec
DELETE FROM supersession_audit_log;

-- name: ResetRegulationDocumentSupersessions :exec
UPDATE documents
SET status = 'ACTIVE', superseded_by_id = '', updated_at = CURRENT_TIMESTAMP
WHERE type IN ('QUY_CHE', 'QUY_DINH', 'QUYET_DINH', 'DE_AN', 'KE_HOACH', 'CTDT', 'SO_TAY')
  AND (status != 'ACTIVE' OR (superseded_by_id IS NOT NULL AND superseded_by_id != ''));

-- name: ResetRegulationRuleSupersessions :exec
UPDATE document_rules
SET status = 'ACTIVE'
WHERE status != 'ACTIVE'
  AND document_id IN (SELECT id FROM documents WHERE type IN ('QUY_CHE', 'QUY_DINH', 'QUYET_DINH', 'DE_AN', 'KE_HOACH', 'CTDT', 'SO_TAY'));

-- name: ResetRegulationChunkSupersessions :exec
UPDATE document_chunks
SET status = 'ACTIVE'
WHERE status != 'ACTIVE'
  AND document_id IN (SELECT id FROM documents WHERE type IN ('QUY_CHE', 'QUY_DINH', 'QUYET_DINH', 'DE_AN', 'KE_HOACH', 'CTDT', 'SO_TAY'));

-- name: ClearRegulationSupersessionEdges :exec
DELETE FROM kg_edges
WHERE relation_type IN ('SUPERSEDES', 'PARTIALLY_AMENDS')
  AND (
      substr(target_node_id, 5) IN (SELECT id FROM documents WHERE type IN ('QUY_CHE', 'QUY_DINH', 'QUYET_DINH', 'DE_AN', 'KE_HOACH', 'CTDT', 'SO_TAY'))
      OR substr(source_node_id, 5) IN (SELECT id FROM documents WHERE type IN ('QUY_CHE', 'QUY_DINH', 'QUYET_DINH', 'DE_AN', 'KE_HOACH', 'CTDT', 'SO_TAY'))
  );

-- name: ClearRegulationSupersessionAuditLogs :exec
DELETE FROM supersession_audit_log
WHERE superseded_doc_id IN (SELECT id FROM documents WHERE type IN ('QUY_CHE', 'QUY_DINH', 'QUYET_DINH', 'DE_AN', 'KE_HOACH', 'CTDT', 'SO_TAY'))
   OR active_doc_id IN (SELECT id FROM documents WHERE type IN ('QUY_CHE', 'QUY_DINH', 'QUYET_DINH', 'DE_AN', 'KE_HOACH', 'CTDT', 'SO_TAY'));

-- name: ClearAllKGEdges :exec
DELETE FROM kg_edges;

-- name: ClearAllKGNodes :exec
DELETE FROM kg_nodes;

-- name: ClearAllDocuments :exec
DELETE FROM documents;

-- name: ListAllRegulationIDsChronological :many
SELECT id FROM documents
WHERE type IN ('QUY_CHE', 'QUY_DINH', 'QUYET_DINH', 'DE_AN', 'KE_HOACH', 'CTDT', 'SO_TAY')
ORDER BY 
    CASE WHEN publish_date IS NOT NULL AND publish_date != '' THEN publish_date ELSE created_at END ASC,
    created_at ASC;

-- name: ListAllDocumentIdentifiers :many
SELECT id, title, file_path FROM documents;

-- name: ListActiveDocumentsForDedupe :many
SELECT id, title, publish_date, char_count
FROM documents
WHERE status = 'ACTIVE';

-- name: CreateSupersessionAuditRun :one
INSERT INTO supersession_audit_runs (id, status, cursor_doc_id, total_docs, processed)
VALUES (?, 'RUNNING', ?, ?, ?)
RETURNING *;

-- name: GetRunningSupersessionAuditRun :one
SELECT * FROM supersession_audit_runs
WHERE status = 'RUNNING'
ORDER BY started_at DESC, rowid DESC
LIMIT 1;

-- name: GetLatestSupersessionAuditRun :one
SELECT * FROM supersession_audit_runs
ORDER BY started_at DESC, rowid DESC
LIMIT 1;

-- name: UpdateSupersessionAuditRunProgress :exec
UPDATE supersession_audit_runs
SET cursor_doc_id = ?, processed = ?, total_docs = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: CompleteSupersessionAuditRun :exec
UPDATE supersession_audit_runs
SET status = 'COMPLETED', cursor_doc_id = '', processed = total_docs, last_error = '', updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: FailSupersessionAuditRun :exec
UPDATE supersession_audit_runs
SET status = 'FAILED', last_error = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: CountAuditLogPairs :one
SELECT COUNT(*) FROM supersession_audit_log
WHERE active_doc_id = ? AND superseded_doc_id = ?;

-- name: CountAuditLogsByActiveDoc :one
SELECT COUNT(*) FROM supersession_audit_log
WHERE active_doc_id = ?;

-- name: ListDocumentIDsMissingGraphNode :many
SELECT d.id
FROM documents d
WHERE NOT EXISTS (
    SELECT 1 FROM kg_nodes n WHERE n.id = 'doc:' || d.id
)
ORDER BY d.created_at ASC
LIMIT ?;

-- name: ListSupersessionPairsMissingEdge :many
SELECT
    d.superseded_by_id AS active_doc_id,
    d.id AS superseded_doc_id,
    'SUPERSEDES' AS relation_type
FROM documents d
WHERE d.status = 'SUPERSEDED'
  AND d.superseded_by_id IS NOT NULL
  AND trim(d.superseded_by_id) != ''
  AND NOT EXISTS (
      SELECT 1 FROM kg_edges e
      WHERE e.source_node_id = 'doc:' || d.superseded_by_id
        AND e.target_node_id = 'doc:' || d.id
        AND e.relation_type IN ('SUPERSEDES', 'PARTIALLY_AMENDS')
  )
ORDER BY d.created_at ASC
LIMIT ?;

-- name: ListAuditLogPairsMissingEdge :many
SELECT
    l.active_doc_id,
    l.superseded_doc_id,
    CASE
        WHEN SUM(CASE WHEN l.rule_type IN ('DOCUMENT_REPLACEMENT', 'ADMIN_OVERRIDE') THEN 1 ELSE 0 END) > 0
        THEN 'SUPERSEDES'
        ELSE 'PARTIALLY_AMENDS'
    END AS relation_type
FROM supersession_audit_log l
WHERE NOT EXISTS (
    SELECT 1 FROM kg_edges e
    WHERE e.source_node_id = 'doc:' || l.active_doc_id
      AND e.target_node_id = 'doc:' || l.superseded_doc_id
      AND e.relation_type IN ('SUPERSEDES', 'PARTIALLY_AMENDS')
)
GROUP BY l.active_doc_id, l.superseded_doc_id
ORDER BY MIN(l.id) ASC
LIMIT ?;

-- name: GetDocumentAuditFingerprint :one
SELECT fingerprint FROM document_audit_state WHERE document_id = ?;

-- name: UpsertDocumentAuditFingerprint :exec
INSERT INTO document_audit_state (document_id, fingerprint, audited_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(document_id) DO UPDATE SET fingerprint = excluded.fingerprint, audited_at = CURRENT_TIMESTAMP;

-- name: CountDocumentAuditStates :one
SELECT COUNT(*) FROM document_audit_state;

-- name: ClearDocumentAuditStates :exec
DELETE FROM document_audit_state;

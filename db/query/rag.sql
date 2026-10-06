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

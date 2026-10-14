-- name: CreateTLUClawSession :one
INSERT INTO tluclaw_sessions (id, admin_user_id, title, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTLUClawSessionByID :one
SELECT id, admin_user_id, title, created_at, updated_at
FROM tluclaw_sessions
WHERE id = ?;

-- name: ListTLUClawSessionIDsByAdmin :many
SELECT id
FROM tluclaw_sessions
WHERE admin_user_id = ?
ORDER BY updated_at DESC
LIMIT ? OFFSET ?;

-- name: GetTLUClawSessionsByIDs :many
SELECT id, admin_user_id, title, created_at, updated_at
FROM tluclaw_sessions
WHERE id IN (sqlc.slice('ids'))
ORDER BY updated_at DESC;

-- name: UpdateTLUClawSessionTitle :exec
UPDATE tluclaw_sessions
SET title = ?, updated_at = ?
WHERE id = ?;

-- name: TouchTLUClawSession :exec
UPDATE tluclaw_sessions
SET updated_at = ?
WHERE id = ?;

-- name: DeleteTLUClawSession :exec
DELETE FROM tluclaw_sessions
WHERE id = ?;

-- name: CreateTLUClawMessage :one
INSERT INTO tluclaw_messages (id, session_id, role, content, reasoning_content, tool_calls_json, tool_call_id, metadata_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListTLUClawMessagesBySessionID :many
SELECT id, session_id, role, content, reasoning_content, tool_calls_json, tool_call_id, metadata_json, created_at
FROM tluclaw_messages
WHERE session_id = ?
ORDER BY created_at ASC;

-- name: CreateTLUClawSnapshot :one
INSERT INTO tluclaw_action_snapshots (
    id, batch_id, session_id, operator_id, action_type, summary,
    diff_preview_json, blast_radius_json, previous_state_json, new_state_json,
    status, confirmation_nonce, nonce_expires_at, applied_at, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetTLUClawSnapshotByID :one
SELECT id, batch_id, session_id, operator_id, action_type, summary,
       diff_preview_json, blast_radius_json, previous_state_json, new_state_json,
       status, confirmation_nonce, nonce_expires_at, applied_at, created_at, updated_at
FROM tluclaw_action_snapshots
WHERE id = ?;

-- name: GetTLUClawSnapshotByNonce :one
SELECT id, batch_id, session_id, operator_id, action_type, summary,
       diff_preview_json, blast_radius_json, previous_state_json, new_state_json,
       status, confirmation_nonce, nonce_expires_at, applied_at, created_at, updated_at
FROM tluclaw_action_snapshots
WHERE confirmation_nonce = ?;

-- name: GetTLUClawSnapshotByBatchID :one
SELECT id, batch_id, session_id, operator_id, action_type, summary,
       diff_preview_json, blast_radius_json, previous_state_json, new_state_json,
       status, confirmation_nonce, nonce_expires_at, applied_at, created_at, updated_at
FROM tluclaw_action_snapshots
WHERE batch_id = ?;

-- name: ListTLUClawSnapshotsBySessionID :many
SELECT id, batch_id, session_id, operator_id, action_type, summary,
       diff_preview_json, blast_radius_json, previous_state_json, new_state_json,
       status, confirmation_nonce, nonce_expires_at, applied_at, created_at, updated_at
FROM tluclaw_action_snapshots
WHERE session_id = ?
ORDER BY created_at DESC;

-- name: UpdateTLUClawSnapshotStatus :exec
UPDATE tluclaw_action_snapshots
SET status = ?, applied_at = ?, updated_at = ?
WHERE id = ?;

-- name: ListLatestCommittedTLUClawSnapshots :many
SELECT id, batch_id, session_id, operator_id, action_type, summary,
       diff_preview_json, blast_radius_json, previous_state_json, new_state_json,
       status, confirmation_nonce, nonce_expires_at, applied_at, created_at, updated_at
FROM tluclaw_action_snapshots
WHERE status = 'COMMITTED'
ORDER BY applied_at DESC
LIMIT ?;

-- name: ListTLUClawSnapshotsPaginated :many
SELECT id, batch_id, session_id, operator_id, action_type, summary,
       diff_preview_json, blast_radius_json, previous_state_json, new_state_json,
       status, confirmation_nonce, nonce_expires_at, applied_at, created_at, updated_at
FROM tluclaw_action_snapshots
WHERE (status = sqlc.arg('status') OR sqlc.arg('status') = '')
ORDER BY created_at DESC
LIMIT sqlc.arg('limit_val') OFFSET sqlc.arg('offset_val');

-- name: CountTLUClawSnapshots :one
SELECT COUNT(*) FROM tluclaw_action_snapshots
WHERE (status = sqlc.arg('status') OR sqlc.arg('status') = '');


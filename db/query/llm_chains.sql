-- name: GetChain :one
SELECT id, chain_type, name, description, is_active, failure_threshold, cooldown_seconds, retry_count, created_at, updated_at
FROM llm_chains
WHERE id = ?;

-- name: GetActiveChainByType :one
SELECT id, chain_type, name, description, is_active, failure_threshold, cooldown_seconds, retry_count, created_at, updated_at
FROM llm_chains
WHERE chain_type = ? AND is_active = 1
LIMIT 1;

-- name: ListChains :many
SELECT id, chain_type, name, description, is_active, failure_threshold, cooldown_seconds, retry_count, created_at, updated_at
FROM llm_chains
ORDER BY chain_type ASC, name ASC;

-- name: CreateChain :one
INSERT INTO llm_chains (id, chain_type, name, description, is_active, failure_threshold, cooldown_seconds, retry_count)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, chain_type, name, description, is_active, failure_threshold, cooldown_seconds, retry_count, created_at, updated_at;

-- name: UpdateChain :one
UPDATE llm_chains
SET name = ?, description = ?, is_active = ?, failure_threshold = ?, cooldown_seconds = ?, retry_count = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING id, chain_type, name, description, is_active, failure_threshold, cooldown_seconds, retry_count, created_at, updated_at;

-- name: DeleteChain :exec
DELETE FROM llm_chains
WHERE id = ?;

-- name: ListChainNodesWithDetails :many
SELECT 
    cn.id AS node_id,
    cn.chain_id,
    cn.model_id,
    cn.priority,
    cn.is_active AS node_is_active,
    m.name AS model_name,
    m.model_key,
    m.model_type,
    m.context_length,
    m.supports_vision,
    m.max_images,
    p.id AS provider_id,
    p.name AS provider_name,
    p.provider_type,
    p.base_url,
    p.api_key_ciphertext,
    p.timeout_seconds,
    p.max_retries,
    p.allow_private_networks
FROM llm_chain_nodes cn
JOIN llm_models m ON cn.model_id = m.id
JOIN llm_providers p ON m.provider_id = p.id
WHERE cn.chain_id = ?
ORDER BY cn.priority ASC;

-- name: AddChainNode :one
INSERT INTO llm_chain_nodes (id, chain_id, model_id, priority, is_active)
VALUES (?, ?, ?, ?, ?)
RETURNING id, chain_id, model_id, priority, is_active, created_at, updated_at;

-- name: RemoveChainNode :exec
DELETE FROM llm_chain_nodes
WHERE id = ?;

-- name: UpdateChainNodePriority :exec
UPDATE llm_chain_nodes
SET priority = ?, is_active = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: ClearChainNodes :exec
DELETE FROM llm_chain_nodes
WHERE chain_id = ?;

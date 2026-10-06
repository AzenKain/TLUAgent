-- name: ListLLMProviders :many
SELECT * FROM llm_providers ORDER BY created_at ASC;

-- name: GetLLMProviderByID :one
SELECT * FROM llm_providers WHERE id = ? LIMIT 1;

-- name: GetDefaultLLMProvider :one
SELECT * FROM llm_providers WHERE is_default = 1 AND is_active = 1 LIMIT 1;

-- name: CreateLLMProvider :one
INSERT INTO llm_providers (
    id, name, provider_type, base_url, api_key_ciphertext, is_active, is_default, custom_headers_json,
    timeout_seconds, max_retries, retry_initial_wait_ms, retry_max_wait_ms, allow_private_networks
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateLLMProvider :one
UPDATE llm_providers
SET name = ?, provider_type = ?, base_url = ?, api_key_ciphertext = ?, is_active = ?, is_default = ?, custom_headers_json = ?,
    timeout_seconds = ?, max_retries = ?, retry_initial_wait_ms = ?, retry_max_wait_ms = ?, allow_private_networks = ?
WHERE id = ?
RETURNING *;

-- name: SetDefaultLLMProvider :exec
UPDATE llm_providers SET is_default = CASE WHEN id = ? THEN 1 ELSE 0 END;

-- name: DeleteLLMProvider :exec
DELETE FROM llm_providers WHERE id = ?;

-- name: ListLLMModels :many
SELECT * FROM llm_models ORDER BY order_index ASC, created_at ASC;

-- name: ListActiveLLMModelsByType :many
SELECT * FROM llm_models WHERE is_active = 1 AND model_type = ? ORDER BY order_index ASC, created_at ASC;

-- name: ListLLMModelsByProviderID :many
SELECT * FROM llm_models WHERE provider_id = ? ORDER BY order_index ASC, created_at ASC;

-- name: GetLLMModelByID :one
SELECT * FROM llm_models WHERE id = ? LIMIT 1;

-- name: GetLLMModelByKey :one
SELECT * FROM llm_models WHERE model_key = ? LIMIT 1;

-- name: GetDefaultLLMModelByType :one
SELECT * FROM llm_models WHERE is_default = 1 AND is_active = 1 AND model_type = ? LIMIT 1;

-- name: CreateLLMModel :one
INSERT INTO llm_models (
    id, provider_id, name, model_key, model_type, is_active, is_default,
    order_index, context_length, vision_mode, supports_vision, max_images,
    thinking_enabled, thinking_budget, effort_level
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateLLMModel :one
UPDATE llm_models
SET provider_id = ?, name = ?, model_key = ?, model_type = ?, is_active = ?, is_default = ?,
    order_index = ?, context_length = ?, vision_mode = ?, supports_vision = ?, max_images = ?,
    thinking_enabled = ?, thinking_budget = ?, effort_level = ?
WHERE id = ?
RETURNING *;

-- name: UpdateModelVisionCapability :one
UPDATE llm_models
SET vision_mode = ?, supports_vision = ?, max_images = ?
WHERE id = ?
RETURNING *;

-- name: SetDefaultLLMModel :exec
UPDATE llm_models SET is_default = CASE WHEN id = ? THEN 1 ELSE 0 END WHERE model_type = ?;

-- name: DeleteLLMModel :exec
DELETE FROM llm_models WHERE id = ?;

-- name: GetLLMModelWithProvider :one
SELECT 
    m.id AS model_id,
    m.name AS model_name,
    m.model_key,
    m.model_type,
    m.is_active AS model_is_active,
    m.is_default AS model_is_default,
    m.order_index,
    m.context_length,
    m.vision_mode,
    m.supports_vision,
    m.max_images,
    m.thinking_enabled,
    m.thinking_budget,
    m.effort_level,
    m.created_at AS model_created_at,
    m.updated_at AS model_updated_at,
    p.id AS provider_id,
    p.name AS provider_name,
    p.provider_type,
    p.base_url,
    p.api_key_ciphertext,
    p.is_active AS provider_is_active,
    p.custom_headers_json,
    p.timeout_seconds,
    p.max_retries,
    p.retry_initial_wait_ms,
    p.retry_max_wait_ms,
    p.allow_private_networks
FROM llm_models m
JOIN llm_providers p ON m.provider_id = p.id
WHERE m.id = ? LIMIT 1;

-- name: ListActiveChatModelsWithProvider :many
SELECT 
    m.id AS model_id,
    m.name AS model_name,
    m.model_key,
    m.model_type,
    m.is_default AS model_is_default,
    m.order_index,
    m.context_length,
    m.vision_mode,
    m.supports_vision,
    m.max_images,
    m.thinking_enabled,
    m.thinking_budget,
    m.effort_level,
    p.id AS provider_id,
    p.name AS provider_name,
    p.provider_type
FROM llm_models m
JOIN llm_providers p ON m.provider_id = p.id
WHERE m.is_active = 1 AND p.is_active = 1 AND m.model_type = 'chat'
ORDER BY m.order_index ASC, m.created_at ASC;

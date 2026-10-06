-- name: GetAgentPrompt :one
SELECT id, title, content, is_active, created_at, updated_at, updated_by
FROM agent_prompts
WHERE id = ?;

-- name: ListAgentPrompts :many
SELECT id, title, content, is_active, created_at, updated_at, updated_by
FROM agent_prompts
ORDER BY id ASC;

-- name: ListActiveAgentPrompts :many
SELECT id, title, content, is_active, created_at, updated_at, updated_by
FROM agent_prompts
WHERE is_active = 1
ORDER BY id ASC;

-- name: UpsertAgentPrompt :exec
INSERT INTO agent_prompts (id, title, content, is_active, updated_at, updated_by)
VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, ?)
ON CONFLICT(id) DO UPDATE SET
    title = excluded.title,
    content = excluded.content,
    is_active = excluded.is_active,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = excluded.updated_by;

-- name: GetAgentSkill :one
SELECT id, name, description, content, tool_definition, is_enabled, priority, created_at, updated_at, updated_by
FROM agent_skills
WHERE id = ?;

-- name: ListAgentSkills :many
SELECT id, name, description, content, tool_definition, is_enabled, priority, created_at, updated_at, updated_by
FROM agent_skills
ORDER BY priority DESC, id ASC;

-- name: ListEnabledAgentSkills :many
SELECT id, name, description, content, tool_definition, is_enabled, priority, created_at, updated_at, updated_by
FROM agent_skills
WHERE is_enabled = 1
ORDER BY priority DESC, id ASC;

-- name: UpsertAgentSkill :exec
INSERT INTO agent_skills (id, name, description, content, tool_definition, is_enabled, priority, created_at, updated_at, updated_by)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    content = excluded.content,
    tool_definition = excluded.tool_definition,
    is_enabled = excluded.is_enabled,
    priority = excluded.priority,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = excluded.updated_by;

-- name: UpdateAgentSkillEnabled :exec
UPDATE agent_skills
SET is_enabled = ?, updated_at = CURRENT_TIMESTAMP, updated_by = ?
WHERE id = ?;

-- name: DeleteAgentSkill :exec
DELETE FROM agent_skills
WHERE id = ?;

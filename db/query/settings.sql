-- name: GetAppSetting :one
SELECT key, value_json, updated_at
FROM app_settings
WHERE key = ?;

-- name: UpsertAppSetting :exec
INSERT INTO app_settings (key, value_json)
VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json;

-- name: GetSetupState :one
SELECT key, value, updated_at
FROM setup_state
WHERE key = ?;

-- name: UpsertSetupState :exec
INSERT INTO setup_state (key, value)
VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

-- name: ClaimInitialSetup :execrows
INSERT INTO setup_state (key, value)
VALUES ('completed', 'in_progress')
ON CONFLICT(key) DO UPDATE SET value = 'in_progress'
WHERE setup_state.value NOT IN ('true', 'in_progress');

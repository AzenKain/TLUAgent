-- name: UpsertUser :one
INSERT INTO users (
    id,
    email,
    full_name,
    student_code,
    avatar_url,
    password_hash,
    auth_provider
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
)
ON CONFLICT(email)
DO UPDATE SET
    full_name = COALESCE(excluded.full_name, users.full_name),
    student_code = COALESCE(excluded.student_code, users.student_code),
    avatar_url = COALESCE(excluded.avatar_url, users.avatar_url)
RETURNING id, email, full_name, student_code, avatar_url, password_hash, auth_provider, token_version, refresh_token, is_deleted, created_at, updated_at;

-- name: CreateUser :one
INSERT INTO users (
    id,
    email,
    full_name,
    student_code,
    password_hash,
    auth_provider
) VALUES (
    ?, ?, ?, ?, ?, ?
)
RETURNING id, email, full_name, student_code, avatar_url, password_hash, auth_provider, token_version, refresh_token, is_deleted, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, email, full_name, student_code, avatar_url, password_hash, auth_provider, token_version, refresh_token, is_deleted, created_at, updated_at
FROM users
WHERE id = ? AND is_deleted = 0;

-- name: GetUserByIDWithoutDeleted :one
SELECT id, email, full_name, student_code, avatar_url, password_hash, auth_provider, token_version, refresh_token, is_deleted, created_at, updated_at
FROM users
WHERE id = ?;

-- name: GetUserByEmail :one
SELECT id, email, full_name, student_code, avatar_url, password_hash, auth_provider, token_version, refresh_token, is_deleted, created_at, updated_at
FROM users
WHERE email = ? AND is_deleted = 0;

-- name: UpdateUserProfile :one
UPDATE users
SET
    full_name = CASE WHEN sqlc.narg('full_name') IS NOT NULL THEN sqlc.narg('full_name') ELSE full_name END,
    student_code = CASE WHEN sqlc.narg('student_code') IS NOT NULL THEN sqlc.narg('student_code') ELSE student_code END,
    avatar_url = CASE WHEN sqlc.narg('avatar_url') IS NOT NULL THEN sqlc.narg('avatar_url') ELSE avatar_url END,
    updated_at = datetime('now')
WHERE id = sqlc.arg('id') AND is_deleted = 0
RETURNING id, email, full_name, student_code, avatar_url, password_hash, auth_provider, token_version, refresh_token, is_deleted, created_at, updated_at;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = ?,
    token_version = token_version + 1,
    refresh_token = NULL,
    updated_at = datetime('now')
WHERE id = ? AND is_deleted = 0;

-- name: UpdateUserRefreshToken :exec
UPDATE users
SET refresh_token = ?
WHERE id = ? AND is_deleted = 0;

-- name: RotateUserRefreshToken :execrows
UPDATE users
SET refresh_token = sqlc.arg('new_refresh_token')
WHERE id = sqlc.arg('id')
  AND is_deleted = 0
  AND refresh_token = sqlc.arg('current_refresh_token');

-- name: GetUserTokenVersion :one
SELECT token_version
FROM users
WHERE id = ? AND is_deleted = 0;

-- name: UpdateUserTokenVersion :exec
UPDATE users
SET token_version = ?
WHERE id = ? AND is_deleted = 0;

-- name: RevokeUserSessions :exec
UPDATE users
SET token_version = token_version + 1,
    refresh_token = NULL,
    updated_at = datetime('now')
WHERE id = ? AND is_deleted = 0;

-- name: DeleteUser :exec
UPDATE users
SET is_deleted = 1,
    token_version = token_version + 1,
    refresh_token = NULL,
    updated_at = datetime('now')
WHERE id = ?;

-- name: RestoreUser :exec
UPDATE users
SET is_deleted = 0,
    updated_at = datetime('now')
WHERE id = ?;

-- name: GetUserRoles :many
SELECT r.id, r.name, r.description, r.is_system, r.is_admin, r.is_banned, r.auto_assign, r.position, r.is_deleted, r.created_at, r.updated_at
FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = ? AND r.is_deleted = 0
ORDER BY r.position DESC, r.name ASC;

-- name: GetUsersByIDs :many
SELECT id, email, full_name, student_code, avatar_url, password_hash, auth_provider, token_version, refresh_token, is_deleted, created_at, updated_at
FROM users
WHERE id IN (sqlc.slice('ids'));

-- name: SearchUserIDs :many
SELECT u.id
FROM users u
WHERE
    (sqlc.narg('is_deleted') IS NULL OR u.is_deleted = sqlc.narg('is_deleted'))
    AND (sqlc.narg('role_id') IS NULL OR EXISTS (
        SELECT 1 FROM user_roles ur
        WHERE ur.user_id = u.id AND ur.role_id = sqlc.narg('role_id')
    ))
    AND (sqlc.narg('search_text') IS NULL OR
        lower(u.email) LIKE '%' || lower(sqlc.narg('search_text')) || '%' OR
        lower(COALESCE(u.full_name, '')) LIKE '%' || lower(sqlc.narg('search_text')) || '%' OR
        lower(COALESCE(u.student_code, '')) LIKE '%' || lower(sqlc.narg('search_text')) || '%'
    )
ORDER BY u.created_at DESC, u.id ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountUsers :one
SELECT count(*)
FROM users u
WHERE
    (sqlc.narg('is_deleted') IS NULL OR u.is_deleted = sqlc.narg('is_deleted'))
    AND (sqlc.narg('role_id') IS NULL OR EXISTS (
        SELECT 1 FROM user_roles ur
        WHERE ur.user_id = u.id AND ur.role_id = sqlc.narg('role_id')
    ))
    AND (sqlc.narg('search_text') IS NULL OR
        lower(u.email) LIKE '%' || lower(sqlc.narg('search_text')) || '%' OR
        lower(COALESCE(u.full_name, '')) LIKE '%' || lower(sqlc.narg('search_text')) || '%' OR
        lower(COALESCE(u.student_code, '')) LIKE '%' || lower(sqlc.narg('search_text')) || '%'
    );

-- name: GetUserRolePermissions :many
SELECT rp.id, rp.role_id, rp.permission_key, rp.effect, rp.conditions_json, rp.created_at, rp.updated_at
FROM role_permissions rp
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = ?
ORDER BY rp.role_id ASC, rp.permission_key ASC;

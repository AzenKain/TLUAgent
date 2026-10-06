-- name: GetStudentProfileByUserID :one
SELECT * FROM student_academic_profiles
WHERE user_id = ? LIMIT 1;

-- name: UpsertStudentProfile :one
INSERT INTO student_academic_profiles (
    user_id, major, cohort, academic_standing, completed_credits,
    cumulative_gpa, target_gpa, advisor_notes, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
    major = excluded.major,
    cohort = excluded.cohort,
    academic_standing = excluded.academic_standing,
    completed_credits = excluded.completed_credits,
    cumulative_gpa = excluded.cumulative_gpa,
    target_gpa = excluded.target_gpa,
    advisor_notes = excluded.advisor_notes,
    updated_at = excluded.updated_at
RETURNING *;

-- name: ListUserMemoriesByUserID :many
SELECT * FROM user_memories
WHERE user_id = ? AND is_active = 1
ORDER BY updated_at DESC;

-- name: ListUserMemoryIDsByUserID :many
SELECT id FROM user_memories
WHERE user_id = ? AND is_active = 1
ORDER BY updated_at DESC;

-- name: GetUserMemoryByID :one
SELECT * FROM user_memories
WHERE id = ? LIMIT 1;

-- name: GetUserMemoriesByIDs :many
SELECT * FROM user_memories
WHERE id IN (sqlc.slice('ids'));

-- name: UpsertUserMemory :one
INSERT INTO user_memories (
    id, user_id, category, memory_key, memory_value,
    confidence, source, is_active, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, memory_key) DO UPDATE SET
    memory_value = excluded.memory_value,
    category = excluded.category,
    confidence = excluded.confidence,
    source = excluded.source,
    is_active = excluded.is_active,
    updated_at = excluded.updated_at
RETURNING *;

-- name: DeleteUserMemory :execrows
DELETE FROM user_memories
WHERE id = ? AND user_id = ?;

-- name: CountUserMemories :one
SELECT COUNT(*) FROM user_memories
WHERE user_id = ? AND is_active = 1;

-- name: DeleteStudentProfile :execrows
DELETE FROM student_academic_profiles
WHERE user_id = ?;

-- name: ClearAllUserMemories :execrows
DELETE FROM user_memories
WHERE user_id = ?;

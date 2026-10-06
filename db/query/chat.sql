-- name: CreateConversation :one
INSERT INTO chat_conversations (id, user_id, title, model_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetConversationByID :one
SELECT id, user_id, title, model_id, created_at, updated_at
FROM chat_conversations
WHERE id = ? LIMIT 1;

-- name: GetConversationsByIDs :many
SELECT id, user_id, title, model_id, created_at, updated_at
FROM chat_conversations
WHERE id IN (sqlc.slice('ids'));

-- name: ListConversationIDsByUserID :many
SELECT id
FROM chat_conversations
WHERE user_id = ?
ORDER BY updated_at DESC
LIMIT ? OFFSET ?;

-- name: CountConversationsByUserID :one
SELECT COUNT(*)
FROM chat_conversations
WHERE user_id = ?;

-- name: UpdateConversationTitle :execrows
UPDATE chat_conversations
SET title = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: TouchConversation :exec
UPDATE chat_conversations
SET updated_at = ?
WHERE id = ?;

-- name: DeleteConversation :execrows
DELETE FROM chat_conversations
WHERE id = ? AND user_id = ?;

-- name: DeleteConversationAdmin :execrows
DELETE FROM chat_conversations
WHERE id = ?;

-- name: CreateMessage :one
INSERT INTO chat_messages (id, conversation_id, sender, content, sources_json, images_json, feedback, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListMessagesByConversationID :many
SELECT id, conversation_id, sender, content, sources_json, images_json, feedback, created_at
FROM chat_messages
WHERE conversation_id = ?
ORDER BY created_at ASC;

-- name: UpdateMessageFeedback :execrows
UPDATE chat_messages
SET feedback = ?
WHERE chat_messages.id = ? AND conversation_id IN (SELECT id FROM chat_conversations WHERE user_id = ?);

-- name: GetAdminTotalChats :one
SELECT COUNT(*) FROM chat_conversations;

-- name: GetAdminMessageStats :one
SELECT 
    COUNT(*) AS total_messages,
    CAST(COALESCE(SUM(CASE WHEN feedback = 'up' THEN 1 ELSE 0 END), 0) AS INTEGER) AS thumbs_up,
    CAST(COALESCE(SUM(CASE WHEN feedback = 'down' THEN 1 ELSE 0 END), 0) AS INTEGER) AS thumbs_down
FROM chat_messages;

-- name: ListAdminConversationsNoFeedback :many
SELECT 
    c.id, 
    c.user_id, 
    u.email AS user_email, 
    COALESCE(u.full_name, '') AS user_full_name, 
    COALESCE(u.student_code, '') AS student_code, 
    c.title, 
    COALESCE(c.model_id, '') AS model_id, 
    COUNT(m.id) AS message_count,
    CAST(COALESCE(SUM(CASE WHEN m.feedback = 'up' THEN 1 ELSE 0 END), 0) AS INTEGER) AS feedback_up_count,
    CAST(COALESCE(SUM(CASE WHEN m.feedback = 'down' THEN 1 ELSE 0 END), 0) AS INTEGER) AS feedback_down_count,
    c.created_at, 
    c.updated_at
FROM chat_conversations c
JOIN users u ON c.user_id = u.id
LEFT JOIN chat_messages m ON c.id = m.conversation_id
WHERE (sqlc.narg('search_pattern') IS NULL OR (
    (u.full_name LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
    (u.student_code LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
    (u.email LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
    (c.title LIKE sqlc.narg('search_pattern') ESCAPE '\')
))
  AND (sqlc.narg('model_id') IS NULL OR c.model_id = sqlc.narg('model_id'))
GROUP BY c.id
ORDER BY c.updated_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountAdminConversationsNoFeedback :one
SELECT COUNT(*) FROM (
    SELECT c.id
    FROM chat_conversations c
    JOIN users u ON c.user_id = u.id
    LEFT JOIN chat_messages m ON c.id = m.conversation_id
    WHERE (sqlc.narg('search_pattern') IS NULL OR (
        (u.full_name LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
        (u.student_code LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
        (u.email LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
        (c.title LIKE sqlc.narg('search_pattern') ESCAPE '\')
    ))
      AND (sqlc.narg('model_id') IS NULL OR c.model_id = sqlc.narg('model_id'))
    GROUP BY c.id
);

-- name: ListAdminConversationsFeedbackUp :many
SELECT 
    c.id, 
    c.user_id, 
    u.email AS user_email, 
    COALESCE(u.full_name, '') AS user_full_name, 
    COALESCE(u.student_code, '') AS student_code, 
    c.title, 
    COALESCE(c.model_id, '') AS model_id, 
    COUNT(m.id) AS message_count,
    CAST(COALESCE(SUM(CASE WHEN m.feedback = 'up' THEN 1 ELSE 0 END), 0) AS INTEGER) AS feedback_up_count,
    CAST(COALESCE(SUM(CASE WHEN m.feedback = 'down' THEN 1 ELSE 0 END), 0) AS INTEGER) AS feedback_down_count,
    c.created_at, 
    c.updated_at
FROM chat_conversations c
JOIN users u ON c.user_id = u.id
LEFT JOIN chat_messages m ON c.id = m.conversation_id
WHERE (sqlc.narg('search_pattern') IS NULL OR (
    (u.full_name LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
    (u.student_code LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
    (u.email LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
    (c.title LIKE sqlc.narg('search_pattern') ESCAPE '\')
))
  AND (sqlc.narg('model_id') IS NULL OR c.model_id = sqlc.narg('model_id'))
GROUP BY c.id
HAVING SUM(CASE WHEN m.feedback = 'up' THEN 1 ELSE 0 END) > 0
ORDER BY c.updated_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountAdminConversationsFeedbackUp :one
SELECT COUNT(*) FROM (
    SELECT c.id
    FROM chat_conversations c
    JOIN users u ON c.user_id = u.id
    LEFT JOIN chat_messages m ON c.id = m.conversation_id
    WHERE (sqlc.narg('search_pattern') IS NULL OR (
        (u.full_name LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
        (u.student_code LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
        (u.email LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
        (c.title LIKE sqlc.narg('search_pattern') ESCAPE '\')
    ))
      AND (sqlc.narg('model_id') IS NULL OR c.model_id = sqlc.narg('model_id'))
    GROUP BY c.id
    HAVING SUM(CASE WHEN m.feedback = 'up' THEN 1 ELSE 0 END) > 0
);

-- name: ListAdminConversationsFeedbackDown :many
SELECT 
    c.id, 
    c.user_id, 
    u.email AS user_email, 
    COALESCE(u.full_name, '') AS user_full_name, 
    COALESCE(u.student_code, '') AS student_code, 
    c.title, 
    COALESCE(c.model_id, '') AS model_id, 
    COUNT(m.id) AS message_count,
    CAST(COALESCE(SUM(CASE WHEN m.feedback = 'up' THEN 1 ELSE 0 END), 0) AS INTEGER) AS feedback_up_count,
    CAST(COALESCE(SUM(CASE WHEN m.feedback = 'down' THEN 1 ELSE 0 END), 0) AS INTEGER) AS feedback_down_count,
    c.created_at, 
    c.updated_at
FROM chat_conversations c
JOIN users u ON c.user_id = u.id
LEFT JOIN chat_messages m ON c.id = m.conversation_id
WHERE (sqlc.narg('search_pattern') IS NULL OR (
    (u.full_name LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
    (u.student_code LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
    (u.email LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
    (c.title LIKE sqlc.narg('search_pattern') ESCAPE '\')
))
  AND (sqlc.narg('model_id') IS NULL OR c.model_id = sqlc.narg('model_id'))
GROUP BY c.id
HAVING SUM(CASE WHEN m.feedback = 'down' THEN 1 ELSE 0 END) > 0
ORDER BY c.updated_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountAdminConversationsFeedbackDown :one
SELECT COUNT(*) FROM (
    SELECT c.id
    FROM chat_conversations c
    JOIN users u ON c.user_id = u.id
    LEFT JOIN chat_messages m ON c.id = m.conversation_id
    WHERE (sqlc.narg('search_pattern') IS NULL OR (
        (u.full_name LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
        (u.student_code LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
        (u.email LIKE sqlc.narg('search_pattern') ESCAPE '\') OR 
        (c.title LIKE sqlc.narg('search_pattern') ESCAPE '\')
    ))
      AND (sqlc.narg('model_id') IS NULL OR c.model_id = sqlc.narg('model_id'))
    GROUP BY c.id
    HAVING SUM(CASE WHEN m.feedback = 'down' THEN 1 ELSE 0 END) > 0
);

-- name: CreateNotification :one
INSERT INTO user_notifications (
    id, user_id, inquiry_id, title, content, type, is_read, created_at
) VALUES (?, ?, ?, ?, ?, ?, 0, ?)
RETURNING *;

-- name: GetNotificationByID :one
SELECT * FROM user_notifications
WHERE id = ? LIMIT 1;

-- name: GetNotificationsByIDs :many
SELECT * FROM user_notifications
WHERE id IN (sqlc.slice('ids'));

-- name: ListNotificationIDsByUserID :many
SELECT id FROM user_notifications
WHERE user_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM user_notifications
WHERE user_id = ? AND is_read = 0;

-- name: MarkNotificationAsRead :exec
UPDATE user_notifications
SET is_read = 1
WHERE id = ? AND user_id = ?;

-- name: MarkAllNotificationsAsRead :exec
UPDATE user_notifications
SET is_read = 1
WHERE user_id = ?;

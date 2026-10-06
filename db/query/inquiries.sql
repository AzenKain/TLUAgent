-- name: CreateInquiry :one
INSERT INTO advisory_inquiries (
    id, user_id, conversation_id, student_name, student_code, student_class,
    question, context, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, ?)
RETURNING *;

-- name: GetInquiryByID :one
SELECT * FROM advisory_inquiries
WHERE id = ? LIMIT 1;

-- name: GetInquiriesByIDs :many
SELECT * FROM advisory_inquiries
WHERE id IN (sqlc.slice('ids'));

-- name: ListInquiryIDsByUserID :many
SELECT id FROM advisory_inquiries
WHERE user_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountInquiriesByUserID :one
SELECT COUNT(*) FROM advisory_inquiries
WHERE user_id = ?;

-- name: ListInquiryIDs :many
SELECT id FROM advisory_inquiries
WHERE (sqlc.narg('status') IS NULL OR sqlc.narg('status') = '' OR status = sqlc.narg('status'))
  AND (sqlc.narg('search') IS NULL OR sqlc.narg('search') = '' OR (
      question LIKE ('%' || sqlc.narg('search') || '%') OR
      student_name LIKE ('%' || sqlc.narg('search') || '%') OR
      student_code LIKE ('%' || sqlc.narg('search') || '%') OR
      student_class LIKE ('%' || sqlc.narg('search') || '%')
  ))
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountInquiries :one
SELECT COUNT(*) FROM advisory_inquiries
WHERE (sqlc.narg('status') IS NULL OR sqlc.narg('status') = '' OR status = sqlc.narg('status'))
  AND (sqlc.narg('search') IS NULL OR sqlc.narg('search') = '' OR (
      question LIKE ('%' || sqlc.narg('search') || '%') OR
      student_name LIKE ('%' || sqlc.narg('search') || '%') OR
      student_code LIKE ('%' || sqlc.narg('search') || '%') OR
      student_class LIKE ('%' || sqlc.narg('search') || '%')
  ));

-- name: AnswerInquiry :one
UPDATE advisory_inquiries
SET status = 'ANSWERED',
    teacher_id = ?,
    teacher_name = ?,
    teacher_reply = ?,
    answered_at = ?,
    knowledge_chunk_id = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: ExpireInquiry :one
UPDATE advisory_inquiries
SET status = 'EXPIRED',
    is_expired = 1,
    superseded_by_doc_id = ?,
    expired_reason = ?,
    expired_at = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: ListActiveAnsweredInquiries :many
SELECT * FROM advisory_inquiries
WHERE status = 'ANSWERED' AND is_expired = 0
ORDER BY answered_at DESC;

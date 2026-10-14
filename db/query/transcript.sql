-- name: ListTranscriptGradeIDsByUserID :many
SELECT id FROM student_course_grades
WHERE user_id = ?
ORDER BY course_code ASC;

-- name: GetTranscriptGradesByIDs :many
SELECT * FROM student_course_grades
WHERE id IN (sqlc.slice('ids'))
ORDER BY course_code ASC;

-- name: GetTranscriptGradeByID :one
SELECT * FROM student_course_grades
WHERE id = ?;

-- name: GetTranscriptGradeByUserAndCourse :one
SELECT * FROM student_course_grades
WHERE user_id = ? AND course_code = ?;

-- name: UpsertTranscriptGrade :one
INSERT INTO student_course_grades (
    id, user_id, course_code, course_name, credits, score_10, score_4,
    grade_letter, is_passed, is_retake, term_taken, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now')
)
ON CONFLICT(user_id, course_code) DO UPDATE SET
    course_name = excluded.course_name,
    credits = excluded.credits,
    score_10 = excluded.score_10,
    score_4 = excluded.score_4,
    grade_letter = excluded.grade_letter,
    is_passed = excluded.is_passed,
    is_retake = excluded.is_retake,
    term_taken = excluded.term_taken,
    updated_at = datetime('now')
RETURNING *;

-- name: DeleteTranscriptGrade :exec
DELETE FROM student_course_grades
WHERE id = ?;

-- name: DeleteTranscriptByUserID :exec
DELETE FROM student_course_grades
WHERE user_id = ?;

-- name: CountTranscriptGradesByUserID :one
SELECT COUNT(*) FROM student_course_grades
WHERE user_id = ?;

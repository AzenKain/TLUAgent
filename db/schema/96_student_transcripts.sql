-- Student Transcripts and Academic Course Grades Schema

CREATE TABLE IF NOT EXISTS student_course_grades (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_code TEXT NOT NULL,
    course_name TEXT NOT NULL DEFAULT '',
    credits REAL NOT NULL DEFAULT 0.0,
    score_10 REAL NOT NULL DEFAULT 0.0,
    score_4 REAL NOT NULL DEFAULT 0.0,
    grade_letter TEXT NOT NULL DEFAULT '',
    is_passed INTEGER NOT NULL DEFAULT 0,
    is_retake INTEGER NOT NULL DEFAULT 0,
    term_taken TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, course_code)
);

CREATE INDEX IF NOT EXISTS idx_student_course_grades_user ON student_course_grades(user_id);
CREATE INDEX IF NOT EXISTS idx_student_course_grades_user_passed ON student_course_grades(user_id, is_passed);

INSERT INTO permissions (key, description) VALUES
    ('transcript.read', 'Xem bảng điểm học vụ sinh viên'),
    ('transcript.manage', 'Cập nhật và ghi đè bảng điểm học vụ sinh viên')
ON CONFLICT(key) DO UPDATE SET description = excluded.description;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-admin-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('transcript.read', 'transcript.manage')
WHERE r.name = 'ADMIN'
ON CONFLICT(role_id, permission_key) DO NOTHING;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-advisor-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('transcript.read', 'transcript.manage')
WHERE r.name = 'ADVISOR'
ON CONFLICT(role_id, permission_key) DO NOTHING;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-student-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('transcript.read', 'transcript.manage')
WHERE r.name = 'STUDENT'
ON CONFLICT(role_id, permission_key) DO NOTHING;

-- Advisory Inquiries, Self-Evolution, and Student Notifications Schema

CREATE TABLE IF NOT EXISTS advisory_inquiries (
    id VARCHAR(64) PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id TEXT REFERENCES chat_conversations(id) ON DELETE SET NULL,
    student_name TEXT NOT NULL DEFAULT '',
    student_code TEXT NOT NULL DEFAULT '',
    student_class TEXT NOT NULL DEFAULT '',
    question TEXT NOT NULL,
    context TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    teacher_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    teacher_name TEXT NOT NULL DEFAULT '',
    teacher_reply TEXT NOT NULL DEFAULT '',
    answered_at DATETIME,
    knowledge_chunk_id VARCHAR(64) REFERENCES document_chunks(id) ON DELETE SET NULL,
    superseded_by_doc_id VARCHAR(64) REFERENCES documents(id) ON DELETE SET NULL,
    is_expired INTEGER NOT NULL DEFAULT 0,
    expired_reason TEXT NOT NULL DEFAULT '',
    expired_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_advisory_inquiries_status ON advisory_inquiries(status);
CREATE INDEX IF NOT EXISTS idx_advisory_inquiries_user ON advisory_inquiries(user_id);
CREATE INDEX IF NOT EXISTS idx_advisory_inquiries_teacher ON advisory_inquiries(teacher_id);
CREATE INDEX IF NOT EXISTS idx_advisory_inquiries_chunk ON advisory_inquiries(knowledge_chunk_id);
CREATE INDEX IF NOT EXISTS idx_advisory_inquiries_expired ON advisory_inquiries(is_expired);

CREATE TABLE IF NOT EXISTS user_notifications (
    id VARCHAR(64) PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    inquiry_id VARCHAR(64) REFERENCES advisory_inquiries(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    type VARCHAR(32) NOT NULL DEFAULT 'INQUIRY_ANSWERED',
    is_read INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON user_notifications(user_id, is_read);
CREATE INDEX IF NOT EXISTS idx_notifications_created ON user_notifications(created_at DESC);

INSERT INTO permissions (key, description) VALUES
    ('inquiry.read', 'Xem câu hỏi và phản hồi cố vấn học tập'),
    ('inquiry.create', 'Gửi câu hỏi tới cố vấn học tập'),
    ('inquiry.answer', 'Giải đáp câu hỏi và cập nhật tri thức cố vấn'),
    ('inquiry.manage', 'Quản trị các giải đáp và vòng đời tri thức')
ON CONFLICT(key) DO UPDATE SET description = excluded.description;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-admin-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('inquiry.read', 'inquiry.create', 'inquiry.answer', 'inquiry.manage')
WHERE r.name = 'ADMIN'
ON CONFLICT(role_id, permission_key) DO NOTHING;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-advisor-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('inquiry.read', 'inquiry.create', 'inquiry.answer')
WHERE r.name = 'ADVISOR'
ON CONFLICT(role_id, permission_key) DO NOTHING;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-student-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('inquiry.read', 'inquiry.create')
WHERE r.name = 'STUDENT'
ON CONFLICT(role_id, permission_key) DO NOTHING;

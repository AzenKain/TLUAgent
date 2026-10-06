-- Student Academic Profiles and Long-Term Memory Schema

CREATE TABLE IF NOT EXISTS student_academic_profiles (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    major TEXT NOT NULL DEFAULT '',
    cohort TEXT NOT NULL DEFAULT '',
    academic_standing TEXT NOT NULL DEFAULT 'NORMAL',
    completed_credits INTEGER NOT NULL DEFAULT 0,
    cumulative_gpa REAL NOT NULL DEFAULT 0.0,
    target_gpa REAL NOT NULL DEFAULT 0.0,
    advisor_notes TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_memories (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category TEXT NOT NULL DEFAULT 'GENERAL',
    memory_key TEXT NOT NULL,
    memory_value TEXT NOT NULL,
    confidence REAL NOT NULL DEFAULT 1.0,
    source TEXT NOT NULL DEFAULT 'CHAT',
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, memory_key)
);

CREATE INDEX IF NOT EXISTS idx_user_memories_user ON user_memories(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_user_memories_category ON user_memories(user_id, category);

INSERT INTO permissions (key, description) VALUES
    ('memory.read', 'Xem hồ sơ ký ức học vụ sinh viên'),
    ('memory.manage', 'Quản trị hồ sơ và ký ức dài hạn của sinh viên')
ON CONFLICT(key) DO UPDATE SET description = excluded.description;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-admin-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('memory.read', 'memory.manage')
WHERE r.name = 'ADMIN'
ON CONFLICT(role_id, permission_key) DO NOTHING;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-advisor-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('memory.read', 'memory.manage')
WHERE r.name = 'ADVISOR'
ON CONFLICT(role_id, permission_key) DO NOTHING;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-student-memory.read', r.id, 'memory.read', 'allow', '{}'
FROM roles r
WHERE r.name = 'STUDENT'
ON CONFLICT(role_id, permission_key) DO NOTHING;

-- TLUClaw Sessions, Messages, and Action Snapshots Schema

CREATE TABLE IF NOT EXISTS tluclaw_sessions (
    id TEXT PRIMARY KEY,
    admin_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL DEFAULT 'Phiên điều hành mới',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tluclaw_messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES tluclaw_sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    reasoning_content TEXT NOT NULL DEFAULT '',
    tool_calls_json TEXT NOT NULL DEFAULT '',
    tool_call_id TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tluclaw_action_snapshots (
    id TEXT PRIMARY KEY,
    batch_id TEXT NOT NULL,
    session_id TEXT NOT NULL REFERENCES tluclaw_sessions(id) ON DELETE CASCADE,
    operator_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action_type TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    diff_preview_json TEXT NOT NULL DEFAULT '{}',
    blast_radius_json TEXT NOT NULL DEFAULT '{}',
    previous_state_json TEXT NOT NULL DEFAULT '{}',
    new_state_json TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'PENDING',
    confirmation_nonce TEXT NOT NULL DEFAULT '',
    nonce_expires_at DATETIME,
    applied_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tluclaw_sessions_admin ON tluclaw_sessions(admin_user_id);
CREATE INDEX IF NOT EXISTS idx_tluclaw_messages_session ON tluclaw_messages(session_id, created_at);
CREATE INDEX IF NOT EXISTS idx_tluclaw_snapshots_batch ON tluclaw_action_snapshots(batch_id);
CREATE INDEX IF NOT EXISTS idx_tluclaw_snapshots_session ON tluclaw_action_snapshots(session_id);
CREATE INDEX IF NOT EXISTS idx_tluclaw_snapshots_nonce ON tluclaw_action_snapshots(confirmation_nonce);
CREATE INDEX IF NOT EXISTS idx_tluclaw_snapshots_status ON tluclaw_action_snapshots(status);

INSERT INTO permissions (key, description) VALUES
    ('tluclaw.read', 'Truy cập và tra cứu thông tin qua Admin TLUClaw'),
    ('tluclaw.manage', 'Thực thi thay đổi dữ liệu học vụ qua Admin TLUClaw')
ON CONFLICT(key) DO UPDATE SET description = excluded.description;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-admin-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('tluclaw.read', 'tluclaw.manage')
WHERE r.name = 'ADMIN'
ON CONFLICT(role_id, permission_key) DO NOTHING;

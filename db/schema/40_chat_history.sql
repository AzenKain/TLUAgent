-- Chat Conversations & Message History
CREATE TABLE IF NOT EXISTS chat_conversations (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    model_id TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chat_conversations_user_updated ON chat_conversations(user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_conversations_updated ON chat_conversations(updated_at DESC);

CREATE TABLE IF NOT EXISTS chat_messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    sender TEXT NOT NULL CHECK(sender IN ('user', 'assistant')),
    content TEXT NOT NULL,
    sources_json TEXT NOT NULL DEFAULT '[]',
    images_json TEXT NOT NULL DEFAULT '[]',
    feedback TEXT CHECK(feedback IN ('up', 'down', NULL)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_conv_created ON chat_messages(conversation_id, created_at ASC);

-- Insert Permission for Chat Management
INSERT INTO permissions (key, description) VALUES
    ('chat.manage', 'Quản trị và thanh tra toàn bộ lịch sử tư vấn')
ON CONFLICT(key) DO UPDATE SET description = excluded.description;

-- Grant chat.manage to ADMIN
INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-admin-chat.manage', r.id, 'chat.manage', 'allow', '{}'
FROM roles r
WHERE r.name = 'ADMIN'
ON CONFLICT(role_id, permission_key) DO NOTHING;

-- Grant chat.manage to ADVISOR
INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-advisor-chat.manage', r.id, 'chat.manage', 'allow', '{}'
FROM roles r
WHERE r.name = 'ADVISOR'
ON CONFLICT(role_id, permission_key) DO NOTHING;

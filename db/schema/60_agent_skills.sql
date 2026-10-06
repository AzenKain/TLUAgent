CREATE TABLE IF NOT EXISTS agent_prompts (
    id VARCHAR(32) PRIMARY KEY,
    title VARCHAR(128) NOT NULL,
    content TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(64)
);

CREATE INDEX IF NOT EXISTS idx_agent_prompts_active ON agent_prompts(is_active);

CREATE TABLE IF NOT EXISTS agent_skills (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL,
    content TEXT NOT NULL,
    tool_definition TEXT,
    is_enabled INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(64)
);

CREATE INDEX IF NOT EXISTS idx_agent_skills_enabled ON agent_skills(is_enabled, priority DESC);

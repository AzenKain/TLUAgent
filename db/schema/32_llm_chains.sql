CREATE TABLE IF NOT EXISTS llm_chains (
    id TEXT PRIMARY KEY,
    chain_type TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_active INTEGER NOT NULL DEFAULT 1,
    failure_threshold INTEGER NOT NULL DEFAULT 3,
    cooldown_seconds INTEGER NOT NULL DEFAULT 300,
    retry_count INTEGER NOT NULL DEFAULT 3,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS llm_chain_nodes (
    id TEXT PRIMARY KEY,
    chain_id TEXT NOT NULL REFERENCES llm_chains(id) ON DELETE CASCADE,
    model_id TEXT NOT NULL REFERENCES llm_models(id) ON DELETE CASCADE,
    priority INTEGER NOT NULL DEFAULT 1,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(chain_id, model_id)
);

CREATE INDEX IF NOT EXISTS idx_chain_nodes_chain_priority ON llm_chain_nodes(chain_id, is_active, priority);
CREATE INDEX IF NOT EXISTS idx_llm_chains_type ON llm_chains(chain_type, is_active);

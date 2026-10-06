CREATE TABLE IF NOT EXISTS llm_providers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    provider_type TEXT NOT NULL,
    base_url TEXT NOT NULL,
    api_key_ciphertext TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1,
    is_default INTEGER NOT NULL DEFAULT 0,
    custom_headers_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS llm_models (
    id TEXT PRIMARY KEY,
    provider_id TEXT NOT NULL REFERENCES llm_providers(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    model_key TEXT NOT NULL,
    model_type TEXT NOT NULL DEFAULT 'chat',
    is_active INTEGER NOT NULL DEFAULT 1,
    is_default INTEGER NOT NULL DEFAULT 0,
    order_index INTEGER NOT NULL DEFAULT 0,
    context_length INTEGER NOT NULL DEFAULT 4096,
    vision_mode TEXT NOT NULL DEFAULT 'manual',
    supports_vision INTEGER NOT NULL DEFAULT 0,
    max_images INTEGER NOT NULL DEFAULT 0,
    thinking_enabled INTEGER NOT NULL DEFAULT 0,
    thinking_budget INTEGER NOT NULL DEFAULT 0,
    effort_level TEXT NOT NULL DEFAULT 'medium',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_llm_models_provider_id ON llm_models (provider_id);
CREATE INDEX IF NOT EXISTS idx_llm_models_active_type ON llm_models (model_type, is_active, order_index);

CREATE TRIGGER IF NOT EXISTS trigger_llm_providers_updated_at
AFTER UPDATE ON llm_providers
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE llm_providers SET updated_at = datetime('now') WHERE id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS trigger_llm_models_updated_at
AFTER UPDATE ON llm_models
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE llm_models SET updated_at = datetime('now') WHERE id = OLD.id;
END;

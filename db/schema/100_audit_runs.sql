CREATE TABLE IF NOT EXISTS supersession_audit_runs (
    id            TEXT PRIMARY KEY,
    status        TEXT NOT NULL DEFAULT 'RUNNING',
    cursor_doc_id TEXT NOT NULL DEFAULT '',
    total_docs    INTEGER NOT NULL DEFAULT 0,
    processed     INTEGER NOT NULL DEFAULT 0,
    last_error    TEXT NOT NULL DEFAULT '',
    started_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_runs_status ON supersession_audit_runs(status);

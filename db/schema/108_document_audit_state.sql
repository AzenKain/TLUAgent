CREATE TABLE IF NOT EXISTS document_audit_state (
    document_id TEXT PRIMARY KEY,
    fingerprint TEXT NOT NULL,
    audited_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

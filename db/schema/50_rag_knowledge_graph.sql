-- RAG Knowledge Base and Knowledge Graph Schema

CREATE TABLE IF NOT EXISTS documents (
    id VARCHAR(64) PRIMARY KEY,
    doc_code VARCHAR(128),
    type VARCHAR(32) NOT NULL,
    title TEXT NOT NULL,
    domain VARCHAR(32) NOT NULL,
    validity_tier VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    superseded_by_id VARCHAR(64),
    department_code VARCHAR(32) NOT NULL,
    publish_date TEXT NOT NULL,
    effective_from TEXT,
    effective_to TEXT,
    academic_year VARCHAR(16) NOT NULL,
    semester VARCHAR(16) NOT NULL,
    applicable_cohort VARCHAR(64) NOT NULL DEFAULT 'ALL',
    priority_level INTEGER NOT NULL DEFAULT 50,
    file_path TEXT NOT NULL,
    char_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
CREATE INDEX IF NOT EXISTS idx_documents_domain ON documents(domain);
CREATE INDEX IF NOT EXISTS idx_documents_cohort ON documents(applicable_cohort);
CREATE INDEX IF NOT EXISTS idx_documents_priority ON documents(priority_level DESC);

CREATE TABLE IF NOT EXISTS document_rules (
    id VARCHAR(64) PRIMARY KEY,
    document_id VARCHAR(64) NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    rule_index INTEGER NOT NULL,
    chapter VARCHAR(128),
    article_number VARCHAR(32),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    superseded_by_id VARCHAR(64),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_doc_rules_doc_id ON document_rules(document_id);

CREATE TABLE IF NOT EXISTS document_chunks (
    id VARCHAR(64) PRIMARY KEY,
    document_id VARCHAR(64) NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    rule_id VARCHAR(64),
    chunk_index INTEGER NOT NULL,
    chunk_type VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    domain VARCHAR(32) NOT NULL,
    applicable_cohort VARCHAR(64) NOT NULL DEFAULT 'ALL',
    academic_year VARCHAR(16) NOT NULL,
    priority_level INTEGER NOT NULL DEFAULT 50,
    embedding BLOB,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chunks_doc_id ON document_chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_chunks_status ON document_chunks(status);
CREATE INDEX IF NOT EXISTS idx_chunks_cohort ON document_chunks(applicable_cohort);
CREATE INDEX IF NOT EXISTS idx_chunks_domain ON document_chunks(domain);

CREATE VIRTUAL TABLE IF NOT EXISTS document_chunks_fts USING fts5(
    chunk_id UNINDEXED,
    content,
    domain UNINDEXED,
    applicable_cohort UNINDEXED,
    status UNINDEXED,
    tokenize='unicode61'
);

CREATE TRIGGER IF NOT EXISTS trg_chunks_ai AFTER INSERT ON document_chunks
BEGIN
    INSERT INTO document_chunks_fts(chunk_id, content, domain, applicable_cohort, status)
    VALUES (new.id, new.content, new.domain, new.applicable_cohort, new.status);
END;

CREATE TRIGGER IF NOT EXISTS trg_chunks_ad AFTER DELETE ON document_chunks
BEGIN
    DELETE FROM document_chunks_fts WHERE chunk_id = old.id;
END;

CREATE TRIGGER IF NOT EXISTS trg_chunks_au AFTER UPDATE ON document_chunks
BEGIN
    DELETE FROM document_chunks_fts WHERE chunk_id = old.id;
    INSERT INTO document_chunks_fts(chunk_id, content, domain, applicable_cohort, status)
    VALUES (new.id, new.content, new.domain, new.applicable_cohort, new.status);
END;

CREATE TABLE IF NOT EXISTS kg_nodes (
    id VARCHAR(64) PRIMARY KEY,
    node_type VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(64),
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_kg_nodes_type ON kg_nodes(node_type);
CREATE INDEX IF NOT EXISTS idx_kg_nodes_status ON kg_nodes(status);

CREATE TABLE IF NOT EXISTS kg_edges (
    id VARCHAR(64) PRIMARY KEY,
    source_node_id VARCHAR(64) NOT NULL REFERENCES kg_nodes(id) ON DELETE CASCADE,
    target_node_id VARCHAR(64) NOT NULL REFERENCES kg_nodes(id) ON DELETE CASCADE,
    relation_type VARCHAR(64) NOT NULL,
    weight REAL DEFAULT 1.0,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_kg_edges_src_rel ON kg_edges(source_node_id, relation_type, status);
CREATE INDEX IF NOT EXISTS idx_kg_edges_tgt_rel ON kg_edges(target_node_id, relation_type, status);

CREATE TABLE IF NOT EXISTS supersession_audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    active_doc_id VARCHAR(64) NOT NULL,
    superseded_doc_id VARCHAR(64) NOT NULL,
    rule_type VARCHAR(64) NOT NULL,
    reason TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO permissions (key, description) VALUES
    ('rag.read', 'Tra cứu cơ sở tri thức RAG và đồ thị học vụ'),
    ('rag.manage', 'Quản trị nạp tài liệu và cấu trúc đồ thị RAG')
ON CONFLICT(key) DO UPDATE SET description = excluded.description;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-admin-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('rag.read', 'rag.manage')
WHERE r.name = 'ADMIN'
ON CONFLICT(role_id, permission_key) DO NOTHING;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-advisor-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('rag.read')
WHERE r.name = 'ADVISOR'
ON CONFLICT(role_id, permission_key) DO NOTHING;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-student-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('rag.read')
WHERE r.name = 'STUDENT'
ON CONFLICT(role_id, permission_key) DO NOTHING;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-guest-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('rag.read')
WHERE r.name = 'GUEST'
ON CONFLICT(role_id, permission_key) DO NOTHING;

CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    progress INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    error_msg TEXT,
    payload_json TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs(type);
CREATE INDEX IF NOT EXISTS idx_jobs_created ON jobs(created_at DESC);

CREATE TABLE IF NOT EXISTS job_schedules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    task_type TEXT NOT NULL,
    payload_json TEXT,
    interval_minutes INTEGER NOT NULL CHECK (interval_minutes >= 1),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    next_run_at DATETIME NOT NULL,
    last_run_at DATETIME,
    last_job_id TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_job_schedules_enabled_next ON job_schedules(enabled, next_run_at);

INSERT INTO permissions (key, description) VALUES
    ('job.read', 'Xem tiến độ và lịch sử tác vụ nền'),
    ('job.manage', 'Quản lý lịch chạy và kích hoạt tác vụ nền')
ON CONFLICT(key) DO UPDATE SET description = excluded.description;

INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-admin-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('job.read', 'job.manage')
WHERE r.name = 'ADMIN'
ON CONFLICT(role_id, permission_key) DO NOTHING;

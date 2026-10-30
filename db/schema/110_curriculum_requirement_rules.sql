CREATE TABLE IF NOT EXISTS curriculum_requirement_rules (
    id TEXT PRIMARY KEY,
    cohort TEXT NOT NULL,
    major_name TEXT NOT NULL,
    rule_type TEXT NOT NULL,
    group_key TEXT NOT NULL DEFAULT '',
    min_credits REAL NOT NULL DEFAULT 0,
    course_codes TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_requirement_rules_cohort_major ON curriculum_requirement_rules (cohort, major_name);

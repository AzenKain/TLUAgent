CREATE TABLE IF NOT EXISTS curriculums (
    id TEXT PRIMARY KEY,
    cohort TEXT NOT NULL,
    academic_year TEXT NOT NULL,
    major_name TEXT NOT NULL,
    major_code TEXT NOT NULL DEFAULT '',
    faculty_code TEXT NOT NULL DEFAULT '',
    aliases TEXT NOT NULL DEFAULT '',
    total_courses INTEGER NOT NULL DEFAULT 0,
    total_credits REAL NOT NULL DEFAULT 0.0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_curriculums_cohort ON curriculums (cohort);
CREATE INDEX IF NOT EXISTS idx_curriculums_major ON curriculums (major_name);
CREATE INDEX IF NOT EXISTS idx_curriculums_major_code ON curriculums (major_code);
CREATE INDEX IF NOT EXISTS idx_curriculums_faculty_code ON curriculums (faculty_code);
CREATE UNIQUE INDEX IF NOT EXISTS idx_curriculums_cohort_major ON curriculums (cohort, major_name);

CREATE TABLE IF NOT EXISTS curriculum_courses (
    id TEXT PRIMARY KEY,
    curriculum_id TEXT NOT NULL REFERENCES curriculums(id) ON DELETE CASCADE,
    cohort TEXT NOT NULL,
    major_name TEXT NOT NULL,
    course_code TEXT NOT NULL,
    course_name TEXT NOT NULL,
    credits REAL NOT NULL DEFAULT 0.0,
    weight REAL NOT NULL DEFAULT 1.0,
    hours TEXT NOT NULL DEFAULT '',
    knowledge_block TEXT NOT NULL DEFAULT '',
    prerequisites_text TEXT NOT NULL DEFAULT '',
    suggested_semester INTEGER NOT NULL DEFAULT 0,
    is_elearning INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_curriculum_courses_curr_id ON curriculum_courses (curriculum_id);
CREATE INDEX IF NOT EXISTS idx_curriculum_courses_code ON curriculum_courses (course_code);
CREATE INDEX IF NOT EXISTS idx_curriculum_courses_cohort_code ON curriculum_courses (cohort, course_code);
CREATE INDEX IF NOT EXISTS idx_curriculum_courses_cohort_major ON curriculum_courses (cohort, major_name);
CREATE INDEX IF NOT EXISTS idx_curriculum_courses_elearning ON curriculum_courses (cohort, is_elearning);

CREATE TABLE IF NOT EXISTS course_prerequisites (
    id TEXT PRIMARY KEY,
    curriculum_course_id TEXT NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
    cohort TEXT NOT NULL,
    major_name TEXT NOT NULL,
    course_code TEXT NOT NULL,
    prerequisite_course_code TEXT NOT NULL,
    min_grade REAL NOT NULL DEFAULT 0.0,
    condition_type TEXT NOT NULL DEFAULT 'PREREQUISITE',
    raw_expression TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_course_prereq_course ON course_prerequisites (cohort, course_code);
CREATE INDEX IF NOT EXISTS idx_course_prereq_target ON course_prerequisites (cohort, prerequisite_course_code);

CREATE TABLE IF NOT EXISTS course_substitutions (
    id TEXT PRIMARY KEY,
    original_course_code TEXT NOT NULL,
    original_course_name TEXT NOT NULL DEFAULT '',
    replacement_course_code TEXT NOT NULL,
    replacement_course_name TEXT NOT NULL DEFAULT '',
    scope_cohort TEXT NOT NULL,
    scope_major TEXT NOT NULL,
    min_cohort INTEGER NOT NULL DEFAULT 0,
    max_cohort INTEGER NOT NULL DEFAULT 999,
    is_bidirectional INTEGER NOT NULL DEFAULT 1,
    is_transitive INTEGER NOT NULL DEFAULT 0,
    credits_original REAL NOT NULL DEFAULT 0.0,
    credits_replacement REAL NOT NULL DEFAULT 0.0,
    rule_type TEXT NOT NULL DEFAULT 'DIRECT_PAIR',
    rule_note TEXT NOT NULL DEFAULT '',
    source_reference TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_course_sub_orig ON course_substitutions (original_course_code, scope_cohort);
CREATE INDEX IF NOT EXISTS idx_course_sub_repl ON course_substitutions (replacement_course_code, scope_cohort);
CREATE INDEX IF NOT EXISTS idx_course_sub_lookup ON course_substitutions (original_course_code, scope_major, scope_cohort);
CREATE INDEX IF NOT EXISTS idx_course_sub_cohort_range ON course_substitutions (original_course_code, min_cohort, max_cohort);

CREATE TABLE IF NOT EXISTS course_group_substitutions (
    id TEXT PRIMARY KEY,
    target_description TEXT NOT NULL,
    replacement_rule_description TEXT NOT NULL,
    scope_cohort TEXT NOT NULL,
    scope_major TEXT NOT NULL,
    min_cohort INTEGER NOT NULL DEFAULT 0,
    max_cohort INTEGER NOT NULL DEFAULT 999,
    allowed_prefixes TEXT NOT NULL DEFAULT '',
    source_reference TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_course_group_sub_scope ON course_group_substitutions (scope_cohort, scope_major);

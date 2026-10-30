-- name: UpsertCurriculum :one
INSERT INTO curriculums (
    id, cohort, academic_year, major_name, major_code, faculty_code, aliases, total_courses, total_credits, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now')
)
ON CONFLICT(cohort, major_name) DO UPDATE SET
    academic_year = excluded.academic_year,
    major_code = excluded.major_code,
    faculty_code = excluded.faculty_code,
    aliases = excluded.aliases,
    total_courses = excluded.total_courses,
    total_credits = excluded.total_credits,
    updated_at = datetime('now')
RETURNING *;

-- name: GetCurriculumByID :one
SELECT * FROM curriculums WHERE id = ? LIMIT 1;

-- name: GetCurriculumByCohortAndMajor :one
SELECT * FROM curriculums WHERE cohort = ? AND major_name = ? LIMIT 1;

-- name: ListCurriculumsByCohort :many
SELECT * FROM curriculums WHERE cohort = ? ORDER BY major_name ASC;

-- name: ListCurriculums :many
SELECT * FROM curriculums ORDER BY cohort DESC, major_name ASC;

-- name: CountCurriculums :one
SELECT COUNT(*) FROM curriculums;

-- name: UpsertCurriculumCourse :one
INSERT INTO curriculum_courses (
    id, curriculum_id, cohort, major_name, course_code, course_name, credits, weight,
    hours, knowledge_block, prerequisites_text, suggested_semester, is_elearning, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now')
)
ON CONFLICT(id) DO UPDATE SET
    curriculum_id = excluded.curriculum_id,
    cohort = excluded.cohort,
    major_name = excluded.major_name,
    course_code = excluded.course_code,
    course_name = excluded.course_name,
    credits = excluded.credits,
    weight = excluded.weight,
    hours = excluded.hours,
    knowledge_block = excluded.knowledge_block,
    prerequisites_text = excluded.prerequisites_text,
    suggested_semester = excluded.suggested_semester,
    is_elearning = excluded.is_elearning,
    updated_at = datetime('now')
RETURNING *;

-- name: GetCurriculumCourseByID :one
SELECT * FROM curriculum_courses WHERE id = ? LIMIT 1;

-- name: GetCurriculumCoursesByCurriculumID :many
SELECT * FROM curriculum_courses WHERE curriculum_id = ? ORDER BY knowledge_block, course_code;

-- name: GetCurriculumCoursesByCohortAndMajor :many
SELECT * FROM curriculum_courses WHERE cohort = ? AND major_name = ? ORDER BY knowledge_block, course_code;

-- name: GetCoursesByCodeAndCohort :many
SELECT * FROM curriculum_courses WHERE cohort = ? AND course_code = ?;

-- name: SearchCurriculumCourses :many
SELECT * FROM curriculum_courses 
WHERE (cohort = sqlc.arg('cohort') OR sqlc.arg('cohort') = '') 
  AND (major_name LIKE '%' || sqlc.arg('major_name') || '%' OR sqlc.arg('major_name') = '') 
  AND (course_code LIKE '%' || sqlc.arg('keyword') || '%' OR course_name LIKE '%' || sqlc.arg('keyword') || '%')
ORDER BY cohort DESC, course_code ASC
LIMIT 50;

-- name: CountCurriculumCourses :one
SELECT COUNT(*) FROM curriculum_courses;

-- name: UpsertCoursePrerequisite :one
INSERT INTO course_prerequisites (
    id, curriculum_course_id, cohort, major_name, course_code, prerequisite_course_code,
    min_grade, condition_type, raw_expression, created_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now')
)
ON CONFLICT(id) DO UPDATE SET
    min_grade = excluded.min_grade,
    condition_type = excluded.condition_type,
    raw_expression = excluded.raw_expression
RETURNING *;

-- name: ListPrerequisitesForCourse :many
SELECT * FROM course_prerequisites 
WHERE (sqlc.arg('cohort') = '' OR cohort = sqlc.arg('cohort')) 
  AND course_code = sqlc.arg('course_code')
ORDER BY prerequisite_course_code ASC;

-- name: ListDependentCourses :many
SELECT * FROM course_prerequisites 
WHERE (sqlc.arg('cohort') = '' OR cohort = sqlc.arg('cohort')) 
  AND prerequisite_course_code = sqlc.arg('prerequisite_course_code')
ORDER BY course_code ASC;

-- name: UpsertCourseSubstitution :one
INSERT INTO course_substitutions (
    id, original_course_code, original_course_name, replacement_course_code, replacement_course_name,
    scope_cohort, scope_major, min_cohort, max_cohort, is_bidirectional, is_transitive,
    credits_original, credits_replacement, rule_type, rule_note, source_reference, created_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now')
)
ON CONFLICT(id) DO UPDATE SET
    original_course_code = excluded.original_course_code,
    original_course_name = excluded.original_course_name,
    replacement_course_code = excluded.replacement_course_code,
    replacement_course_name = excluded.replacement_course_name,
    scope_cohort = excluded.scope_cohort,
    scope_major = excluded.scope_major,
    min_cohort = excluded.min_cohort,
    max_cohort = excluded.max_cohort,
    is_bidirectional = excluded.is_bidirectional,
    is_transitive = excluded.is_transitive,
    credits_original = excluded.credits_original,
    credits_replacement = excluded.credits_replacement,
    rule_type = excluded.rule_type,
    rule_note = excluded.rule_note,
    source_reference = excluded.source_reference
RETURNING *;

-- name: FindDirectSubstitutions :many
SELECT * FROM course_substitutions 
WHERE original_course_code = sqlc.arg('original_course_code') 
  AND (CAST(sqlc.arg('cohort_num') AS INTEGER) = 0 OR (min_cohort <= CAST(sqlc.arg('cohort_num') AS INTEGER) AND max_cohort >= CAST(sqlc.arg('cohort_num') AS INTEGER)))
  AND (CAST(sqlc.arg('scope_major') AS TEXT) = '' OR scope_major = CAST(sqlc.arg('scope_major') AS TEXT) OR scope_major = CAST(sqlc.arg('faculty_code') AS TEXT) OR scope_major = 'ALL')
ORDER BY replacement_course_code ASC;

-- name: ListSubstitutionsByCohortAndMajor :many
SELECT * FROM course_substitutions 
WHERE (CAST(sqlc.arg('cohort_num') AS INTEGER) = 0 OR (min_cohort <= CAST(sqlc.arg('cohort_num') AS INTEGER) AND max_cohort >= CAST(sqlc.arg('cohort_num') AS INTEGER)))
  AND (CAST(sqlc.arg('scope_major') AS TEXT) = '' OR scope_major = CAST(sqlc.arg('scope_major') AS TEXT) OR scope_major = CAST(sqlc.arg('faculty_code') AS TEXT) OR scope_major = 'ALL')
ORDER BY original_course_code ASC, replacement_course_code ASC;

-- name: CountSubstitutions :one
SELECT COUNT(*) FROM course_substitutions;

-- name: CountCoursePrerequisites :one
SELECT COUNT(*) FROM course_prerequisites;

-- name: UpsertCourseGroupSubstitution :one
INSERT INTO course_group_substitutions (
    id, target_description, replacement_rule_description, scope_cohort, scope_major,
    min_cohort, max_cohort, allowed_prefixes, source_reference, created_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now')
)
ON CONFLICT(id) DO UPDATE SET
    target_description = excluded.target_description,
    replacement_rule_description = excluded.replacement_rule_description,
    scope_cohort = excluded.scope_cohort,
    scope_major = excluded.scope_major,
    min_cohort = excluded.min_cohort,
    max_cohort = excluded.max_cohort,
    allowed_prefixes = excluded.allowed_prefixes,
    source_reference = excluded.source_reference
RETURNING *;

-- name: ListGroupSubstitutions :many
SELECT * FROM course_group_substitutions 
WHERE (CAST(sqlc.arg('cohort_num') AS INTEGER) = 0 OR (min_cohort <= CAST(sqlc.arg('cohort_num') AS INTEGER) AND max_cohort >= CAST(sqlc.arg('cohort_num') AS INTEGER)))
  AND (CAST(sqlc.arg('scope_major') AS TEXT) = '' 
       OR scope_major = CAST(sqlc.arg('scope_major') AS TEXT) 
       OR scope_major = CAST(sqlc.arg('faculty_code') AS TEXT) 
       OR scope_major = 'ALL'
       OR (CAST(sqlc.arg('major_aliases') AS TEXT) != '' AND instr(',' || CAST(sqlc.arg('major_aliases') AS TEXT) || ',', ',' || scope_major || ',') > 0))
ORDER BY id ASC;

-- name: ListCohorts :many
SELECT DISTINCT cohort FROM curriculums ORDER BY cohort DESC;

-- name: ListMajorsByCohort :many
SELECT DISTINCT major_name, major_code FROM curriculums WHERE cohort = ? ORDER BY major_name ASC;

-- name: ListAllMajors :many
SELECT DISTINCT major_name, major_code FROM curriculums ORDER BY major_name ASC;

-- name: ListCoursesPaginated :many
SELECT * FROM curriculum_courses 
WHERE (cohort = sqlc.arg('cohort') OR sqlc.arg('cohort') = '') 
  AND (major_name = sqlc.arg('major_name') OR sqlc.arg('major_name') = '') 
  AND (knowledge_block = sqlc.arg('knowledge_block') OR sqlc.arg('knowledge_block') = '')
  AND (suggested_semester = sqlc.arg('suggested_semester') OR sqlc.arg('suggested_semester') = 0)
  AND (is_elearning = sqlc.arg('is_elearning') OR sqlc.arg('is_elearning') = -1)
  AND (CAST(sqlc.arg('keyword') AS text) = '' OR course_code LIKE '%' || CAST(sqlc.arg('keyword') AS text) || '%' OR course_name LIKE '%' || CAST(sqlc.arg('keyword') AS text) || '%')
ORDER BY cohort DESC, suggested_semester ASC, course_code ASC
LIMIT sqlc.arg('limit_val') OFFSET sqlc.arg('offset_val');

-- name: CountCoursesFiltered :one
SELECT COUNT(*) FROM curriculum_courses 
WHERE (cohort = sqlc.arg('cohort') OR sqlc.arg('cohort') = '') 
  AND (major_name = sqlc.arg('major_name') OR sqlc.arg('major_name') = '') 
  AND (knowledge_block = sqlc.arg('knowledge_block') OR sqlc.arg('knowledge_block') = '')
  AND (suggested_semester = sqlc.arg('suggested_semester') OR sqlc.arg('suggested_semester') = 0)
  AND (is_elearning = sqlc.arg('is_elearning') OR sqlc.arg('is_elearning') = -1)
  AND (CAST(sqlc.arg('keyword') AS text) = '' OR course_code LIKE '%' || CAST(sqlc.arg('keyword') AS text) || '%' OR course_name LIKE '%' || CAST(sqlc.arg('keyword') AS text) || '%');

-- name: UpdateCurriculumCourse :one
UPDATE curriculum_courses SET
    course_name = sqlc.arg('course_name'),
    credits = sqlc.arg('credits'),
    weight = sqlc.arg('weight'),
    hours = sqlc.arg('hours'),
    knowledge_block = sqlc.arg('knowledge_block'),
    prerequisites_text = sqlc.arg('prerequisites_text'),
    suggested_semester = sqlc.arg('suggested_semester'),
    is_elearning = sqlc.arg('is_elearning'),
    updated_at = datetime('now')
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteCurriculumCourse :exec
DELETE FROM curriculum_courses WHERE id = ?;

-- name: DeleteCurriculum :exec
DELETE FROM curriculums WHERE id = ?;

-- name: ListSubstitutionsPaginated :many
SELECT * FROM course_substitutions 
WHERE (CAST(sqlc.arg('cohort_num') AS INTEGER) = 0 OR (min_cohort <= CAST(sqlc.arg('cohort_num') AS INTEGER) AND max_cohort >= CAST(sqlc.arg('cohort_num') AS INTEGER)))
  AND (CAST(sqlc.arg('scope_major') AS TEXT) = '' 
       OR scope_major = CAST(sqlc.arg('scope_major') AS TEXT) 
       OR scope_major = CAST(sqlc.arg('faculty_code') AS TEXT) 
       OR scope_major = 'ALL'
       OR (CAST(sqlc.arg('major_aliases') AS TEXT) != '' AND instr(',' || CAST(sqlc.arg('major_aliases') AS TEXT) || ',', ',' || scope_major || ',') > 0))
  AND (CAST(sqlc.arg('keyword') AS text) = '' 
       OR original_course_code LIKE '%' || CAST(sqlc.arg('keyword') AS text) || '%' 
       OR replacement_course_code LIKE '%' || CAST(sqlc.arg('keyword') AS text) || '%'
       OR original_course_name LIKE '%' || CAST(sqlc.arg('keyword') AS text) || '%' 
       OR replacement_course_name LIKE '%' || CAST(sqlc.arg('keyword') AS text) || '%')
ORDER BY min_cohort DESC, original_course_code ASC
LIMIT sqlc.arg('limit_val') OFFSET sqlc.arg('offset_val');

-- name: CountSubstitutionsFiltered :one
SELECT COUNT(*) FROM course_substitutions 
WHERE (CAST(sqlc.arg('cohort_num') AS INTEGER) = 0 OR (min_cohort <= CAST(sqlc.arg('cohort_num') AS INTEGER) AND max_cohort >= CAST(sqlc.arg('cohort_num') AS INTEGER)))
  AND (CAST(sqlc.arg('scope_major') AS TEXT) = '' 
       OR scope_major = CAST(sqlc.arg('scope_major') AS TEXT) 
       OR scope_major = CAST(sqlc.arg('faculty_code') AS TEXT) 
       OR scope_major = 'ALL'
       OR (CAST(sqlc.arg('major_aliases') AS TEXT) != '' AND instr(',' || CAST(sqlc.arg('major_aliases') AS TEXT) || ',', ',' || scope_major || ',') > 0))
  AND (CAST(sqlc.arg('keyword') AS text) = '' 
       OR original_course_code LIKE '%' || CAST(sqlc.arg('keyword') AS text) || '%' 
       OR replacement_course_code LIKE '%' || CAST(sqlc.arg('keyword') AS text) || '%'
       OR original_course_name LIKE '%' || CAST(sqlc.arg('keyword') AS text) || '%' 
       OR replacement_course_name LIKE '%' || CAST(sqlc.arg('keyword') AS text) || '%');

-- name: GetSubstitutionByID :one
SELECT * FROM course_substitutions WHERE id = ? LIMIT 1;

-- name: DeleteCourseSubstitution :execrows
DELETE FROM course_substitutions WHERE id = ?;

-- name: ListAllPrerequisitesForCohortAndMajor :many
SELECT cp.* FROM course_prerequisites cp
WHERE cp.cohort = sqlc.arg('cohort') 
  AND (cp.major_name = sqlc.arg('major_name') OR sqlc.arg('major_name') = '')
ORDER BY cp.course_code ASC;

-- name: ListAllMajorMetadata :many
SELECT DISTINCT cohort, major_name, major_code, faculty_code, aliases
FROM curriculums
ORDER BY cohort DESC, major_name ASC;

-- name: ListMajorsByFaculty :many
SELECT DISTINCT major_code FROM curriculums
WHERE faculty_code = sqlc.arg('faculty_code') AND major_code != ''
ORDER BY major_code ASC;

-- name: FindMajorMapping :one
SELECT major_name, major_code, faculty_code, aliases FROM curriculums 
WHERE (cohort = sqlc.arg('cohort') OR sqlc.arg('cohort') = '')
  AND (major_code = sqlc.arg('major') 
       OR major_name = sqlc.arg('major')
       OR (aliases != '' AND (',' || aliases || ',' LIKE '%,' || sqlc.arg('major') || ',%')))
ORDER BY CASE WHEN cohort = sqlc.arg('cohort') THEN 0 ELSE 1 END
LIMIT 1;

-- name: ListDistinctSubstitutionMajors :many
SELECT DISTINCT scope_major FROM course_substitutions 
WHERE (CAST(sqlc.arg('cohort_num') AS INTEGER) = 0 OR (min_cohort <= CAST(sqlc.arg('cohort_num') AS INTEGER) AND max_cohort >= CAST(sqlc.arg('cohort_num') AS INTEGER)))
ORDER BY scope_major ASC;

-- name: FindCourseMetadataByCode :one
SELECT course_code, course_name, credits, knowledge_block FROM curriculum_courses
WHERE course_code = sqlc.arg('course_code')
ORDER BY cohort DESC
LIMIT 1;

-- name: UpdateMajorMetadataByName :exec
UPDATE curriculums
SET major_code = sqlc.arg('major_code'),
    faculty_code = sqlc.arg('faculty_code'),
    aliases = sqlc.arg('aliases'),
    updated_at = datetime('now')
WHERE major_name = sqlc.arg('major_name') OR (major_code != '' AND major_code = sqlc.arg('major_code'));

-- name: DeleteCoursePrerequisite :exec
DELETE FROM course_prerequisites WHERE id = ?;

-- name: GetCoursePrerequisiteByID :one
SELECT * FROM course_prerequisites WHERE id = ? LIMIT 1;



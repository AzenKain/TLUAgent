-- name: ListRequirementRules :many
SELECT id, cohort, major_name, rule_type, group_key, min_credits, course_codes, note, source, created_at
FROM curriculum_requirement_rules
WHERE cohort = ? AND major_name = ?
ORDER BY rule_type, group_key, id;

-- name: DeleteRequirementRules :exec
DELETE FROM curriculum_requirement_rules WHERE cohort = ? AND major_name = ?;

-- name: InsertRequirementRule :exec
INSERT INTO curriculum_requirement_rules (id, cohort, major_name, rule_type, group_key, min_credits, course_codes, note, source)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

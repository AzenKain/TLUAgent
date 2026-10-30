package services

import (
	"testing"

	"tluagent-web/internal/models"
)

func courseSet(codes map[string]float64) (map[string]*models.CurriculumCourse, []*models.CurriculumCourse) {
	byCode := map[string]*models.CurriculumCourse{}
	var list []*models.CurriculumCourse
	for _, code := range []string{"A1", "A2", "L1", "L2", "Z1", "Z2", "E1", "E2", "E3", "T1", "T2"} {
		if credits, ok := codes[code]; ok {
			c := &models.CurriculumCourse{CourseCode: code, Credits: credits}
			byCode[code] = c
			list = append(list, c)
		}
	}
	return byCode, list
}

func keptCodes(kept []*models.CurriculumCourse) map[string]bool {
	out := map[string]bool{}
	for _, c := range kept {
		out[c.CourseCode] = true
	}
	return out
}

func TestApplyRequirementRulesChoosesOneAlternativeAndFillsToTotal(t *testing.T) {
	byCode, all := courseSet(map[string]float64{"A1": 3, "A2": 3, "L1": 2, "L2": 2, "Z1": 2, "Z2": 2, "E1": 3, "E2": 3, "E3": 3, "T1": 6, "T2": 6})
	rules := []*models.RequirementRule{
		{RuleType: models.RequirementChooseOne, CourseCodes: "L1,L2|Z1,Z2"},
		{RuleType: models.RequirementChooseOne, CourseCodes: "T1|T2"},
		{RuleType: models.RequirementElectivePool, MinCredits: 3, CourseCodes: "E1,E2,E3"},
		{RuleType: models.RequirementTotalMin, MinCredits: 21},
	}
	out := applyRequirementRules(rules, all, map[string]bool{"Z1": true}, 2, byCode, nil)
	kept := keptCodes(out.Kept)

	if kept["L1"] || kept["L2"] || !kept["Z2"] {
		t.Fatalf("the language pair already started must be finished and the other pair dropped: %v", kept)
	}
	if kept["T2"] && kept["T1"] {
		t.Fatalf("only one thesis alternative may remain: %v", kept)
	}
	var total float64 = 2
	for _, c := range out.Kept {
		total += c.Credits
	}
	if total < 21 {
		t.Fatalf("the plan must reach the required total, got %.1f", total)
	}
}

func TestApplyRequirementRulesWithoutRulesKeepsEverything(t *testing.T) {
	_, all := courseSet(map[string]float64{"A1": 3, "E1": 3})
	out := applyRequirementRules(nil, all, map[string]bool{}, 0, nil, nil)
	if len(out.Kept) != 2 {
		t.Fatalf("no rules means no filtering, got %d courses", len(out.Kept))
	}
}

func TestApplyRequirementRulesSkipsPoolWhenAlreadySatisfied(t *testing.T) {
	byCode, all := courseSet(map[string]float64{"E1": 3, "E2": 3, "E3": 3})
	passed := map[string]bool{"E1": true}
	remaining := all[1:]
	rules := []*models.RequirementRule{{RuleType: models.RequirementElectivePool, MinCredits: 3, CourseCodes: "E1,E2,E3"}}
	out := applyRequirementRules(rules, remaining, passed, 3, byCode, nil)
	if len(out.Kept) != 0 {
		t.Fatalf("a satisfied pool must not add more courses, got %v", keptCodes(out.Kept))
	}
}

func TestApplyRequirementRulesUsesCoursesOutsideTheCurriculumAsDone(t *testing.T) {
	byCode, all := courseSet(map[string]float64{"T1": 2})
	rules := []*models.RequirementRule{{RuleType: models.RequirementChooseOne, CourseCodes: "T1|T9"}}
	out := applyRequirementRules(rules, all, map[string]bool{"T9": true}, 3, byCode, nil)
	if len(out.Kept) != 0 {
		t.Fatalf("an alternative finished with a course unknown to the curriculum must satisfy the group, kept %v", keptCodes(out.Kept))
	}

	unfinished := applyRequirementRules(rules, all, map[string]bool{}, 0, byCode, nil)
	if len(unfinished.Kept) != 1 {
		t.Fatalf("an unfinished alternative that contains unknown courses must not be chosen, kept %v", keptCodes(unfinished.Kept))
	}
}

func TestApplyRequirementRulesCountsOutsideAndExcessCreditsAsFreeElectives(t *testing.T) {
	byCode, all := courseSet(map[string]float64{"E1": 3, "E2": 3, "E3": 3})
	rules := []*models.RequirementRule{
		{RuleType: models.RequirementElectivePool, MinCredits: 3, CourseCodes: "E1,E2,E3"},
		{RuleType: models.RequirementFreeElective, MinCredits: 6},
		{RuleType: models.RequirementNonCredit, CourseCodes: "PG*"},
	}

	outside := map[string]float64{"X9": 3, "PG106": 1}
	passed := map[string]bool{"X9": true, "PG106": true, "E1": true, "E2": true}
	out := applyRequirementRules(rules, all[2:], passed, 9, byCode, outside)
	if len(out.Kept) != 0 {
		t.Fatalf("an outside course and an excess pool course (3+3) already cover 6 free credits, got %v", keptCodes(out.Kept))
	}

	short := applyRequirementRules(rules, all[1:], map[string]bool{"E1": true, "PG106": true}, 4, byCode, map[string]float64{"PG106": 1})
	if len(short.Kept) != 2 {
		t.Fatalf("physical education credits do not count, so 6 free credits are still needed from the leftover pool, got %v", keptCodes(short.Kept))
	}
}

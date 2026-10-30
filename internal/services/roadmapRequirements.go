package services

import (
	"fmt"
	"sort"
	"strings"

	"tluagent-web/internal/models"
)

type requirementOutcome struct {
	Kept      []*models.CurriculumCourse
	Notes     []string
	TotalMin  float64
	Satisfied bool
}

func passedCredits(codes []string, passed map[string]bool, byCode map[string]*models.CurriculumCourse) float64 {
	var sum float64
	for _, code := range codes {
		if c, ok := byCode[code]; ok && passed[code] {
			sum += c.Credits
		}
	}
	return sum
}

func missingCredits(codes []string, passed map[string]bool, byCode map[string]*models.CurriculumCourse) float64 {
	var sum float64
	for _, code := range codes {
		if c, ok := byCode[code]; ok && !passed[code] {
			sum += c.Credits
		}
	}
	return sum
}

func pickChooseOne(rule *models.RequirementRule, passed map[string]bool, byCode map[string]*models.CurriculumCourse, drop map[string]bool) {
	sets := rule.CodeSets()
	if len(sets) < 2 {
		return
	}

	best, bestMissing, bestProgress, bestDone := -1, 0.0, 0.0, false
	for i, set := range sets {
		allPassed, unknownOpen := true, false
		for _, code := range set {
			if passed[code] {
				continue
			}
			allPassed = false
			if _, known := byCode[code]; !known {
				unknownOpen = true
			}
		}
		if unknownOpen {
			continue
		}
		missing := missingCredits(set, passed, byCode)
		progress := passedCredits(set, passed, byCode)
		better := best < 0 ||
			(allPassed && !bestDone) ||
			(allPassed == bestDone && (missing < bestMissing || (missing == bestMissing && progress > bestProgress)))
		if better {
			best, bestMissing, bestProgress, bestDone = i, missing, progress, allPassed
		}
	}
	if best < 0 {
		return
	}

	keep := map[string]bool{}
	for _, code := range sets[best] {
		keep[code] = true
	}
	for i, set := range sets {
		if i == best {
			continue
		}
		for _, code := range set {
			if !keep[code] {
				drop[code] = true
			}
		}
	}
}

// applyRequirementRules narrows the remaining courses to what is still needed to graduate under the curriculum's requirement rules.
func applyRequirementRules(
	rules []*models.RequirementRule,
	remaining []*models.CurriculumCourse,
	passed map[string]bool,
	accumulated float64,
	byCode map[string]*models.CurriculumCourse,
	outsideCredits map[string]float64,
) requirementOutcome {
	if len(rules) == 0 {
		return requirementOutcome{Kept: remaining}
	}

	drop := map[string]bool{}
	excluded := map[string]bool{}
	var totalMin float64
	var pools, frees []*models.RequirementRule
	var nonCredit []string

	for _, rule := range rules {
		switch rule.RuleType {
		case models.RequirementChooseOne:
			pickChooseOne(rule, passed, byCode, drop)
		case models.RequirementElectivePool:
			pools = append(pools, rule)
		case models.RequirementFreeElective:
			frees = append(frees, rule)
		case models.RequirementNonCredit:
			for _, set := range rule.CodeSets() {
				nonCredit = append(nonCredit, set...)
			}
		case models.RequirementTotalMin:
			totalMin = rule.MinCredits
			for _, set := range rule.CodeSets() {
				for _, code := range set {
					excluded[code] = true
				}
			}
		}
	}

	isNonCredit := func(code string) bool {
		for _, pattern := range nonCredit {
			if models.MatchesCode(pattern, code) {
				return true
			}
		}
		return false
	}

	remainingByCode := make(map[string]*models.CurriculumCourse, len(remaining))
	for _, c := range remaining {
		remainingByCode[c.CourseCode] = c
	}

	var freeEarned float64
	for code, credits := range outsideCredits {
		if passed[code] && !isNonCredit(code) {
			freeEarned += credits
		}
	}
	for code := range drop {
		if c, ok := byCode[code]; ok && passed[code] && !isNonCredit(code) {
			freeEarned += c.Credits
		}
	}

	var deferred []*models.CurriculumCourse
	var notes []string
	for _, pool := range pools {
		var codes []string
		for _, set := range pool.CodeSets() {
			codes = append(codes, set...)
		}
		earned := passedCredits(codes, passed, byCode)
		if earned > pool.MinCredits {
			freeEarned += earned - pool.MinCredits
		}
		need := pool.MinCredits - earned
		for _, code := range codes {
			c, ok := remainingByCode[code]
			if !ok || drop[code] {
				continue
			}
			if need > 0 {
				need -= c.Credits
				continue
			}
			drop[code] = true
			deferred = append(deferred, c)
		}
		if need > 0 {
			notes = append(notes, fmt.Sprintf("pool %s is short by %.1f credits and has no more courses to offer", pool.GroupKey, need))
		}
	}

	sort.SliceStable(deferred, func(i, j int) bool { return deferred[i].Credits > deferred[j].Credits })
	var promoted []*models.CurriculumCourse
	for _, free := range frees {
		need := free.MinCredits - freeEarned
		for _, c := range deferred {
			if need <= 0 {
				break
			}
			if drop[c.CourseCode] {
				drop[c.CourseCode] = false
				promoted = append(promoted, c)
				need -= c.Credits
			}
		}
		freeEarned = free.MinCredits - need
		if need > 0 {
			notes = append(notes, fmt.Sprintf("choose %.1f more free-elective credits from any course offered by the university", need))
		}
	}
	deferred = deferred[:0]
	for _, c := range remaining {
		if drop[c.CourseCode] {
			deferred = append(deferred, c)
		}
	}

	kept := make([]*models.CurriculumCourse, 0, len(remaining))
	for _, c := range remaining {
		if !drop[c.CourseCode] {
			kept = append(kept, c)
		}
	}

	if totalMin > 0 {
		counted := accumulated
		for code := range excluded {
			if c, ok := byCode[code]; ok && passed[code] {
				counted -= c.Credits
			}
		}
		for _, c := range kept {
			if !excluded[c.CourseCode] {
				counted += c.Credits
			}
		}
		for _, c := range deferred {
			if counted >= totalMin {
				break
			}
			kept = append(kept, c)
			counted += c.Credits
		}
		if counted < totalMin {
			notes = append(notes, fmt.Sprintf("the curriculum can only reach %.1f of the %.1f required credits", counted, totalMin))
		}
	}

	return requirementOutcome{Kept: kept, Notes: notes, TotalMin: totalMin, Satisfied: true}
}

func describeRequirements(outcome requirementOutcome, droppedCredits float64) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Graduation requirement model applied: %.1f optional or alternative credits are not scheduled", droppedCredits)
	if outcome.TotalMin > 0 {
		fmt.Fprintf(&sb, "; the minimum is %.1f credits excluding the listed physical education and defence courses", outcome.TotalMin)
	}
	return sb.String()
}

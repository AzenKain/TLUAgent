package services_test

import (
	"context"
	"strings"
	"testing"

	"tluagent-web/internal/models"
	"tluagent-web/internal/services"
)

func TestOptimizeRoadmapPlacesRequestedCoursesInSummer(t *testing.T) {
	db, repo := setupRoadmapDB(t)
	defer db.Close()

	optimizer := services.NewRoadmapOptimizer(repo)
	res, err := optimizer.OptimizeRoadmap(context.Background(), &models.RoadmapOptimizationRequest{
		Cohort:               "K40",
		MajorName:            "CNTT",
		CurrentSemester:      1,
		TargetGraduationTerm: 8,
		SummerCourses:        []string{" cs102 "},
	})
	if err != nil {
		t.Fatalf("optimize failed: %v", err)
	}

	found := false
	for _, sem := range res.Semesters {
		for _, c := range sem.Courses {
			if strings.EqualFold(c.CourseCode, "CS102") {
				found = true
				if !sem.IsSummer {
					t.Fatalf("CS102 must be scheduled in a summer term, got semester %d", sem.SemesterNumber)
				}
			}
		}
	}
	if !found {
		t.Fatal("CS102 must still be scheduled")
	}
}

func TestOptimizeRoadmapTargetTermChangesPace(t *testing.T) {
	db, repo := setupRoadmapDB(t)
	defer db.Close()
	optimizer := services.NewRoadmapOptimizer(repo)

	span := func(target int) float64 {
		res, err := optimizer.OptimizeRoadmap(context.Background(), &models.RoadmapOptimizationRequest{
			Cohort: "K40", MajorName: "CNTT", CurrentSemester: 1, TargetGraduationTerm: target,
		})
		if err != nil {
			t.Fatalf("optimize failed: %v", err)
		}
		var first float64
		if len(res.Semesters) > 0 {
			first = res.Semesters[0].TotalCredits
		}
		return first
	}
	if fast, slow := span(2), span(12); fast < slow {
		t.Fatalf("a tighter target must not schedule a lighter first semester: fast=%v slow=%v", fast, slow)
	}
}

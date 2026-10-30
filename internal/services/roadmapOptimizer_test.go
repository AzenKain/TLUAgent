package services_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

// TestRoadmapOptimizer verifies DAG scheduling, critical paths, rule waterfalls, and pacing feedback.
func TestRoadmapOptimizer(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	ramCache := cache.NewTheineCache(10 << 20)
	repo := repositories.NewCurriculumRepository(db, ramCache)
	optimizer := services.NewRoadmapOptimizer(repo)
	ctx := context.Background()

	_, err = repo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
		ID:           "curr_k35_cntt",
		Cohort:       "K35",
		AcademicYear: "2022-2023",
		MajorCode:    "CNTT",
		MajorName:    "Công nghệ thông tin",
		TotalCourses: 10,
		TotalCredits: 35.0,
	})
	if err != nil {
		t.Fatalf("UpsertCurriculum failed: %v", err)
	}

	testCourses := []struct {
		id          string
		code        string
		name        string
		credits     float64
		isElearning int64
		sem         int64
	}{
		{"c_cs102", "CS102", "Tin học văn phòng", 3.0, 1, 1},
		{"c_ge111", "GE111", "Tiếng Anh 1", 3.0, 0, 1},
		{"c_ge112", "GE112", "Tiếng Anh 2", 3.0, 0, 2},
		{"c_ge221", "GE221", "Tiếng Anh 3", 3.0, 0, 3},
		{"c_ml113", "ML113", "Triết học Mác - Lênin", 3.0, 1, 1},
		{"c_sh131", "SH131", "Pháp luật đại cương", 2.0, 1, 2},
		{"c_pg121", "PG121", "Giáo dục quốc phòng", 4.0, 0, 2},
		{"c_cs111", "CS111", "Cấu trúc rời rạc", 3.0, 0, 1},
		{"c_cs122", "CS122", "Cấu trúc dữ liệu và giải thuật", 4.0, 0, 2},
		{"c_cs211", "CS211", "Thiết kế giải thuật", 3.0, 0, 3},
		{"c_te201", "TE201", "Kỹ thuật phần mềm", 3.0, 0, 3},
		{"c_te301", "TE301", "Kiểm thử phần mềm", 3.0, 0, 4},
	}

	for _, tc := range testCourses {
		_, err = repo.UpsertCurriculumCourse(ctx, sqlc.UpsertCurriculumCourseParams{
			ID:                tc.id,
			CurriculumID:      "curr_k35_cntt",
			Cohort:            "K35",
			MajorName:         "Công nghệ thông tin",
			CourseCode:        tc.code,
			CourseName:        tc.name,
			Credits:           tc.credits,
			Weight:            1.0,
			Hours:             "45",
			KnowledgeBlock:    "Cơ sở ngành",
			PrerequisitesText: "",
			SuggestedSemester: tc.sem,
			IsElearning:       tc.isElearning,
		})
		if err != nil {
			t.Fatalf("Upsert course %s failed: %v", tc.code, err)
		}
	}

	prereqs := []struct {
		id     string
		course string
		target string
	}{
		{"p1", "CS122", "CS111"},
		{"p2", "CS211", "CS122"},
		{"p3", "TE201", "CS122"},
		{"p4", "TE301", "TE201"},
		{"p5", "GE112", "GE111"},
		{"p6", "GE221", "GE112"},
	}

	for _, p := range prereqs {
		courseID := "c_" + strings.ToLower(p.course)
		_, err = repo.UpsertCoursePrerequisite(ctx, sqlc.UpsertCoursePrerequisiteParams{
			ID:                     p.id,
			CurriculumCourseID:     courseID,
			Cohort:                 "K35",
			MajorName:              "Công nghệ thông tin",
			CourseCode:             p.course,
			PrerequisiteCourseCode: p.target,
			MinGrade:               4.0,
			ConditionType:          "PREREQUISITE",
		})
		if err != nil {
			t.Fatalf("Upsert prereq %s -> %s failed: %v", p.target, p.course, err)
		}
	}

	t.Run("EvaluateDiagnosticQuiz", func(t *testing.T) {
		res, err := optimizer.EvaluateDiagnosticQuiz(ctx, &models.PersonalityQuizAnswers{
			PacePreference:     "fast",
			PrimaryStrength:    "programming",
			DeliveryPreference: "elearning",
			SummerWillingness:  true,
		})
		if err != nil {
			t.Fatalf("EvaluateDiagnosticQuiz failed: %v", err)
		}
		if res.TargetCreditsPerSemester != 19.0 {
			t.Errorf("Expected 19.0 target credits, got %f", res.TargetCreditsPerSemester)
		}
		if res.RecommendedGraduationTerm != 7 {
			t.Errorf("Expected term 7 for fast pace, got %d", res.RecommendedGraduationTerm)
		}
		if res.CompetencyRadar["programming"] != 90.0 {
			t.Errorf("Expected 90.0 programming strength, got %f", res.CompetencyRadar["programming"])
		}
	})

	t.Run("OptimizeRoadmap_Freshman", func(t *testing.T) {
		req := &models.RoadmapOptimizationRequest{
			Cohort:               "K35",
			MajorName:            "Công nghệ thông tin",
			CurrentSemester:      0,
			TargetGraduationTerm: 8,
			Transcript:           []models.StudentCourseGrade{},
		}

		res, err := optimizer.OptimizeRoadmap(ctx, req)
		if err != nil {
			t.Fatalf("OptimizeRoadmap failed: %v", err)
		}

		if len(res.Semesters) == 0 {
			t.Fatalf("Expected scheduled semesters, got 0")
		}

		firstSem := res.Semesters[0]
		foundCS111 := false
		foundGE111 := false
		for _, c := range firstSem.Courses {
			if c.CourseCode == "CS111" {
				foundCS111 = true
			}
			if c.CourseCode == "GE111" {
				foundGE111 = true
			}
			if c.CourseCode == "CS122" {
				t.Errorf("CS122 scheduled in semester 1 before prerequisite CS111!")
			}
		}

		if !foundCS111 {
			t.Errorf("CS111 should be in semester 1 as foundation node")
		}
		if !foundGE111 {
			t.Errorf("GE111 should be in semester 1")
		}
	})

	t.Run("OptimizeRoadmap_RetakePrioritizationAndDowngradeWarning", func(t *testing.T) {
		req := &models.RoadmapOptimizationRequest{
			Cohort:               "K35",
			MajorName:            "Công nghệ thông tin",
			CurrentSemester:      1,
			TargetGraduationTerm: 8,
			Transcript: []models.StudentCourseGrade{
				{CourseCode: "CS111", Score: 2.5, Grade: "F", IsRetake: false},
				{CourseCode: "GE111", Score: 7.0, Grade: "B", IsRetake: false},
				{CourseCode: "EXTRA_CS", Score: 8.0, Grade: "A", Credits: 3.0},
				{CourseCode: "JA111", Score: 8.0, Grade: "B", Credits: 3.0},
				{CourseCode: "JA112", Score: 8.0, Grade: "B", Credits: 3.0},
				{CourseCode: "JA221", Score: 8.0, Grade: "B", Credits: 3.0},
			},
		}

		res, err := optimizer.OptimizeRoadmap(ctx, req)
		if err != nil {
			t.Fatalf("OptimizeRoadmap failed: %v", err)
		}

		nextSem := res.Semesters[0]
		foundRetakeCS111 := false
		for _, c := range nextSem.Courses {
			if c.CourseCode == "CS111" && c.IsRetake {
				foundRetakeCS111 = true
			}
		}
		if !foundRetakeCS111 {
			t.Errorf("CS111 was failed and should be scheduled with top priority in semester 2")
		}

		if len(res.FreeElectives.CrossMajorCourses) == 0 {
			t.Errorf("Expected EXTRA_CS to be detected in FreeElectives waterfall")
		}
		if len(res.FreeElectives.ExtraLanguages) == 0 {
			t.Errorf("Expected extra Japanese course to spill over into FreeElectives waterfall")
		}
	})

	t.Run("OptimizeRoadmap_CompletedAll", func(t *testing.T) {
		var allPassed []models.StudentCourseGrade
		for _, tc := range testCourses {
			allPassed = append(allPassed, models.StudentCourseGrade{
				CourseCode: tc.code,
				Score:      8.0,
				Grade:      "B",
			})
		}

		req := &models.RoadmapOptimizationRequest{
			Cohort:          "K35",
			MajorName:       "Công nghệ thông tin",
			CurrentSemester: 8,
			Transcript:      allPassed,
		}

		res, err := optimizer.OptimizeRoadmap(ctx, req)
		if err != nil {
			t.Fatalf("OptimizeRoadmap failed: %v", err)
		}
		if res.RemainingCredits != 0 {
			t.Errorf("Expected 0 remaining credits, got %f", res.RemainingCredits)
		}
		if len(res.Semesters) != 0 {
			t.Errorf("Expected 0 remaining semesters, got %d", len(res.Semesters))
		}
	})

	t.Run("OptimizeRoadmap_CourseSubstitution", func(t *testing.T) {
		_, err := repo.UpsertCourseSubstitution(ctx, sqlc.UpsertCourseSubstitutionParams{
			ID:                    "sub_cs111_alt",
			OriginalCourseCode:    "CS111",
			OriginalCourseName:    "Cấu trúc rời rạc",
			ReplacementCourseCode: "CS111_ALT",
			ReplacementCourseName: "Cấu trúc rời rạc mới",
			ScopeCohort:           "<=k35",
			ScopeMajor:            "CNTT",
			IsBidirectional:       1,
			IsTransitive:          0,
			CreditsOriginal:       3.0,
			CreditsReplacement:    3.0,
			RuleType:              "EQUIVALENT",
		})
		if err != nil {
			t.Fatalf("UpsertCourseSubstitution failed: %v", err)
		}

		req := &models.RoadmapOptimizationRequest{
			Cohort:          "K35",
			MajorName:       "Công nghệ thông tin",
			CurrentSemester: 1,
			Transcript: []models.StudentCourseGrade{
				{
					CourseCode: "CS111_ALT",
					Score:      8.5,
					Grade:      "A",
					Credits:    3.0,
				},
				{
					CourseCode: "GE111",
					Score:      8.0,
					Grade:      "B+",
					Credits:    3.0,
				},
			},
		}

		res, err := optimizer.OptimizeRoadmap(ctx, req)
		if err != nil {
			t.Fatalf("OptimizeRoadmap with substitution failed: %v", err)
		}

		for _, sem := range res.Semesters {
			for _, c := range sem.Courses {
				if c.CourseCode == "CS111" {
					t.Errorf("CS111 should be satisfied via CS111_ALT substitute, but was scheduled")
				}
			}
		}

		foundCS122 := false
		for _, sem := range res.Semesters {
			for _, c := range sem.Courses {
				if c.CourseCode == "CS122" {
					foundCS122 = true
					break
				}
			}
		}
		if !foundCS122 {
			t.Errorf("CS122 should be scheduled once CS111 was satisfied via substitution")
		}
	})
}

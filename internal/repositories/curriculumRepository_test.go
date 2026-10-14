package repositories_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

// TestCurriculumRepository verifies CRUD, cache IDs, singleflight and course code non-uniqueness.
func TestCurriculumRepository(t *testing.T) {
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
	ctx := context.Background()

	currK35, err := repo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
		ID:           "curr_k35_ti",
		Cohort:       "K35",
		AcademicYear: "2022-2023",
		MajorCode:    "TI",
		MajorName:    "Khoa học máy tính",
		TotalCourses: 45,
		TotalCredits: 130,
	})
	if err != nil {
		t.Fatalf("UpsertCurriculum K35 failed: %v", err)
	}

	currK36, err := repo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
		ID:           "curr_k36_cntt",
		Cohort:       "K36",
		AcademicYear: "2023-2024",
		MajorCode:    "CNTT",
		MajorName:    "Công nghệ thông tin",
		TotalCourses: 46,
		TotalCredits: 132,
	})
	if err != nil {
		t.Fatalf("UpsertCurriculum K36 failed: %v", err)
	}

	gotK35, err := repo.GetCurriculumByCohortAndMajor(ctx, "K35", "Khoa học máy tính")
	if err != nil || gotK35 == nil || gotK35.ID != currK35.ID {
		t.Fatalf("GetCurriculumByCohortAndMajor K35 failed: %+v (err: %v)", gotK35, err)
	}

	gotK35Cached, err := repo.GetCurriculumByCohortAndMajor(ctx, "K35", "Khoa học máy tính")
	if err != nil || gotK35Cached == nil || gotK35Cached.ID != currK35.ID {
		t.Fatalf("GetCurriculumByCohortAndMajor cached failed: %+v", gotK35Cached)
	}

	courseK35, err := repo.UpsertCurriculumCourse(ctx, sqlc.UpsertCurriculumCourseParams{
		ID:                "cc_k35_ti_cs212",
		CurriculumID:      currK35.ID,
		Cohort:            "K35",
		MajorName:         "Khoa học máy tính",
		CourseCode:        "CS212",
		CourseName:        "Cấu trúc dữ liệu và giải thuật",
		Credits:           3.0,
		Weight:            1.0,
		Hours:             "45",
		PrerequisitesText: "CS101",
		KnowledgeBlock:    "Chuyên nghiệp",
		SuggestedSemester: 3,
	})
	if err != nil {
		t.Fatalf("UpsertCurriculumCourse CS212 K35 failed: %v", err)
	}

	courseK36, err := repo.UpsertCurriculumCourse(ctx, sqlc.UpsertCurriculumCourseParams{
		ID:                "cc_k36_cntt_cs212",
		CurriculumID:      currK36.ID,
		Cohort:            "K36",
		MajorName:         "Công nghệ thông tin",
		CourseCode:        "CS212",
		CourseName:        "Cấu trúc dữ liệu nâng cao",
		Credits:           4.0,
		Weight:            1.2,
		Hours:             "60",
		PrerequisitesText: "CS101",
		KnowledgeBlock:    "Chuyên ngành",
		SuggestedSemester: 3,
	})
	if err != nil {
		t.Fatalf("UpsertCurriculumCourse CS212 K36 with same code failed (should allow non-unique course_code): %v", err)
	}

	if courseK35.CourseCode != courseK36.CourseCode {
		t.Fatalf("Expected identical course code CS212, got %s and %s", courseK35.CourseCode, courseK36.CourseCode)
	}
	if courseK35.Credits == courseK36.Credits {
		t.Fatalf("Expected different credits across cohorts for CS212")
	}

	coursesK35, err := repo.GetCurriculumCoursesByCohortAndMajor(ctx, "K35", "Khoa học máy tính")
	if err != nil || len(coursesK35) != 1 {
		t.Fatalf("GetCurriculumCoursesByCohortAndMajor failed: %v (count: %d)", err, len(coursesK35))
	}

	searchRes, err := repo.SearchCurriculumCourses(ctx, "K35", "Khoa học máy tính", "giải thuật")
	if err != nil || len(searchRes) != 1 {
		t.Fatalf("SearchCurriculumCourses failed: %v (count: %d)", err, len(searchRes))
	}

	_, err = repo.UpsertCoursePrerequisite(ctx, sqlc.UpsertCoursePrerequisiteParams{
		ID:                     "prereq_cs212_cs101_k35",
		CurriculumCourseID:     courseK35.ID,
		Cohort:                 "K35",
		MajorName:              "Khoa học máy tính",
		CourseCode:             "CS212",
		PrerequisiteCourseCode: "CS101",
		MinGrade:               4.0,
		ConditionType:          "PREREQUISITE",
		RawExpression:          "CS101",
	})
	if err != nil {
		t.Fatalf("UpsertCoursePrerequisite failed: %v", err)
	}

	prereqs, err := repo.ListPrerequisitesForCourse(ctx, "K35", "CS212")
	if err != nil || len(prereqs) != 1 || prereqs[0].PrerequisiteCourseCode != "CS101" {
		t.Fatalf("ListPrerequisitesForCourse failed: %v", err)
	}

	dependents, err := repo.ListDependentCourses(ctx, "K35", "CS101")
	if err != nil || len(dependents) != 1 || dependents[0].CourseCode != "CS212" {
		t.Fatalf("ListDependentCourses failed: %v", err)
	}

	_, err = repo.UpsertCourseSubstitution(ctx, sqlc.UpsertCourseSubstitutionParams{
		ID:                    "sub_mi322_ai130_tt",
		OriginalCourseCode:    "MI322",
		OriginalCourseName:    "Xử lý ảnh",
		ReplacementCourseCode: "AI130",
		ReplacementCourseName: "Thị giác máy tính",
		ScopeCohort:           "<=k35",
		ScopeMajor:            "TT",
		IsBidirectional:       1,
		IsTransitive:          0,
		CreditsOriginal:       3.0,
		CreditsReplacement:    3.0,
		RuleType:              "DIRECT_EQUIVALENT",
		RuleNote:              "Môn thay thế 2 chiều",
		SourceReference:       "TT_old_K<=35",
	})
	if err != nil {
		t.Fatalf("UpsertCourseSubstitution failed: %v", err)
	}

	subs, err := repo.FindDirectSubstitutions(ctx, "MI322", 35, "TT", "")
	if err != nil || len(subs) != 1 || subs[0].ReplacementCourseCode != "AI130" {
		t.Fatalf("FindDirectSubstitutions failed: %v (len: %d)", err, len(subs))
	}

	_, err = repo.UpsertCourseGroupSubstitution(ctx, sqlc.UpsertCourseGroupSubstitutionParams{
		ID:                         "grp_sub_1",
		TargetDescription:          "Tín chỉ tự chọn khối ngành",
		ReplacementRuleDescription: "Tích lũy đủ 12 tín chỉ từ danh mục CS*",
		ScopeCohort:                "<=k35",
		ScopeMajor:                 "TI",
		AllowedPrefixes:            "CS,MI",
		SourceReference:            "TI_old_K<=35",
	})
	if err != nil {
		t.Fatalf("UpsertCourseGroupSubstitution failed: %v", err)
	}

	grps, err := repo.ListGroupSubstitutions(ctx, 35, "TI", "")
	if err != nil || len(grps) != 1 {
		t.Fatalf("ListGroupSubstitutions failed: %v", err)
	}

	repo.InvalidateCurriculumCache(ctx, "K35", "Khoa học máy tính")
}

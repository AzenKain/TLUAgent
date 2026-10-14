package services_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

// TestCurriculumService verifies domain orchestration, substitutions, prerequisites and formatting.
func TestCurriculumService(t *testing.T) {
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
	svc := services.NewCurriculumService(repo)
	ctx := context.Background()

	_, err = repo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
		ID:           "curr_k35_tt",
		Cohort:       "K35",
		AcademicYear: "2022-2023",
		MajorCode:    "TT",
		MajorName:    "Trí tuệ nhân tạo",
		TotalCourses: 40,
		TotalCredits: 130,
	})
	if err != nil {
		t.Fatalf("UpsertCurriculum failed: %v", err)
	}

	c1, err := repo.UpsertCurriculumCourse(ctx, sqlc.UpsertCurriculumCourseParams{
		ID:                "cc_k35_tt_mi322",
		CurriculumID:      "curr_k35_tt",
		Cohort:            "K35",
		MajorName:         "Trí tuệ nhân tạo",
		CourseCode:        "MI322",
		CourseName:        "Xử lý ảnh và thị giác máy tính",
		Credits:           3.0,
		Weight:            1.0,
		Hours:             "45",
		PrerequisitesText: "MI201",
		KnowledgeBlock:    "Chuyên ngành",
		SuggestedSemester: 5,
	})
	if err != nil {
		t.Fatalf("UpsertCurriculumCourse MI322 failed: %v", err)
	}

	_, err = repo.UpsertCurriculumCourse(ctx, sqlc.UpsertCurriculumCourseParams{
		ID:                "cc_k35_tt_ai130",
		CurriculumID:      "curr_k35_tt",
		Cohort:            "K35",
		MajorName:         "Trí tuệ nhân tạo",
		CourseCode:        "AI130",
		CourseName:        "Thị giác máy tính",
		Credits:           3.0,
		Weight:            1.0,
		Hours:             "45",
		PrerequisitesText: "MI201",
		KnowledgeBlock:    "Chuyên ngành",
		SuggestedSemester: 5,
	})
	if err != nil {
		t.Fatalf("UpsertCurriculumCourse AI130 failed: %v", err)
	}

	curr, courses, err := svc.GetCurriculum(ctx, "K35", "Trí tuệ nhân tạo")
	if err != nil || curr == nil || len(courses) != 2 {
		t.Fatalf("GetCurriculum failed: %v (count: %d)", err, len(courses))
	}

	currFmt := svc.FormatCurriculumContext(courses)
	if !strings.Contains(currFmt, "MI322") || !strings.Contains(currFmt, "AI130") {
		t.Fatalf("FormatCurriculumContext missing course codes: %s", currFmt)
	}

	searchRes, err := svc.SearchCourses(ctx, "K35", "Trí tuệ nhân tạo", "thị giác")
	if err != nil || len(searchRes) != 2 {
		t.Fatalf("SearchCourses failed: %v (len: %d)", err, len(searchRes))
	}

	_, err = repo.UpsertCourseSubstitution(ctx, sqlc.UpsertCourseSubstitutionParams{
		ID:                    "sub_mi322_ai130_tt_1",
		OriginalCourseCode:    "MI322",
		OriginalCourseName:    "Xử lý ảnh và thị giác máy tính",
		ReplacementCourseCode: "AI130",
		ReplacementCourseName: "Thị giác máy tính",
		ScopeCohort:           "<=k35",
		ScopeMajor:            "TT",
		IsBidirectional:       1,
		IsTransitive:          0,
		CreditsOriginal:       3.0,
		CreditsReplacement:    3.0,
		RuleType:              "DIRECT_EQUIVALENT",
		RuleNote:              "Môn thay thế tương đương 2 chiều",
		SourceReference:       "TT_old_K<=35",
	})
	if err != nil {
		t.Fatalf("UpsertCourseSubstitution 1 failed: %v", err)
	}

	_, err = repo.UpsertCourseSubstitution(ctx, sqlc.UpsertCourseSubstitutionParams{
		ID:                    "sub_mi322_ai130_tt_2",
		OriginalCourseCode:    "AI130",
		OriginalCourseName:    "Thị giác máy tính",
		ReplacementCourseCode: "MI322",
		ReplacementCourseName: "Xử lý ảnh và thị giác máy tính",
		ScopeCohort:           "<=k35",
		ScopeMajor:            "TT",
		IsBidirectional:       1,
		IsTransitive:          0,
		CreditsOriginal:       3.0,
		CreditsReplacement:    3.0,
		RuleType:              "DIRECT_EQUIVALENT",
		RuleNote:              "Môn thay thế tương đương 2 chiều",
		SourceReference:       "TT_old_K<=35",
	})
	if err != nil {
		t.Fatalf("UpsertCourseSubstitution 2 failed: %v", err)
	}

	subsMI, _, err := svc.GetSubstitutions(ctx, "MI322", "K35", "Trí tuệ nhân tạo")
	if err != nil || len(subsMI) != 1 || subsMI[0].ReplacementCourseCode != "AI130" {
		t.Fatalf("GetSubstitutions for MI322 failed: %v (len: %d)", err, len(subsMI))
	}

	subsAI, _, err := svc.GetSubstitutions(ctx, "AI130", "K35", "Trí tuệ nhân tạo")
	if err != nil || len(subsAI) != 1 || subsAI[0].ReplacementCourseCode != "MI322" {
		t.Fatalf("GetSubstitutions for AI130 failed (bidirectional check): %v (len: %d)", err, len(subsAI))
	}

	subFmt := svc.FormatSubstitutionsContext("MI322", subsMI, nil)
	if !strings.Contains(subFmt, "AI130") || !strings.Contains(subFmt, "bắc cầu") {
		t.Fatalf("FormatSubstitutionsContext invalid: %s", subFmt)
	}

	_, err = repo.UpsertCoursePrerequisite(ctx, sqlc.UpsertCoursePrerequisiteParams{
		ID:                     "prereq_mi322_mi201_k35",
		CurriculumCourseID:     c1.ID,
		Cohort:                 "K35",
		MajorName:              "Trí tuệ nhân tạo",
		CourseCode:             "MI322",
		PrerequisiteCourseCode: "MI201",
		MinGrade:               4.0,
		ConditionType:          "PREREQUISITE",
		RawExpression:          "MI201",
	})
	if err != nil {
		t.Fatalf("UpsertCoursePrerequisite failed: %v", err)
	}

	prereqs, dependents, err := svc.GetPrerequisites(ctx, "MI322", "K35")
	if err != nil || len(prereqs) != 1 || prereqs[0].PrerequisiteCourseCode != "MI201" {
		t.Fatalf("GetPrerequisites for MI322 failed: %v (len: %d)", err, len(prereqs))
	}
	if len(dependents) != 0 {
		t.Fatalf("Expected 0 dependents for MI322, got %d", len(dependents))
	}

	prereqFmt := svc.FormatPrerequisitesContext("MI322", "K35", prereqs, dependents)
	if !strings.Contains(prereqFmt, "MI201") {
		t.Fatalf("FormatPrerequisitesContext invalid: %s", prereqFmt)
	}
}

// TestDynamicMajorAndCohortResolution verifies data-driven major alias matching and cohort ranges.
func TestDynamicMajorAndCohortResolution(t *testing.T) {
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
	svc := services.NewCurriculumService(repo)
	ctx := context.Background()

	_, err = repo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
		ID:           "curr_k24_tc",
		Cohort:       "K24",
		AcademicYear: "2011-2012",
		MajorCode:    "TC",
		MajorName:    "Mạng máy tính và viễn thông",
		FacultyCode:  "CNTT",
		Aliases:      "MMT,TC,VIEN THONG,DATACOMM",
		TotalCourses: 40,
		TotalCredits: 130,
	})
	if err != nil {
		t.Fatalf("UpsertCurriculum K24 TC failed: %v", err)
	}

	_, err = repo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
		ID:           "curr_k36_tc",
		Cohort:       "K36",
		AcademicYear: "2023-2024",
		MajorCode:    "TC",
		MajorName:    "Mạng máy tính và truyền thông dữ liệu",
		FacultyCode:  "CNTT",
		Aliases:      "MMT,TC,DATACOMM,TRUYEN THONG",
		TotalCourses: 42,
		TotalCredits: 135,
	})
	if err != nil {
		t.Fatalf("UpsertCurriculum K36 TC failed: %v", err)
	}

	_, err = repo.UpsertCourseSubstitution(ctx, sqlc.UpsertCourseSubstitutionParams{
		ID:                    "sub_net101_k35",
		OriginalCourseCode:    "NET101",
		OriginalCourseName:    "Mạng căn bản",
		ReplacementCourseCode: "NET102",
		ReplacementCourseName: "Mạng máy tính nâng cao",
		ScopeCohort:           "<=k35",
		ScopeMajor:            "TC",
		IsBidirectional:       1,
		IsTransitive:          0,
		CreditsOriginal:       3.0,
		CreditsReplacement:    3.0,
	})
	if err != nil {
		t.Fatalf("UpsertCourseSubstitution <=k35 failed: %v", err)
	}

	_, err = repo.UpsertCourseSubstitution(ctx, sqlc.UpsertCourseSubstitutionParams{
		ID:                    "sub_net101_k38_k40",
		OriginalCourseCode:    "NET101",
		OriginalCourseName:    "Mạng căn bản",
		ReplacementCourseCode: "NET201",
		ReplacementCourseName: "Mạng thế hệ mới",
		ScopeCohort:           "K38-K40",
		ScopeMajor:            "TC",
		IsBidirectional:       1,
		IsTransitive:          0,
		CreditsOriginal:       3.0,
		CreditsReplacement:    3.0,
	})
	if err != nil {
		t.Fatalf("UpsertCourseSubstitution K38-K40 failed: %v", err)
	}

	subsK24, _, err := svc.GetSubstitutions(ctx, "NET101", "K24", "MMT")
	if err != nil || len(subsK24) != 1 || subsK24[0].ReplacementCourseCode != "NET102" {
		t.Fatalf("K24 with alias MMT should match <=k35 rule: %v, len: %d", err, len(subsK24))
	}

	subsNoAccent, _, err := svc.GetSubstitutions(ctx, "NET101", "K24", "mang may tinh va vien thong")
	if err != nil || len(subsNoAccent) != 1 || subsNoAccent[0].ReplacementCourseCode != "NET102" {
		t.Fatalf("K24 with unaccented name should match <=k35 rule: %v, len: %d", err, len(subsNoAccent))
	}

	subsK39, _, err := svc.GetSubstitutions(ctx, "NET101", "K39", "TC")
	if err != nil || len(subsK39) != 1 || subsK39[0].ReplacementCourseCode != "NET201" {
		t.Fatalf("K39 with code TC should match K38-K40 rule: %v, len: %d", err, len(subsK39))
	}

	subsK36, _, err := svc.GetSubstitutions(ctx, "NET101", "K36", "TC")
	if err != nil || len(subsK36) != 0 {
		t.Fatalf("K36 should match 0 rules for NET101, got: %d", len(subsK36))
	}
}

package services_test

import (
	"context"
	"database/sql"
	"testing"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"

	_ "modernc.org/sqlite"
)

func setupRoadmapDB(t *testing.T) (*sql.DB, repositories.CurriculumRepository) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}
	repo := repositories.NewCurriculumRepository(db, cache.NewRamCache())
	if _, err := repo.UpsertCurriculum(context.Background(), sqlc.UpsertCurriculumParams{
		ID:           "CURR_K40_CNTT",
		Cohort:       "K40",
		AcademicYear: "2026",
		MajorName:    "CNTT",
		MajorCode:    "CNTT",
		FacultyCode:  "ALL",
	}); err != nil {
		t.Fatalf("failed to seed curriculum: %v", err)
	}
	courses := []struct {
		code   string
		block  string
		credit float64
	}{{"CS101", "Cơ sở ngành", 3}, {"CS102", "Cơ sở ngành", 3}, {"CS201", "Chuyên ngành", 4}}
	for i, c := range courses {
		if _, err := repo.UpsertCurriculumCourse(context.Background(), sqlc.UpsertCurriculumCourseParams{
			ID:                "c_" + c.code,
			CurriculumID:      "CURR_K40_CNTT",
			Cohort:            "K40",
			MajorName:         "CNTT",
			CourseCode:        c.code,
			CourseName:        c.code,
			Credits:           c.credit,
			Weight:            1,
			Hours:             "45",
			KnowledgeBlock:    c.block,
			SuggestedSemester: int64(i + 1),
		}); err != nil {
			t.Fatalf("failed to seed course: %v", err)
		}
	}
	return db, repo
}

func TestOptimizeRoadmapRequiresCohort(t *testing.T) {
	db, repo := setupRoadmapDB(t)
	defer db.Close()

	optimizer := services.NewRoadmapOptimizer(repo)
	_, err := optimizer.OptimizeRoadmap(context.Background(), &models.RoadmapOptimizationRequest{
		MajorName:            "CNTT",
		TargetGraduationTerm: 8,
		CurrentSemester:      1,
	})
	if err == nil {
		t.Fatalf("expected an error when cohort is missing")
	}
	if !errorsIsMissingCohort(err) {
		t.Fatalf("expected ErrMissingCohort, got %v", err)
	}
}

func errorsIsMissingCohort(err error) bool {
	type unwrapper interface{ Unwrap() error }
	for err != nil {
		if err == apperrors.ErrMissingCohort {
			return true
		}
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func TestOptimizeRoadmapDowngradeWarningOnlyForHonors(t *testing.T) {
	db, repo := setupRoadmapDB(t)
	defer db.Close()

	transcript := []models.StudentCourseGrade{
		{CourseCode: "CS101", Credits: 3, Grade: "F", Score: 2, IsRetake: true},
	}

	optimizerHonors := services.NewRoadmapOptimizer(repo)
	resHonors, err := optimizerHonors.OptimizeRoadmap(context.Background(), &models.RoadmapOptimizationRequest{
		Cohort:               "K40",
		MajorName:            "CNTT",
		CurrentSemester:      1,
		TargetGraduationTerm: 8,
		AcademicStanding:     "XUAT_SAC",
		Transcript:           transcript,
	})
	if err != nil {
		t.Fatalf("optimize failed: %v", err)
	}
	if !resHonors.DowngradeRisk.IsAtRisk || resHonors.DowngradeRisk.WarningMessage == "" {
		t.Fatalf("expected downgrade risk warning for honors student, got %+v", resHonors.DowngradeRisk)
	}

	resAverage, err := optimizerHonors.OptimizeRoadmap(context.Background(), &models.RoadmapOptimizationRequest{
		Cohort:               "K40",
		MajorName:            "CNTT",
		CurrentSemester:      1,
		TargetGraduationTerm: 8,
		AcademicStanding:     "TRUNG_BINH",
		Transcript:           transcript,
	})
	if err != nil {
		t.Fatalf("optimize failed: %v", err)
	}
	if resAverage.DowngradeRisk.IsAtRisk || resAverage.DowngradeRisk.WarningMessage != "" {
		t.Fatalf("expected no downgrade warning for average student, got %+v", resAverage.DowngradeRisk)
	}
	if resAverage.DowngradeRisk.Note == "" {
		t.Fatalf("expected an explanatory note for non-honors student")
	}
}

func TestOptimizeRoadmapAcademicStandingFromProfile(t *testing.T) {
	db, repo := setupRoadmapDB(t)
	defer db.Close()

	if _, err := db.Exec(`INSERT OR IGNORE INTO users (id, email, full_name, password_hash) VALUES ('stu1','s@tlu.edu.vn','S','h')`); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	memoryRepo := repositories.NewMemoryRepository(db, cache.NewRamCache())
	if _, err := memoryRepo.UpsertStudentProfile(context.Background(), &models.StudentProfile{
		UserID:           "stu1",
		Major:            "CNTT",
		Cohort:           "K40",
		AcademicStanding: "GIOI",
	}); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	optimizer := services.NewRoadmapOptimizer(repo, memoryRepo)
	res, err := optimizer.OptimizeRoadmap(context.Background(), &models.RoadmapOptimizationRequest{
		UserID:               "stu1",
		Cohort:               "K40",
		MajorName:            "CNTT",
		CurrentSemester:      1,
		TargetGraduationTerm: 8,
		Transcript: []models.StudentCourseGrade{
			{CourseCode: "CS101", Credits: 3, Grade: "F", Score: 2, IsRetake: true},
		},
	})
	if err != nil {
		t.Fatalf("optimize failed: %v", err)
	}
	if !res.DowngradeRisk.IsAtRisk {
		t.Fatalf("expected profile-based honors standing to trigger downgrade risk, got %+v", res.DowngradeRisk)
	}
}

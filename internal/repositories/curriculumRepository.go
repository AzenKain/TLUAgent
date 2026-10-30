package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
)

// CurriculumRepository defines the interface for curriculum and substitution queries.
type CurriculumRepository interface {
	UpsertCurriculum(ctx context.Context, params sqlc.UpsertCurriculumParams) (*models.Curriculum, error)
	GetCurriculumByID(ctx context.Context, id string) (*models.Curriculum, error)
	GetCurriculumByCohortAndMajor(ctx context.Context, cohort, majorName string) (*models.Curriculum, error)
	ListCurriculumsByCohort(ctx context.Context, cohort string) ([]*models.Curriculum, error)
	ListCurriculums(ctx context.Context) ([]*models.Curriculum, error)
	CountCurriculums(ctx context.Context) (int64, error)
	DeleteCurriculum(ctx context.Context, id string) error
	ListCohorts(ctx context.Context) ([]string, error)
	ListMajorsByCohort(ctx context.Context, cohort string) ([]sqlc.ListMajorsByCohortRow, error)
	ListAllMajors(ctx context.Context) ([]sqlc.ListAllMajorsRow, error)
	ListAllMajorMetadata(ctx context.Context) ([]sqlc.ListAllMajorMetadataRow, error)
	ListMajorsByFaculty(ctx context.Context, facultyCode string) ([]string, error)
	FindMajorMapping(ctx context.Context, cohort, major string) (*sqlc.FindMajorMappingRow, error)
	ListDistinctSubstitutionMajors(ctx context.Context, cohortNum int64) ([]string, error)
	UpdateMajorMetadataByName(ctx context.Context, majorName, majorCode, facultyCode, aliases string) error

	UpsertCurriculumCourse(ctx context.Context, params sqlc.UpsertCurriculumCourseParams) (*models.CurriculumCourse, error)
	UpdateCurriculumCourse(ctx context.Context, params sqlc.UpdateCurriculumCourseParams) (*models.CurriculumCourse, error)
	DeleteCurriculumCourse(ctx context.Context, id string) error
	GetCurriculumCourseByID(ctx context.Context, id string) (*models.CurriculumCourse, error)
	GetCurriculumCoursesByCurriculumID(ctx context.Context, curriculumID string) ([]*models.CurriculumCourse, error)
	GetCurriculumCoursesByCohortAndMajor(ctx context.Context, cohort, majorName string) ([]*models.CurriculumCourse, error)
	GetCoursesByCodeAndCohort(ctx context.Context, cohort, courseCode string) ([]*models.CurriculumCourse, error)
	SearchCurriculumCourses(ctx context.Context, cohort, majorName, keyword string) ([]*models.CurriculumCourse, error)
	ListCoursesPaginated(ctx context.Context, params sqlc.ListCoursesPaginatedParams) ([]*models.CurriculumCourse, error)
	CountCoursesFiltered(ctx context.Context, params sqlc.CountCoursesFilteredParams) (int64, error)
	CountCurriculumCourses(ctx context.Context) (int64, error)
	FindCourseMetadataByCode(ctx context.Context, courseCode string) (*sqlc.FindCourseMetadataByCodeRow, error)

	UpsertCoursePrerequisite(ctx context.Context, params sqlc.UpsertCoursePrerequisiteParams) (*models.CoursePrerequisite, error)
	GetCoursePrerequisiteByID(ctx context.Context, id string) (*models.CoursePrerequisite, error)
	DeleteCoursePrerequisite(ctx context.Context, id string) error
	ListPrerequisitesForCourse(ctx context.Context, cohort, courseCode string) ([]*models.CoursePrerequisite, error)
	ListDependentCourses(ctx context.Context, cohort, prerequisiteCode string) ([]*models.CoursePrerequisite, error)
	ListAllPrerequisitesForCohortAndMajor(ctx context.Context, cohort, majorName string) ([]*models.CoursePrerequisite, error)

	UpsertCourseSubstitution(ctx context.Context, params sqlc.UpsertCourseSubstitutionParams) (*models.CourseSubstitution, error)
	FindDirectSubstitutions(ctx context.Context, originalCourseCode string, cohortNum int64, scopeMajor, facultyCode string) ([]*models.CourseSubstitution, error)
	ListSubstitutionsByCohortAndMajor(ctx context.Context, cohortNum int64, scopeMajor, facultyCode string) ([]*models.CourseSubstitution, error)
	ListSubstitutionsPaginated(ctx context.Context, params sqlc.ListSubstitutionsPaginatedParams) ([]*models.CourseSubstitution, error)
	CountSubstitutionsFiltered(ctx context.Context, params sqlc.CountSubstitutionsFilteredParams) (int64, error)
	GetSubstitutionByID(ctx context.Context, id string) (*models.CourseSubstitution, error)
	DeleteCourseSubstitution(ctx context.Context, id string) error
	CountSubstitutions(ctx context.Context) (int64, error)
	CountCoursePrerequisites(ctx context.Context) (int64, error)
	ListRequirementRules(ctx context.Context, cohort, majorName string) ([]*models.RequirementRule, error)
	ReplaceRequirementRules(ctx context.Context, cohort, majorName string, rules []*models.RequirementRule) error

	UpsertCourseGroupSubstitution(ctx context.Context, params sqlc.UpsertCourseGroupSubstitutionParams) (*models.CourseGroupSubstitution, error)
	ListGroupSubstitutions(ctx context.Context, cohortNum int64, scopeMajor, facultyCode string) ([]*models.CourseGroupSubstitution, error)

	InvalidateCurriculumCache(ctx context.Context, cohort, majorName string)
	WithTx(tx *sql.Tx) CurriculumRepository
}

type curriculumRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

// NewCurriculumRepository initializes a new curriculum repository instance.
func NewCurriculumRepository(db sqlc.DBTX, c cache.Cache) CurriculumRepository {
	return &curriculumRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *curriculumRepository) WithTx(tx *sql.Tx) CurriculumRepository {
	return &curriculumRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sfg:  &singleflight.Group{},
	}
}

func (r *curriculumRepository) InvalidateCurriculumCache(ctx context.Context, cohort, majorName string) {
	if r.c == nil {
		return
	}
	_ = r.c.DelByPattern(ctx, "curr:*")
	_ = r.c.DelByPattern(ctx, "sub:*")
}

func (r *curriculumRepository) UpsertCurriculum(ctx context.Context, params sqlc.UpsertCurriculumParams) (*models.Curriculum, error) {
	row, err := r.q.UpsertCurriculum(ctx, params)
	if err != nil {
		return nil, err
	}
	r.InvalidateCurriculumCache(ctx, params.Cohort, params.MajorName)
	var m models.Curriculum
	return m.FromSqlc(row), nil
}

func (r *curriculumRepository) GetCurriculumByID(ctx context.Context, id string) (*models.Curriculum, error) {
	key := cache.BuildKey("curr", "id", id)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached models.Curriculum
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return &cached, nil
			}
		}
		row, err := r.q.GetCurriculumByID(ctx, id)
		if err != nil {
			return nil, err
		}
		var m models.Curriculum
		res := m.FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.Curriculum), nil
}

func (r *curriculumRepository) GetCurriculumByCohortAndMajor(ctx context.Context, cohort, majorName string) (*models.Curriculum, error) {
	key := cache.BuildKey("curr", "cohort_major", cohort, majorName)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached models.Curriculum
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return &cached, nil
			}
		}
		row, err := r.q.GetCurriculumByCohortAndMajor(ctx, sqlc.GetCurriculumByCohortAndMajorParams{
			Cohort:    cohort,
			MajorName: majorName,
		})
		if err != nil {
			return nil, err
		}
		var m models.Curriculum
		res := m.FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.Curriculum), nil
}

func (r *curriculumRepository) ListCurriculumsByCohort(ctx context.Context, cohort string) ([]*models.Curriculum, error) {
	key := cache.BuildKey("curr", "cohort_list", cohort)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.Curriculum
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListCurriculumsByCohort(ctx, cohort)
		if err != nil {
			return nil, err
		}
		res := make([]*models.Curriculum, len(rows))
		for i, row := range rows {
			var m models.Curriculum
			res[i] = m.FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.Curriculum), nil
}

func (r *curriculumRepository) ListCurriculums(ctx context.Context) ([]*models.Curriculum, error) {
	key := cache.BuildKey("curr", "all")
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.Curriculum
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListCurriculums(ctx)
		if err != nil {
			return nil, err
		}
		res := make([]*models.Curriculum, len(rows))
		for i, row := range rows {
			var m models.Curriculum
			res[i] = m.FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.Curriculum), nil
}

func (r *curriculumRepository) CountCurriculums(ctx context.Context) (int64, error) {
	key := cache.BuildKey("curr", "count_all")
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached int64
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		cnt, err := r.q.CountCurriculums(ctx)
		if err != nil {
			return int64(0), err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, cnt, 0)
		}
		return cnt, nil
	})
	if err != nil {
		return 0, err
	}
	return val.(int64), nil
}

func (r *curriculumRepository) UpsertCurriculumCourse(ctx context.Context, params sqlc.UpsertCurriculumCourseParams) (*models.CurriculumCourse, error) {
	row, err := r.q.UpsertCurriculumCourse(ctx, params)
	if err != nil {
		return nil, err
	}
	r.InvalidateCurriculumCache(ctx, params.Cohort, params.MajorName)
	var m models.CurriculumCourse
	return m.FromSqlc(row), nil
}

func (r *curriculumRepository) GetCurriculumCourseByID(ctx context.Context, id string) (*models.CurriculumCourse, error) {
	key := cache.BuildKey("curr", "course", "id", id)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached models.CurriculumCourse
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return &cached, nil
			}
		}
		row, err := r.q.GetCurriculumCourseByID(ctx, id)
		if err != nil {
			return nil, err
		}
		var m models.CurriculumCourse
		res := m.FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.CurriculumCourse), nil
}

func (r *curriculumRepository) GetCurriculumCoursesByCurriculumID(ctx context.Context, curriculumID string) ([]*models.CurriculumCourse, error) {
	key := cache.BuildKey("curr", "courses_by_curriculum_id", curriculumID)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.CurriculumCourse
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.GetCurriculumCoursesByCurriculumID(ctx, curriculumID)
		if err != nil {
			return nil, err
		}
		res := make([]*models.CurriculumCourse, len(rows))
		for i, row := range rows {
			var m models.CurriculumCourse
			res[i] = m.FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.CurriculumCourse), nil
}

func (r *curriculumRepository) GetCurriculumCoursesByCohortAndMajor(ctx context.Context, cohort, majorName string) ([]*models.CurriculumCourse, error) {
	key := cache.BuildKey("curr", "courses", cohort, majorName)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.CurriculumCourse
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.GetCurriculumCoursesByCohortAndMajor(ctx, sqlc.GetCurriculumCoursesByCohortAndMajorParams{
			Cohort:    cohort,
			MajorName: majorName,
		})
		if err != nil {
			return nil, err
		}
		res := make([]*models.CurriculumCourse, len(rows))
		for i, row := range rows {
			var m models.CurriculumCourse
			res[i] = m.FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.CurriculumCourse), nil
}

func (r *curriculumRepository) GetCoursesByCodeAndCohort(ctx context.Context, cohort, courseCode string) ([]*models.CurriculumCourse, error) {
	key := cache.BuildKey("curr", "courses_by_code", cohort, courseCode)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.CurriculumCourse
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.GetCoursesByCodeAndCohort(ctx, sqlc.GetCoursesByCodeAndCohortParams{
			Cohort:     cohort,
			CourseCode: courseCode,
		})
		if err != nil {
			return nil, err
		}
		res := make([]*models.CurriculumCourse, len(rows))
		for i, row := range rows {
			var m models.CurriculumCourse
			res[i] = m.FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.CurriculumCourse), nil
}

func (r *curriculumRepository) SearchCurriculumCourses(ctx context.Context, cohort, majorName, keyword string) ([]*models.CurriculumCourse, error) {
	key := cache.BuildKey("curr", "search_courses", cohort, majorName, keyword)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.CurriculumCourse
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.SearchCurriculumCourses(ctx, sqlc.SearchCurriculumCoursesParams{
			Cohort:    cohort,
			MajorName: sql.NullString{String: majorName, Valid: majorName != ""},
			Keyword:   sql.NullString{String: keyword, Valid: keyword != ""},
		})
		if err != nil {
			return nil, err
		}
		res := make([]*models.CurriculumCourse, len(rows))
		for i, row := range rows {
			var m models.CurriculumCourse
			res[i] = m.FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.CurriculumCourse), nil
}

func (r *curriculumRepository) CountCurriculumCourses(ctx context.Context) (int64, error) {
	key := cache.BuildKey("curr", "count_courses_all")
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached int64
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		cnt, err := r.q.CountCurriculumCourses(ctx)
		if err != nil {
			return int64(0), err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, cnt, 0)
		}
		return cnt, nil
	})
	if err != nil {
		return 0, err
	}
	return val.(int64), nil
}

func (r *curriculumRepository) UpsertCoursePrerequisite(ctx context.Context, params sqlc.UpsertCoursePrerequisiteParams) (*models.CoursePrerequisite, error) {
	row, err := r.q.UpsertCoursePrerequisite(ctx, params)
	if err != nil {
		return nil, err
	}
	r.InvalidateCurriculumCache(ctx, params.Cohort, "")
	var m models.CoursePrerequisite
	return m.FromSqlc(row), nil
}

func (r *curriculumRepository) GetCoursePrerequisiteByID(ctx context.Context, id string) (*models.CoursePrerequisite, error) {
	key := cache.BuildKey("curr", "prereq", "id", id)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached models.CoursePrerequisite
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return &cached, nil
			}
		}
		row, err := r.q.GetCoursePrerequisiteByID(ctx, id)
		if err != nil {
			return nil, err
		}
		var m models.CoursePrerequisite
		res := m.FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.CoursePrerequisite), nil
}

func (r *curriculumRepository) DeleteCoursePrerequisite(ctx context.Context, id string) error {
	err := r.q.DeleteCoursePrerequisite(ctx, id)
	if err != nil {
		return err
	}
	r.InvalidateCurriculumCache(ctx, "", "")
	return nil
}

func (r *curriculumRepository) ListPrerequisitesForCourse(ctx context.Context, cohort, courseCode string) ([]*models.CoursePrerequisite, error) {
	key := cache.BuildKey("curr", "prereqs_course", cohort, courseCode)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.CoursePrerequisite
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListPrerequisitesForCourse(ctx, sqlc.ListPrerequisitesForCourseParams{
			Cohort:     cohort,
			CourseCode: courseCode,
		})
		if err != nil {
			return nil, err
		}
		res := make([]*models.CoursePrerequisite, len(rows))
		for i, row := range rows {
			var m models.CoursePrerequisite
			res[i] = m.FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.CoursePrerequisite), nil
}

func (r *curriculumRepository) ListDependentCourses(ctx context.Context, cohort, prerequisiteCode string) ([]*models.CoursePrerequisite, error) {
	key := cache.BuildKey("curr", "dependent_courses", cohort, prerequisiteCode)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.CoursePrerequisite
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListDependentCourses(ctx, sqlc.ListDependentCoursesParams{
			Cohort:                 cohort,
			PrerequisiteCourseCode: prerequisiteCode,
		})
		if err != nil {
			return nil, err
		}
		res := make([]*models.CoursePrerequisite, len(rows))
		for i, row := range rows {
			var m models.CoursePrerequisite
			res[i] = m.FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.CoursePrerequisite), nil
}

// ParseCohortRange extracts numeric bounds (min, max) from a cohort scope string.
func ParseCohortRange(scopeCohort string) (int64, int64) {
	c := strings.ToLower(strings.TrimSpace(scopeCohort))
	if c == "" || c == "all" {
		return 0, 999
	}
	var nums []int64
	var current int64
	inNum := false
	for _, r := range c {
		if unicode.IsDigit(r) {
			current = current*10 + int64(r-'0')
			inNum = true
		} else {
			if inNum {
				nums = append(nums, current)
				current = 0
				inNum = false
			}
		}
	}
	if inNum {
		nums = append(nums, current)
	}

	if strings.Contains(c, "<") && len(nums) > 0 {
		return 1, nums[0]
	}
	if strings.Contains(c, ">") && len(nums) > 0 {
		return nums[0], 999
	}
	if len(nums) >= 2 {
		return nums[0], nums[1]
	}
	if len(nums) == 1 {
		return nums[0], nums[0]
	}
	return 0, 999
}

func (r *curriculumRepository) UpsertCourseSubstitution(ctx context.Context, params sqlc.UpsertCourseSubstitutionParams) (*models.CourseSubstitution, error) {
	if params.MinCohort == 0 && params.MaxCohort == 0 {
		params.MinCohort, params.MaxCohort = ParseCohortRange(params.ScopeCohort)
	}
	row, err := r.q.UpsertCourseSubstitution(ctx, params)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.DelByPattern(ctx, "sub:*")
	}
	var m models.CourseSubstitution
	return m.FromSqlc(row), nil
}

func (r *curriculumRepository) FindDirectSubstitutions(ctx context.Context, originalCourseCode string, cohortNum int64, scopeMajor, facultyCode string) ([]*models.CourseSubstitution, error) {
	key := cache.BuildKey("sub", "direct", originalCourseCode, strconv.FormatInt(cohortNum, 10), scopeMajor, facultyCode)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.CourseSubstitution
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.FindDirectSubstitutions(ctx, sqlc.FindDirectSubstitutionsParams{
			OriginalCourseCode: originalCourseCode,
			CohortNum:          cohortNum,
			ScopeMajor:         scopeMajor,
			FacultyCode:        facultyCode,
		})
		if err != nil {
			return nil, err
		}
		res := make([]*models.CourseSubstitution, len(rows))
		for i, row := range rows {
			var m models.CourseSubstitution
			res[i] = m.FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.CourseSubstitution), nil
}

func (r *curriculumRepository) ListSubstitutionsByCohortAndMajor(ctx context.Context, cohortNum int64, scopeMajor, facultyCode string) ([]*models.CourseSubstitution, error) {
	key := cache.BuildKey("sub", "list", strconv.FormatInt(cohortNum, 10), scopeMajor, facultyCode)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.CourseSubstitution
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListSubstitutionsByCohortAndMajor(ctx, sqlc.ListSubstitutionsByCohortAndMajorParams{
			CohortNum:   cohortNum,
			ScopeMajor:  scopeMajor,
			FacultyCode: facultyCode,
		})
		if err != nil {
			return nil, err
		}
		res := make([]*models.CourseSubstitution, len(rows))
		for i, row := range rows {
			var m models.CourseSubstitution
			res[i] = m.FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.CourseSubstitution), nil
}

func (r *curriculumRepository) CountCoursePrerequisites(ctx context.Context) (int64, error) {
	key := cache.BuildKey("prereq", "count_all")
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached int64
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		cnt, err := r.q.CountCoursePrerequisites(ctx)
		if err != nil {
			return int64(0), err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, cnt, constants.NormalCacheDuration)
		}
		return cnt, nil
	})
	if err != nil {
		return 0, err
	}
	return val.(int64), nil
}

func (r *curriculumRepository) CountSubstitutions(ctx context.Context) (int64, error) {
	key := cache.BuildKey("sub", "count_all")
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached int64
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		cnt, err := r.q.CountSubstitutions(ctx)
		if err != nil {
			return int64(0), err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, cnt, 0)
		}
		return cnt, nil
	})
	if err != nil {
		return 0, err
	}
	return val.(int64), nil
}

func (r *curriculumRepository) UpsertCourseGroupSubstitution(ctx context.Context, params sqlc.UpsertCourseGroupSubstitutionParams) (*models.CourseGroupSubstitution, error) {
	if params.MinCohort == 0 && params.MaxCohort == 0 {
		params.MinCohort, params.MaxCohort = ParseCohortRange(params.ScopeCohort)
	}
	row, err := r.q.UpsertCourseGroupSubstitution(ctx, params)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.DelByPattern(ctx, "sub:*")
	}
	var m models.CourseGroupSubstitution
	return m.FromSqlc(row), nil
}

func (r *curriculumRepository) ListGroupSubstitutions(ctx context.Context, cohortNum int64, scopeMajor, facultyCode string) ([]*models.CourseGroupSubstitution, error) {
	key := cache.BuildKey("sub", "group_list", strconv.FormatInt(cohortNum, 10), scopeMajor, facultyCode)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.CourseGroupSubstitution
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListGroupSubstitutions(ctx, sqlc.ListGroupSubstitutionsParams{
			CohortNum:   cohortNum,
			ScopeMajor:  scopeMajor,
			FacultyCode: facultyCode,
		})
		if err != nil {
			return nil, err
		}
		res := make([]*models.CourseGroupSubstitution, len(rows))
		for i, row := range rows {
			var m models.CourseGroupSubstitution
			res[i] = m.FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.CourseGroupSubstitution), nil
}

func (r *curriculumRepository) DeleteCurriculum(ctx context.Context, id string) error {
	curr, err := r.GetCurriculumByID(ctx, id)
	if err == nil && curr != nil {
		r.InvalidateCurriculumCache(ctx, curr.Cohort, curr.MajorName)
	}
	return r.q.DeleteCurriculum(ctx, id)
}

func (r *curriculumRepository) ListCohorts(ctx context.Context) ([]string, error) {
	key := cache.BuildKey("curr", "cohorts_all")
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []string
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListCohorts(ctx)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, rows, 0)
		}
		return rows, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]string), nil
}

func (r *curriculumRepository) ListMajorsByCohort(ctx context.Context, cohort string) ([]sqlc.ListMajorsByCohortRow, error) {
	key := cache.BuildKey("curr", "majors_by_cohort", cohort)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []sqlc.ListMajorsByCohortRow
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListMajorsByCohort(ctx, cohort)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, rows, 0)
		}
		return rows, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]sqlc.ListMajorsByCohortRow), nil
}

func (r *curriculumRepository) ListAllMajors(ctx context.Context) ([]sqlc.ListAllMajorsRow, error) {
	key := cache.BuildKey("curr", "majors_all")
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []sqlc.ListAllMajorsRow
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListAllMajors(ctx)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, rows, 0)
		}
		return rows, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]sqlc.ListAllMajorsRow), nil
}

func (r *curriculumRepository) UpdateCurriculumCourse(ctx context.Context, params sqlc.UpdateCurriculumCourseParams) (*models.CurriculumCourse, error) {
	row, err := r.q.UpdateCurriculumCourse(ctx, params)
	if err != nil {
		return nil, err
	}
	r.InvalidateCurriculumCache(ctx, row.Cohort, row.MajorName)
	var m models.CurriculumCourse
	return m.FromSqlc(row), nil
}

func (r *curriculumRepository) DeleteCurriculumCourse(ctx context.Context, id string) error {
	course, err := r.GetCurriculumCourseByID(ctx, id)
	if err == nil && course != nil {
		r.InvalidateCurriculumCache(ctx, course.Cohort, course.MajorName)
	}
	return r.q.DeleteCurriculumCourse(ctx, id)
}

func (r *curriculumRepository) ListCoursesPaginated(ctx context.Context, params sqlc.ListCoursesPaginatedParams) ([]*models.CurriculumCourse, error) {
	key := cache.BuildKey("curr", "courses_paged", params.Cohort, params.MajorName, params.KnowledgeBlock, strconv.FormatInt(params.SuggestedSemester, 10), strconv.FormatInt(params.IsElearning, 10), params.Keyword, strconv.FormatInt(params.LimitVal, 10), strconv.FormatInt(params.OffsetVal, 10))
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.CurriculumCourse
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListCoursesPaginated(ctx, params)
		if err != nil {
			return nil, err
		}
		res := make([]*models.CurriculumCourse, len(rows))
		for i, row := range rows {
			var m models.CurriculumCourse
			res[i] = m.FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.CurriculumCourse), nil
}

func (r *curriculumRepository) CountCoursesFiltered(ctx context.Context, params sqlc.CountCoursesFilteredParams) (int64, error) {
	key := cache.BuildKey("curr", "courses_count_filtered", params.Cohort, params.MajorName, params.KnowledgeBlock, strconv.FormatInt(params.SuggestedSemester, 10), strconv.FormatInt(params.IsElearning, 10), params.Keyword)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached int64
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		cnt, err := r.q.CountCoursesFiltered(ctx, params)
		if err != nil {
			return int64(0), err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, cnt, 0)
		}
		return cnt, nil
	})
	if err != nil {
		return 0, err
	}
	return val.(int64), nil
}

func (r *curriculumRepository) ListAllPrerequisitesForCohortAndMajor(ctx context.Context, cohort, majorName string) ([]*models.CoursePrerequisite, error) {
	key := cache.BuildKey("curr", "prereqs_cohort_major", cohort, majorName)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.CoursePrerequisite
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListAllPrerequisitesForCohortAndMajor(ctx, sqlc.ListAllPrerequisitesForCohortAndMajorParams{
			Cohort:    cohort,
			MajorName: majorName,
		})
		if err != nil {
			return nil, err
		}
		res := make([]*models.CoursePrerequisite, len(rows))
		for i, row := range rows {
			var m models.CoursePrerequisite
			res[i] = m.FromSqlc(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.CoursePrerequisite), nil
}

func (r *curriculumRepository) ListSubstitutionsPaginated(ctx context.Context, params sqlc.ListSubstitutionsPaginatedParams) ([]*models.CourseSubstitution, error) {
	key := cache.BuildKey("sub", "paged", strconv.FormatInt(params.CohortNum, 10), params.ScopeMajor, params.FacultyCode, params.MajorAliases, params.Keyword, strconv.FormatInt(params.LimitVal, 10), strconv.FormatInt(params.OffsetVal, 10))
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.CourseSubstitution
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListSubstitutionsPaginated(ctx, params)
		if err != nil {
			return nil, err
		}
		res := make([]*models.CourseSubstitution, len(rows))
		for i, row := range rows {
			var m models.CourseSubstitution
			res[i] = m.FromSqlc(row)
		}
		if r.c != nil && !r.inTx && len(res) > 0 {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.CourseSubstitution), nil
}

func (r *curriculumRepository) CountSubstitutionsFiltered(ctx context.Context, params sqlc.CountSubstitutionsFilteredParams) (int64, error) {
	key := cache.BuildKey("sub", "count_filtered", strconv.FormatInt(params.CohortNum, 10), params.ScopeMajor, params.FacultyCode, params.MajorAliases, params.Keyword)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached int64
			if err := r.c.Get(ctx, key, &cached); err == nil && cached > 0 {
				return cached, nil
			}
		}
		cnt, err := r.q.CountSubstitutionsFiltered(ctx, params)
		if err != nil {
			return int64(0), err
		}
		if r.c != nil && !r.inTx && cnt > 0 {
			_ = r.c.Set(ctx, key, cnt, 0)
		}
		return cnt, nil
	})
	if err != nil {
		return 0, err
	}
	return val.(int64), nil
}

func (r *curriculumRepository) GetSubstitutionByID(ctx context.Context, id string) (*models.CourseSubstitution, error) {
	key := cache.BuildKey("sub", "id", id)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached models.CourseSubstitution
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return &cached, nil
			}
		}
		row, err := r.q.GetSubstitutionByID(ctx, id)
		if err != nil {
			return nil, err
		}
		var m models.CourseSubstitution
		res := m.FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, 0)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.CourseSubstitution), nil
}

func (r *curriculumRepository) DeleteCourseSubstitution(ctx context.Context, id string) error {
	if r.c != nil {
		_ = r.c.DelByPattern(ctx, "sub:*")
	}
	rows, err := r.q.DeleteCourseSubstitution(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *curriculumRepository) FindMajorMapping(ctx context.Context, cohort, major string) (*sqlc.FindMajorMappingRow, error) {
	key := cache.BuildKey("curr", "major_mapping", cohort, strings.ToLower(strings.TrimSpace(major)))
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached sqlc.FindMajorMappingRow
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return &cached, nil
			}
		}
		row, err := r.q.FindMajorMapping(ctx, sqlc.FindMajorMappingParams{
			Cohort: cohort,
			Major:  major,
		})
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, row, 0)
		}
		return &row, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*sqlc.FindMajorMappingRow), nil
}

func (r *curriculumRepository) ListDistinctSubstitutionMajors(ctx context.Context, cohortNum int64) ([]string, error) {
	key := cache.BuildKey("sub", "distinct_majors", strconv.FormatInt(cohortNum, 10))
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []string
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListDistinctSubstitutionMajors(ctx, cohortNum)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, rows, 0)
		}
		return rows, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]string), nil
}

func (r *curriculumRepository) ListAllMajorMetadata(ctx context.Context) ([]sqlc.ListAllMajorMetadataRow, error) {
	key := cache.BuildKey("curr", "major_metadata_all")
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []sqlc.ListAllMajorMetadataRow
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListAllMajorMetadata(ctx)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, rows, 0)
		}
		return rows, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]sqlc.ListAllMajorMetadataRow), nil
}

func (r *curriculumRepository) ListMajorsByFaculty(ctx context.Context, facultyCode string) ([]string, error) {
	key := cache.BuildKey("curr", "majors_by_faculty", facultyCode)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached []string
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListMajorsByFaculty(ctx, facultyCode)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, rows, 0)
		}
		return rows, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]string), nil
}

func (r *curriculumRepository) FindCourseMetadataByCode(ctx context.Context, courseCode string) (*sqlc.FindCourseMetadataByCodeRow, error) {
	codeUpper := strings.ToUpper(strings.TrimSpace(courseCode))
	key := cache.BuildKey("curr", "course_meta", codeUpper)
	val, err, _ := r.sfg.Do(key, func() (interface{}, error) {
		if r.c != nil && !r.inTx {
			var cached sqlc.FindCourseMetadataByCodeRow
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return &cached, nil
			}
		}
		row, err := r.q.FindCourseMetadataByCode(ctx, codeUpper)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, row, 0)
		}
		return &row, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*sqlc.FindCourseMetadataByCodeRow), nil
}

func (r *curriculumRepository) UpdateMajorMetadataByName(ctx context.Context, majorName, majorCode, facultyCode, aliases string) error {
	err := r.q.UpdateMajorMetadataByName(ctx, sqlc.UpdateMajorMetadataByNameParams{
		MajorName:   majorName,
		MajorCode:   majorCode,
		FacultyCode: facultyCode,
		Aliases:     aliases,
	})
	if err == nil {
		r.InvalidateCurriculumCache(ctx, "", majorName)
		if r.c != nil && !r.inTx {
			_ = r.c.Del(ctx, cache.BuildKey("curr", "majors_all"))
			_ = r.c.Del(ctx, cache.BuildKey("curr", "all"))
			_ = r.c.DelByPattern(ctx, "curr:majors_by_cohort:*")
		}
	}
	return err
}

func (r *curriculumRepository) ListRequirementRules(ctx context.Context, cohort, majorName string) ([]*models.RequirementRule, error) {
	key := cache.BuildKey("curr", "requirement_rules", cohort, majorName)
	val, err, _ := r.sfg.Do(key, func() (any, error) {
		if r.c != nil && !r.inTx {
			var cached []*models.RequirementRule
			if err := r.c.Get(ctx, key, &cached); err == nil {
				return cached, nil
			}
		}
		rows, err := r.q.ListRequirementRules(ctx, sqlc.ListRequirementRulesParams{Cohort: cohort, MajorName: majorName})
		if err != nil {
			return nil, err
		}
		res := make([]*models.RequirementRule, len(rows))
		for i, row := range rows {
			res[i] = &models.RequirementRule{
				ID: row.ID, Cohort: row.Cohort, MajorName: row.MajorName, RuleType: row.RuleType,
				GroupKey: row.GroupKey, MinCredits: row.MinCredits, CourseCodes: row.CourseCodes,
				Note: row.Note, Source: row.Source,
			}
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.RequirementRule), nil
}

func (r *curriculumRepository) ReplaceRequirementRules(ctx context.Context, cohort, majorName string, rules []*models.RequirementRule) error {
	if err := r.q.DeleteRequirementRules(ctx, sqlc.DeleteRequirementRulesParams{Cohort: cohort, MajorName: majorName}); err != nil {
		return err
	}
	for i, rule := range rules {
		err := r.q.InsertRequirementRule(ctx, sqlc.InsertRequirementRuleParams{
			ID:          fmt.Sprintf("req_%s_%s_%d", strings.ToLower(cohort), strings.ToLower(strings.ReplaceAll(majorName, " ", "_")), i),
			Cohort:      cohort,
			MajorName:   majorName,
			RuleType:    rule.RuleType,
			GroupKey:    rule.GroupKey,
			MinCredits:  rule.MinCredits,
			CourseCodes: rule.CourseCodes,
			Note:        rule.Note,
			Source:      rule.Source,
		})
		if err != nil {
			return err
		}
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("curr", "requirement_rules", cohort, majorName))
	}
	return nil
}

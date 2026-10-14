package controllers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func setupCurriculumTestController(t *testing.T) (*controllers.CurriculumController, services.CurriculumService, *http.ServeMux, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	require.NoError(t, err)

	err = database.ApplySchema(db)
	require.NoError(t, err)

	ramCache := cache.NewTheineCache(10 << 20)
	currRepo := repositories.NewCurriculumRepository(db, ramCache)
	currSvc := services.NewCurriculumService(currRepo)
	roadmapOptimizer := services.NewRoadmapOptimizer(currRepo)
	transcriptRepo := repositories.NewTranscriptRepository(db, ramCache)
	memoryRepo := repositories.NewMemoryRepository(db, ramCache)
	userRepo := repositories.NewUserRepository(db, ramCache)
	transcriptSvc := services.NewTranscriptService(transcriptRepo, memoryRepo, nil)
	ctrl := controllers.NewCurriculumController(currSvc, roadmapOptimizer, transcriptSvc, userRepo, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/curriculum/my-roadmap", ctrl.HandleGetMyRoadmap)
	mux.HandleFunc("POST /api/curriculums/roadmap/optimize", ctrl.HandleOptimizeRoadmap)
	mux.HandleFunc("POST /api/curriculums/roadmap/diagnostic", ctrl.HandleEvaluateDiagnosticQuiz)
	mux.HandleFunc("GET /api/admin/curriculums/stats", ctrl.HandleGetStats)
	mux.HandleFunc("GET /api/admin/curriculums/cohorts", ctrl.HandleListCohorts)
	mux.HandleFunc("GET /api/admin/curriculums/majors", ctrl.HandleListMajors)
	mux.HandleFunc("GET /api/admin/curriculums/visualization", ctrl.HandleGetVisualization)

	mux.HandleFunc("GET /api/admin/curriculums/courses", ctrl.HandleListCourses)
	mux.HandleFunc("GET /api/admin/curriculums/courses/{id}", ctrl.HandleGetCourse)
	mux.HandleFunc("POST /api/admin/curriculums/courses", ctrl.HandleCreateCourse)
	mux.HandleFunc("PUT /api/admin/curriculums/courses/{id}", ctrl.HandleUpdateCourse)
	mux.HandleFunc("DELETE /api/admin/curriculums/courses/{id}", ctrl.HandleDeleteCourse)

	mux.HandleFunc("GET /api/admin/curriculums/substitutions", ctrl.HandleListSubstitutions)
	mux.HandleFunc("GET /api/admin/curriculums/substitutions/{id}", ctrl.HandleGetSubstitution)
	mux.HandleFunc("POST /api/admin/curriculums/substitutions", ctrl.HandleCreateSubstitution)
	mux.HandleFunc("PUT /api/admin/curriculums/substitutions/{id}", ctrl.HandleUpdateSubstitution)
	mux.HandleFunc("DELETE /api/admin/curriculums/substitutions/{id}", ctrl.HandleDeleteSubstitution)

	return ctrl, currSvc, mux, db
}

// TestCurriculumController_Endpoints exercises all admin curriculum, course, and substitution handlers.
func TestCurriculumController_Endpoints(t *testing.T) {
	_, currSvc, mux, db := setupCurriculumTestController(t)
	defer db.Close()
	ctx := context.Background()

	currRepo := repositories.NewCurriculumRepository(db, cache.NewTheineCache(10<<20))
	_, err := currRepo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
		ID:           "curr_k36_cntt",
		Cohort:       "K36",
		AcademicYear: "2023-2024",
		MajorName:    "Công nghệ thông tin",
		MajorCode:    "CNTT",
		TotalCourses: 2,
		TotalCredits: 7.0,
	})
	require.NoError(t, err)

	createdCourse, err := currSvc.CreateCourse(ctx, &request.CreateCourseRequest{
		Cohort:            "K36",
		MajorName:         "Công nghệ thông tin",
		CourseCode:        "CF213",
		CourseName:        "Cấu trúc dữ liệu và giải thuật",
		Credits:           4.0,
		Weight:            1.8,
		Hours:             "60",
		KnowledgeBlock:    "Cơ sở ngành",
		PrerequisitesText: "CS122",
		SuggestedSemester: 3,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/curriculums/stats", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var statsResp response.CommonResponse
	err = json.Unmarshal(rec.Body.Bytes(), &statsResp)
	require.NoError(t, err)
	require.True(t, statsResp.Status)

	req = httptest.NewRequest(http.MethodGet, "/api/admin/curriculums/cohorts", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/admin/curriculums/courses?cohort=K36", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/admin/curriculums/courses/"+createdCourse.ID, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	createPayload, _ := json.Marshal(request.CreateCourseRequest{
		Cohort:            "K36",
		MajorName:         "Công nghệ thông tin",
		CourseCode:        "CS122",
		CourseName:        "Kỹ thuật lập trình",
		Credits:           3.0,
		Weight:            1.0,
		Hours:             "45",
		KnowledgeBlock:    "Cơ sở ngành",
		PrerequisitesText: "",
		SuggestedSemester: 2,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/admin/curriculums/courses", bytes.NewReader(createPayload))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	updatePayload, _ := json.Marshal(request.UpdateCourseRequest{
		CourseName:        "Cấu trúc dữ liệu và giải thuật nâng cao",
		Credits:           4.5,
		Weight:            1.8,
		Hours:             "75",
		KnowledgeBlock:    "Cơ sở ngành",
		PrerequisitesText: "CS122",
		SuggestedSemester: 3,
	})
	req = httptest.NewRequest(http.MethodPut, "/api/admin/curriculums/courses/"+createdCourse.ID, bytes.NewReader(updatePayload))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/admin/curriculums/visualization?cohort=K36&major="+url.QueryEscape("Công nghệ thông tin"), nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var vizResp response.CommonResponse
	err = json.Unmarshal(rec.Body.Bytes(), &vizResp)
	require.NoError(t, err)
	require.True(t, vizResp.Status)

	subPayload, _ := json.Marshal(request.CreateSubstitutionRequest{
		OriginalCourseCode:    "MI322",
		OriginalCourseName:    "Xử lý ảnh",
		ReplacementCourseCode: "AI130",
		ReplacementCourseName: "Thị giác máy tính",
		ScopeCohort:           "<=k35",
		ScopeMajor:            "TT",
		IsBidirectional:       true,
		CreditsOriginal:       3.0,
		CreditsReplacement:    3.0,
		RuleType:              "DIRECT_EQUIVALENT",
		RuleNote:              "Môn thay thế 2 chiều",
		SourceReference:       "TT_old_K<=35",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/admin/curriculums/substitutions", bytes.NewReader(subPayload))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/admin/curriculums/substitutions", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodDelete, "/api/admin/curriculums/courses/"+createdCourse.ID, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	quizPayload, _ := json.Marshal(models.PersonalityQuizAnswers{
		PacePreference:     "fast",
		PrimaryStrength:    "programming",
		DeliveryPreference: "elearning",
		SummerWillingness:  true,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/curriculums/roadmap/diagnostic", bytes.NewReader(quizPayload))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	roadmapPayload, _ := json.Marshal(models.RoadmapOptimizationRequest{
		Cohort:               "K36",
		MajorName:            "Công nghệ thông tin",
		CurrentSemester:      1,
		TargetGraduationTerm: 8,
		Transcript: []models.StudentCourseGrade{
			{CourseCode: "CS111", Score: 8.0, Grade: "B"},
		},
	})
	req = httptest.NewRequest(http.MethodPost, "/api/curriculums/roadmap/optimize", bytes.NewReader(roadmapPayload))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	unauthReq := httptest.NewRequest(http.MethodGet, "/api/curriculum/my-roadmap", nil)
	unauthRec := httptest.NewRecorder()
	mux.ServeHTTP(unauthRec, unauthReq)
	require.Equal(t, http.StatusUnauthorized, unauthRec.Code)

	authCtx := context.WithValue(context.Background(), middlewares.UserIDContextKey, "test-user-123")
	authReq := httptest.NewRequest(http.MethodGet, "/api/curriculum/my-roadmap?cohort=K36&major="+url.QueryEscape("Công nghệ thông tin"), nil).WithContext(authCtx)
	authRec := httptest.NewRecorder()
	mux.ServeHTTP(authRec, authReq)
	require.Equal(t, http.StatusOK, authRec.Code)
}

// TestCurriculumController_HandleGetMyRoadmap_NoCohortGuessingFromStudentCode verifies student code never guesses cohort.
func TestCurriculumController_HandleGetMyRoadmap_NoCohortGuessingFromStudentCode(t *testing.T) {
	_, _, mux, db := setupCurriculumTestController(t)
	defer db.Close()
	ctx := context.Background()

	ramCache := cache.NewTheineCache(10 << 20)
	userRepo := repositories.NewUserRepository(db, ramCache)

	testUserID := "student-uuid-heuristic-roadmap-test"
	_, err := userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           testUserID,
		Email:        "student_a36@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Nguyen Thi A", Valid: true},
		StudentCode:  sql.NullString{String: "A36777", Valid: true},
		AuthProvider: "local",
	})
	require.NoError(t, err)

	authCtx := context.WithValue(context.Background(), middlewares.UserIDContextKey, testUserID)
	req := httptest.NewRequest(http.MethodGet, "/api/curriculum/my-roadmap", nil).WithContext(authCtx)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Status bool `json:"status"`
		Data   struct {
			Summary struct {
				Cohort    string `json:"cohort"`
				MajorName string `json:"major_name"`
			} `json:"summary"`
		} `json:"data"`
	}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Empty(t, resp.Data.Summary.Cohort, "Cohort should remain empty and never be guessed from StudentCode")
}

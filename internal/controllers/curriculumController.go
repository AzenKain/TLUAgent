package controllers

import (
	"net/http"
	"strings"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/validator"
)

// CurriculumController manages academic programs, courses, prerequisites, substitutions, and roadmaps.
type CurriculumController struct {
	svc           services.CurriculumService
	optimizer     services.RoadmapOptimizer
	transcriptSvc services.TranscriptService
	userRepo      repositories.UserRepository
	memorySvc     services.MemoryService
}

// NewCurriculumController creates a new CurriculumController instance.
func NewCurriculumController(
	svc services.CurriculumService,
	optimizer services.RoadmapOptimizer,
	transcriptSvc services.TranscriptService,
	userRepo repositories.UserRepository,
	memorySvc services.MemoryService,
) *CurriculumController {
	return &CurriculumController{
		svc:           svc,
		optimizer:     optimizer,
		transcriptSvc: transcriptSvc,
		userRepo:      userRepo,
		memorySvc:     memorySvc,
	}
}

func (c *CurriculumController) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	stats, err := c.svc.GetStats(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   stats,
	})
}

func (c *CurriculumController) HandleListCohorts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	cohorts, err := c.svc.ListCohorts(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   cohorts,
	})
}

func (c *CurriculumController) HandleListMajors(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.ListMajorsQueryDto
	if errs := validator.ValidateQueryDto(r, &dto); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	if dto.Cohort != "" {
		rows, err := c.svc.ListMajorsByCohort(ctx, dto.Cohort)
		if err != nil {
			apperrors.HandleError(w, err)
			return
		}
		writeJSONResponse(w, http.StatusOK, response.CommonResponse{
			Status: true,
			Data:   rows,
		})
		return
	}

	rows, err := c.svc.ListAllMajors(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   rows,
	})
}

func (c *CurriculumController) HandleGetVisualization(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.GetVisualizationQueryDto
	if errs := validator.ValidateQueryDto(r, &dto); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	data, err := c.svc.GetVisualizationData(ctx, dto.Cohort, dto.Major)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   data,
	})
}

func (c *CurriculumController) HandleListCourses(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.ListCoursesQueryDto
	if errs := validator.ValidateQueryDto(r, &dto); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	page := dto.Page
	if page < 1 {
		page = 1
	}
	limit := dto.Limit
	if limit < 1 {
		limit = 50
	}

	isElearning := -1
	if dto.IsElearning == "1" || strings.EqualFold(dto.IsElearning, "true") {
		isElearning = 1
	} else if dto.IsElearning == "0" || strings.EqualFold(dto.IsElearning, "false") {
		isElearning = 0
	}

	courses, total, err := c.svc.ListCoursesPaginated(ctx, dto.Cohort, dto.Major, dto.KnowledgeBlock, dto.Keyword, dto.Semester, isElearning, page, limit)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data: &response.CurriculumCourseListResponse{
			Items: courses,
			Total: total,
			Page:  page,
			Limit: limit,
		},
	})
}

func (c *CurriculumController) HandleGetCourse(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing course ID"))
		return
	}

	course, err := c.svc.GetCourseByID(ctx, id)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}
	if course == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrNotFound, "Course not found"))
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   course,
	})
}

func (c *CurriculumController) HandleCreateCourse(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var req request.CreateCourseRequest
	if errs := validator.ValidateBodyDto(r, &req); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	course, err := c.svc.CreateCourse(ctx, &req)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, response.CommonResponse{
		Status:  true,
		Message: "Course created successfully",
		Data:    course,
	})
}

func (c *CurriculumController) HandleUpdateCourse(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing course ID"))
		return
	}

	var req request.UpdateCourseRequest
	if errs := validator.ValidateBodyDto(r, &req); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	course, err := c.svc.UpdateCourse(ctx, id, &req)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Course updated successfully",
		Data:    course,
	})
}

func (c *CurriculumController) HandleDeleteCourse(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing course ID"))
		return
	}

	if err := c.svc.DeleteCourse(ctx, id); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Course deleted successfully",
	})
}

func (c *CurriculumController) HandleListSubstitutions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.ListSubstitutionsQueryDto
	if errs := validator.ValidateQueryDto(r, &dto); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	page := dto.Page
	if page < 1 {
		page = 1
	}
	limit := dto.Limit
	if limit < 1 {
		limit = 50
	}

	subs, total, err := c.svc.ListSubstitutionsPaginated(ctx, dto.Cohort, dto.Major, dto.Keyword, page, limit)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data: &response.CourseSubstitutionListResponse{
			Items: subs,
			Total: total,
			Page:  page,
			Limit: limit,
		},
	})
}

func (c *CurriculumController) HandleGetSubstitution(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing substitution ID"))
		return
	}

	sub, err := c.svc.GetSubstitutionByID(ctx, id)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}
	if sub == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrNotFound, "Substitution not found"))
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   sub,
	})
}

func (c *CurriculumController) HandleCreateSubstitution(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var req request.CreateSubstitutionRequest
	if errs := validator.ValidateBodyDto(r, &req); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	sub, err := c.svc.CreateSubstitution(ctx, &req)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, response.CommonResponse{
		Status:  true,
		Message: "Substitution rule created successfully",
		Data:    sub,
	})
}

func (c *CurriculumController) HandleUpdateSubstitution(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing substitution ID"))
		return
	}

	var req request.UpdateSubstitutionRequest
	if errs := validator.ValidateBodyDto(r, &req); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	sub, err := c.svc.UpdateSubstitution(ctx, id, &req)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Substitution rule updated successfully",
		Data:    sub,
	})
}

func (c *CurriculumController) HandleDeleteSubstitution(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing substitution ID"))
		return
	}

	if err := c.svc.DeleteSubstitution(ctx, id); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Substitution rule deleted successfully",
	})
}

func (c *CurriculumController) HandleOptimizeRoadmap(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	if c.optimizer == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrInternalError, "Roadmap optimizer is not configured"))
		return
	}

	var req models.RoadmapOptimizationRequest
	if errs := validator.ValidateBodyDto(r, &req); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	res, err := c.optimizer.OptimizeRoadmap(ctx, &req)
	if err != nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

func (c *CurriculumController) HandleEvaluateDiagnosticQuiz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	if c.optimizer == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrInternalError, "Roadmap optimizer is not configured"))
		return
	}

	var answers models.PersonalityQuizAnswers
	if errs := validator.ValidateBodyDto(r, &answers); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	res, err := c.optimizer.EvaluateDiagnosticQuiz(ctx, &answers)
	if err != nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

// HandleGetMyRoadmap resolves and compiles the personal study roadmap for the authenticated student.
func (c *CurriculumController) HandleGetMyRoadmap(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	uid := getUserID(r)
	if uid == "" || uid == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "authentication required"))
		return
	}

	var grades []*response.StudentCourseGradeItem
	if c.transcriptSvc != nil {
		if summary, err := c.transcriptSvc.GetTranscriptSummary(ctx, uid); err == nil && summary != nil {
			grades = summary.Grades
		}
	}

	cohort := strings.TrimSpace(r.URL.Query().Get("cohort"))
	major := strings.TrimSpace(r.URL.Query().Get("major"))

	if (cohort == "" || major == "") && c.memorySvc != nil {
		if profile, err := c.memorySvc.GetStudentProfile(ctx, uid); err == nil && profile != nil {
			if cohort == "" && profile.Cohort != "" {
				cohort = profile.Cohort
			}
			if major == "" && profile.Major != "" {
				major = profile.Major
			}
		}
	}

	if cohort == "" || major == "" {
		var completedCredits float64
		var attemptedCredits float64
		var weightedScore10 float64
		var weightedScore4 float64
		var gpaCredits float64
		passedCount := 0
		failedCount := 0
		for _, g := range grades {
			attemptedCredits += g.Credits
			if g.IsPassed && g.GradeLetter != "F" {
				completedCredits += g.Credits
				passedCount++
			} else {
				failedCount++
			}
			if g.Score10 > 0 && g.Credits > 0 {
				weightedScore10 += g.Score10 * g.Credits
				weightedScore4 += g.Score4 * g.Credits
				gpaCredits += g.Credits
			}
		}
		var cpa10, cpa4 float64
		if gpaCredits > 0 {
			cpa10 = weightedScore10 / gpaCredits
			cpa4 = weightedScore4 / gpaCredits
		}
		writeJSONResponse(w, http.StatusOK, response.CommonResponse{
			Status: true,
			Data: &response.StudentPersonalRoadmapDataResponse{
				Summary: &response.StudentRoadmapSummaryResponse{
					Cohort:                cohort,
					MajorName:             major,
					TotalCreditsCompleted: completedCredits,
					CumulativeGPA10:       cpa10,
					CumulativeGPA4:        cpa4,
					TotalCourses:          len(grades),
					PassedCoursesCount:    passedCount,
					FailedCoursesCount:    failedCount,
				},
				Semesters: []*response.StudentSemesterGroupResponse{},
			},
		})
		return
	}

	roadmapData, err := c.svc.GetStudentPersonalRoadmap(ctx, uid, cohort, major, grades, c.optimizer)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   roadmapData,
	})
}

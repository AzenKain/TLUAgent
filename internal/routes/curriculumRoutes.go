package routes

import (
	"net/http"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/constants"
)

// RegisterCurriculumRoutes configures HTTP routing for curriculum, course, and substitution management.
func RegisterCurriculumRoutes(
	mux *http.ServeMux,
	ctrl *controllers.CurriculumController,
	userRepo repositories.UserRepository,
	permCache services.PermissionCache,
) {
	jwtAuth := middlewares.JWTAccess(userRepo)
	requireRead := middlewares.RequirePermission(permCache, constants.PermCurriculumRead)
	requireManage := middlewares.RequirePermission(permCache, constants.PermCurriculumManage)

	protectRead := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireRead(http.HandlerFunc(handler)))
	}

	protectManage := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireManage(http.HandlerFunc(handler)))
	}

	// Overview, Stats, and Visualization
	mux.Handle("GET /api/admin/curriculums/stats", protectRead(ctrl.HandleGetStats))
	mux.Handle("GET /api/admin/curriculums/cohorts", protectRead(ctrl.HandleListCohorts))
	mux.Handle("GET /api/admin/curriculums/majors", protectRead(ctrl.HandleListMajors))
	mux.Handle("GET /api/admin/curriculums/visualization", protectRead(ctrl.HandleGetVisualization))

	// Graduation requirement rules
	mux.Handle("GET /api/admin/curriculums/requirements", protectRead(ctrl.HandleListRequirementRules))
	mux.Handle("PUT /api/admin/curriculums/requirements", protectManage(ctrl.HandleReplaceRequirementRules))

	// Courses CRUD
	mux.Handle("GET /api/admin/curriculums/courses", protectRead(ctrl.HandleListCourses))
	mux.Handle("GET /api/admin/curriculums/courses/{id}", protectRead(ctrl.HandleGetCourse))
	mux.Handle("POST /api/admin/curriculums/courses", protectManage(ctrl.HandleCreateCourse))
	mux.Handle("PUT /api/admin/curriculums/courses/{id}", protectManage(ctrl.HandleUpdateCourse))
	mux.Handle("DELETE /api/admin/curriculums/courses/{id}", protectManage(ctrl.HandleDeleteCourse))

	// Substitutions CRUD
	mux.Handle("GET /api/admin/curriculums/substitutions", protectRead(ctrl.HandleListSubstitutions))
	mux.Handle("GET /api/admin/curriculums/substitutions/{id}", protectRead(ctrl.HandleGetSubstitution))
	mux.Handle("POST /api/admin/curriculums/substitutions", protectManage(ctrl.HandleCreateSubstitution))
	mux.Handle("PUT /api/admin/curriculums/substitutions/{id}", protectManage(ctrl.HandleUpdateSubstitution))
	mux.Handle("DELETE /api/admin/curriculums/substitutions/{id}", protectManage(ctrl.HandleDeleteSubstitution))

	// Study Roadmap Optimization & Diagnostic Quiz
	mux.Handle("GET /api/curriculum/my-roadmap", jwtAuth(http.HandlerFunc(ctrl.HandleGetMyRoadmap)))
	mux.Handle("GET /api/curriculums/my-roadmap", jwtAuth(http.HandlerFunc(ctrl.HandleGetMyRoadmap)))
	mux.Handle("POST /api/curriculums/roadmap/optimize", jwtAuth(http.HandlerFunc(ctrl.HandleOptimizeRoadmap)))
	mux.Handle("POST /api/curriculums/roadmap/diagnostic", jwtAuth(http.HandlerFunc(ctrl.HandleEvaluateDiagnosticQuiz)))
	mux.Handle("POST /api/admin/curriculums/roadmap/optimize", protectRead(ctrl.HandleOptimizeRoadmap))
	mux.Handle("POST /api/admin/curriculums/roadmap/diagnostic", protectRead(ctrl.HandleEvaluateDiagnosticQuiz))
}

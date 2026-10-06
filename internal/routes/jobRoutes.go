package routes

import (
	"net/http"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/constants"
)

// RegisterJobRoutes registers background job and schedule management routes.
func RegisterJobRoutes(mux *http.ServeMux, jobCtrl *controllers.JobController, userRepo repositories.UserRepository, permCache services.PermissionCache) {
	jwtAuth := middlewares.JWTAccess(userRepo)
	requireJobManage := middlewares.RequirePermission(permCache, constants.PermJobManage)

	protect := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireJobManage(http.HandlerFunc(handler)))
	}

	mux.Handle("GET /api/admin/jobs", protect(jobCtrl.ListJobs))
	mux.Handle("GET /api/admin/jobs/tasks", protect(jobCtrl.ListTasks))
	mux.Handle("GET /api/admin/jobs/{id}", protect(jobCtrl.GetJob))
	mux.Handle("POST /api/admin/jobs/trigger", protect(jobCtrl.Trigger))

	mux.Handle("GET /api/admin/jobs/schedules", protect(jobCtrl.ListSchedules))
	mux.Handle("POST /api/admin/jobs/schedules", protect(jobCtrl.CreateSchedule))
	mux.Handle("PUT /api/admin/jobs/schedules/{id}", protect(jobCtrl.UpdateSchedule))
	mux.Handle("DELETE /api/admin/jobs/schedules/{id}", protect(jobCtrl.DeleteSchedule))
	mux.Handle("POST /api/admin/jobs/schedules/{id}/run-now", protect(jobCtrl.RunScheduleNow))
}

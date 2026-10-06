package routes

import (
	"net/http"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
)

// RegisterMemoryRoutes registers student profile and long-term memory routes.
func RegisterMemoryRoutes(mux *http.ServeMux, memoryCtrl *controllers.MemoryController, userRepo repositories.UserRepository, permCache services.PermissionCache) {
	jwtAuth := middlewares.JWTAccess(userRepo)
	requireManage := middlewares.RequirePermission(permCache, "memory.manage")

	protectManage := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireManage(http.HandlerFunc(handler)))
	}

	protectUser := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(http.HandlerFunc(handler))
	}

	mux.Handle("GET /api/user/profile", protectUser(memoryCtrl.GetUserProfile))
	mux.Handle("PUT /api/user/profile", protectUser(memoryCtrl.UpdateUserProfile))
	mux.Handle("DELETE /api/user/profile", protectUser(memoryCtrl.ClearUserProfileAndMemories))
	mux.Handle("GET /api/user/memories", protectUser(memoryCtrl.ListUserMemories))
	mux.Handle("POST /api/user/memories", protectUser(memoryCtrl.CreateUserMemory))
	mux.Handle("DELETE /api/user/memories/{id}", protectUser(memoryCtrl.DeleteUserMemory))

	mux.Handle("GET /api/admin/users/{id}/memories", protectManage(memoryCtrl.GetAdminStudentProfile))
}

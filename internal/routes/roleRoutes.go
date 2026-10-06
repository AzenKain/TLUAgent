package routes

import (
	"net/http"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/constants"
)

func RegisterRoleRoutes(mux *http.ServeMux, roleCtrl *controllers.RoleController, userRepo repositories.UserRepository, permCache services.PermissionCache) {
	jwtAuth := middlewares.JWTAccess(userRepo)
	requireRoleManage := middlewares.RequirePermission(permCache, constants.PermRoleManage)

	protect := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireRoleManage(http.HandlerFunc(handler)))
	}

	mux.Handle("GET /api/permissions", protect(roleCtrl.ListPermissions))
	mux.Handle("GET /api/roles", protect(roleCtrl.ListRoles))
	mux.Handle("GET /api/roles/{id}", protect(roleCtrl.GetRole))
	mux.Handle("POST /api/roles", protect(roleCtrl.CreateRole))
	mux.Handle("PUT /api/roles/{id}", protect(roleCtrl.UpdateRole))
	mux.Handle("PUT /api/roles/{id}/permissions", protect(roleCtrl.UpdateRolePermissions))
	mux.Handle("DELETE /api/roles/{id}", protect(roleCtrl.DeleteRole))
	mux.Handle("POST /api/roles/reorder", protect(roleCtrl.ReorderRoles))
}

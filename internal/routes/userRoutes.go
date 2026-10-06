package routes

import (
	"net/http"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/constants"
)

func RegisterUserRoutes(mux *http.ServeMux, userCtrl *controllers.UserController, userRepo repositories.UserRepository, permCache services.PermissionCache) {
	jwtAuth := middlewares.JWTAccess(userRepo)
	requireUserManage := middlewares.RequirePermission(permCache, constants.PermUserManage)

	protectSelf := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(http.HandlerFunc(handler))
	}

	protectManage := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireUserManage(http.HandlerFunc(handler)))
	}

	// Current user endpoints
	mux.Handle("GET /api/users/current", protectSelf(userCtrl.GetUserCurrent))
	mux.Handle("PUT /api/users/current", protectSelf(userCtrl.UpdateProfile))
	mux.Handle("POST /api/users/current/password", protectSelf(userCtrl.ChangePassword))
	mux.Handle("POST /api/users/current/revoke-sessions", protectSelf(userCtrl.RevokeUserSessions))

	// User management endpoints (requires user.manage permission)
	mux.Handle("POST /api/users", protectManage(userCtrl.CreateUser))
	mux.Handle("GET /api/users", protectManage(userCtrl.SearchUser))
	mux.Handle("GET /api/users/{id}", protectManage(userCtrl.GetUserByID))
	mux.Handle("PUT /api/users/{id}", protectManage(userCtrl.AdminUpdateProfile))
	mux.Handle("PUT /api/users/{id}/roles", protectManage(userCtrl.ChangeRoleUser))
	mux.Handle("POST /api/users/{id}/password", protectManage(userCtrl.AdminResetPassword))
	mux.Handle("DELETE /api/users/{id}", protectManage(userCtrl.DeleteUser))
	mux.Handle("POST /api/users/{id}/restore", protectManage(userCtrl.RestoreUser))
	mux.Handle("POST /api/users/{id}/revoke-sessions", protectManage(userCtrl.RevokeUserSessions))
}

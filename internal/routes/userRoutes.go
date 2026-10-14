package routes

import (
	"net/http"
	"time"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/constants"
)

func RegisterUserRoutes(mux *http.ServeMux, userCtrl *controllers.UserController, userRepo repositories.UserRepository, permCache services.PermissionCache) {
	jwtAuth := middlewares.JWTAccess(userRepo)
	requireUserManage := middlewares.RequirePermission(permCache, constants.PermUserManage)
	passwordRateLimiter := middlewares.NewRateLimiter(5, time.Minute)

	protectSelf := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(http.HandlerFunc(handler))
	}

	protectPassword := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(passwordRateLimiter.Middleware()(http.HandlerFunc(handler)))
	}

	protectManage := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireUserManage(http.HandlerFunc(handler)))
	}

	protectManagePassword := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireUserManage(passwordRateLimiter.Middleware()(http.HandlerFunc(handler))))
	}

	mux.Handle("GET /api/users/current", protectSelf(userCtrl.GetUserCurrent))
	mux.Handle("PUT /api/users/current", protectSelf(userCtrl.UpdateProfile))
	mux.Handle("POST /api/users/current/password", protectPassword(userCtrl.ChangePassword))
	mux.Handle("POST /api/users/current/revoke-sessions", protectSelf(userCtrl.RevokeUserSessions))

	mux.Handle("POST /api/users", protectManage(userCtrl.CreateUser))
	mux.Handle("GET /api/users", protectManage(userCtrl.SearchUser))
	mux.Handle("GET /api/users/{id}", protectManage(userCtrl.GetUserByID))
	mux.Handle("PUT /api/users/{id}", protectManage(userCtrl.AdminUpdateProfile))
	mux.Handle("PUT /api/users/{id}/roles", protectManage(userCtrl.ChangeRoleUser))
	mux.Handle("POST /api/users/{id}/password", protectManagePassword(userCtrl.AdminResetPassword))
	mux.Handle("DELETE /api/users/{id}", protectManage(userCtrl.DeleteUser))
	mux.Handle("POST /api/users/{id}/restore", protectManage(userCtrl.RestoreUser))
	mux.Handle("POST /api/users/{id}/revoke-sessions", protectManage(userCtrl.RevokeUserSessions))
}

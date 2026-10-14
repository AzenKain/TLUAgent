package routes

import (
	"net/http"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/constants"
)

// RegisterTLUClawRoutes configures HTTP endpoints for administrative autonomous operation.
func RegisterTLUClawRoutes(
	mux *http.ServeMux,
	ctrl *controllers.TLUClawController,
	userRepo repositories.UserRepository,
	permCache services.PermissionCache,
) {
	jwtAuth := middlewares.JWTAccess(userRepo)
	requireRead := middlewares.RequirePermission(permCache, constants.PermTLUClawRead)
	requireManage := middlewares.RequirePermission(permCache, constants.PermTLUClawManage)

	protectRead := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireRead(http.HandlerFunc(handler)))
	}

	protectManage := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireManage(http.HandlerFunc(handler)))
	}

	// Chat & Two-Phase Execution
	mux.Handle("POST /api/admin/tluclaw/chat", protectManage(ctrl.HandleSendMessage))
	mux.Handle("POST /api/admin/tluclaw/confirm", protectManage(ctrl.HandleConfirmAction))
	mux.Handle("POST /api/admin/tluclaw/cancel", protectManage(ctrl.HandleCancelAction))
	mux.Handle("POST /api/admin/tluclaw/rollback", protectManage(ctrl.HandleRollbackAction))

	// Sessions Management
	mux.Handle("GET /api/admin/tluclaw/sessions", protectRead(ctrl.HandleListSessions))
	mux.Handle("POST /api/admin/tluclaw/sessions", protectManage(ctrl.HandleCreateSession))
	mux.Handle("GET /api/admin/tluclaw/sessions/{id}", protectRead(ctrl.HandleGetSessionDetail))
	mux.Handle("DELETE /api/admin/tluclaw/sessions/{id}", protectManage(ctrl.HandleDeleteSession))

	// Snapshots Timeline
	mux.Handle("GET /api/admin/tluclaw/snapshots", protectRead(ctrl.HandleListSnapshots))
}

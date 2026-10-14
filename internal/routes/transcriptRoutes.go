package routes

import (
	"net/http"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/constants"
)

// RegisterTranscriptRoutes configures HTTP routing for student transcripts, OCR diffs, and batch persistence.
func RegisterTranscriptRoutes(
	mux *http.ServeMux,
	ctrl *controllers.TranscriptController,
	userRepo repositories.UserRepository,
	permCache services.PermissionCache,
) {
	jwtAuth := middlewares.JWTAccess(userRepo)
	requireRead := middlewares.RequirePermission(permCache, constants.PermTranscriptRead)
	requireManage := middlewares.RequirePermission(permCache, constants.PermTranscriptManage)

	protectRead := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireRead(http.HandlerFunc(handler)))
	}

	protectManage := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireManage(http.HandlerFunc(handler)))
	}

	mux.Handle("GET /api/transcript/my", protectRead(ctrl.HandleGetMyTranscript))
	mux.Handle("POST /api/transcript/extract-images", protectManage(ctrl.HandleExtractImages))
	mux.Handle("POST /api/transcript/batch-save", protectManage(ctrl.HandleBatchSave))
}

package routes

import (
	"net/http"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/constants"
)

// RegisterEmbeddingRoutes registers admin management routes for embedding settings and ONNX models.
func RegisterEmbeddingRoutes(mux *http.ServeMux, embeddingCtrl *controllers.EmbeddingController, userRepo repositories.UserRepository, permCache services.PermissionCache) {
	jwtAuth := middlewares.JWTAccess(userRepo)
	requireLLMManage := middlewares.RequirePermission(permCache, constants.PermLLMManage)

	protectManage := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireLLMManage(http.HandlerFunc(handler)))
	}

	mux.Handle("GET /api/admin/embedding/settings", protectManage(embeddingCtrl.GetSettings))
	mux.Handle("PUT /api/admin/embedding/settings", protectManage(embeddingCtrl.UpdateSettings))

	mux.Handle("GET /api/admin/embedding/onnx/models", protectManage(embeddingCtrl.ListModels))
	mux.Handle("GET /api/admin/embedding/onnx/catalog", protectManage(embeddingCtrl.GetCatalog))
	mux.Handle("POST /api/admin/embedding/onnx/download", protectManage(embeddingCtrl.StartDownload))
	mux.Handle("GET /api/admin/embedding/onnx/download/status", protectManage(embeddingCtrl.DownloadStatus))
	mux.Handle("POST /api/admin/embedding/onnx/models/{modelId}/validate", protectManage(embeddingCtrl.ValidateModel))
	mux.Handle("POST /api/admin/embedding/onnx/models/{modelId}/activate", protectManage(embeddingCtrl.ActivateModel))
	mux.Handle("DELETE /api/admin/embedding/onnx/models/{modelId}", protectManage(embeddingCtrl.DeleteModel))
}

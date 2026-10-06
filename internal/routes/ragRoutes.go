package routes

import (
	"net/http"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/constants"
)

// RegisterRAGRoutes configures HTTP routing for RAG retrieval and administration endpoints.
func RegisterRAGRoutes(
	mux *http.ServeMux,
	ragCtrl *controllers.RAGController,
	userRepo repositories.UserRepository,
	permCache services.PermissionCache,
) {
	jwtAuth := middlewares.JWTAccess(userRepo)
	requireRead := middlewares.RequirePermission(permCache, constants.PermRAGRead)
	requireManage := middlewares.RequirePermission(permCache, constants.PermRAGManage)

	protectRead := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireRead(http.HandlerFunc(handler)))
	}

	protectManage := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireManage(http.HandlerFunc(handler)))
	}

	guardrailGuard := middlewares.GuardrailQuery()

	mux.Handle("POST /api/rag/search", guardrailGuard(protectRead(ragCtrl.HandleSearch)))
	mux.Handle("GET /api/rag/documents", protectRead(ragCtrl.HandleGetDocuments))
	mux.Handle("GET /api/rag/stats", protectRead(ragCtrl.HandleGetStats))

	mux.Handle("GET /api/admin/documents", protectManage(ragCtrl.HandleGetDocuments))
	mux.Handle("GET /api/admin/documents/{id}", protectManage(ragCtrl.HandleGetDocumentDetail))
	mux.Handle("PATCH /api/admin/documents/{id}/status", protectManage(ragCtrl.HandleUpdateDocumentStatus))
	mux.Handle("GET /api/admin/knowledge-graph", protectManage(ragCtrl.HandleGetKnowledgeGraph))
	mux.Handle("POST /api/admin/knowledge-graph/edges", protectManage(ragCtrl.HandleUpsertKGEdge))
	mux.Handle("DELETE /api/admin/knowledge-graph/edges/{id}", protectManage(ragCtrl.HandleDeleteKGEdge))
	mux.Handle("GET /api/admin/rag/stats", protectManage(ragCtrl.HandleGetStats))
	mux.Handle("GET /api/admin/rag/audit-logs", protectManage(ragCtrl.HandleGetAuditLogs))
	mux.Handle("POST /api/admin/rag/sync", protectManage(ragCtrl.HandleSyncIndex))
}

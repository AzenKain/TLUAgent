package routes

import (
	"net/http"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/constants"
)

// RegisterLLMRoutes registers admin management routes and client discovery routes for LLM.
func RegisterLLMRoutes(mux *http.ServeMux, llmCtrl *controllers.LLMController, userRepo repositories.UserRepository, permCache services.PermissionCache) {
	jwtAuth := middlewares.JWTAccess(userRepo)
	optionalJWT := middlewares.OptionalJWTAccess(userRepo)
	requireLLMManage := middlewares.RequirePermission(permCache, constants.PermLLMManage)

	protectManage := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireLLMManage(http.HandlerFunc(handler)))
	}

	mux.Handle("GET /api/chat/models", optionalJWT(http.HandlerFunc(llmCtrl.GetActiveChatModels)))

	mux.Handle("GET /api/admin/llm/providers", protectManage(llmCtrl.ListProviders))
	mux.Handle("GET /api/admin/llm/providers/{id}", protectManage(llmCtrl.GetProvider))
	mux.Handle("POST /api/admin/llm/providers", protectManage(llmCtrl.CreateProvider))
	mux.Handle("PUT /api/admin/llm/providers/{id}", protectManage(llmCtrl.UpdateProvider))
	mux.Handle("DELETE /api/admin/llm/providers/{id}", protectManage(llmCtrl.DeleteProvider))
	mux.Handle("POST /api/admin/llm/providers/{id}/default", protectManage(llmCtrl.SetDefaultProvider))

	mux.Handle("GET /api/admin/llm/models", protectManage(llmCtrl.ListModels))
	mux.Handle("POST /api/admin/llm/models", protectManage(llmCtrl.CreateModel))
	mux.Handle("PUT /api/admin/llm/models/{id}", protectManage(llmCtrl.UpdateModel))
	mux.Handle("DELETE /api/admin/llm/models/{id}", protectManage(llmCtrl.DeleteModel))
	mux.Handle("POST /api/admin/llm/models/probe-vision", protectManage(llmCtrl.ProbeVision))

	mux.Handle("GET /api/admin/llm/chains", protectManage(llmCtrl.ListChains))
	mux.Handle("GET /api/admin/llm/chains/{id}", protectManage(llmCtrl.GetChain))
	mux.Handle("POST /api/admin/llm/chains", protectManage(llmCtrl.CreateChain))
	mux.Handle("PUT /api/admin/llm/chains/{id}", protectManage(llmCtrl.UpdateChain))
	mux.Handle("DELETE /api/admin/llm/chains/{id}", protectManage(llmCtrl.DeleteChain))
	mux.Handle("POST /api/admin/llm/chains/{id}/nodes", protectManage(llmCtrl.AddChainNode))
	mux.Handle("DELETE /api/admin/llm/chains/{id}/nodes/{nodeId}", protectManage(llmCtrl.RemoveChainNode))
	mux.Handle("PUT /api/admin/llm/chains/{id}/nodes/{nodeId}", protectManage(llmCtrl.UpdateChainNodePriority))
	mux.Handle("POST /api/admin/llm/chains/{id}/nodes/{nodeId}/reset", protectManage(llmCtrl.ResetNodeCircuitBreaker))
	mux.Handle("POST /api/admin/llm/test", protectManage(llmCtrl.TestModel))
}

package routes

import (
	"net/http"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/constants"
)

// RegisterAgentRoutes registers admin management routes for agent prompts and skills.
func RegisterAgentRoutes(mux *http.ServeMux, agentCtrl *controllers.AgentController, userRepo repositories.UserRepository, permCache services.PermissionCache) {
	jwtAuth := middlewares.JWTAccess(userRepo)
	requireManage := middlewares.RequirePermission(permCache, constants.PermLLMManage)

	protectManage := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireManage(http.HandlerFunc(handler)))
	}

	mux.Handle("GET /api/admin/agent/prompts", protectManage(agentCtrl.ListPrompts))
	mux.Handle("GET /api/admin/agent/prompts/{id}", protectManage(agentCtrl.GetPrompt))
	mux.Handle("PUT /api/admin/agent/prompts/{id}", protectManage(agentCtrl.UpdatePrompt))
	mux.Handle("POST /api/admin/agent/prompts/{id}/reset", protectManage(agentCtrl.ResetPrompt))

	mux.Handle("GET /api/admin/agent/skills", protectManage(agentCtrl.ListSkills))
	mux.Handle("GET /api/admin/agent/skills/{id}", protectManage(agentCtrl.GetSkill))
	mux.Handle("POST /api/admin/agent/skills", protectManage(agentCtrl.CreateSkill))
	mux.Handle("PUT /api/admin/agent/skills/{id}", protectManage(agentCtrl.UpdateSkill))
	mux.Handle("PATCH /api/admin/agent/skills/{id}/toggle", protectManage(agentCtrl.ToggleSkill))
	mux.Handle("DELETE /api/admin/agent/skills/{id}", protectManage(agentCtrl.DeleteSkill))
	mux.Handle("POST /api/admin/agent/skills/{id}/reset", protectManage(agentCtrl.ResetSkill))

	mux.Handle("POST /api/admin/agent/preview", protectManage(agentCtrl.PreviewPrompt))
	mux.Handle("GET /api/admin/agent/compactor", protectManage(agentCtrl.GetCompactorSettings))
	mux.Handle("PUT /api/admin/agent/compactor", protectManage(agentCtrl.UpdateCompactorSettings))
}

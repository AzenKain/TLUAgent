package routes

import (
	"net/http"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/constants"
)

// RegisterAPIRoutes registers health, user chat sessions, and institutional chat audit routes.
func RegisterAPIRoutes(
	mux *http.ServeMux,
	healthCtrl *controllers.HealthController,
	chatCtrl *controllers.ChatController,
	adminChatCtrl *controllers.AdminChatController,
	userRepo repositories.UserRepository,
	permCache services.PermissionCache,
	chatQuota services.ChatQuotaService,
) {
	optionalJWT := middlewares.OptionalJWTAccess(userRepo)
	jwtAuth := middlewares.JWTAccess(userRepo)
	requireChatAsk := middlewares.RequireAnyPermission(permCache, constants.PermChatAsk)
	requireChatManage := middlewares.RequirePermission(permCache, constants.PermChatManage)
	chatQuotaGuard := middlewares.ChatQuota(chatQuota)
	guardrailGuard := middlewares.GuardrailQuery()

	protectChatManage := func(handler http.HandlerFunc) http.Handler {
		return jwtAuth(requireChatManage(http.HandlerFunc(handler)))
	}

	mux.HandleFunc("GET /api/health", healthCtrl.HandleHealth)
	mux.HandleFunc("GET /api/info", healthCtrl.HandleInfo)

	mux.Handle("POST /api/chat", optionalJWT(requireChatAsk(guardrailGuard(chatQuotaGuard(http.HandlerFunc(chatCtrl.HandleChat))))))
	mux.Handle("POST /api/chat/stream", optionalJWT(requireChatAsk(guardrailGuard(chatQuotaGuard(http.HandlerFunc(chatCtrl.HandleChatStream))))))
	mux.HandleFunc("POST /api/chat/guest/leave", chatCtrl.HandleGuestLeave)

	mux.Handle("GET /api/chat/sessions", jwtAuth(http.HandlerFunc(chatCtrl.HandleListSessions)))
	mux.Handle("POST /api/chat/sessions", jwtAuth(http.HandlerFunc(chatCtrl.HandleCreateSession)))
	mux.Handle("GET /api/chat/sessions/{id}", jwtAuth(http.HandlerFunc(chatCtrl.HandleGetSession)))
	mux.Handle("PATCH /api/chat/sessions/{id}", jwtAuth(http.HandlerFunc(chatCtrl.HandleUpdateSessionTitle)))
	mux.Handle("DELETE /api/chat/sessions/{id}", jwtAuth(http.HandlerFunc(chatCtrl.HandleDeleteSession)))
	mux.Handle("POST /api/chat/messages/{id}/feedback", jwtAuth(http.HandlerFunc(chatCtrl.HandleMessageFeedback)))

	mux.Handle("GET /api/admin/chats", protectChatManage(adminChatCtrl.HandleListConversations))
	mux.Handle("GET /api/admin/chats/stats", protectChatManage(adminChatCtrl.HandleGetStats))
	mux.Handle("GET /api/admin/chats/{id}", protectChatManage(adminChatCtrl.HandleGetConversationDetail))
	mux.Handle("DELETE /api/admin/chats/{id}", protectChatManage(adminChatCtrl.HandleDeleteConversation))
}

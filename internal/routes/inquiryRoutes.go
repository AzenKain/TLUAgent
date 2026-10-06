package routes

import (
	"net/http"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/constants"
)

// RegisterInquiryRoutes registers student inquiry, lecturer advice, and notification endpoints.
func RegisterInquiryRoutes(
	mux *http.ServeMux,
	inquiryCtrl *controllers.InquiryController,
	notifCtrl *controllers.NotificationController,
	userRepo repositories.UserRepository,
	permCache services.PermissionCache,
) {
	jwtAuth := middlewares.JWTAccess(userRepo)
	requireInquiryRead := middlewares.RequirePermission(permCache, constants.PermInquiryRead)
	requireInquiryCreate := middlewares.RequirePermission(permCache, constants.PermInquiryCreate)
	requireInquiryAnswer := middlewares.RequirePermission(permCache, constants.PermInquiryAnswer)

	protectStudentRead := func(h http.HandlerFunc) http.Handler {
		return jwtAuth(requireInquiryRead(http.HandlerFunc(h)))
	}
	protectStudentCreate := func(h http.HandlerFunc) http.Handler {
		return jwtAuth(requireInquiryCreate(http.HandlerFunc(h)))
	}
	protectAdvisorAnswer := func(h http.HandlerFunc) http.Handler {
		return jwtAuth(requireInquiryAnswer(http.HandlerFunc(h)))
	}

	guardrailGuard := middlewares.GuardrailQuery()

	mux.Handle("POST /api/inquiries", guardrailGuard(protectStudentCreate(inquiryCtrl.CreateInquiry)))
	mux.Handle("GET /api/student/inquiries", protectStudentRead(inquiryCtrl.GetStudentInquiries))
	mux.Handle("GET /api/inquiries/{id}", protectStudentRead(inquiryCtrl.GetInquiryDetail))

	mux.Handle("GET /api/advisor/inquiries", protectAdvisorAnswer(inquiryCtrl.GetTeacherInquiries))
	mux.Handle("GET /api/advisor/inquiries/{id}", protectAdvisorAnswer(inquiryCtrl.GetInquiryDetail))
	mux.Handle("POST /api/advisor/inquiries/{id}/answer", protectAdvisorAnswer(inquiryCtrl.AnswerInquiry))
	mux.Handle("POST /api/advisor/inquiries/{id}/expire", protectAdvisorAnswer(inquiryCtrl.ExpireInquiry))

	mux.Handle("GET /api/student/notifications", jwtAuth(http.HandlerFunc(notifCtrl.ListNotifications)))
	mux.Handle("PATCH /api/student/notifications/{id}/read", jwtAuth(http.HandlerFunc(notifCtrl.MarkAsRead)))
	mux.Handle("POST /api/student/notifications/read-all", jwtAuth(http.HandlerFunc(notifCtrl.MarkAllAsRead)))
	mux.Handle("GET /api/student/notifications/unread-count", jwtAuth(http.HandlerFunc(notifCtrl.GetUnreadCount)))
}

package controllers

import (
	"net/http"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/validator"
)

// NotificationController handles student notifications and read receipts.
type NotificationController struct {
	svc services.NotificationService
}

// NewNotificationController creates an instance of NotificationController.
func NewNotificationController(svc services.NotificationService) *NotificationController {
	return &NotificationController{svc: svc}
}

func (c *NotificationController) ListNotifications(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	userID := getUserID(r)
	if userID == "" || userID == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
		return
	}

	var dto request.ListNotificationsDto
	if err := validator.ValidateQueryDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	items, unread, err := c.svc.ListUserNotifications(ctx, userID, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	res := make([]*response.NotificationResponse, 0, len(items))
	for _, item := range items {
		res = append(res, toNotificationResponse(item))
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data: map[string]any{
			"items":        res,
			"unread_count": unread,
			"page":         dto.GetPage(),
			"limit":        dto.GetLimit(20),
		},
	})
}

func (c *NotificationController) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing notification id"))
		return
	}

	userID := getUserID(r)
	if userID == "" || userID == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
		return
	}

	if err := c.svc.MarkAsRead(ctx, id, userID); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Notification marked as read",
	})
}

func (c *NotificationController) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	userID := getUserID(r)
	if userID == "" || userID == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
		return
	}

	if err := c.svc.MarkAllAsRead(ctx, userID); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "All notifications marked as read",
	})
}

func (c *NotificationController) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	userID := getUserID(r)
	if userID == "" || userID == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
		return
	}

	count, err := c.svc.GetUnreadCount(ctx, userID)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data: response.UnreadCountResponse{
			UnreadCount: count,
		},
	})
}

func toNotificationResponse(m *models.UserNotification) *response.NotificationResponse {
	if m == nil {
		return nil
	}
	return &response.NotificationResponse{
		ID:        m.ID,
		UserID:    m.UserID,
		InquiryID: m.InquiryID,
		Title:     m.Title,
		Content:   m.Content,
		Type:      m.Type,
		IsRead:    m.IsRead,
		CreatedAt: m.CreatedAt,
	}
}

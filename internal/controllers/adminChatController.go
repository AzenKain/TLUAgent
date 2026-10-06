package controllers

import (
	"net/http"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/validator"
)

// AdminChatController manages institutional conversational audit endpoints.
type AdminChatController struct {
	chatSessionService services.ChatSessionService
}

// NewAdminChatController instantiates an admin chat controller.
func NewAdminChatController(session services.ChatSessionService) *AdminChatController {
	return &AdminChatController{
		chatSessionService: session,
	}
}

// HandleListConversations searches and filters institutional chat conversations.
func (c *AdminChatController) HandleListConversations(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.ListAdminConversationsDto
	if err := validator.ValidateQueryDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	summaries, total, err := c.chatSessionService.ListAdminConversations(ctx, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	res := response.BuildPaginatedResponse(summaries, int64(total), dto.GetPage(), dto.GetLimit(20))
	writeJSONResponse(w, http.StatusOK, res)
}

// HandleGetConversationDetail retrieves full conversational context and student info.
func (c *AdminChatController) HandleGetConversationDetail(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	sessionID := r.PathValue("id")
	if sessionID == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing session ID"))
		return
	}

	detail, err := c.chatSessionService.GetAdminConversationDetail(ctx, sessionID)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   detail,
	})
}

// HandleDeleteConversation deletes a conversation from institutional history.
func (c *AdminChatController) HandleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	sessionID := r.PathValue("id")
	if sessionID == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing session ID"))
		return
	}

	if err := c.chatSessionService.DeleteAdminConversation(ctx, sessionID); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Conversation deleted successfully",
	})
}

// HandleGetStats computes analytical metrics for student-AI advisory conversations.
func (c *AdminChatController) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	stats, err := c.chatSessionService.GetAdminStats(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   stats,
	})
}

package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/guardrail"
	"tluagent-web/pkg/jsonx"
	"tluagent-web/pkg/validator"
)

func generateGuestSessionID() string {
	if v7, err := uuid.NewV7(); err == nil {
		return "guest_" + v7.String()
	}
	return "guest_" + uuid.NewString()
}

// ChatController exposes conversational advisory HTTP handlers.
type ChatController struct {
	advisoryService    services.AdvisoryService
	chatSessionService services.ChatSessionService
	streamLimiter      *services.StreamLimiter
}

// NewChatController creates an instance of ChatController.
func NewChatController(advisory services.AdvisoryService, chatSession services.ChatSessionService, streamLimiter *services.StreamLimiter) *ChatController {
	return &ChatController{
		advisoryService:    advisory,
		chatSessionService: chatSession,
		streamLimiter:      streamLimiter,
	}
}

// HandleChat processes synchronous chat consultation.
func (c *ChatController) HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONResponse(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "method not allowed",
		})
		return
	}

	ctx, cancel := requestContext(r, longRequestTimeout)
	defer cancel()

	var req request.ChatRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	if chk := guardrail.CheckQuery(req.Query); chk.Decision != guardrail.DecisionAllow {
		writeJSONResponse(w, http.StatusOK, response.ChatResponse{
			Query:     req.Query,
			Reply:     chk.Response,
			Timestamp: time.Now().Format("15:04:05"),
		})
		return
	}

	for _, h := range req.History {
		if chk := guardrail.CheckQuery(h.Content); chk.Decision == guardrail.DecisionBlockedInjection {
			writeJSONResponse(w, http.StatusOK, response.ChatResponse{
				Query:     req.Query,
				Reply:     chk.Response,
				Timestamp: time.Now().Format("15:04:05"),
			})
			return
		}
	}

	uid := getUserID(r)
	if uid != "" && uid != "0" && req.SessionID != "" && c.chatSessionService != nil {
		if err := c.chatSessionService.ValidateSessionOwnership(ctx, req.SessionID, uid); err != nil {
			apperrors.HandleError(w, err)
			return
		}
	}
	res, err := c.advisoryService.Consult(ctx, uid, &req)
	if err != nil {
		log.Error().Err(err).Msg("failed to process advisory consultation")
		writeJSONResponse(w, http.StatusInternalServerError, response.CommonResponse{
			Status:  false,
			Message: "an error occurred while processing inquiry",
		})
		return
	}

	if uid != "" && uid != "0" && c.chatSessionService != nil {
		sessID, recErr := c.chatSessionService.RecordExchange(ctx, &request.RecordChatExchangeDto{
			SessionID:   req.SessionID,
			UserID:      uid,
			ModelID:     req.ModelID,
			UserQuery:   req.Query,
			BotReply:    res.Reply,
			UserImages:  req.Images,
			BotSources:  res.Sources,
			CanEscalate: res.CanEscalate,
		})
		if recErr != nil {
			log.Error().Err(recErr).Msg("failed to record chat exchange")
		} else {
			res.SessionID = sessID
		}
	} else {
		sessID := req.SessionID
		if sessID == "" {
			sessID = generateGuestSessionID()
		}
		c.advisoryService.RecordGuestExchange(ctx, sessID, req.Query, res.Reply)
		res.SessionID = sessID
	}

	writeJSONResponse(w, http.StatusOK, res)
}

// HandleChatStream processes streaming SSE chat consultation.
func (c *ChatController) HandleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONResponse(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "method not allowed",
		})
		return
	}

	var req request.ChatRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	if chk := guardrail.CheckQuery(req.Query); chk.Decision != guardrail.DecisionAllow {
		writeJSONResponse(w, http.StatusOK, response.ChatResponse{
			Query:     req.Query,
			Reply:     chk.Response,
			Timestamp: time.Now().Format("15:04:05"),
		})
		return
	}

	for _, h := range req.History {
		if chk := guardrail.CheckQuery(h.Content); chk.Decision == guardrail.DecisionBlockedInjection {
			writeJSONResponse(w, http.StatusOK, response.ChatResponse{
				Query:     req.Query,
				Reply:     chk.Response,
				Timestamp: time.Now().Format("15:04:05"),
			})
			return
		}
	}

	releaseStream, ok := c.tryAcquireStream(r)
	if !ok {
		writeJSONResponse(w, http.StatusTooManyRequests, response.CommonResponse{
			Status:  false,
			Message: "too many concurrent chat requests; please try again later",
		})
		return
	}
	defer releaseStream()

	ctx, cancel := requestContext(r, streamRequestTimeout)
	defer cancel()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONResponse(w, http.StatusInternalServerError, response.CommonResponse{
			Status:  false,
			Message: "streaming not supported",
		})
		return
	}

	uid := getUserID(r)
	if uid != "" && uid != "0" && req.SessionID != "" && c.chatSessionService != nil {
		if err := c.chatSessionService.ValidateSessionOwnership(ctx, req.SessionID, uid); err != nil {
			apperrors.HandleError(w, err)
			return
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})
	flusher.Flush()

	if (uid == "" || uid == "0") && req.SessionID != "" && len(req.History) == 0 {
		if !c.advisoryService.HasGuestSession(ctx, req.SessionID) {
			rehydratePayload, _ := jsonx.Marshal(services.StreamEvent{
				Type:      "rehydrate_required",
				SessionID: req.SessionID,
			})
			_, _ = fmt.Fprintf(w, "data: %s\n\n", rehydratePayload)
			flusher.Flush()
			return
		}
	}

	onEvent := func(event services.StreamEvent) error {
		payload, err := jsonx.Marshal(event)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "data: %s\n\n", payload)
		if err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	res, err := c.advisoryService.ConsultStream(ctx, uid, &req, onEvent)
	if err != nil {
		log.Error().Err(err).Msg("failed in advisory streaming")
		errPayload, _ := jsonx.Marshal(services.StreamEvent{
			Type:    "error",
			Message: "an error occurred during conversation; please try again",
		})
		_, _ = fmt.Fprintf(w, "data: %s\n\n", errPayload)
		flusher.Flush()
		return
	}

	var sessionID string
	recordCtx, recordCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer recordCancel()

	if uid != "" && uid != "0" && c.chatSessionService != nil {
		var recErr error
		sessionID, recErr = c.chatSessionService.RecordExchange(recordCtx, &request.RecordChatExchangeDto{
			SessionID:   req.SessionID,
			UserID:      uid,
			ModelID:     req.ModelID,
			UserQuery:   req.Query,
			BotReply:    res.Reply,
			UserImages:  req.Images,
			BotSources:  res.Sources,
			CanEscalate: res.CanEscalate,
		})
		if recErr != nil {
			log.Error().Err(recErr).Msg("failed to record stream chat exchange")
		}
	} else {
		sessionID = req.SessionID
		if sessionID == "" {
			sessionID = generateGuestSessionID()
		}
		c.advisoryService.RecordGuestExchange(recordCtx, sessionID, req.Query, res.Reply)
	}

	donePayload, _ := jsonx.Marshal(services.StreamEvent{
		Type:      "done",
		Sources:   res.Sources,
		Timestamp: res.Timestamp,
		SessionID: sessionID,
	})
	_, _ = fmt.Fprintf(w, "data: %s\n\n", donePayload)
	flusher.Flush()
}

// HandleListSessions retrieves paginated chat conversations for the authenticated user.
func (c *ChatController) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	uid := getUserID(r)
	if uid == "" || uid == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "unauthorized"))
		return
	}

	var dto request.ListUserSessionsDto
	if err := validator.ValidateQueryDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	conversations, total, err := c.chatSessionService.ListUserSessions(ctx, uid, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	res := response.BuildPaginatedResponse(conversations, int64(total), dto.GetPage(), dto.GetLimit(20))
	writeJSONResponse(w, http.StatusOK, res)
}

// HandleCreateSession creates a new conversational session for the authenticated user.
func (c *ChatController) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	uid := getUserID(r)
	if uid == "" || uid == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "unauthorized"))
		return
	}

	var req request.CreateSessionRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	conv, err := c.chatSessionService.CreateSession(ctx, uid, req.Title, req.ModelID)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, response.CommonResponse{
		Status:  true,
		Message: "chat session created successfully",
		Data:    conv,
	})
}

// HandleGetSession retrieves messages and details of a specific conversation.
func (c *ChatController) HandleGetSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	uid := getUserID(r)
	if uid == "" || uid == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "unauthorized"))
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "missing conversation session id"))
		return
	}

	detail, err := c.chatSessionService.GetSessionDetails(ctx, sessionID, uid)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   detail,
	})
}

// HandleUpdateSessionTitle modifies the title of an existing conversation.
func (c *ChatController) HandleUpdateSessionTitle(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	uid := getUserID(r)
	if uid == "" || uid == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "unauthorized"))
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "missing conversation session id"))
		return
	}

	var req request.UpdateSessionTitleRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	if err := c.chatSessionService.UpdateSessionTitle(ctx, sessionID, uid, req.Title); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "session title updated successfully",
	})
}

// HandleDeleteSession removes an existing conversation and associated messages.
func (c *ChatController) HandleDeleteSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	uid := getUserID(r)
	if uid == "" || uid == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "unauthorized"))
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "missing conversation session id"))
		return
	}

	if err := c.chatSessionService.DeleteUserSession(ctx, sessionID, uid); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "session deleted successfully",
	})
}

// HandleMessageFeedback updates rating feedback for a specific chat message.
func (c *ChatController) HandleMessageFeedback(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	uid := getUserID(r)
	if uid == "" || uid == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "unauthorized"))
		return
	}

	messageID := r.PathValue("id")
	if messageID == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "missing message id"))
		return
	}

	var req request.MessageFeedbackRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	if err := c.chatSessionService.UpdateFeedback(ctx, messageID, uid, req.Feedback); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "message feedback updated successfully",
	})
}

func (c *ChatController) tryAcquireStream(r *http.Request) (func(), bool) {
	if c.streamLimiter == nil {
		return func() {}, true
	}
	return c.streamLimiter.TryAcquire(streamClientKey(r))
}

func streamClientKey(r *http.Request) string {
	if uid := getUserID(r); uid != "" && uid != "0" {
		return "user:" + uid
	}
	return "client:" + middlewares.ClientIP(r)
}

// HandleGuestLeave evicts a guest session from RAM cache when the browser window closes.
func (c *ChatController) HandleGuestLeave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONResponse(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "method not allowed",
		})
		return
	}
	var payload struct {
		SessionID string `json:"session_id" validate:"required,min=1,max=128"`
	}
	if errs := validator.ValidateBodyDto(r, &payload); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()
	if payload.SessionID != "" {
		c.advisoryService.EvictGuestSession(ctx, payload.SessionID)
	}
	w.WriteHeader(http.StatusOK)
}

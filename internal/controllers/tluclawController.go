package controllers

import (
	"net/http"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/validator"
)

// TLUClawController coordinates administrative chat sessions, proposals, two-phase commits, and rollbacks.
type TLUClawController struct {
	svc services.TLUClawService
}

// NewTLUClawController creates a new instance of TLUClawController.
func NewTLUClawController(svc services.TLUClawService) *TLUClawController {
	return &TLUClawController{svc: svc}
}

// HandleSendMessage processes an administrative chat message with optional media attachments.
func (c *TLUClawController) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, longRequestTimeout)
	defer cancel()

	adminID := getUserID(r)
	if adminID == "" {
		writeJSONResponse(w, http.StatusUnauthorized, response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
		return
	}

	var req request.TLUClawChatRequest
	if errs := validator.ValidateBodyDto(r, &req); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	resp, err := c.svc.ProcessAdminMessage(ctx, adminID, &req)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   resp,
	})
}

// HandleConfirmAction executes Phase 2 transactional commit using a valid cryptographic nonce.
func (c *TLUClawController) HandleConfirmAction(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	adminID := getUserID(r)
	if adminID == "" {
		writeJSONResponse(w, http.StatusUnauthorized, response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
		return
	}

	var req request.TLUClawConfirmRequest
	if errs := validator.ValidateBodyDto(r, &req); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	resp, err := c.svc.ConfirmSnapshot(ctx, adminID, &req)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   resp,
	})
}

// HandleCancelAction marks a staged preview snapshot as cancelled.
func (c *TLUClawController) HandleCancelAction(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	adminID := getUserID(r)
	if adminID == "" {
		writeJSONResponse(w, http.StatusUnauthorized, response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
		return
	}

	var req request.TLUClawCancelRequest
	if errs := validator.ValidateBodyDto(r, &req); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	resp, err := c.svc.CancelSnapshot(ctx, adminID, &req)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   resp,
	})
}

// HandleRollbackAction inverts and reverses a committed batch snapshot.
func (c *TLUClawController) HandleRollbackAction(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	adminID := getUserID(r)
	if adminID == "" {
		writeJSONResponse(w, http.StatusUnauthorized, response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
		return
	}

	var req request.TLUClawRollbackRequest
	if errs := validator.ValidateBodyDto(r, &req); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	resp, err := c.svc.RollbackBatch(ctx, adminID, req.BatchID)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   resp,
	})
}

// HandleCreateSession initializes a new administrative dialogue session.
func (c *TLUClawController) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	adminID := getUserID(r)
	if adminID == "" {
		writeJSONResponse(w, http.StatusUnauthorized, response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
		return
	}

	var req request.TLUClawCreateSessionRequest
	if errs := validator.ValidateBodyDto(r, &req); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	sess, err := c.svc.CreateSession(ctx, adminID, req.Title)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, response.CommonResponse{
		Status: true,
		Data:   sess,
	})
}

// HandleListSessions returns paginated sessions created by the current admin.
func (c *TLUClawController) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	adminID := getUserID(r)
	if adminID == "" {
		writeJSONResponse(w, http.StatusUnauthorized, response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
		return
	}

	var queryDto request.PaginationDto
	_ = validator.ValidateQueryDto(r, &queryDto)
	limit := int32(queryDto.GetLimit(20))
	offset := int32(queryDto.GetOffset(20))

	sessions, err := c.svc.ListSessions(ctx, adminID, limit, offset)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   sessions,
	})
}

// HandleGetSessionDetail retrieves the session transcript and staged preview snapshots.
func (c *TLUClawController) HandleGetSessionDetail(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	adminID := getUserID(r)
	if adminID == "" {
		writeJSONResponse(w, http.StatusUnauthorized, response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status:  false,
			Message: "missing session id",
		})
		return
	}

	detail, err := c.svc.GetSessionDetail(ctx, sessionID, adminID)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   detail,
	})
}

// HandleDeleteSession removes an administrative dialogue session.
func (c *TLUClawController) HandleDeleteSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	adminID := getUserID(r)
	if adminID == "" {
		writeJSONResponse(w, http.StatusUnauthorized, response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status:  false,
			Message: "missing session id",
		})
		return
	}

	if err := c.svc.DeleteSession(ctx, sessionID, adminID); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "session deleted successfully",
	})
}

// HandleListSnapshots retrieves paginated operational snapshots with optional status filter.
func (c *TLUClawController) HandleListSnapshots(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	adminID := getUserID(r)
	if adminID == "" {
		writeJSONResponse(w, http.StatusUnauthorized, response.CommonResponse{
			Status:  false,
			Message: "unauthorized",
		})
		return
	}

	type listSnapshotsQuery struct {
		request.PaginationDto
		Status string `query:"status"`
	}
	var queryDto listSnapshotsQuery
	_ = validator.ValidateQueryDto(r, &queryDto)
	limit := int32(queryDto.GetLimit(20))
	offset := int32(queryDto.GetOffset(20))

	resp, err := c.svc.ListSnapshots(ctx, queryDto.Status, limit, offset)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   resp.Snapshots,
	})
}

package controllers

import (
	"net/http"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/validator"
)

// TranscriptController manages student transcript queries, image OCR diff analysis, and batch persistence.
type TranscriptController struct {
	svc services.TranscriptService
}

// NewTranscriptController creates a new TranscriptController instance.
func NewTranscriptController(svc services.TranscriptService) *TranscriptController {
	return &TranscriptController{svc: svc}
}

func (c *TranscriptController) HandleGetMyTranscript(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	uid := getUserID(r)
	if uid == "" || uid == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "authentication required"))
		return
	}

	summary, err := c.svc.GetTranscriptSummary(ctx, uid)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   summary,
	})
}

func (c *TranscriptController) HandleExtractImages(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout*3)
	defer cancel()

	uid := getUserID(r)
	if uid == "" || uid == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "authentication required"))
		return
	}

	var req request.ExtractTranscriptImagesRequest
	if errs := validator.ValidateBodyDto(r, &req); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	res, err := c.svc.ExtractAndAnalyzeImages(ctx, uid, req.Images)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

func (c *TranscriptController) HandleBatchSave(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	uid := getUserID(r)
	if uid == "" || uid == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "authentication required"))
		return
	}

	var req request.BatchSaveTranscriptRequest
	if errs := validator.ValidateBodyDto(r, &req); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	res, err := c.svc.BatchSaveTranscriptGrades(ctx, uid, req.Grades)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

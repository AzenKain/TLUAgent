package controllers

import (
	"net/http"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/validator"
)

type EmbeddingController struct {
	svc services.EmbeddingService
}

// NewEmbeddingController creates a new EmbeddingController instance.
func NewEmbeddingController(svc services.EmbeddingService) *EmbeddingController {
	return &EmbeddingController{svc: svc}
}

func (c *EmbeddingController) GetSettings(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	settings, err := c.svc.GetSettings(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}
	writeEmbeddingData(w, settings)
}

func (c *EmbeddingController) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.UpdateEmbeddingSettingsRequest
	if !validateEmbeddingBody(w, r, &dto) {
		return
	}
	settings, err := c.svc.UpdateSettings(ctx, dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}
	writeEmbeddingMessage(w, "Embedding settings updated successfully", settings)
}

func (c *EmbeddingController) ListModels(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	models, err := c.svc.ListOnnxModels(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}
	writeEmbeddingData(w, models)
}

func (c *EmbeddingController) GetCatalog(w http.ResponseWriter, r *http.Request) {
	writeEmbeddingData(w, c.svc.GetCatalog())
}

func (c *EmbeddingController) StartDownload(w http.ResponseWriter, r *http.Request) {
	var dto request.EmbeddingDownloadRequest
	if !validateEmbeddingBody(w, r, &dto) {
		return
	}
	result, err := c.svc.StartDownload(dto.PresetID)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}
	writeEmbeddingMessage(w, "Download job started", result)
}

func (c *EmbeddingController) DownloadStatus(w http.ResponseWriter, r *http.Request) {
	writeEmbeddingData(w, response.EmbeddingDownloadStatusResponse{Jobs: c.svc.ListDownloadJobs()})
}

func (c *EmbeddingController) ValidateModel(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, longRequestTimeout)
	defer cancel()

	modelID := r.PathValue("modelId")
	if modelID == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing model id"))
		return
	}
	result, err := c.svc.ValidateModel(ctx, modelID)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}
	writeEmbeddingData(w, result)
}

func (c *EmbeddingController) ActivateModel(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	modelID := r.PathValue("modelId")
	if modelID == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing model id"))
		return
	}
	settings, err := c.svc.ActivateModel(ctx, modelID)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}
	writeEmbeddingMessage(w, "Model activated successfully", settings)
}

func (c *EmbeddingController) DeleteModel(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	modelID := r.PathValue("modelId")
	if modelID == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing model id"))
		return
	}
	if err := c.svc.DeleteModel(ctx, modelID); err != nil {
		apperrors.HandleError(w, err)
		return
	}
	writeEmbeddingMessage(w, "Model deleted successfully", nil)
}

func writeEmbeddingData(w http.ResponseWriter, data any) {
	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   data,
	})
}

func writeEmbeddingMessage(w http.ResponseWriter, message string, data any) {
	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: message,
		Data:    data,
	})
}

func validateEmbeddingBody(w http.ResponseWriter, r *http.Request, dto any) bool {
	if errs := validator.ValidateBodyDto(r, dto); errs != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: errs,
		})
		return false
	}
	return true
}

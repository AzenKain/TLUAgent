package controllers

import (
	"net/http"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/validator"
)

type LLMController struct {
	svc services.LLMConfigService
}

// NewLLMController creates a new LLMController instance.
func NewLLMController(svc services.LLMConfigService) *LLMController {
	return &LLMController{svc: svc}
}

func (c *LLMController) ListProviders(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	provs, err := c.svc.ListProviders(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   provs,
	})
}

func (c *LLMController) GetProvider(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing provider id"))
		return
	}

	prov, err := c.svc.GetProvider(ctx, id)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   prov,
	})
}

func (c *LLMController) CreateProvider(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.CreateLLMProviderRequest
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	prov, err := c.svc.CreateProvider(ctx, dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, response.CommonResponse{
		Status:  true,
		Message: "Provider created successfully",
		Data:    prov,
	})
}

func (c *LLMController) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing provider id"))
		return
	}

	var dto request.UpdateLLMProviderRequest
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	prov, err := c.svc.UpdateProvider(ctx, id, dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Provider updated successfully",
		Data:    prov,
	})
}

func (c *LLMController) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing provider id"))
		return
	}

	if err := c.svc.DeleteProvider(ctx, id); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Provider deleted successfully",
	})
}

func (c *LLMController) SetDefaultProvider(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing provider id"))
		return
	}

	if err := c.svc.SetDefaultProvider(ctx, id); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Default provider updated successfully",
	})
}

func (c *LLMController) ListModels(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.ListLLMModelsQueryDto
	if err := validator.ValidateQueryDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	models, err := c.svc.ListModels(ctx, dto.ProviderID)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   models,
	})
}

func (c *LLMController) CreateModel(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.CreateLLMModelRequest
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	model, err := c.svc.CreateModel(ctx, dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, response.CommonResponse{
		Status:  true,
		Message: "Model created successfully",
		Data:    model,
	})
}

func (c *LLMController) UpdateModel(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing model id"))
		return
	}

	var dto request.UpdateLLMModelRequest
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	model, err := c.svc.UpdateModel(ctx, id, dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Model updated successfully",
		Data:    model,
	})
}

func (c *LLMController) DeleteModel(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing model id"))
		return
	}

	if err := c.svc.DeleteModel(ctx, id); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Model deleted successfully",
	})
}

func (c *LLMController) ProbeVision(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, longRequestTimeout)
	defer cancel()

	var dto request.ProbeVisionRequest
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	probe, err := c.svc.ProbeModelVisionCapability(ctx, dto.ProviderID, dto.ModelKey)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   probe,
	})
}

func (c *LLMController) GetActiveChatModels(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	models, err := c.svc.GetActiveChatModels(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	var data any = models
	if !isAuthenticatedCaller(r) {
		publicModels := make([]response.ChatModelPublicDTO, len(models))
		for i, m := range models {
			publicModels[i] = response.ChatModelPublicDTO{
				ID:             m.ID,
				Name:           m.Name,
				ContextLength:  m.ContextLength,
				SupportsVision: m.SupportsVision,
			}
		}
		data = publicModels
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   data,
	})
}

func isAuthenticatedCaller(r *http.Request) bool {
	claims := middlewares.GetUserClaims(r.Context())
	return claims != nil && claims.UId != "" && claims.UId != "0"
}

func (c *LLMController) ListChains(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	chains, err := c.svc.ListChains(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   chains,
	})
}

func (c *LLMController) GetChain(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing chain id"))
		return
	}

	chain, err := c.svc.GetChain(ctx, id)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   chain,
	})
}

func (c *LLMController) CreateChain(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.CreateLLMChainRequest
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	chain, err := c.svc.CreateChain(ctx, dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, response.CommonResponse{
		Status: true,
		Data:   chain,
	})
}

func (c *LLMController) UpdateChain(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing chain id"))
		return
	}

	var dto request.UpdateLLMChainRequest
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	chain, err := c.svc.UpdateChain(ctx, id, dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   chain,
	})
}

func (c *LLMController) DeleteChain(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing chain id"))
		return
	}

	if err := c.svc.DeleteChain(ctx, id); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Chain deleted successfully",
	})
}

func (c *LLMController) AddChainNode(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing chain id"))
		return
	}

	var dto request.AddChainNodeRequest
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	node, err := c.svc.AddChainNode(ctx, id, dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, response.CommonResponse{
		Status: true,
		Data:   node,
	})
}

func (c *LLMController) RemoveChainNode(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	chainID := r.PathValue("id")
	nodeID := r.PathValue("nodeId")
	if chainID == "" || nodeID == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing chain id or node id"))
		return
	}

	if err := c.svc.RemoveChainNode(ctx, chainID, nodeID); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Chain node removed successfully",
	})
}

func (c *LLMController) UpdateChainNodePriority(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	chainID := r.PathValue("id")
	nodeID := r.PathValue("nodeId")
	if chainID == "" || nodeID == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing chain id or node id"))
		return
	}

	var dto request.UpdateChainNodePriorityRequest
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	if err := c.svc.UpdateChainNodePriority(ctx, chainID, nodeID, dto); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Chain node updated successfully",
	})
}

func (c *LLMController) ResetNodeCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	chainID := r.PathValue("id")
	nodeID := r.PathValue("nodeId")
	if chainID == "" || nodeID == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing chain id or node id"))
		return
	}

	if err := c.svc.ResetNodeCircuitBreaker(ctx, chainID, nodeID); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Node circuit breaker reset successfully",
	})
}

// TestModel executes an operational test on an LLM model or chain.
func (c *LLMController) TestModel(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, longRequestTimeout)
	defer cancel()

	var dto request.TestLLMModelRequest
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	res, err := c.svc.TestModel(ctx, dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

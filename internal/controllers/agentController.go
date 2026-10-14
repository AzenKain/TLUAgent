package controllers

import (
	"net/http"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/validator"
)

// AgentController handles admin operations for agent prompts, persona, and modular skills.
type AgentController struct {
	svc services.AgentService
}

// NewAgentController creates an instance of AgentController.
func NewAgentController(svc services.AgentService) *AgentController {
	return &AgentController{svc: svc}
}

func (c *AgentController) ListPrompts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	prompts, err := c.svc.GetPrompts(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   prompts,
	})
}

func (c *AgentController) GetPrompt(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing prompt id"))
		return
	}

	prompt, err := c.svc.GetPrompt(ctx, id)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   prompt,
	})
}

func (c *AgentController) UpdatePrompt(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing prompt id"))
		return
	}

	var req request.UpdateAgentPromptRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	updatedBy := "admin"
	if claims := getUserClaims(r); claims != nil {
		updatedBy = claims.UId
	}

	prompt, err := c.svc.UpdatePrompt(ctx, id, req.Title, req.Content, updatedBy)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Prompt updated successfully",
		Data:    prompt,
	})
}

func (c *AgentController) ResetPrompt(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing prompt id"))
		return
	}

	updatedBy := "admin"
	if claims := getUserClaims(r); claims != nil {
		updatedBy = claims.UId
	}

	prompt, err := c.svc.ResetPrompt(ctx, id, updatedBy)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Prompt reset to default successfully",
		Data:    prompt,
	})
}

func (c *AgentController) ListSkills(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	skills, err := c.svc.GetSkills(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   skills,
	})
}

func (c *AgentController) GetSkill(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing skill id"))
		return
	}

	skill, err := c.svc.GetSkill(ctx, id)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   skill,
	})
}

func (c *AgentController) CreateSkill(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var req request.CreateAgentSkillRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	updatedBy := "admin"
	if claims := getUserClaims(r); claims != nil {
		updatedBy = claims.UId
	}

	created, err := c.svc.CreateSkill(ctx, &req, updatedBy)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, response.CommonResponse{
		Status:  true,
		Message: "Skill created successfully",
		Data:    created,
	})
}

func (c *AgentController) UpdateSkill(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing skill id"))
		return
	}

	var req request.UpdateAgentSkillRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	updatedBy := "admin"
	if claims := getUserClaims(r); claims != nil {
		updatedBy = claims.UId
	}

	skill, err := c.svc.UpdateSkill(ctx, id, &req, updatedBy)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Skill updated successfully",
		Data:    skill,
	})
}

func (c *AgentController) ToggleSkill(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing skill id"))
		return
	}

	var req request.ToggleAgentSkillRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	updatedBy := "admin"
	if claims := getUserClaims(r); claims != nil {
		updatedBy = claims.UId
	}

	if err := c.svc.ToggleSkill(ctx, id, req.IsEnabled, updatedBy); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Skill state toggled successfully",
	})
}

func (c *AgentController) DeleteSkill(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing skill id"))
		return
	}

	if err := c.svc.DeleteSkill(ctx, id); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Skill deleted successfully",
	})
}

func (c *AgentController) ResetSkill(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing skill id"))
		return
	}

	updatedBy := "admin"
	if claims := getUserClaims(r); claims != nil {
		updatedBy = claims.UId
	}

	skill, err := c.svc.ResetSkill(ctx, id, updatedBy)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Skill reset to default successfully",
		Data:    skill,
	})
}

func (c *AgentController) PreviewPrompt(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var req request.PreviewAgentPromptRequest
	_ = validator.ValidateBodyDto(r, &req)

	preview, err := c.svc.PreviewSystemPrompt(ctx, req.StudentCohort, req.IncludeSampleRAG)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   response.PreviewAgentPromptResponse{Prompt: preview},
	})
}

func (c *AgentController) GetCompactorSettings(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	settings, err := c.svc.GetCompactorSettings(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   settings,
	})
}

func (c *AgentController) UpdateCompactorSettings(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var req request.UpdateCompactorSettingsRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	settings, err := c.svc.UpdateCompactorSettings(ctx, &req)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Compactor settings updated successfully",
		Data:    settings,
	})
}

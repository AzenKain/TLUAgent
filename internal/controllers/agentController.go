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

	res := make([]*response.AgentPromptResponse, 0, len(prompts))
	for _, p := range prompts {
		res = append(res, p.ToResponse())
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   res,
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
		Data:   prompt.ToResponse(),
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
		Data:    prompt.ToResponse(),
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
		Data:    prompt.ToResponse(),
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

	res := make([]*response.AgentSkillResponse, 0, len(skills))
	for _, s := range skills {
		res = append(res, s.ToResponse())
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   res,
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
		Data:   skill.ToResponse(),
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

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}
	priority := 0
	if req.Priority != nil {
		priority = *req.Priority
	}

	skill := &models.AgentSkill{
		ID:             req.ID,
		Name:           req.Name,
		Description:    req.Description,
		Content:        req.Content,
		ToolDefinition: req.ToolDefinition,
		IsEnabled:      isEnabled,
		Priority:       priority,
		UpdatedBy:      updatedBy,
	}

	created, err := c.svc.CreateSkill(ctx, skill)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, response.CommonResponse{
		Status:  true,
		Message: "Skill created successfully",
		Data:    created.ToResponse(),
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

	priority := 0
	if req.Priority != nil {
		priority = *req.Priority
	}

	skill, err := c.svc.UpdateSkill(ctx, id, req.Name, req.Description, req.Content, req.ToolDefinition, priority, updatedBy)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Skill updated successfully",
		Data:    skill.ToResponse(),
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
		Data:    skill.ToResponse(),
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

package controllers

import (
	"net/http"
	"time"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/validator"
)

// MemoryController handles student profile and long-term memory operations.
type MemoryController struct {
	svc services.MemoryService
}

// NewMemoryController creates an instance of MemoryController.
func NewMemoryController(svc services.MemoryService) *MemoryController {
	return &MemoryController{svc: svc}
}

// GetUserProfile retrieves the current authenticated student's academic profile and active memories.
func (c *MemoryController) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	userID := getUserID(r)
	if userID == "" || userID == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	profile, err := c.svc.GetStudentProfile(ctx, userID)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	memories, err := c.svc.ListUserMemories(ctx, userID)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data: map[string]any{
			"profile":  profile,
			"memories": memories,
		},
	})
}

// UpdateUserProfile updates academic standing, major, or target goals.
func (c *MemoryController) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	userID := getUserID(r)
	if userID == "" || userID == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	var dto request.UpsertStudentProfileDto
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	profile := &models.StudentProfile{
		UserID:           userID,
		Major:            dto.Major,
		Cohort:           dto.Cohort,
		AcademicStanding: dto.AcademicStanding,
		CompletedCredits: dto.CompletedCredits,
		CumulativeGPA:    dto.CumulativeGPA,
		TargetGPA:        dto.TargetGPA,
		AdvisorNotes:     dto.AdvisorNotes,
		UpdatedAt:        time.Now().UTC(),
	}

	updated, err := c.svc.UpsertStudentProfile(ctx, profile)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Profile updated successfully",
		Data:    updated,
	})
}

// ListUserMemories lists all retained memory facts for the user.
func (c *MemoryController) ListUserMemories(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	userID := getUserID(r)
	if userID == "" || userID == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	memories, err := c.svc.ListUserMemories(ctx, userID)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   memories,
	})
}

// CreateUserMemory manually records or saves a student memory item.
func (c *MemoryController) CreateUserMemory(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	userID := getUserID(r)
	if userID == "" || userID == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	var dto request.SaveUserMemoryDto
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	item := &models.UserMemoryItem{
		UserID:      userID,
		Category:    dto.Category,
		MemoryKey:   dto.MemoryKey,
		MemoryValue: dto.MemoryValue,
		Confidence:  dto.Confidence,
		Source:      dto.Source,
	}
	if item.Source == "" {
		item.Source = "USER_DECLARED"
	}

	saved, err := c.svc.SaveUserMemory(ctx, item)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Memory saved successfully",
		Data:    saved,
	})
}

// DeleteUserMemory removes a student memory item.
func (c *MemoryController) DeleteUserMemory(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	userID := getUserID(r)
	if userID == "" || userID == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing memory id"))
		return
	}

	if err := c.svc.DeleteUserMemory(ctx, id, userID); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Memory deleted successfully",
	})
}

// ClearUserProfileAndMemories resets the user's student academic profile and retained memories.
func (c *MemoryController) ClearUserProfileAndMemories(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	userID := getUserID(r)
	if userID == "" || userID == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	if err := c.svc.ClearStudentMemoryAndProfile(ctx, userID); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Student profile and memories cleared successfully",
	})
}

// GetAdminStudentProfile allows advisors/admins to view a student's profile and memory facts.
func (c *MemoryController) GetAdminStudentProfile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	targetUserID := r.PathValue("id")
	if targetUserID == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing student user id"))
		return
	}

	profile, err := c.svc.GetStudentProfile(ctx, targetUserID)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	memories, err := c.svc.ListUserMemories(ctx, targetUserID)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data: map[string]any{
			"profile":  profile,
			"memories": memories,
		},
	})
}

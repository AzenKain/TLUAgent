package controllers

import (
	"net/http"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/validator"
)

type UserController struct {
	userService services.UserService
}

func NewUserController(userService services.UserService) *UserController {
	return &UserController{userService: userService}
}

func (c *UserController) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	claims := getUserClaims(r)
	if claims == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	var dto request.CreateUserDto
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	user, err := c.userService.CreateUser(ctx, claims, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, response.CommonResponse{
		Status:  true,
		Message: "User created successfully",
		Data:    user,
	})
}

func (c *UserController) SearchUser(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.SearchUserDto
	if err := validator.ValidateQueryDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	res, err := c.userService.SearchUser(ctx, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, res)
}

func (c *UserController) GetUserByID(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing user id"))
		return
	}

	user, err := c.userService.GetUserByID(ctx, id)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   user,
	})
}

func (c *UserController) GetUserCurrent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	claims := getUserClaims(r)
	if claims == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	user, err := c.userService.GetUserCurrent(ctx, claims.UId)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   user,
	})
}

func (c *UserController) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	claims := getUserClaims(r)
	if claims == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	var dto request.UpdateProfileDto
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	user, err := c.userService.UpdateProfile(ctx, claims.UId, claims, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Profile updated successfully",
		Data:    user,
	})
}

func (c *UserController) AdminUpdateProfile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	claims := getUserClaims(r)
	if claims == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing user id"))
		return
	}

	var dto request.UpdateProfileDto
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	user, err := c.userService.UpdateProfile(ctx, id, claims, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "User updated successfully",
		Data:    user,
	})
}

func (c *UserController) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	claims := getUserClaims(r)
	if claims == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	var dto request.ChangePasswordDto
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	if err := c.userService.ChangePassword(ctx, claims.UId, &dto); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Password changed successfully",
	})
}

func (c *UserController) AdminResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	claims := getUserClaims(r)
	if claims == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing user id"))
		return
	}

	var dto request.ResetPasswordDto
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	if err := c.userService.AdminResetPassword(ctx, id, claims, &dto); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Password reset successfully",
	})
}

func (c *UserController) ChangeRoleUser(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	claims := getUserClaims(r)
	if claims == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing user id"))
		return
	}

	var dto request.ChangeRoleDto
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	user, err := c.userService.ChangeRoleUser(ctx, id, claims, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "User roles updated successfully",
		Data:    user,
	})
}

func (c *UserController) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	claims := getUserClaims(r)
	if claims == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing user id"))
		return
	}

	if claims.UId == id {
		writeJSONResponse(w, http.StatusForbidden, response.CommonResponse{
			Status:  false,
			Message: "You cannot delete yourself",
		})
		return
	}

	if err := c.userService.DeleteUser(ctx, id, claims); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "User deleted successfully",
	})
}

func (c *UserController) RestoreUser(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	claims := getUserClaims(r)
	if claims == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing user id"))
		return
	}

	user, err := c.userService.RestoreUser(ctx, id, claims)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "User restored successfully",
		Data:    user,
	})
}

func (c *UserController) RevokeUserSessions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	claims := getUserClaims(r)
	if claims == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	targetID := r.PathValue("id")
	if targetID == "current" || targetID == "" {
		targetID = claims.UId
	}

	if err := c.userService.RevokeUserSessions(ctx, targetID, claims); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "User sessions revoked successfully",
	})
}

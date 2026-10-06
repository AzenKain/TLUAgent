package controllers

import (
	"net/http"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/validator"
)

type RoleController struct {
	roleService services.RoleService
}

func NewRoleController(roleService services.RoleService) *RoleController {
	return &RoleController{roleService: roleService}
}

func (c *RoleController) ListRoles(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	roles, err := c.roleService.GetAllRole(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   roles,
	})
}

func (c *RoleController) GetRole(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing role id"))
		return
	}

	role, err := c.roleService.GetRoleByID(ctx, id)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   role,
	})
}

func (c *RoleController) CreateRole(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.CreateRoleDto
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	role, err := c.roleService.CreateRole(ctx, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Role created successfully",
		Data:    role,
	})
}

func (c *RoleController) UpdateRole(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing role id"))
		return
	}

	var dto request.UpdateRoleDto
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	role, err := c.roleService.UpdateRole(ctx, id, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Role updated successfully",
		Data:    role,
	})
}

func (c *RoleController) UpdateRolePermissions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing role id"))
		return
	}

	var dto request.UpdateRolePermissionsDto
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	role, err := c.roleService.UpdateRolePermissions(ctx, id, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Role permissions updated successfully",
		Data:    role,
	})
}

func (c *RoleController) DeleteRole(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing role id"))
		return
	}

	if err := c.roleService.DeleteRole(ctx, id); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Role deleted successfully",
	})
}

func (c *RoleController) ListPermissions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	permissions, err := c.roleService.GetPermissions(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   permissions,
	})
}

func (c *RoleController) ReorderRoles(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.ReorderRolesDto
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	if err := c.roleService.ReorderRoles(ctx, &dto); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Roles reordered successfully",
	})
}

package controllers

import (
	"net/http"
	"slices"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/guardrail"
	"tluagent-web/pkg/validator"
)

// InquiryController handles student academic inquiries and advisor responses.
type InquiryController struct {
	svc services.InquiryService
}

// NewInquiryController creates an instance of InquiryController.
func NewInquiryController(svc services.InquiryService) *InquiryController {
	return &InquiryController{svc: svc}
}

func (c *InquiryController) CreateInquiry(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var req request.CreateInquiryRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	if chk := guardrail.CheckQuery(req.Question + " " + req.Context); chk.Decision == guardrail.DecisionBlockedInjection {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status:  false,
			Message: chk.Response,
		})
		return
	}

	userID := getUserID(r)
	if userID == "" || userID == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Authentication required to create inquiry"))
		return
	}

	inq, err := c.svc.CreateInquiry(ctx, userID, &req)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, response.CommonResponse{
		Status: true,
		Data:   inq,
	})
}

func (c *InquiryController) GetInquiryDetail(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing inquiry id"))
		return
	}

	inq, err := c.svc.GetInquiry(ctx, id)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	claims := getUserClaims(r)
	if claims == nil {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	isAdvisorOrAdmin := slices.Contains(claims.Roles, constants.RoleTypeAdmin) || slices.Contains(claims.Roles, constants.RoleTypeAdvisor)
	if !isAdvisorOrAdmin && inq.UserID != claims.UId {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrForbidden, "Forbidden"))
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   inq,
	})
}

func (c *InquiryController) GetStudentInquiries(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	userID := getUserID(r)
	if userID == "" || userID == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Authentication required"))
		return
	}

	var dto request.ListStudentInquiriesDto
	if err := validator.ValidateQueryDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	items, total, err := c.svc.ListStudentInquiries(ctx, userID, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data: &response.InquiryListResponse{
			Items: items,
			Total: total,
			Page:  dto.GetPage(),
			Limit: dto.GetLimit(20),
		},
	})
}

func (c *InquiryController) GetTeacherInquiries(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.ListTeacherInquiriesDto
	if err := validator.ValidateQueryDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	items, total, err := c.svc.ListTeacherInquiries(ctx, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data: &response.InquiryListResponse{
			Items: items,
			Total: total,
			Page:  dto.GetPage(),
			Limit: dto.GetLimit(20),
		},
	})
}

func (c *InquiryController) AnswerInquiry(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing inquiry id"))
		return
	}

	var req request.AnswerInquiryRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	teacherID := getUserID(r)
	if teacherID == "" || teacherID == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Authentication required to answer inquiry"))
		return
	}

	answered, err := c.svc.AnswerInquiry(ctx, teacherID, id, &req)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   answered,
	})
}

func (c *InquiryController) ExpireInquiry(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "Missing inquiry id"))
		return
	}

	var req request.ExpireInquiryRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	if err := c.svc.ExpireInquiry(ctx, id, req.SupersededByDocID, req.Reason); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Inquiry marked as expired successfully",
	})
}

package controllers

import (
	"net/http"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
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
		Data:   toInquiryResponse(inq),
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

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   toInquiryResponse(inq),
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

	res := make([]*response.InquiryResponse, 0, len(items))
	for _, item := range items {
		res = append(res, toInquiryResponse(item))
	}

	page := dto.GetPage()
	limit := dto.GetLimit(20)
	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data: map[string]any{
			"items": res,
			"total": total,
			"page":  page,
			"limit": limit,
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

	res := make([]*response.InquiryResponse, 0, len(items))
	for _, item := range items {
		res = append(res, toInquiryResponse(item))
	}

	page := dto.GetPage()
	limit := dto.GetLimit(20)
	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data: map[string]any{
			"items": res,
			"total": total,
			"page":  page,
			"limit": limit,
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
		Data:   toInquiryResponse(answered),
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

func toInquiryResponse(m *models.AdvisoryInquiry) *response.InquiryResponse {
	if m == nil {
		return nil
	}
	return &response.InquiryResponse{
		ID:                m.ID,
		UserID:            m.UserID,
		ConversationID:    m.ConversationID,
		StudentName:       m.StudentName,
		StudentCode:       m.StudentCode,
		StudentClass:      m.StudentClass,
		Question:          m.Question,
		Context:           m.Context,
		Status:            string(m.Status),
		TeacherID:         m.TeacherID,
		TeacherName:       m.TeacherName,
		TeacherReply:      m.TeacherReply,
		AnsweredAt:        m.AnsweredAt,
		KnowledgeChunkID:  m.KnowledgeChunkID,
		SupersededByDocID: m.SupersededByDocID,
		IsExpired:         m.IsExpired,
		ExpiredReason:     m.ExpiredReason,
		ExpiredAt:         m.ExpiredAt,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

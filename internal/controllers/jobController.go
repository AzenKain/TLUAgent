package controllers

import (
	"net/http"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/validator"
)

type JobController struct {
	jobs      services.JobService
	schedules services.JobScheduleService
}

func NewJobController(jobs services.JobService, schedules services.JobScheduleService) *JobController {
	return &JobController{
		jobs:      jobs,
		schedules: schedules,
	}
}

func (c *JobController) ListJobs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.ListJobsDto
	if err := validator.ValidateQueryDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	jobs, total, err := c.jobs.ListJobs(ctx, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data: map[string]any{
			"items": jobs,
			"total": total,
		},
	})
}

func (c *JobController) GetJob(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "missing job id"))
		return
	}

	job, err := c.jobs.GetJob(ctx, id)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   job,
	})
}

func (c *JobController) ListTasks(w http.ResponseWriter, r *http.Request) {
	tasks := c.jobs.ListTasks()
	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   tasks,
	})
}

func (c *JobController) Trigger(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.TriggerJobRequest
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	job, err := c.jobs.Trigger(ctx, dto.Type, dto.PayloadJSON)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusAccepted, response.CommonResponse{
		Status:  true,
		Message: "Job triggered successfully",
		Data:    job,
	})
}

func (c *JobController) ListSchedules(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	schedules, err := c.schedules.List(ctx)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   schedules,
	})
}

func (c *JobController) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.UpsertJobScheduleRequest
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	res, err := c.schedules.Create(ctx, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, response.CommonResponse{
		Status:  true,
		Message: "Job schedule created successfully",
		Data:    res,
	})
}

func (c *JobController) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "missing schedule id"))
		return
	}

	var dto request.UpsertJobScheduleRequest
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	res, err := c.schedules.Update(ctx, id, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Job schedule updated successfully",
		Data:    res,
	})
}

func (c *JobController) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "missing schedule id"))
		return
	}

	if err := c.schedules.Delete(ctx, id); err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Job schedule deleted successfully",
	})
}

func (c *JobController) RunScheduleNow(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrBadRequest, "missing schedule id"))
		return
	}

	job, err := c.schedules.RunNow(ctx, id)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusAccepted, response.CommonResponse{
		Status:  true,
		Message: "Job triggered from schedule",
		Data:    job,
	})
}

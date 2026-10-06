package models

import (
	"time"

	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/pkg/convert"
)

type JobEntity struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Progress    int64     `json:"progress"`
	Total       int64     `json:"total"`
	ErrorMsg    *string   `json:"error_msg"`
	PayloadJSON *string   `json:"payload_json"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (e *JobEntity) FromSqlc(res sqlc.Job) *JobEntity {
	e.ID = res.ID
	e.Type = res.Type
	e.Status = res.Status
	e.Progress = res.Progress
	e.Total = res.Total
	e.ErrorMsg = convert.NullStringToStrPtr(res.ErrorMsg)
	e.PayloadJSON = convert.NullStringToStrPtr(res.PayloadJson)
	e.CreatedAt = res.CreatedAt
	e.UpdatedAt = res.UpdatedAt
	return e
}

func (e *JobEntity) ToResponse() *response.JobResponse {
	if e == nil {
		return nil
	}
	return &response.JobResponse{
		ID:          e.ID,
		Type:        e.Type,
		Status:      e.Status,
		Progress:    e.Progress,
		Total:       e.Total,
		ErrorMsg:    e.ErrorMsg,
		PayloadJSON: e.PayloadJSON,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func JobEntitiesToResponse(entities []*JobEntity) []*response.JobResponse {
	out := make([]*response.JobResponse, len(entities))
	for i, e := range entities {
		out[i] = e.ToResponse()
	}
	return out
}

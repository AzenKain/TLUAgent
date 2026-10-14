package request

type TriggerJobRequest struct {
	Type        string `json:"type" validate:"required,min=2,max=64"`
	PayloadJSON string `json:"payload_json,omitempty" validate:"omitempty,max=65536"`
}

type UpsertJobScheduleRequest struct {
	Name            string `json:"name" validate:"required,min=2,max=100"`
	TaskType        string `json:"task_type,omitempty" validate:"omitempty,max=64"`
	JobType         string `json:"job_type,omitempty" validate:"omitempty,max=64"`
	PayloadJSON     string `json:"payload_json,omitempty" validate:"omitempty,max=65536"`
	IntervalMinutes int64  `json:"interval_minutes,omitempty" validate:"omitempty,gte=0,lte=525600"`
	IntervalSec     int64  `json:"interval_sec,omitempty" validate:"omitempty,gte=0,lte=31536000"`
	Enabled         bool   `json:"enabled"`
	IsActive        *bool  `json:"is_active,omitempty" validate:"omitempty"`
}

type ListJobsDto struct {
	Status string `json:"status,omitempty" query:"status" validate:"omitempty,max=50"`
	Type   string `json:"type,omitempty" query:"type" validate:"omitempty,max=64"`
	Limit  int64  `json:"limit,omitempty" query:"limit" validate:"omitempty,min=1,max=100"`
	Offset int64  `json:"offset,omitempty" query:"offset" validate:"omitempty,min=0"`
}

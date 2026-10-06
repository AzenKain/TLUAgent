package response

import "time"

type JobResponse struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Progress    int64     `json:"progress"`
	Total       int64     `json:"total"`
	ErrorMsg    *string   `json:"error_msg,omitempty"`
	PayloadJSON *string   `json:"payload_json,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type JobScheduleResponse struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	TaskType        string     `json:"task_type"`
	JobType         string     `json:"job_type"`
	PayloadJSON     *string    `json:"payload_json,omitempty"`
	IntervalMinutes int64      `json:"interval_minutes"`
	IntervalSec     int64      `json:"interval_sec"`
	Enabled         bool       `json:"enabled"`
	IsActive        bool       `json:"is_active"`
	NextRunAt       time.Time  `json:"next_run_at"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	LastJobID       *string    `json:"last_job_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type JobTaskResponse struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

package response

type EmbeddingRuntimeInfo struct {
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	LibPresent bool   `json:"lib_present"`
	LibPath    string `json:"lib_path"`
}

type EmbeddingSettingsResponse struct {
	Provider          string               `json:"provider"`
	ActiveOnnxModelID *string              `json:"active_onnx_model_id"`
	Runtime           EmbeddingRuntimeInfo `json:"runtime"`
}

type OnnxModelResponse struct {
	ModelID     string  `json:"model_id"`
	DisplayName string  `json:"display_name"`
	Dim         int     `json:"dim"`
	SizeBytes   int64   `json:"size_bytes"`
	Status      string  `json:"status"`
	Error       *string `json:"error"`
	ValidatedAt *string `json:"validated_at"`
	IsActive    bool    `json:"is_active"`
	SHA256      *string `json:"sha256"`
}

type EmbeddingDownloadStartedResponse struct {
	JobID string `json:"job_id"`
}

type EmbeddingDownloadJobResponse struct {
	JobID           string `json:"job_id"`
	PresetID        string `json:"preset_id"`
	State           string `json:"state"`
	DownloadedBytes int64  `json:"downloaded_bytes"`
	TotalBytes      int64  `json:"total_bytes"`
	Error           string `json:"error"`
}

type EmbeddingDownloadStatusResponse struct {
	Jobs []EmbeddingDownloadJobResponse `json:"jobs"`
}

type EmbeddingValidationResponse struct {
	OK        bool    `json:"ok"`
	Dim       int     `json:"dim"`
	LatencyMs float64 `json:"latency_ms"`
	Error     string  `json:"error"`
}

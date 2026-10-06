package request

type UpdateEmbeddingSettingsRequest struct {
	Provider          string  `json:"provider" validate:"required,oneof=api onnx"`
	ActiveOnnxModelID *string `json:"active_onnx_model_id" validate:"omitempty,min=1,max=200"`
}

type EmbeddingDownloadRequest struct {
	PresetID string `json:"preset_id" validate:"required,min=1,max=200"`
}

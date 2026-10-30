package request

type CreateLLMProviderRequest struct {
	ID                   string `json:"id" validate:"required,min=2,max=64"`
	Name                 string `json:"name" validate:"required,min=2,max=100"`
	ProviderType         string `json:"provider_type" validate:"required,oneof=openai gemini openrouter custom"`
	BaseURL              string `json:"base_url" validate:"required,base_url"`
	APIKey               string `json:"api_key" validate:"required,min=1"`
	IsActive             *bool  `json:"is_active" validate:"omitempty"`
	IsDefault            *bool  `json:"is_default" validate:"omitempty"`
	CustomHeadersJSON    string `json:"custom_headers_json" validate:"omitempty,max=65536"`
	TimeoutSeconds       *int   `json:"timeout_seconds" validate:"omitempty,min=1,max=600"`
	MaxRetries           *int   `json:"max_retries" validate:"omitempty,min=0,max=10"`
	RetryInitialWaitMs   *int   `json:"retry_initial_wait_ms" validate:"omitempty,min=0,max=60000"`
	RetryMaxWaitMs       *int   `json:"retry_max_wait_ms" validate:"omitempty,min=0,max=60000"`
	AllowPrivateNetworks *bool  `json:"allow_private_networks" validate:"omitempty"`
}

type UpdateLLMProviderRequest struct {
	Name                 string `json:"name" validate:"required,min=2,max=100"`
	ProviderType         string `json:"provider_type" validate:"required,oneof=openai gemini openrouter custom"`
	BaseURL              string `json:"base_url" validate:"required,base_url"`
	APIKey               string `json:"api_key" validate:"omitempty"`
	IsActive             *bool  `json:"is_active" validate:"omitempty"`
	IsDefault            *bool  `json:"is_default" validate:"omitempty"`
	CustomHeadersJSON    string `json:"custom_headers_json" validate:"omitempty,max=65536"`
	TimeoutSeconds       *int   `json:"timeout_seconds" validate:"omitempty,min=1,max=600"`
	MaxRetries           *int   `json:"max_retries" validate:"omitempty,min=0,max=10"`
	RetryInitialWaitMs   *int   `json:"retry_initial_wait_ms" validate:"omitempty,min=0,max=60000"`
	RetryMaxWaitMs       *int   `json:"retry_max_wait_ms" validate:"omitempty,min=0,max=60000"`
	AllowPrivateNetworks *bool  `json:"allow_private_networks" validate:"omitempty"`
}

type CreateLLMModelRequest struct {
	ID              string `json:"id" validate:"required,min=2,max=64"`
	ProviderID      string `json:"provider_id" validate:"required,min=1,max=64"`
	Name            string `json:"name" validate:"required,min=2,max=100"`
	ModelKey        string `json:"model_key" validate:"required,min=1,max=256"`
	ModelType       string `json:"model_type" validate:"required,oneof=chat embedding rerank"`
	IsActive        *bool  `json:"is_active" validate:"omitempty"`
	IsDefault       *bool  `json:"is_default" validate:"omitempty"`
	OrderIndex      int    `json:"order_index" validate:"min=0,max=10000"`
	ContextLength   int    `json:"context_length" validate:"min=0,max=10000000"`
	VisionMode      string `json:"vision_mode" validate:"required,oneof=auto manual"`
	SupportsVision  *bool  `json:"supports_vision" validate:"omitempty"`
	MaxImages       int    `json:"max_images" validate:"min=0,max=20"`
	ThinkingEnabled *bool  `json:"thinking_enabled" validate:"omitempty"`
	ThinkingBudget  int    `json:"thinking_budget" validate:"min=0"`
	EffortLevel     string `json:"effort_level" validate:"omitempty,oneof=low medium high"`
}

type UpdateLLMModelRequest struct {
	ProviderID      string `json:"provider_id" validate:"required,min=1,max=64"`
	Name            string `json:"name" validate:"required,min=2,max=100"`
	ModelKey        string `json:"model_key" validate:"required,min=1,max=256"`
	ModelType       string `json:"model_type" validate:"required,oneof=chat embedding rerank"`
	IsActive        *bool  `json:"is_active" validate:"omitempty"`
	IsDefault       *bool  `json:"is_default" validate:"omitempty"`
	OrderIndex      int    `json:"order_index" validate:"min=0,max=10000"`
	ContextLength   int    `json:"context_length" validate:"min=0,max=10000000"`
	VisionMode      string `json:"vision_mode" validate:"required,oneof=auto manual"`
	SupportsVision  *bool  `json:"supports_vision" validate:"omitempty"`
	MaxImages       int    `json:"max_images" validate:"min=0,max=20"`
	ThinkingEnabled *bool  `json:"thinking_enabled" validate:"omitempty"`
	ThinkingBudget  int    `json:"thinking_budget" validate:"min=0"`
	EffortLevel     string `json:"effort_level" validate:"omitempty,oneof=low medium high"`
}

type ProbeVisionRequest struct {
	ProviderID string `json:"provider_id" validate:"required,min=1,max=64"`
	ModelKey   string `json:"model_key" validate:"required,min=1,max=256"`
}

type CreateLLMChainRequest struct {
	ID               string `json:"id" validate:"required,min=2,max=64"`
	ChainType        string `json:"chain_type" validate:"required,oneof=chat rerank embedding audit"`
	Name             string `json:"name" validate:"required,min=2,max=100"`
	Description      string `json:"description" validate:"omitempty,max=500"`
	IsActive         *bool  `json:"is_active" validate:"omitempty"`
	FailureThreshold int    `json:"failure_threshold" validate:"min=1,max=20"`
	CooldownSeconds  int    `json:"cooldown_seconds" validate:"min=5,max=86400"`
	RetryCount       int    `json:"retry_count" validate:"min=0,max=10"`
}

type UpdateLLMChainRequest struct {
	Name             string `json:"name" validate:"required,min=2,max=100"`
	Description      string `json:"description" validate:"omitempty,max=500"`
	IsActive         *bool  `json:"is_active" validate:"omitempty"`
	FailureThreshold int    `json:"failure_threshold" validate:"min=1,max=20"`
	CooldownSeconds  int    `json:"cooldown_seconds" validate:"min=5,max=86400"`
	RetryCount       int    `json:"retry_count" validate:"min=0,max=10"`
}

type AddChainNodeRequest struct {
	ModelID  string `json:"model_id" validate:"required,min=1,max=64"`
	Priority int    `json:"priority" validate:"min=1"`
	IsActive *bool  `json:"is_active" validate:"omitempty"`
}

type UpdateChainNodePriorityRequest struct {
	Priority int   `json:"priority" validate:"min=1"`
	IsActive *bool `json:"is_active" validate:"omitempty"`
}

type TestLLMModelRequest struct {
	ModelID      string   `json:"model_id,omitempty" validate:"omitempty,max=100"`
	ChainID      string   `json:"chain_id,omitempty" validate:"omitempty,max=100"`
	SystemPrompt string   `json:"system_prompt,omitempty" validate:"omitempty,max=50000"`
	Prompt       string   `json:"prompt" validate:"required,min=1"`
	Images       []string `json:"images,omitempty" validate:"omitempty,max=5,dive,image_url"`
	Temperature  *float64 `json:"temperature,omitempty" validate:"omitempty,min=0,max=2"`
	MaxTokens    *int     `json:"max_tokens,omitempty" validate:"omitempty,min=1,max=128000"`
}

type ListLLMModelsQueryDto struct {
	ProviderID string `json:"provider_id,omitempty" query:"provider_id" validate:"omitempty,max=100"`
}

// UpdateVisionSettingsRequest defines parameters for updating vision strategy and fallback model.
type UpdateVisionSettingsRequest struct {
	VisionStrategy      string `json:"vision_strategy" validate:"required,oneof=fallback degrade reject"`
	VisionFallbackModel string `json:"vision_fallback_model" validate:"omitempty,max=256"`
}

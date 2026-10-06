package request

type CreateLLMProviderRequest struct {
	ID                   string `json:"id" validate:"required,min=2,max=64"`
	Name                 string `json:"name" validate:"required,min=2,max=100"`
	ProviderType         string `json:"provider_type" validate:"required,oneof=openai gemini openrouter custom"`
	BaseURL              string `json:"base_url" validate:"required,base_url"`
	APIKey               string `json:"api_key" validate:"required,min=1"`
	IsActive             *bool  `json:"is_active"`
	IsDefault            *bool  `json:"is_default"`
	CustomHeadersJSON    string `json:"custom_headers_json"`
	TimeoutSeconds       *int   `json:"timeout_seconds"`
	MaxRetries           *int   `json:"max_retries"`
	RetryInitialWaitMs   *int   `json:"retry_initial_wait_ms"`
	RetryMaxWaitMs       *int   `json:"retry_max_wait_ms"`
	AllowPrivateNetworks *bool  `json:"allow_private_networks"`
}

type UpdateLLMProviderRequest struct {
	Name                 string `json:"name" validate:"required,min=2,max=100"`
	ProviderType         string `json:"provider_type" validate:"required,oneof=openai gemini openrouter custom"`
	BaseURL              string `json:"base_url" validate:"required,base_url"`
	APIKey               string `json:"api_key"`
	IsActive             *bool  `json:"is_active"`
	IsDefault            *bool  `json:"is_default"`
	CustomHeadersJSON    string `json:"custom_headers_json"`
	TimeoutSeconds       *int   `json:"timeout_seconds"`
	MaxRetries           *int   `json:"max_retries"`
	RetryInitialWaitMs   *int   `json:"retry_initial_wait_ms"`
	RetryMaxWaitMs       *int   `json:"retry_max_wait_ms"`
	AllowPrivateNetworks *bool  `json:"allow_private_networks"`
}

type CreateLLMModelRequest struct {
	ID              string `json:"id" validate:"required,min=2,max=64"`
	ProviderID      string `json:"provider_id" validate:"required"`
	Name            string `json:"name" validate:"required,min=2,max=100"`
	ModelKey        string `json:"model_key" validate:"required,min=1"`
	ModelType       string `json:"model_type" validate:"required,oneof=chat embedding rerank"`
	IsActive        *bool  `json:"is_active"`
	IsDefault       *bool  `json:"is_default"`
	OrderIndex      int    `json:"order_index"`
	ContextLength   int    `json:"context_length"`
	VisionMode      string `json:"vision_mode" validate:"required,oneof=auto manual"`
	SupportsVision  *bool  `json:"supports_vision"`
	MaxImages       int    `json:"max_images" validate:"min=0,max=20"`
	ThinkingEnabled *bool  `json:"thinking_enabled"`
	ThinkingBudget  int    `json:"thinking_budget" validate:"min=0"`
	EffortLevel     string `json:"effort_level" validate:"omitempty,oneof=low medium high"`
}

type UpdateLLMModelRequest struct {
	ProviderID      string `json:"provider_id" validate:"required"`
	Name            string `json:"name" validate:"required,min=2,max=100"`
	ModelKey        string `json:"model_key" validate:"required,min=1"`
	ModelType       string `json:"model_type" validate:"required,oneof=chat embedding rerank"`
	IsActive        *bool  `json:"is_active"`
	IsDefault       *bool  `json:"is_default"`
	OrderIndex      int    `json:"order_index"`
	ContextLength   int    `json:"context_length"`
	VisionMode      string `json:"vision_mode" validate:"required,oneof=auto manual"`
	SupportsVision  *bool  `json:"supports_vision"`
	MaxImages       int    `json:"max_images" validate:"min=0,max=20"`
	ThinkingEnabled *bool  `json:"thinking_enabled"`
	ThinkingBudget  int    `json:"thinking_budget" validate:"min=0"`
	EffortLevel     string `json:"effort_level" validate:"omitempty,oneof=low medium high"`
}

type ProbeVisionRequest struct {
	ProviderID string `json:"provider_id" validate:"required"`
	ModelKey   string `json:"model_key" validate:"required"`
}

type CreateLLMChainRequest struct {
	ID               string `json:"id" validate:"required,min=2,max=64"`
	ChainType        string `json:"chain_type" validate:"required,oneof=chat rerank embedding"`
	Name             string `json:"name" validate:"required,min=2,max=100"`
	Description      string `json:"description"`
	IsActive         *bool  `json:"is_active"`
	FailureThreshold int    `json:"failure_threshold" validate:"min=1,max=20"`
	CooldownSeconds  int    `json:"cooldown_seconds" validate:"min=5,max=86400"`
	RetryCount       int    `json:"retry_count" validate:"min=0,max=10"`
}

type UpdateLLMChainRequest struct {
	Name             string `json:"name" validate:"required,min=2,max=100"`
	Description      string `json:"description"`
	IsActive         *bool  `json:"is_active"`
	FailureThreshold int    `json:"failure_threshold" validate:"min=1,max=20"`
	CooldownSeconds  int    `json:"cooldown_seconds" validate:"min=5,max=86400"`
	RetryCount       int    `json:"retry_count" validate:"min=0,max=10"`
}

type AddChainNodeRequest struct {
	ModelID  string `json:"model_id" validate:"required"`
	Priority int    `json:"priority" validate:"min=1"`
	IsActive *bool  `json:"is_active"`
}

type UpdateChainNodePriorityRequest struct {
	Priority int   `json:"priority" validate:"min=1"`
	IsActive *bool `json:"is_active"`
}

type TestLLMModelRequest struct {
	ModelID      string   `json:"model_id,omitempty"`
	ChainID      string   `json:"chain_id,omitempty"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	Prompt       string   `json:"prompt" validate:"required,min=1"`
	Images       []string `json:"images,omitempty"`
	Temperature  *float64 `json:"temperature,omitempty"`
	MaxTokens    *int     `json:"max_tokens,omitempty"`
}

type ListLLMModelsQueryDto struct {
	ProviderID string `json:"provider_id,omitempty" query:"provider_id" validate:"omitempty,max=100"`
}

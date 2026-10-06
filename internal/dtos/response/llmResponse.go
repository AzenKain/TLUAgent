package response

type LLMProviderResponse struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	ProviderType         string `json:"provider_type"`
	BaseURL              string `json:"base_url"`
	MaskedAPIKey         string `json:"masked_api_key"`
	IsActive             bool   `json:"is_active"`
	IsDefault            bool   `json:"is_default"`
	CustomHeadersJSON    string `json:"custom_headers_json"`
	TimeoutSeconds       int    `json:"timeout_seconds"`
	MaxRetries           int    `json:"max_retries"`
	RetryInitialWaitMs   int    `json:"retry_initial_wait_ms"`
	RetryMaxWaitMs       int    `json:"retry_max_wait_ms"`
	AllowPrivateNetworks bool   `json:"allow_private_networks"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

type LLMModelResponse struct {
	ID              string `json:"id"`
	ProviderID      string `json:"provider_id"`
	ProviderName    string `json:"provider_name,omitempty"`
	Name            string `json:"name"`
	ModelKey        string `json:"model_key"`
	ModelType       string `json:"model_type"`
	IsActive        bool   `json:"is_active"`
	IsDefault       bool   `json:"is_default"`
	OrderIndex      int    `json:"order_index"`
	ContextLength   int    `json:"context_length"`
	VisionMode      string `json:"vision_mode"`
	SupportsVision  bool   `json:"supports_vision"`
	MaxImages       int    `json:"max_images"`
	ThinkingEnabled bool   `json:"thinking_enabled"`
	ThinkingBudget  int    `json:"thinking_budget"`
	EffortLevel     string `json:"effort_level"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type ChatModelClientDTO struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ModelKey        string `json:"model_key"`
	IsDefault       bool   `json:"is_default"`
	ContextLength   int    `json:"context_length"`
	SupportsVision  bool   `json:"supports_vision"`
	MaxImages       int    `json:"max_images"`
	ThinkingEnabled bool   `json:"thinking_enabled"`
	ThinkingBudget  int    `json:"thinking_budget"`
	EffortLevel     string `json:"effort_level"`
	ProviderName    string `json:"provider_name"`
}

// ChatModelPublicDTO is the anonymous-safe subset of the chat model catalog.
type ChatModelPublicDTO struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ContextLength   int    `json:"context_length"`
	SupportsVision  bool   `json:"supports_vision"`
	ThinkingEnabled bool   `json:"thinking_enabled"`
	ThinkingBudget  int    `json:"thinking_budget"`
	EffortLevel     string `json:"effort_level"`
}

type ProbeVisionResponse struct {
	ModelKey           string `json:"model_key"`
	SupportsVision     bool   `json:"supports_vision"`
	SuggestedMaxImages int    `json:"suggested_max_images"`
}

type LLMChainNodeResponse struct {
	ID                       string  `json:"id"`
	ChainID                  string  `json:"chain_id"`
	ModelID                  string  `json:"model_id"`
	Priority                 int     `json:"priority"`
	IsActive                 bool    `json:"is_active"`
	ModelName                string  `json:"model_name"`
	ModelKey                 string  `json:"model_key"`
	ModelType                string  `json:"model_type"`
	ContextLength            int     `json:"context_length"`
	SupportsVision           bool    `json:"supports_vision"`
	MaxImages                int     `json:"max_images"`
	ProviderID               string  `json:"provider_id"`
	ProviderName             string  `json:"provider_name"`
	ProviderType             string  `json:"provider_type"`
	BaseURL                  string  `json:"base_url"`
	TimeoutSeconds           int     `json:"timeout_seconds"`
	MaxRetries               int     `json:"max_retries"`
	HealthState              string  `json:"health_state"`
	ConsecutiveFailures      int     `json:"consecutive_failures"`
	CooldownUntil            *string `json:"cooldown_until,omitempty"`
	RemainingCooldownSeconds int64   `json:"remaining_cooldown_seconds"`
}

type LLMChainResponse struct {
	ID               string                 `json:"id"`
	ChainType        string                 `json:"chain_type"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	IsActive         bool                   `json:"is_active"`
	FailureThreshold int                    `json:"failure_threshold"`
	CooldownSeconds  int                    `json:"cooldown_seconds"`
	RetryCount       int                    `json:"retry_count"`
	Nodes            []LLMChainNodeResponse `json:"nodes"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
}

type TestLLMModelResponse struct {
	Success          bool           `json:"success"`
	ModelID          string         `json:"model_id,omitempty"`
	ModelName        string         `json:"model_name,omitempty"`
	ModelKey         string         `json:"model_key,omitempty"`
	ProviderName     string         `json:"provider_name,omitempty"`
	ChainID          string         `json:"chain_id,omitempty"`
	NodeUsed         string         `json:"node_used,omitempty"`
	Content          string         `json:"content"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	LatencyMs        int64          `json:"latency_ms"`
	PromptTokens     int            `json:"prompt_tokens"`
	CompletionTokens int            `json:"completion_tokens"`
	ReasoningTokens  int            `json:"reasoning_tokens"`
	TotalTokens      int            `json:"total_tokens"`
	FinishReason     string         `json:"finish_reason,omitempty"`
	Error            string         `json:"error,omitempty"`
	RawResponse      map[string]any `json:"raw_response,omitempty"`
}

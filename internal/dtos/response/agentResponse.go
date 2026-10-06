package response

import (
	"time"
)

// AgentPromptResponse represents an agent prompt returned by the API.
type AgentPromptResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by,omitempty"`
}

// AgentSkillResponse represents an agent skill returned by the API.
type AgentSkillResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Content        string    `json:"content"`
	ToolDefinition string    `json:"tool_definition,omitempty"`
	IsEnabled      bool      `json:"is_enabled"`
	Priority       int       `json:"priority"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      string    `json:"updated_by,omitempty"`
}

// PreviewAgentPromptResponse represents the rendered prompt preview response.
type PreviewAgentPromptResponse struct {
	Prompt string `json:"prompt"`
}

// CompactorSettingsResponse represents compactor parameters for admin.
type CompactorSettingsResponse struct {
	MaxContextTokens      int     `json:"max_context_tokens"`
	CompactThresholdRatio float64 `json:"compact_threshold_ratio"`
	KeepRecentTurns       int     `json:"keep_recent_turns"`
}

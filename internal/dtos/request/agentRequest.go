package request

// UpdateAgentPromptRequest represents the payload to update an agent prompt.
type UpdateAgentPromptRequest struct {
	Title   string `json:"title" validate:"required,min=1,max=128"`
	Content string `json:"content" validate:"required,min=1"`
}

// CreateAgentSkillRequest represents the payload to create a new agent skill.
type CreateAgentSkillRequest struct {
	ID             string `json:"id" validate:"required,min=2,max=64"`
	Name           string `json:"name" validate:"required,min=2,max=128"`
	Description    string `json:"description" validate:"required"`
	Content        string `json:"content" validate:"required,min=1"`
	ToolDefinition string `json:"tool_definition"`
	IsEnabled      *bool  `json:"is_enabled"`
	Priority       *int   `json:"priority"`
}

// UpdateAgentSkillRequest represents the payload to update an existing agent skill.
type UpdateAgentSkillRequest struct {
	Name           string `json:"name" validate:"required,min=2,max=128"`
	Description    string `json:"description" validate:"required"`
	Content        string `json:"content" validate:"required,min=1"`
	ToolDefinition string `json:"tool_definition"`
	Priority       *int   `json:"priority"`
}

// ToggleAgentSkillRequest represents the payload to toggle a skill's enabled state.
type ToggleAgentSkillRequest struct {
	IsEnabled bool `json:"is_enabled"`
}

// PreviewAgentPromptRequest represents the payload to preview an assembled system prompt.
type PreviewAgentPromptRequest struct {
	StudentCohort    string `json:"student_cohort"`
	IncludeSampleRAG bool   `json:"include_sample_rag"`
}

// UpdateCompactorSettingsRequest represents admin payload to configure context token compactor.
type UpdateCompactorSettingsRequest struct {
	MaxContextTokens      int     `json:"max_context_tokens" validate:"required,min=1000,max=2000000"`
	CompactThresholdRatio float64 `json:"compact_threshold_ratio" validate:"required,gt=0,lte=1"`
	KeepRecentTurns       int     `json:"keep_recent_turns" validate:"required,min=2,max=100"`
}

package request

// ChatMessageDTO represents a message payload sent during conversational advisory.
type ChatMessageDTO struct {
	Role    string   `json:"role" validate:"omitempty,oneof=user assistant"`
	Content string   `json:"content" validate:"max=65536"`
	Images  []string `json:"images,omitempty" validate:"omitempty,max=5,dive,image_url"`
}

// ChatRequest encapsulates advisory query parameters and history context.
type ChatRequest struct {
	Query           string           `json:"query" validate:"required,min=1,max=4000"`
	SessionID       string           `json:"session_id,omitempty"`
	ModelID         string           `json:"model_id,omitempty"`
	Images          []string         `json:"images,omitempty" validate:"omitempty,max=5,dive,image_url"`
	History         []ChatMessageDTO `json:"history,omitempty" validate:"omitempty,max=50,dive"`
	ThinkingEnabled *bool            `json:"thinking_enabled,omitempty"`
	ThinkingBudget  *int             `json:"thinking_budget,omitempty"`
	EffortLevel     string           `json:"effort_level,omitempty"`
}

// CreateSessionRequest encapsulates input for creating a conversation.
type CreateSessionRequest struct {
	Title   string `json:"title" validate:"required,min=1,max=200"`
	ModelID string `json:"model_id,omitempty"`
}

// UpdateSessionTitleRequest encapsulates input for updating a conversation title.
type UpdateSessionTitleRequest struct {
	Title string `json:"title" validate:"required,min=1,max=200"`
}

// MessageFeedbackRequest encapsulates feedback payload for an assistant reply.
type MessageFeedbackRequest struct {
	Feedback string `json:"feedback" validate:"required,oneof=up down clear"`
}

// ListUserSessionsDto encapsulates query parameters for listing user chat sessions.
type ListUserSessionsDto struct {
	PaginationDto
}

// ListAdminConversationsDto encapsulates query parameters for admin conversational audit.
type ListAdminConversationsDto struct {
	PaginationDto
	Q        string `json:"q,omitempty" query:"q" validate:"omitempty,max=200"`
	Feedback string `json:"feedback,omitempty" query:"feedback" validate:"omitempty,oneof=liked disliked"`
	ModelID  string `json:"model_id,omitempty" query:"model_id" validate:"omitempty,max=100"`
}

// RecordChatExchangeDto encapsulates data for saving an exchange in chat history.
type RecordChatExchangeDto struct {
	SessionID  string   `json:"session_id"`
	UserID     string   `json:"user_id"`
	ModelID    string   `json:"model_id"`
	UserQuery  string   `json:"user_query"`
	BotReply   string   `json:"bot_reply"`
	UserImages []string `json:"user_images,omitempty"`
	BotSources []string `json:"bot_sources,omitempty"`
}

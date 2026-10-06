package models

// ChatConversationEntity represents a user chat session stored in the database.
type ChatConversationEntity struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Title     string `json:"title"`
	ModelID   string `json:"model_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ChatMessageEntity represents an individual message in a chat conversation.
type ChatMessageEntity struct {
	ID             string   `json:"id"`
	ConversationID string   `json:"conversation_id"`
	Sender         string   `json:"sender"`
	Content        string   `json:"content"`
	Sources        []string `json:"sources"`
	Images         []string `json:"images"`
	Feedback       string   `json:"feedback,omitempty"`
	CreatedAt      string   `json:"created_at"`
}

// AdminConversationSummary represents aggregated conversation metadata for administration.
type AdminConversationSummary struct {
	ID                string `json:"id"`
	UserID            string `json:"user_id"`
	UserEmail         string `json:"user_email"`
	UserFullName      string `json:"user_name"`
	StudentCode       string `json:"student_code"`
	Title             string `json:"title"`
	ModelID           string `json:"model_id"`
	MessageCount      int    `json:"total_messages"`
	FeedbackUpCount   int    `json:"thumbs_up"`
	FeedbackDownCount int    `json:"thumbs_down"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// ConversationDetailResponse bundles conversation metadata with its message trajectory.
type ConversationDetailResponse struct {
	Conversation *ChatConversationEntity `json:"conversation"`
	User         *UserEntity             `json:"user,omitempty"`
	Messages     []*ChatMessageEntity    `json:"messages"`
}

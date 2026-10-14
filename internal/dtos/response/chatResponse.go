package response

type ChatResponse struct {
	SessionID   string                              `json:"session_id,omitempty"`
	Query       string                              `json:"query"`
	Reply       string                              `json:"reply"`
	Sources     []string                            `json:"sources"`
	Timestamp   string                              `json:"timestamp"`
	CanEscalate bool                                `json:"can_escalate,omitempty"`
	Roadmap     *RoadmapOptimizationResultResponse `json:"roadmap,omitempty"`
	DidCompact  bool                                `json:"did_compact,omitempty"`
}

// ChatSessionResponse mirrors the serialized shape of a chat conversation entity.
type ChatSessionResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Title     string `json:"title"`
	ModelID   string `json:"model_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ChatMessageResponse mirrors the serialized shape of a chat message entity.
type ChatMessageResponse struct {
	ID             string   `json:"id"`
	ConversationID string   `json:"conversation_id"`
	Sender         string   `json:"sender"`
	Content        string   `json:"content"`
	Sources        []string `json:"sources"`
	Images         []string `json:"images"`
	Feedback       string   `json:"feedback,omitempty"`
	CanEscalate    bool     `json:"can_escalate"`
	CreatedAt      string   `json:"created_at"`
}

// ConversationUserResponse mirrors the serialized shape of a user entity embedded in conversation detail.
type ConversationUserResponse struct {
	ID           string                `json:"id"`
	Email        string                `json:"email"`
	FullName     string                `json:"full_name"`
	StudentCode  string                `json:"student_code"`
	AvatarUrl    string                `json:"avatar_url"`
	AuthProvider string                `json:"auth_provider"`
	TokenVersion int32                 `json:"token_version"`
	Roles        []*RoleSimpleResponse `json:"roles"`
	IsDeleted    bool                  `json:"is_deleted"`
	CreatedAt    string                `json:"created_at"`
	UpdatedAt    string                `json:"updated_at"`
}

// ConversationDetailResponse bundles conversation metadata with its message trajectory.
type ConversationDetailResponse struct {
	Conversation *ChatSessionResponse      `json:"conversation"`
	User         *ConversationUserResponse `json:"user,omitempty"`
	Messages     []*ChatMessageResponse    `json:"messages"`
}

// AdminConversationSummaryResponse mirrors the serialized shape of an admin conversation summary.
type AdminConversationSummaryResponse struct {
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

// AdminChatStatsResponse carries aggregated chat moderation metrics.
type AdminChatStatsResponse struct {
	TotalConversations int     `json:"total_conversations"`
	TotalMessages      int     `json:"total_messages"`
	FeedbackUp         int     `json:"feedback_up"`
	FeedbackDown       int     `json:"feedback_down"`
	SatisfactionRate   float64 `json:"satisfaction_rate"`
}

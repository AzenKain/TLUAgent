package response

type ChatResponse struct {
	SessionID string   `json:"session_id,omitempty"`
	Query     string   `json:"query"`
	Reply     string   `json:"reply"`
	Sources     []string `json:"sources"`
	Timestamp   string   `json:"timestamp"`
	CanEscalate bool     `json:"can_escalate,omitempty"`
}

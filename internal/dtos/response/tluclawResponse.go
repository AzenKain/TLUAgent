package response

import (
	"time"
)

// TLUClawDiffItemResponse represents an atomic difference for client rendering.
type TLUClawDiffItemResponse struct {
	EntityType string `json:"entity_type"`
	Action     string `json:"action"`
	Key        string `json:"key"`
	Name       string `json:"name,omitempty"`
	OldValue   string `json:"old_value,omitempty"`
	NewValue   string `json:"new_value,omitempty"`
	Note       string `json:"note,omitempty"`
}

// TLUClawBlastRadiusResponse encapsulates impact analysis for client rendering.
type TLUClawBlastRadiusResponse struct {
	AffectedCohorts    []string `json:"affected_cohorts,omitempty"`
	AffectedMajors     []string `json:"affected_majors,omitempty"`
	SupersededDocIDs   []string `json:"superseded_doc_ids,omitempty"`
	SupersededDocNames []string `json:"superseded_doc_names,omitempty"`
	ImpactSummary      string   `json:"impact_summary"`
	RiskLevel          string   `json:"risk_level"`
}

// TLUClawPreviewCardResponse encapsulates preview card details for client cards.
type TLUClawPreviewCardResponse struct {
	SnapshotID  string                     `json:"snapshot_id"`
	BatchID     string                     `json:"batch_id"`
	ActionType  string                     `json:"action_type"`
	Summary     string                     `json:"summary"`
	Nonce       string                     `json:"nonce"`
	ExpiresAt   time.Time                  `json:"expires_at"`
	DiffItems   []TLUClawDiffItemResponse   `json:"diff_items"`
	BlastRadius TLUClawBlastRadiusResponse `json:"blast_radius"`
}

// TLUClawChatResponse encapsulates the synthesized assistant output and any staged preview card.
type TLUClawChatResponse struct {
	SessionID        string                      `json:"session_id"`
	MessageID        string                      `json:"message_id"`
	Reply            string                      `json:"reply"`
	ReasoningContent string                      `json:"reasoning_content,omitempty"`
	PreviewCard      *TLUClawPreviewCardResponse `json:"preview_card,omitempty"`
	CreatedAt        time.Time                   `json:"created_at"`
	DidCompact       bool                        `json:"did_compact,omitempty"`
}

// TLUClawActionResponse encapsulates operation status after confirmation or cancellation.
type TLUClawActionResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	SnapshotID string `json:"snapshot_id"`
	BatchID    string `json:"batch_id,omitempty"`
	Status     string `json:"status"`
}

// TLUClawSessionResponse represents an administrative session summary for client lists.
type TLUClawSessionResponse struct {
	ID          string    `json:"id"`
	AdminUserID string    `json:"admin_user_id"`
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TLUClawMessageResponse represents a conversation message for transcript rendering.
type TLUClawMessageResponse struct {
	ID               string    `json:"id"`
	SessionID        string    `json:"session_id"`
	Role             string    `json:"role"`
	Content          string    `json:"content"`
	ReasoningContent string    `json:"reasoning_content,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	DidCompact       bool      `json:"did_compact,omitempty"`
}

// TLUClawSessionDetailResponse encapsulates a session and its ordered message transcript.
type TLUClawSessionDetailResponse struct {
	Session  *TLUClawSessionResponse       `json:"session"`
	Messages []*TLUClawMessageResponse     `json:"messages"`
	Cards    []*TLUClawPreviewCardResponse `json:"snapshots,omitempty"`
}

// TLUClawSnapshotItemResponse represents an audit snapshot record for timeline rendering.
type TLUClawSnapshotItemResponse struct {
	ID          string                     `json:"id"`
	BatchID     string                     `json:"batch_id"`
	SessionID   string                     `json:"session_id"`
	OperatorID  string                     `json:"operator_id"`
	ActionType  string                     `json:"action_type"`
	Summary     string                     `json:"summary"`
	DiffItems   []TLUClawDiffItemResponse   `json:"diff_items"`
	BlastRadius TLUClawBlastRadiusResponse `json:"blast_radius"`
	Status      string                     `json:"status"`
	AppliedAt   *time.Time                 `json:"applied_at,omitempty"`
	CreatedAt   time.Time                  `json:"created_at"`
}

// TLUClawSnapshotListResponse encapsulates paginated snapshots.
type TLUClawSnapshotListResponse struct {
	Snapshots []*TLUClawSnapshotItemResponse `json:"snapshots"`
	Total     int64                          `json:"total"`
}

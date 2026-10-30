package request

// TLUClawAttachmentDTO represents a file or document payload attached to an admin prompt.
type TLUClawAttachmentDTO struct {
	FileName string `json:"file_name" validate:"required,max=255"`
	FileType string `json:"file_type" validate:"required,max=64"`
	Data     string `json:"data" validate:"required"`
}

// TLUClawChatRequest encapsulates prompt, attachments, and session details for admin interaction.
type TLUClawChatRequest struct {
	SessionID   string                 `json:"session_id,omitempty" validate:"omitempty,max=128"`
	Message     string                 `json:"message" validate:"required,min=1,max=10000"`
	Images      []string               `json:"images,omitempty" validate:"omitempty,max=10"`
	Attachments []TLUClawAttachmentDTO `json:"attachments,omitempty" validate:"omitempty,max=10"`

	ThinkingEnabled *bool  `json:"thinking_enabled,omitempty"`
	EffortLevel     string `json:"effort_level,omitempty" validate:"omitempty,oneof=low medium high"`
}

// TLUClawConfirmRequest encapsulates validation parameters to commit a staged snapshot.
type TLUClawConfirmRequest struct {
	SnapshotID   string   `json:"snapshot_id" validate:"required,max=128"`
	Nonce        string   `json:"nonce" validate:"required,max=128"`
	SelectedKeys []string `json:"selected_keys,omitempty" validate:"omitempty"`
}

// TLUClawCancelRequest encapsulates parameters to discard a staged snapshot.
type TLUClawCancelRequest struct {
	SnapshotID string `json:"snapshot_id" validate:"required,max=128"`
	Nonce      string `json:"nonce" validate:"required,max=128"`
}

// TLUClawCreateSessionRequest defines the input payload for creating a new TLUClaw admin session.
type TLUClawCreateSessionRequest struct {
	Title string `json:"title" validate:"required,min=1,max=200"`
}

// TLUClawRollbackRequest encapsulates input parameters to reverse a committed batch.
type TLUClawRollbackRequest struct {
	BatchID string `json:"batch_id" validate:"required,max=128"`
}

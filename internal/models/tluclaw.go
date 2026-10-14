package models

import (
	"encoding/json"
	"strings"
	"time"

	"tluagent-web/internal/dtos/response"
)

// TLUClawSession represents an interactive operational session for administrators.
type TLUClawSession struct {
	ID          string    `json:"id"`
	AdminUserID string    `json:"admin_user_id"`
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse converts a TLUClawSession domain model to response DTO.
func (s *TLUClawSession) ToResponse() *response.TLUClawSessionResponse {
	if s == nil {
		return nil
	}
	return &response.TLUClawSessionResponse{
		ID:          s.ID,
		AdminUserID: s.AdminUserID,
		Title:       s.Title,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

// TLUClawMessage represents a single dialogue turn within a TLUClaw session.
type TLUClawMessage struct {
	ID               string    `json:"id"`
	SessionID        string    `json:"session_id"`
	Role             string    `json:"role"`
	Content          string    `json:"content"`
	ReasoningContent string    `json:"reasoning_content,omitempty"`
	ToolCallsJSON    string    `json:"tool_calls_json,omitempty"`
	ToolCallID       string    `json:"tool_call_id,omitempty"`
	MetadataJSON     string    `json:"metadata_json,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// ToResponse converts a TLUClawMessage domain model to response DTO.
func (m *TLUClawMessage) ToResponse() *response.TLUClawMessageResponse {
	if m == nil {
		return nil
	}
	var didCompact bool
	if m.MetadataJSON != "" && strings.Contains(m.MetadataJSON, `"did_compact":true`) {
		didCompact = true
	}
	return &response.TLUClawMessageResponse{
		ID:               m.ID,
		SessionID:        m.SessionID,
		Role:             m.Role,
		Content:          m.Content,
		ReasoningContent: m.ReasoningContent,
		CreatedAt:        m.CreatedAt,
		DidCompact:       didCompact,
	}
}

// TLUClawActionSnapshot represents a staged or committed state change with audit details.
type TLUClawActionSnapshot struct {
	ID                string     `json:"id"`
	BatchID           string     `json:"batch_id"`
	SessionID         string     `json:"session_id"`
	OperatorID        string     `json:"operator_id"`
	ActionType        string     `json:"action_type"`
	Summary           string     `json:"summary"`
	DiffPreviewJSON   string     `json:"diff_preview_json"`
	BlastRadiusJSON   string     `json:"blast_radius_json"`
	PreviousStateJSON string     `json:"previous_state_json"`
	NewStateJSON      string     `json:"new_state_json"`
	Status            string     `json:"status"`
	ConfirmationNonce string     `json:"confirmation_nonce"`
	NonceExpiresAt    *time.Time `json:"nonce_expires_at,omitempty"`
	AppliedAt         *time.Time `json:"applied_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// ToPreviewResponse converts an action snapshot to a preview card response.
func (s *TLUClawActionSnapshot) ToPreviewResponse() *response.TLUClawPreviewCardResponse {
	if s == nil {
		return nil
	}
	var diffs []response.TLUClawDiffItemResponse
	_ = json.Unmarshal([]byte(s.DiffPreviewJSON), &diffs)

	var blast response.TLUClawBlastRadiusResponse
	_ = json.Unmarshal([]byte(s.BlastRadiusJSON), &blast)

	var exp time.Time
	if s.NonceExpiresAt != nil {
		exp = *s.NonceExpiresAt
	}

	return &response.TLUClawPreviewCardResponse{
		SnapshotID:  s.ID,
		BatchID:     s.BatchID,
		ActionType:  s.ActionType,
		Summary:     s.Summary,
		Nonce:       s.ConfirmationNonce,
		ExpiresAt:   exp,
		DiffItems:   diffs,
		BlastRadius: blast,
	}
}

// TLUClawDiffItem represents an atomic property or record difference for preview display.
type TLUClawDiffItem struct {
	EntityType string `json:"entity_type"`
	Action     string `json:"action"`
	Key        string `json:"key"`
	Name       string `json:"name,omitempty"`
	OldValue   string `json:"old_value,omitempty"`
	NewValue   string `json:"new_value,omitempty"`
	Note       string `json:"note,omitempty"`
}

// ToResponse converts a diff item to response DTO.
func (d *TLUClawDiffItem) ToResponse() response.TLUClawDiffItemResponse {
	return response.TLUClawDiffItemResponse{
		EntityType: d.EntityType,
		Action:     d.Action,
		Key:        d.Key,
		Name:       d.Name,
		OldValue:   d.OldValue,
		NewValue:   d.NewValue,
		Note:       d.Note,
	}
}

// TLUClawBlastRadius encapsulates affected entities and supersession impact analysis.
type TLUClawBlastRadius struct {
	AffectedCohorts    []string `json:"affected_cohorts,omitempty"`
	AffectedMajors     []string `json:"affected_majors,omitempty"`
	SupersededDocIDs   []string `json:"superseded_doc_ids,omitempty"`
	SupersededDocNames []string `json:"superseded_doc_names,omitempty"`
	ImpactSummary      string   `json:"impact_summary"`
	RiskLevel          string   `json:"risk_level"`
}

// ToResponse converts blast radius to response DTO.
func (b *TLUClawBlastRadius) ToResponse() response.TLUClawBlastRadiusResponse {
	return response.TLUClawBlastRadiusResponse{
		AffectedCohorts:    b.AffectedCohorts,
		AffectedMajors:     b.AffectedMajors,
		SupersededDocIDs:   b.SupersededDocIDs,
		SupersededDocNames: b.SupersededDocNames,
		ImpactSummary:      b.ImpactSummary,
		RiskLevel:          b.RiskLevel,
	}
}

// TLUClawPreviewCardData encapsulates the complete interactive payload sent to admin chat UI.
type TLUClawPreviewCardData struct {
	SnapshotID  string             `json:"snapshot_id"`
	BatchID     string             `json:"batch_id"`
	ActionType  string             `json:"action_type"`
	Summary     string             `json:"summary"`
	Nonce       string             `json:"nonce"`
	ExpiresAt   time.Time          `json:"expires_at"`
	DiffItems   []TLUClawDiffItem  `json:"diff_items"`
	BlastRadius TLUClawBlastRadius `json:"blast_radius"`
}

// ToResponse converts preview card data to response DTO.
func (c *TLUClawPreviewCardData) ToResponse() *response.TLUClawPreviewCardResponse {
	if c == nil {
		return nil
	}
	diffs := make([]response.TLUClawDiffItemResponse, len(c.DiffItems))
	for i, d := range c.DiffItems {
		diffs[i] = d.ToResponse()
	}
	return &response.TLUClawPreviewCardResponse{
		SnapshotID:  c.SnapshotID,
		BatchID:     c.BatchID,
		ActionType:  c.ActionType,
		Summary:     c.Summary,
		Nonce:       c.Nonce,
		ExpiresAt:   c.ExpiresAt,
		DiffItems:   diffs,
		BlastRadius: c.BlastRadius.ToResponse(),
	}
}

// ToSnapshotItemResponse converts an action snapshot to item response DTO.
func (s *TLUClawActionSnapshot) ToSnapshotItemResponse() *response.TLUClawSnapshotItemResponse {
	if s == nil {
		return nil
	}
	var diffs []TLUClawDiffItem
	_ = json.Unmarshal([]byte(s.DiffPreviewJSON), &diffs)

	diffResps := make([]response.TLUClawDiffItemResponse, len(diffs))
	for i, d := range diffs {
		diffResps[i] = d.ToResponse()
	}

	var blast TLUClawBlastRadius
	_ = json.Unmarshal([]byte(s.BlastRadiusJSON), &blast)

	return &response.TLUClawSnapshotItemResponse{
		ID:          s.ID,
		BatchID:     s.BatchID,
		SessionID:   s.SessionID,
		OperatorID:  s.OperatorID,
		ActionType:  s.ActionType,
		Summary:     s.Summary,
		DiffItems:   diffResps,
		BlastRadius: blast.ToResponse(),
		Status:      s.Status,
		AppliedAt:   s.AppliedAt,
		CreatedAt:   s.CreatedAt,
	}
}

// TLUClawCurriculumPriorSnapshotState encapsulates prior curriculum state for rollback.
type TLUClawCurriculumPriorSnapshotState struct {
	Cohort          string              `json:"cohort"`
	MajorCode       string              `json:"major_code"`
	ExistingCourses []*CurriculumCourse `json:"existing_courses"`
}

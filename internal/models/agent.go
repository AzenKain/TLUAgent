package models

import (
	"time"

	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/pkg/convert"
)

// AgentPrompt represents a customizable core agent instruction prompt.
type AgentPrompt struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by,omitempty"`
}

// FromSqlc converts a sqlc AgentPrompt row to an AgentPrompt domain model.
func (p *AgentPrompt) FromSqlc(row sqlc.AgentPrompt) *AgentPrompt {
	p.ID = row.ID
	p.Title = row.Title
	p.Content = row.Content
	p.IsActive = row.IsActive == 1
	p.CreatedAt = row.CreatedAt
	p.UpdatedAt = row.UpdatedAt
	p.UpdatedBy = convert.NullStringToString(row.UpdatedBy)
	return p
}

// ToResponse converts an AgentPrompt domain model to response DTO.
func (p *AgentPrompt) ToResponse() *response.AgentPromptResponse {
	if p == nil {
		return nil
	}
	return &response.AgentPromptResponse{
		ID:        p.ID,
		Title:     p.Title,
		Content:   p.Content,
		IsActive:  p.IsActive,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		UpdatedBy: p.UpdatedBy,
	}
}

// AgentSkill represents an individual modular capability in the agent skills system.
type AgentSkill struct {
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

// FromSqlc converts a sqlc AgentSkill row to an AgentSkill domain model.
func (s *AgentSkill) FromSqlc(row sqlc.AgentSkill) *AgentSkill {
	s.ID = row.ID
	s.Name = row.Name
	s.Description = row.Description
	s.Content = row.Content
	s.ToolDefinition = convert.NullStringToString(row.ToolDefinition)
	s.IsEnabled = row.IsEnabled == 1
	s.Priority = int(row.Priority)
	s.CreatedAt = row.CreatedAt
	s.UpdatedAt = row.UpdatedAt
	s.UpdatedBy = convert.NullStringToString(row.UpdatedBy)
	return s
}

// ToResponse converts an AgentSkill domain model to response DTO.
func (s *AgentSkill) ToResponse() *response.AgentSkillResponse {
	if s == nil {
		return nil
	}
	return &response.AgentSkillResponse{
		ID:             s.ID,
		Name:           s.Name,
		Description:    s.Description,
		Content:        s.Content,
		ToolDefinition: s.ToolDefinition,
		IsEnabled:      s.IsEnabled,
		Priority:       s.Priority,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
		UpdatedBy:      s.UpdatedBy,
	}
}

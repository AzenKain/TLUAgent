package models

import (
	"tluagent-web/internal/dtos/response"
)

type AdvisoryRuleEntity struct {
	ID       string   `json:"id"`
	Category string   `json:"category"`
	Keywords []string `json:"keywords"`
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Sources  []string `json:"sources"`
}

func (e *AdvisoryRuleEntity) ToResponse(query, timestamp string) *response.ChatResponse {
	if e == nil {
		return nil
	}
	return &response.ChatResponse{
		Query:     query,
		Reply:     e.Content,
		Sources:   e.Sources,
		Timestamp: timestamp,
	}
}

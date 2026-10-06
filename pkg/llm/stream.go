package llm

import (
	"errors"
	"io"
	"sort"
	"strings"
)

type StreamAssembler struct {
	ID               string
	Model            string
	Role             Role
	Content          strings.Builder
	ReasoningContent strings.Builder
	FinishReason     string
	Usage            TokenUsage
	toolCallsMap     map[int]*toolCallAccumulator
}

type toolCallAccumulator struct {
	id        string
	callType  string
	name      strings.Builder
	arguments strings.Builder
}

func NewStreamAssembler() *StreamAssembler {
	return &StreamAssembler{
		Role:         RoleAssistant,
		toolCallsMap: make(map[int]*toolCallAccumulator),
	}
}

func (a *StreamAssembler) AppendChunk(chunk *StreamChunk) {
	if chunk == nil {
		return
	}
	if a.ID == "" && chunk.ID != "" {
		a.ID = chunk.ID
	}
	if a.Model == "" && chunk.Model != "" {
		a.Model = chunk.Model
	}
	if chunk.Delta.Role != "" {
		a.Role = chunk.Delta.Role
	}
	if chunk.Delta.Content != "" {
		a.Content.WriteString(chunk.Delta.Content)
	}
	if chunk.Delta.ReasoningContent != "" {
		a.ReasoningContent.WriteString(chunk.Delta.ReasoningContent)
	}
	if chunk.FinishReason != "" {
		a.FinishReason = chunk.FinishReason
	}
	if chunk.Usage != nil {
		a.Usage = *chunk.Usage
	}

	for _, tc := range chunk.Delta.ToolCalls {
		acc, exists := a.toolCallsMap[tc.Index]
		if !exists {
			acc = &toolCallAccumulator{
				id:       tc.ID,
				callType: tc.Type,
			}
			a.toolCallsMap[tc.Index] = acc
		}
		if tc.ID != "" && acc.id == "" {
			acc.id = tc.ID
		}
		if tc.Type != "" && acc.callType == "" {
			acc.callType = tc.Type
		}
		if tc.Function.Name != "" {
			acc.name.WriteString(tc.Function.Name)
		}
		if tc.Function.Arguments != "" {
			acc.arguments.WriteString(tc.Function.Arguments)
		}
	}
}

func (a *StreamAssembler) ToResponse() *ChatResponse {
	var indices []int
	for idx := range a.toolCallsMap {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	var toolCalls []ToolCall
	for _, idx := range indices {
		acc := a.toolCallsMap[idx]
		callType := acc.callType
		if callType == "" {
			callType = "function"
		}
		toolCalls = append(toolCalls, ToolCall{
			ID:   acc.id,
			Type: callType,
			Function: FunctionCall{
				Name:      acc.name.String(),
				Arguments: acc.arguments.String(),
			},
		})
	}

	return &ChatResponse{
		ID:    a.ID,
		Model: a.Model,
		Message: Message{
			Role:             a.Role,
			Content:          a.Content.String(),
			ReasoningContent: a.ReasoningContent.String(),
			ToolCalls:        toolCalls,
		},
		FinishReason: a.FinishReason,
		Usage:        a.Usage,
	}
}

// ConsumeStream reads all chunks from a stream and builds an aggregated ChatResponse.
func ConsumeStream(reader StreamReader, onChunk StreamHandler) (*ChatResponse, error) {
	if reader == nil {
		return nil, errors.New("stream reader is nil")
	}
	defer reader.Close()

	assembler := NewStreamAssembler()

	for {
		chunk, err := reader.Recv()
		if errors.Is(err, io.EOF) || errors.Is(err, ErrStreamClosed) {
			break
		}
		if err != nil {
			return nil, err
		}

		assembler.AppendChunk(chunk)

		if onChunk != nil {
			if err := onChunk(chunk); err != nil {
				return nil, err
			}
		}
	}

	return assembler.ToResponse(), nil
}

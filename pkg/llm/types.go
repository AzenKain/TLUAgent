package llm

import (
	"context"
	"io"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type ContentPartType string

const (
	ContentPartText  ContentPartType = "text"
	ContentPartImage ContentPartType = "image"
)

type ImageSource struct {
	URL      string `json:"url,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

type ContentPart struct {
	Type  ContentPartType `json:"type"`
	Text  string          `json:"text,omitempty"`
	Image *ImageSource    `json:"image,omitempty"`
}

type Message struct {
	Role             Role          `json:"role"`
	Content          string        `json:"content,omitempty"`
	Parts            []ContentPart `json:"parts,omitempty"`
	Name             string        `json:"name,omitempty"`
	ToolCalls        []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID       string        `json:"tool_call_id,omitempty"`
	ReasoningContent string        `json:"reasoning_content,omitempty"`
}

type FunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type ToolDefinition struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCallChunk struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type ToolCallChunk struct {
	Index    int               `json:"index"`
	ID       string            `json:"id,omitempty"`
	Type     string            `json:"type,omitempty"`
	Function FunctionCallChunk `json:"function,omitempty"`
}

type MessageDelta struct {
	Role             Role            `json:"role,omitempty"`
	Content          string          `json:"content,omitempty"`
	ReasoningContent string          `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCallChunk `json:"tool_calls,omitempty"`
}

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	ReasoningTokens  int `json:"reasoning_tokens,omitempty"`
}

type ChatRequest struct {
	Model            string           `json:"model,omitempty"`
	Messages         []Message        `json:"messages"`
	Tools            []ToolDefinition `json:"tools,omitempty"`
	ToolChoice       any              `json:"tool_choice,omitempty"`
	Temperature      *float64         `json:"temperature,omitempty"`
	TopP             *float64         `json:"top_p,omitempty"`
	MaxTokens        *int             `json:"max_tokens,omitempty"`
	Stop             []string         `json:"stop,omitempty"`
	Stream           bool             `json:"stream,omitempty"`
	JSONMode         bool             `json:"json_mode,omitempty"`
	SystemPrompt     string           `json:"system_prompt,omitempty"`
	ThinkingDisabled bool             `json:"thinking_disabled,omitempty"`
	ThinkingEnabled  *bool            `json:"thinking_enabled,omitempty"`
	ThinkingBudget   *int             `json:"thinking_budget,omitempty"`
	ReasoningEffort  string           `json:"reasoning_effort,omitempty"`
}

type ChatResponse struct {
	ID           string     `json:"id,omitempty"`
	Model        string     `json:"model"`
	Message      Message    `json:"message"`
	FinishReason string     `json:"finish_reason,omitempty"`
	Usage        TokenUsage `json:"usage"`
}

type StreamChunk struct {
	ID           string       `json:"id,omitempty"`
	Model        string       `json:"model,omitempty"`
	Delta        MessageDelta `json:"delta"`
	FinishReason string       `json:"finish_reason,omitempty"`
	Usage        *TokenUsage  `json:"usage,omitempty"`
}

type StreamReader interface {
	Recv() (*StreamChunk, error)
	Close() error
}

type StreamHandler func(chunk *StreamChunk) error

type Provider interface {
	Name() string
	DefaultModel() string
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req *ChatRequest) (StreamReader, error)
}

func SystemMessage(content string) Message {
	return Message{Role: RoleSystem, Content: content}
}

func UserMessage(content string) Message {
	return Message{Role: RoleUser, Content: content}
}

func AssistantMessage(content string) Message {
	return Message{Role: RoleAssistant, Content: content}
}

func ToolMessage(toolCallID, name, content string) Message {
	return Message{
		Role:       RoleTool,
		ToolCallID: toolCallID,
		Name:       name,
		Content:    content,
	}
}

func UserMessageWithImageURL(text, imageURL string) Message {
	return Message{
		Role:    RoleUser,
		Content: text,
		Parts: []ContentPart{
			{Type: ContentPartText, Text: text},
			{Type: ContentPartImage, Image: &ImageSource{URL: imageURL}},
		},
	}
}

func UserMessageWithImageBase64(text, mimeType, base64Data string) Message {
	return Message{
		Role:    RoleUser,
		Content: text,
		Parts: []ContentPart{
			{Type: ContentPartText, Text: text},
			{Type: ContentPartImage, Image: &ImageSource{MimeType: mimeType, Data: base64Data}},
		},
	}
}

var ErrStreamClosed = io.EOF

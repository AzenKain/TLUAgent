package compactor

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

type Message struct {
	Role    MessageRole `json:"role"`
	Content string      `json:"content"`
}

type SummarizerFunc func(ctx context.Context, messagesToCompact []Message) (string, error)

type Config struct {
	MaxContextTokens      int
	CompactThresholdRatio float64
	KeepRecentTurns       int
	Summarizer            SummarizerFunc
}

// DefaultConfig returns recommended compactor parameters with 250k token context window support.
func DefaultConfig() Config {
	return Config{
		MaxContextTokens:      250000,
		CompactThresholdRatio: 0.80,
		KeepRecentTurns:       10,
		Summarizer:            DefaultSummarizer,
	}
}

// DynamicConfig builds a compaction configuration tailored for specified context token capacity.
func DynamicConfig(maxTokens int, keepRecentTurns int) Config {
	if maxTokens <= 0 {
		maxTokens = 250000
	}
	if keepRecentTurns <= 0 {
		keepRecentTurns = 10
	}
	return Config{
		MaxContextTokens:      maxTokens,
		CompactThresholdRatio: 0.80,
		KeepRecentTurns:       keepRecentTurns,
		Summarizer:            DefaultSummarizer,
	}
}

// CreateLLMSummarizer wraps an abstractive LLM callback with fallback to extractive summarizer.
func CreateLLMSummarizer(llmFn func(ctx context.Context, prompt string) (string, error)) SummarizerFunc {
	return func(ctx context.Context, messages []Message) (string, error) {
		if len(messages) == 0 {
			return "", nil
		}
		if llmFn == nil {
			return DefaultSummarizer(ctx, messages)
		}
		var prompt strings.Builder
		prompt.WriteString("You are an executive context summarizer for an academic advisory AI.\n")
		prompt.WriteString("Summarize the following prior conversation turns compactly, preserving key student facts, academic terms, and course codes:\n\n")
		for _, m := range messages {
			if m.Role == RoleSystem {
				continue
			}
			prompt.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
		}
		prompt.WriteString("\nConcise Summary:")
		summary, err := llmFn(ctx, prompt.String())
		if err != nil || strings.TrimSpace(summary) == "" {
			return DefaultSummarizer(ctx, messages)
		}
		return "[CONVERSATION HISTORY EXECUTIVE SUMMARY]\n" + strings.TrimSpace(summary), nil
	}
}

// EstimateTokens calculates approximate token counts for multilingual text including Vietnamese.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	var nonAsciiRunes int
	var asciiWords int
	inWord := false

	for _, r := range text {
		if r > 127 {
			nonAsciiRunes++
			inWord = false
		} else if unicode.IsSpace(r) {
			inWord = false
		} else {
			if !inWord {
				asciiWords++
				inWord = true
			}
		}
	}

	asciiTokens := int(float64(asciiWords) * 1.3)
	unicodeTokens := int(float64(nonAsciiRunes) * 0.8)
	total := asciiTokens + unicodeTokens
	if total < 1 && len(text) > 0 {
		return 1
	}
	return total
}

// CalculateHistoryTokens returns total estimated tokens for message list.
func CalculateHistoryTokens(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += 4 + EstimateTokens(m.Content)
	}
	return total
}

// DefaultSummarizer creates an extractive bulleted summary of conversational turns.
func DefaultSummarizer(_ context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}
	var sb strings.Builder
	sb.WriteString("[CONVERSATION HISTORY SUMMARY]\n")
	for _, m := range messages {
		if m.Role == RoleSystem {
			continue
		}
		trimmed := strings.TrimSpace(m.Content)
		if len(trimmed) > 160 {
			trimmed = trimmed[:160] + "..."
		}
		sb.WriteString(fmt.Sprintf("- %s: %s\n", m.Role, trimmed))
	}
	return sb.String(), nil
}

// ShouldCompact checks whether message tokens exceed threshold ratio.
func ShouldCompact(messages []Message, cfg Config) bool {
	if cfg.MaxContextTokens <= 0 {
		return false
	}
	ratio := cfg.CompactThresholdRatio
	if ratio <= 0 || ratio > 1.0 {
		ratio = 0.75
	}
	threshold := int(float64(cfg.MaxContextTokens) * ratio)
	return CalculateHistoryTokens(messages) > threshold
}

// CompactHistory shrinks older dialogue turns into an executive summary.
func CompactHistory(ctx context.Context, messages []Message, cfg Config) ([]Message, bool, error) {
	if len(messages) <= cfg.KeepRecentTurns+1 || !ShouldCompact(messages, cfg) {
		return messages, false, nil
	}

	systemMessages := make([]Message, 0, 2)
	dialogueMessages := make([]Message, 0, len(messages))

	for _, m := range messages {
		if m.Role == RoleSystem {
			systemMessages = append(systemMessages, m)
		} else {
			dialogueMessages = append(dialogueMessages, m)
		}
	}

	keep := cfg.KeepRecentTurns
	if keep >= len(dialogueMessages) {
		return messages, false, nil
	}

	splitIdx := len(dialogueMessages) - keep
	toCompact := dialogueMessages[:splitIdx]
	recent := dialogueMessages[splitIdx:]

	summarizer := cfg.Summarizer
	if summarizer == nil {
		summarizer = DefaultSummarizer
	}

	summaryText, err := summarizer(ctx, toCompact)
	if err != nil {
		return nil, false, fmt.Errorf("failed to summarize context: %w", err)
	}

	result := make([]Message, 0, len(systemMessages)+1+len(recent))
	result = append(result, systemMessages...)
	result = append(result, Message{
		Role:    RoleSystem,
		Content: summaryText,
	})
	result = append(result, recent...)

	return result, true, nil
}

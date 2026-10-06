package compactor

import (
	"context"
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	if EstimateTokens("") != 0 {
		t.Errorf("expected 0 tokens for empty string")
	}

	enText := "Hello world, how are you doing today?"
	enTokens := EstimateTokens(enText)
	if enTokens <= 0 || enTokens > 30 {
		t.Errorf("unexpected token count for English: %d", enTokens)
	}

	viText := "Training regulations for the credit-based system of Thang Long University"
	viTokens := EstimateTokens(viText)
	if viTokens <= 0 || viTokens > 40 {
		t.Errorf("unexpected token count for long text: %d", viTokens)
	}
}

func TestCalculateHistoryTokens(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "You are an academic advisor."},
		{Role: RoleUser, Content: "How many credits are required?"},
		{Role: RoleAssistant, Content: "You need 130 credits to graduate."},
	}

	total := CalculateHistoryTokens(msgs)
	if total < 20 {
		t.Errorf("expected at least 20 tokens, got %d", total)
	}
}

func TestCompactHistoryNotTriggered(t *testing.T) {
	cfg := Config{
		MaxContextTokens:      1000,
		CompactThresholdRatio: 0.75,
		KeepRecentTurns:       2,
	}

	msgs := []Message{
		{Role: RoleSystem, Content: "You are an academic advisor."},
		{Role: RoleUser, Content: "Hi"},
		{Role: RoleAssistant, Content: "Hello"},
	}

	compacted, didCompact, err := CompactHistory(context.Background(), msgs, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if didCompact {
		t.Errorf("expected no compaction for short history")
	}
	if len(compacted) != len(msgs) {
		t.Errorf("expected length %d, got %d", len(msgs), len(compacted))
	}
}

func TestCompactHistoryTriggered(t *testing.T) {
	cfg := Config{
		MaxContextTokens:      50,
		CompactThresholdRatio: 0.5,
		KeepRecentTurns:       2,
	}

	msgs := []Message{
		{Role: RoleSystem, Content: "System prompt"},
		{Role: RoleUser, Content: "Long question 1 about registration rules and prerequisites"},
		{Role: RoleAssistant, Content: "Detailed answer 1 explaining prerequisite policies in full"},
		{Role: RoleUser, Content: "Long question 2 about graduation GPA requirements"},
		{Role: RoleAssistant, Content: "Detailed answer 2 explaining minimum GPA criteria"},
		{Role: RoleUser, Content: "Recent question"},
		{Role: RoleAssistant, Content: "Recent answer"},
	}

	compacted, didCompact, err := CompactHistory(context.Background(), msgs, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !didCompact {
		t.Fatalf("expected compaction to be triggered")
	}

	if len(compacted) != 4 {
		t.Fatalf("expected 4 messages (1 sys + 1 summary + 2 recent), got %d", len(compacted))
	}

	if compacted[0].Role != RoleSystem || compacted[0].Content != "System prompt" {
		t.Errorf("system prompt was altered")
	}

	if compacted[1].Role != RoleSystem || !strings.Contains(compacted[1].Content, "[CONVERSATION HISTORY SUMMARY]") {
		t.Errorf("summary message not generated correctly")
	}

	if compacted[2].Content != "Recent question" || compacted[3].Content != "Recent answer" {
		t.Errorf("recent turns were not preserved")
	}
}

func TestCustomSummarizer(t *testing.T) {
	customCalled := false
	cfg := Config{
		MaxContextTokens:      30,
		CompactThresholdRatio: 0.5,
		KeepRecentTurns:       1,
		Summarizer: func(ctx context.Context, messagesToCompact []Message) (string, error) {
			customCalled = true
			return "CUSTOM_SUMMARY", nil
		},
	}

	msgs := []Message{
		{Role: RoleSystem, Content: "Sys"},
		{Role: RoleUser, Content: "Turn 1"},
		{Role: RoleAssistant, Content: "Turn 2"},
		{Role: RoleUser, Content: "Turn 3"},
	}

	compacted, didCompact, err := CompactHistory(context.Background(), msgs, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !didCompact || !customCalled {
		t.Errorf("custom summarizer was not invoked")
	}
	if compacted[1].Content != "CUSTOM_SUMMARY" {
		t.Errorf("expected CUSTOM_SUMMARY, got %s", compacted[1].Content)
	}
}

package llm

import (
	"context"
	"testing"
	"time"
)

type recordingProvider struct {
	name string
	seen []*ChatRequest
}

func (p *recordingProvider) Name() string         { return p.name }
func (p *recordingProvider) DefaultModel() string { return p.name + "-default" }
func (p *recordingProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	copied := *req
	p.seen = append(p.seen, &copied)
	return &ChatResponse{Message: Message{Role: RoleAssistant, Content: "ok from " + p.name}}, nil
}
func (p *recordingProvider) ChatStream(ctx context.Context, req *ChatRequest) (StreamReader, error) {
	return nil, nil
}

func TestChatWithChain_UsesAuditNodeModelAndReasoning(t *testing.T) {
	chatProv := &recordingProvider{name: "chat"}
	auditProv := &recordingProvider{name: "audit"}
	m := NewManager()
	m.Register("chat", chatProv)

	enabled, budget := true, 8192
	chain := NewLLMChain("audit-chain", ChainTypeAudit, "audit", 3, time.Minute)
	if err := chain.SetNodes([]ChainNode{{
		ID: "n1", Name: "step", ProviderID: "audit", ProviderName: "audit", ModelKey: "step/step-5-preview",
		IsActive: true, Provider: auditProv, Thinking: &enabled, ThinkBudget: &budget, Effort: "medium",
	}}); err != nil {
		t.Fatalf("set nodes: %v", err)
	}
	m.RegisterChain(chain)

	resp, err := m.ChatWithChain(context.Background(), ChainTypeAudit, &ChatRequest{Messages: []Message{UserMessage("hi")}})
	if err != nil || resp.Message.Content != "ok from audit" {
		t.Fatalf("expected the audit node to answer, got %+v err=%v", resp, err)
	}
	got := auditProv.seen[0]
	if got.Model != "step/step-5-preview" || got.ThinkingEnabled == nil || !*got.ThinkingEnabled || got.ThinkingBudget == nil || *got.ThinkingBudget != 8192 || got.ReasoningEffort != "medium" {
		t.Fatalf("audit node model and reasoning settings must reach the provider, got %+v", got)
	}
	if len(chatProv.seen) != 0 {
		t.Fatalf("the chat provider must stay untouched")
	}
}

func TestChatWithChain_KeepsExplicitReasoningAndFallsBackWithoutChain(t *testing.T) {
	chatProv := &recordingProvider{name: "chat"}
	auditProv := &recordingProvider{name: "audit"}
	m := NewManager()
	m.Register("chat", chatProv)

	if _, err := m.ChatWithChain(context.Background(), ChainTypeAudit, &ChatRequest{Messages: []Message{UserMessage("hi")}}); err != nil {
		t.Fatalf("without an audit chain the request must fall back to chat: %v", err)
	}
	if len(chatProv.seen) != 1 {
		t.Fatalf("expected the chat provider to serve the fallback")
	}

	enabled, budget := true, 4000
	chain := NewLLMChain("audit-chain", ChainTypeAudit, "audit", 3, time.Minute)
	_ = chain.SetNodes([]ChainNode{{ID: "n1", ModelKey: "m", IsActive: true, Provider: auditProv, Thinking: &enabled, ThinkBudget: &budget, Effort: "high"}})
	m.RegisterChain(chain)
	off := false
	if _, err := m.ChatWithChain(context.Background(), ChainTypeAudit, &ChatRequest{Messages: []Message{UserMessage("hi")}, ThinkingEnabled: &off}); err != nil {
		t.Fatalf("chat with explicit reasoning: %v", err)
	}
	if got := auditProv.seen[0]; got.ThinkingEnabled == nil || *got.ThinkingEnabled || got.ReasoningEffort != "" {
		t.Fatalf("explicit request settings must win over node defaults, got %+v", got)
	}
}

type toolOnlyProvider struct {
	recordingProvider
}

func (p *toolOnlyProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	copied := *req
	p.seen = append(p.seen, &copied)
	return &ChatResponse{Model: "audit-model", Message: Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "propose", Arguments: "{}"}}}}}, nil
}

func TestChatWithChain_ToolCallOnlyReplyCountsAsSuccess(t *testing.T) {
	chatProv := &recordingProvider{name: "chat"}
	auditProv := &toolOnlyProvider{recordingProvider{name: "audit"}}
	m := NewManager()
	m.Register("chat", chatProv)
	chain := NewLLMChain("audit-chain", ChainTypeAudit, "audit", 3, time.Minute)
	_ = chain.SetNodes([]ChainNode{{ID: "n1", ModelKey: "audit-model", IsActive: true, Provider: auditProv}})
	m.RegisterChain(chain)

	for i := 0; i < 5; i++ {
		resp, err := m.ChatWithChain(context.Background(), ChainTypeAudit, &ChatRequest{Messages: []Message{UserMessage("decide")}})
		if err != nil || len(resp.Message.ToolCalls) != 1 || resp.Model != "audit-model" {
			t.Fatalf("a tool-call reply from the audit node must be returned as is, got %+v err=%v", resp, err)
		}
	}
	if len(chatProv.seen) != 0 {
		t.Fatalf("the chat model must never serve audit requests while the audit node answers, got %d calls", len(chatProv.seen))
	}
	if len(auditProv.seen) != 5 {
		t.Fatalf("each request must hit the audit node exactly once, got %d", len(auditProv.seen))
	}
}

func TestManagerChat_ToolCallReplyOnChatChainIsServedOnce(t *testing.T) {
	chatProv := &toolOnlyProvider{recordingProvider{name: "chat"}}
	m := NewManager()
	m.Register("chat", chatProv)
	chain := NewLLMChain("chat-chain", ChainTypeChat, "chat", 3, time.Minute)
	_ = chain.SetNodes([]ChainNode{{ID: "n1", ModelKey: "chat-model", IsActive: true, Provider: chatProv}})
	m.RegisterChain(chain)

	if _, err := m.Chat(context.Background(), &ChatRequest{Messages: []Message{UserMessage("which rule applies?")}}); err != nil {
		t.Fatalf("chat with tool call failed: %v", err)
	}
	if len(chatProv.seen) != 1 {
		t.Fatalf("a tool-call turn must cost one upstream request, got %d", len(chatProv.seen))
	}
}

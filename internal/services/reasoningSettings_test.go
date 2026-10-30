package services

import (
	"reflect"
	"testing"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/pkg/llm"
)

func TestBuildChatRequest_ReasoningFollowsModelSettingsOnly(t *testing.T) {
	svc := &advisoryService{}

	off := svc.buildChatRequest("model-a", &sqlc.GetLLMModelWithProviderRow{ThinkingEnabled: 0, ThinkingBudget: 4096, EffortLevel: "high"}, nil)
	if off.ThinkingEnabled == nil || *off.ThinkingEnabled || off.ThinkingBudget != nil || off.ReasoningEffort != "" {
		t.Fatalf("a model with thinking off must send no budget or effort, got %+v", off)
	}

	on := svc.buildChatRequest("model-b", &sqlc.GetLLMModelWithProviderRow{ThinkingEnabled: 1, ThinkingBudget: 4096, EffortLevel: "high"}, nil)
	if on.ThinkingEnabled == nil || !*on.ThinkingEnabled || on.ThinkingBudget == nil || *on.ThinkingBudget != 4096 || on.ReasoningEffort != "high" {
		t.Fatalf("a model with thinking on must carry its configured budget and effort, got %+v", on)
	}

	none := svc.buildChatRequest("", nil, nil)
	if none.ThinkingEnabled != nil || none.ThinkingBudget != nil || none.ReasoningEffort != "" {
		t.Fatalf("without a model row nothing is set so the chain node settings decide, got %+v", none)
	}
}

func TestChatRequestDTO_HasNoUserReasoningControls(t *testing.T) {
	typ := reflect.TypeOf(request.ChatRequest{})
	for _, name := range []string{"ThinkingEnabled", "ThinkingBudget", "EffortLevel"} {
		if _, found := typ.FieldByName(name); found {
			t.Fatalf("the main chat request must not expose %s to users", name)
		}
	}
}

func TestApplyAdminReasoning(t *testing.T) {
	yes, no := true, false

	req := &llm.ChatRequest{}
	applyAdminReasoning(req, nil, "high")
	if req.ThinkingEnabled != nil || req.ReasoningEffort != "" {
		t.Fatalf("no admin choice must leave the model settings in charge, got %+v", req)
	}

	req = &llm.ChatRequest{}
	applyAdminReasoning(req, &yes, "")
	if req.ThinkingEnabled == nil || !*req.ThinkingEnabled || req.ReasoningEffort != "medium" {
		t.Fatalf("thinking on without an effort defaults to medium, got %+v", req)
	}

	req = &llm.ChatRequest{}
	applyAdminReasoning(req, &yes, "high")
	if req.ReasoningEffort != "high" {
		t.Fatalf("the admin's effort must be used, got %q", req.ReasoningEffort)
	}

	req = &llm.ChatRequest{}
	applyAdminReasoning(req, &no, "high")
	if req.ThinkingEnabled == nil || *req.ThinkingEnabled || req.ReasoningEffort != "" {
		t.Fatalf("thinking off must carry no effort, got %+v", req)
	}
}

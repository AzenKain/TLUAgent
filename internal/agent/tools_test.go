package agent

import (
	"context"
	"testing"

	"tluagent-web/pkg/llm"
)

func TestParseToolDefinitionsSingle(t *testing.T) {
	raw := `{"name":"search_regulations","description":"Search rules","parameters":{"type":"object"}}`
	defs, err := ParseToolDefinitions(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing single tool: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0].Function.Name != "search_regulations" {
		t.Errorf("expected function name search_regulations, got %s", defs[0].Function.Name)
	}
	if defs[0].Type != "function" {
		t.Errorf("expected type function, got %s", defs[0].Type)
	}
}

func TestParseToolDefinitionsArray(t *testing.T) {
	raw := `[
		{"name":"tool_a","description":"Tool A"},
		{"name":"tool_b","description":"Tool B"}
	]`
	defs, err := ParseToolDefinitions(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing array tool: %v", err)
	}
	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(defs))
	}
	if defs[0].Function.Name != "tool_a" || defs[1].Function.Name != "tool_b" {
		t.Errorf("unexpected tool names: %v, %v", defs[0].Function.Name, defs[1].Function.Name)
	}
}

func TestGetDefaultToolDefinitions(t *testing.T) {
	defs := GetDefaultToolDefinitions()
	if len(defs) < 5 {
		t.Fatalf("expected at least 5 default tools, got %d", len(defs))
	}

	foundMap := make(map[string]bool)
	for _, d := range defs {
		foundMap[d.Function.Name] = true
	}

	expectedTools := []string{
		"search_regulations",
		"lookup_curriculum",
		"lookup_course_substitutions",
		"lookup_course_prerequisites",
		"lookup_student_transcript_and_roadmap",
		"optimize_study_roadmap",
		"update_student_academic_preferences",
		"lookup_tuition_fee",
		"escalate_inquiry",
	}

	for _, name := range expectedTools {
		if !foundMap[name] {
			t.Errorf("expected tool %s to be discovered from default skills, but was missing", name)
		}
	}
}

func TestGetDefaultTLUClawToolDefinitions(t *testing.T) {
	defs := GetDefaultTLUClawToolDefinitions()
	if len(defs) != 11 {
		t.Fatalf("expected 11 TLU Claw tools, got %d", len(defs))
	}
	foundMap := make(map[string]bool)
	for _, d := range defs {
		foundMap[d.Function.Name] = true
	}
	expectedTools := []string{
		"tluclaw_curriculum_operations",
		"tluclaw_document_operations",
		"tluclaw_audit_operations",
		"tluclaw_substitution_operations",
		"tluclaw_student_academic_operations",
		"tluclaw_user_operations",
		"tluclaw_inquiry_operations",
		"tluclaw_system_operations",
		"tluclaw_admin_memory_operations",
		"tluclaw_notification_operations",
		"tluclaw_job_operations",
	}
	for _, name := range expectedTools {
		if !foundMap[name] {
			t.Errorf("expected TLU Claw tool %s was missing", name)
		}
	}
}

func TestGetRegulatoryDissectionToolDefinitions(t *testing.T) {
	defs := GetRegulatoryDissectionToolDefinitions()
	if len(defs) != 6 {
		t.Fatalf("expected 6 regulatory dissection tools, got %d", len(defs))
	}
	foundMap := make(map[string]bool)
	for _, d := range defs {
		foundMap[d.Function.Name] = true
	}
	expectedTools := []string{
		"search_regulations_fts",
		"inspect_document_articles",
		"compare_article_diff",
		"search_knowledge_graph",
		"inspect_cohort_applicability",
		"propose_granular_supersession",
	}
	for _, name := range expectedTools {
		if !foundMap[name] {
			t.Errorf("expected regulatory dissection tool %s was missing", name)
		}
	}
}

func TestToolRegistry(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register("dummy_tool", func(tcx *ToolContext, tc llm.ToolCall) (*ToolResult, error) {
		return &ToolResult{
			Content: "dummy content",
			Title:   "Dummy Title",
			Sources: []string{"Source A"},
		}, nil
	})

	if !reg.HasTool("dummy_tool") {
		t.Errorf("expected reg to have dummy_tool")
	}

	tcx := &ToolContext{
		Context: context.Background(),
		UserID:  "test-user",
	}

	res, err := reg.Execute(tcx, llm.ToolCall{
		ID: "call-1",
		Function: llm.FunctionCall{
			Name: "dummy_tool",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error executing dummy_tool: %v", err)
	}
	if res.Content != "dummy content" || res.Title != "Dummy Title" || len(res.Sources) != 1 {
		t.Errorf("unexpected tool execution result: %+v", res)
	}

	_, notFoundErr := reg.Execute(tcx, llm.ToolCall{
		ID: "call-2",
		Function: llm.FunctionCall{
			Name: "non_existent_tool",
		},
	})
	if notFoundErr == nil {
		t.Errorf("expected error when executing non-existent tool, got nil")
	}
}

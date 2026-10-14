package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"tluagent-web/internal/agent/defaults"
	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/memory"
)

// ToolResult represents the standardized outcome of executing an agent skill tool.
type ToolResult struct {
	Content string   `json:"content"`
	Title   string   `json:"title"`
	Sources []string `json:"sources,omitempty"`
	Data    any      `json:"data,omitempty"`
}

// ToolContext encapsulates runtime context passed to tool handlers.
type ToolContext struct {
	Context context.Context
	UserID  string
	Profile *memory.StudentAcademicProfile
}

// ToolHandler represents an execution function for a specific agent tool.
type ToolHandler func(tcx *ToolContext, tc llm.ToolCall) (*ToolResult, error)

// ToolRegistry manages registration and dynamic execution of academic advisor tools.
type ToolRegistry interface {
	Register(name string, handler ToolHandler)
	Execute(tcx *ToolContext, tc llm.ToolCall) (*ToolResult, error)
	HasTool(name string) bool
}

type toolRegistry struct {
	mu       sync.RWMutex
	handlers map[string]ToolHandler
}

// NewToolRegistry creates a new concurrent-safe ToolRegistry instance.
func NewToolRegistry() ToolRegistry {
	return &toolRegistry{
		handlers: make(map[string]ToolHandler),
	}
}

func (r *toolRegistry) Register(name string, handler ToolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[strings.TrimSpace(name)] = handler
}

func (r *toolRegistry) HasTool(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.handlers[strings.TrimSpace(name)]
	return exists
}

func (r *toolRegistry) Execute(tcx *ToolContext, tc llm.ToolCall) (*ToolResult, error) {
	r.mu.RLock()
	handler, exists := r.handlers[strings.TrimSpace(tc.Function.Name)]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool handler not found for: %s", tc.Function.Name)
	}
	return handler(tcx, tc)
}

// ParseToolDefinitions parses raw JSON string into one or more LLM tool definitions.
func ParseToolDefinitions(raw string) ([]llm.ToolDefinition, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var toolDefs []llm.ToolDefinition
		if err := json.Unmarshal([]byte(trimmed), &toolDefs); err == nil && len(toolDefs) > 0 && toolDefs[0].Function.Name != "" {
			for i := range toolDefs {
				if toolDefs[i].Type == "" {
					toolDefs[i].Type = "function"
				}
			}
			return toolDefs, nil
		}

		var fnDefs []llm.FunctionDefinition
		if err := json.Unmarshal([]byte(trimmed), &fnDefs); err == nil && len(fnDefs) > 0 && fnDefs[0].Name != "" {
			res := make([]llm.ToolDefinition, len(fnDefs))
			for i, fn := range fnDefs {
				res[i] = llm.ToolDefinition{
					Type:     "function",
					Function: fn,
				}
			}
			return res, nil
		}

		var rawMaps []map[string]any
		if err := json.Unmarshal([]byte(trimmed), &rawMaps); err == nil {
			var res []llm.ToolDefinition
			for _, m := range rawMaps {
				if def, ok := parseSingleToolMap(m); ok {
					res = append(res, def)
				}
			}
			if len(res) > 0 {
				return res, nil
			}
		}
	} else {
		var toolDef llm.ToolDefinition
		if err := json.Unmarshal([]byte(trimmed), &toolDef); err == nil && toolDef.Function.Name != "" {
			if toolDef.Type == "" {
				toolDef.Type = "function"
			}
			return []llm.ToolDefinition{toolDef}, nil
		}

		var fnDef llm.FunctionDefinition
		if err := json.Unmarshal([]byte(trimmed), &fnDef); err == nil && fnDef.Name != "" {
			return []llm.ToolDefinition{{
				Type:     "function",
				Function: fnDef,
			}}, nil
		}

		var rawMap map[string]any
		if err := json.Unmarshal([]byte(trimmed), &rawMap); err == nil {
			if def, ok := parseSingleToolMap(rawMap); ok {
				return []llm.ToolDefinition{def}, nil
			}
		}
	}

	return nil, fmt.Errorf("unable to parse tool definition: %s", trimmed)
}

func parseSingleToolMap(m map[string]any) (llm.ToolDefinition, bool) {
	if fnRaw, ok := m["function"].(map[string]any); ok {
		name, _ := fnRaw["name"].(string)
		desc, _ := fnRaw["description"].(string)
		params := fnRaw["parameters"]
		if name != "" {
			return llm.ToolDefinition{
				Type: "function",
				Function: llm.FunctionDefinition{
					Name:        name,
					Description: desc,
					Parameters:  params,
				},
			}, true
		}
	}

	name, _ := m["name"].(string)
	if name != "" {
		desc, _ := m["description"].(string)
		params := m["parameters"]
		return llm.ToolDefinition{
			Type: "function",
			Function: llm.FunctionDefinition{
				Name:        name,
				Description: desc,
				Parameters:  params,
			},
		}, true
	}

	return llm.ToolDefinition{}, false
}

// GetDefaultToolDefinitions retrieves all default tool definitions discovered from embedded skill files.
func GetDefaultToolDefinitions() []llm.ToolDefinition {
	var result []llm.ToolDefinition
	seen := make(map[string]bool)
	for _, meta := range defaults.GetDefaultSkillsMetadata() {
		if meta.ToolDef == "" {
			continue
		}
		defs, err := ParseToolDefinitions(meta.ToolDef)
		if err != nil {
			continue
		}
		for _, def := range defs {
			if def.Function.Name != "" && !seen[def.Function.Name] {
				seen[def.Function.Name] = true
				result = append(result, def)
			}
		}
	}
	return result
}

// GetDefaultTLUClawToolDefinitions retrieves tool definitions dedicated to TLU Claw administrative operations.
func GetDefaultTLUClawToolDefinitions() []llm.ToolDefinition {
	var result []llm.ToolDefinition
	for _, def := range GetDefaultToolDefinitions() {
		if strings.HasPrefix(def.Function.Name, "tluclaw_") {
			result = append(result, def)
		}
	}
	return result
}

// GetDefaultAdvisorToolDefinitions retrieves tool definitions dedicated to student academic advising.
func GetDefaultAdvisorToolDefinitions() []llm.ToolDefinition {
	var result []llm.ToolDefinition
	for _, def := range GetDefaultToolDefinitions() {
		if !strings.HasPrefix(def.Function.Name, "tluclaw_") && !isRegulatoryDissectionTool(def.Function.Name) {
			result = append(result, def)
		}
	}
	return result
}

// GetRegulatoryDissectionToolDefinitions retrieves tool definitions dedicated to regulatory dissection and supersession auditing.
func GetRegulatoryDissectionToolDefinitions() []llm.ToolDefinition {
	for _, meta := range defaults.GetDefaultSkillsMetadata() {
		if meta.ID == defaults.SkillRegulatoryDissectionOpsID && meta.ToolDef != "" {
			if defs, err := ParseToolDefinitions(meta.ToolDef); err == nil && len(defs) > 0 {
				return defs
			}
		}
	}
	return nil
}

func isRegulatoryDissectionTool(name string) bool {
	switch name {
	case "search_regulations_fts",
		"inspect_document_articles",
		"compare_article_diff",
		"search_knowledge_graph",
		"inspect_cohort_applicability",
		"propose_granular_supersession":
		return true
	default:
		return false
	}
}


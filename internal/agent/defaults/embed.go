package defaults

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"sync"
)

//go:embed soul.md rule.md skills/*.md
var DefaultsFS embed.FS

const (
	PromptSoulID = "soul"
	PromptRuleID = "rule"

	SkillSearchRegulationsID        = "search_regulations"
	SkillCurriculumAdvisoryID       = "curriculum_advisory"
	SkillStudyRoadmapOptimizationID = "study_roadmap_optimization"
	SkillAcademicStandingID         = "academic_standing"
	SkillTuitionCalcID              = "tuition_calculation"
	SkillCourseRegID                = "course_registration"
	SkillInquiryEscalationID        = "inquiry_escalation"

	SkillTLUClawCurriculumOpsID      = "tluclaw_curriculum_ops"
	SkillTLUClawDocSupersessionOpsID = "tluclaw_document_supersession_ops"
	SkillTLUClawRollbackAuditOpsID   = "tluclaw_rollback_audit_ops"
	SkillTLUClawSubstitutionOpsID    = "tluclaw_substitution_ops"
	SkillTLUClawStudentAcademicOpsID = "tluclaw_student_academic_ops"
	SkillTLUClawUserOpsID            = "tluclaw_user_ops"
	SkillTLUClawNotificationOpsID    = "tluclaw_notification_ops"
	SkillTLUClawJobOpsID             = "tluclaw_job_ops"
	SkillTLUClawInquiryOpsID         = "tluclaw_inquiry_ops"
	SkillTLUClawSystemOpsID          = "tluclaw_system_ops"
	SkillTLUClawAdminMemoryOpsID     = "tluclaw_admin_memory_ops"
	SkillRegulatoryDissectionOpsID   = "regulatory_dissection_ops"
)

// DefaultPromptMeta holds metadata for embedded default prompts.
type DefaultPromptMeta struct {
	ID    string
	Title string
	Path  string
}

// DefaultSkillMeta holds metadata for embedded default skills.
type DefaultSkillMeta struct {
	ID          string
	Name        string
	Description string
	Path        string
	ToolDef     string
	Priority    int
}

var (
	cachedSkillsOnce sync.Once
	cachedSkills     []DefaultSkillMeta
)

// GetDefaultPromptsMetadata returns all default prompt definitions.
func GetDefaultPromptsMetadata() []DefaultPromptMeta {
	return []DefaultPromptMeta{
		{
			ID:    PromptSoulID,
			Title: "Căn Tính & Sứ Mệnh Cố Vấn Học Vụ TLU",
			Path:  "soul.md",
		},
		{
			ID:    PromptRuleID,
			Title: "Quy Tắc Hoạt Động & Ranh Giới Pháp Lý",
			Path:  "rule.md",
		},
	}
}

// DiscoverDefaultSkills dynamically discovers and parses all default skill definitions from embedded markdown files.
func DiscoverDefaultSkills() ([]DefaultSkillMeta, error) {
	entries, err := fs.ReadDir(DefaultsFS, "skills")
	if err != nil {
		return nil, fmt.Errorf("read skills directory: %w", err)
	}

	var skills []DefaultSkillMeta
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := "skills/" + entry.Name()
		data, err := DefaultsFS.ReadFile(filePath)
		if err != nil {
			continue
		}

		meta, _ := parseSkillFrontmatter(string(data), filePath, entry.Name())
		skills = append(skills, meta)
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Priority > skills[j].Priority
	})

	return skills, nil
}

// GetDefaultSkillsMetadata returns discovered default skill definitions.
func GetDefaultSkillsMetadata() []DefaultSkillMeta {
	cachedSkillsOnce.Do(func() {
		skills, err := DiscoverDefaultSkills()
		if err == nil {
			cachedSkills = skills
		}
	})
	return cachedSkills
}

func parseSkillFrontmatter(raw string, filePath, fileName string) (DefaultSkillMeta, string) {
	meta := DefaultSkillMeta{
		ID:       strings.TrimSuffix(fileName, ".md"),
		Path:     filePath,
		Priority: 50,
	}

	lines := strings.Split(raw, "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		return meta, raw
	}

	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		return meta, raw
	}

	fmLines := lines[1:endIdx]
	body := strings.Join(lines[endIdx+1:], "\n")

	var inToolDef bool
	var toolDefLines []string

	for _, line := range fmLines {
		trimmed := strings.TrimSpace(line)
		if inToolDef {
			if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
				toolDefLines = append(toolDefLines, trimmed)
				continue
			}
			inToolDef = false
		}

		if strings.HasPrefix(trimmed, "id:") {
			meta.ID = strings.TrimSpace(strings.TrimPrefix(trimmed, "id:"))
		} else if strings.HasPrefix(trimmed, "name:") {
			meta.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
		} else if strings.HasPrefix(trimmed, "description:") {
			meta.Description = strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
		} else if strings.HasPrefix(trimmed, "priority:") {
			if p, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(trimmed, "priority:"))); err == nil {
				meta.Priority = p
			}
		} else if strings.HasPrefix(trimmed, "tool_definition:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "tool_definition:"))
			if val == "|" || val == ">" || val == "" {
				inToolDef = true
			} else {
				meta.ToolDef = val
			}
		}
	}

	if len(toolDefLines) > 0 {
		meta.ToolDef = strings.Join(toolDefLines, "\n")
	}

	return meta, strings.TrimSpace(body)
}

// ReadDefaultFile reads content of an embedded default file (stripping frontmatter for skills).
func ReadDefaultFile(path string) (string, error) {
	data, err := DefaultsFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded default file %s: %w", path, err)
	}

	content := string(data)
	if strings.HasPrefix(path, "skills/") {
		_, body := parseSkillFrontmatter(content, path, path)
		if body != "" {
			return body, nil
		}
	}
	return content, nil
}

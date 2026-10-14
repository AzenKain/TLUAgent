package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"tluagent-web/internal/agent"
	"tluagent-web/internal/agent/defaults"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/memory"
)

// AgentService manages agent personas, rules, and modular skill instructions.
type AgentService interface {
	SeedDefaultsIfEmpty(ctx context.Context) error

	GetPrompts(ctx context.Context) ([]*response.AgentPromptResponse, error)
	GetPrompt(ctx context.Context, id string) (*response.AgentPromptResponse, error)
	UpdatePrompt(ctx context.Context, id string, title, content string, updatedBy string) (*response.AgentPromptResponse, error)
	ResetPrompt(ctx context.Context, id string, updatedBy string) (*response.AgentPromptResponse, error)

	GetSkills(ctx context.Context) ([]*response.AgentSkillResponse, error)
	GetSkill(ctx context.Context, id string) (*response.AgentSkillResponse, error)
	CreateSkill(ctx context.Context, req *request.CreateAgentSkillRequest, updatedBy string) (*response.AgentSkillResponse, error)
	UpdateSkill(ctx context.Context, id string, req *request.UpdateAgentSkillRequest, updatedBy string) (*response.AgentSkillResponse, error)
	ToggleSkill(ctx context.Context, id string, isEnabled bool, updatedBy string) error
	DeleteSkill(ctx context.Context, id string) error
	ResetSkill(ctx context.Context, id string, updatedBy string) (*response.AgentSkillResponse, error)

	GetActiveToolDefinitions(ctx context.Context) ([]llm.ToolDefinition, error)
	BuildSystemPrompt(ctx context.Context, profile *memory.StudentAcademicProfile, ragItems []*response.RAGSearchResultItemResponse, nonce ...string) (string, []string, error)
	PreviewSystemPrompt(ctx context.Context, studentCohort string, includeSampleRAG bool) (string, error)
	GetCompactorSettings(ctx context.Context) (*response.CompactorSettingsResponse, error)
	UpdateCompactorSettings(ctx context.Context, req *request.UpdateCompactorSettingsRequest) (*response.CompactorSettingsResponse, error)
}

type agentService struct {
	agentRepo    repositories.AgentRepository
	skillLoader  agent.SkillLoader
	ragService   RAGService
	settingsRepo repositories.SettingsRepository
}

// NewAgentService creates a new instance of AgentService.
func NewAgentService(
	agentRepo repositories.AgentRepository,
	ragService RAGService,
	settingsRepo repositories.SettingsRepository,
) AgentService {
	return &agentService{
		agentRepo:    agentRepo,
		skillLoader:  agent.NewDBSkillLoader(agentRepo),
		ragService:   ragService,
		settingsRepo: settingsRepo,
	}
}

func (s *agentService) SeedDefaultsIfEmpty(ctx context.Context) error {
	for _, meta := range defaults.GetDefaultPromptsMetadata() {
		existing, err := s.agentRepo.GetPrompt(ctx, meta.ID)
		if err != nil {
			log.Warn().Err(err).Str("prompt_id", meta.ID).Msg("Failed to check existing prompt during seeding")
			continue
		}
		if existing == nil {
			content, readErr := defaults.ReadDefaultFile(meta.Path)
			if readErr != nil {
				log.Warn().Err(readErr).Str("path", meta.Path).Msg("Failed to read default prompt file")
				continue
			}
			prompt := &models.AgentPrompt{
				ID:        meta.ID,
				Title:     meta.Title,
				Content:   content,
				IsActive:  true,
				UpdatedBy: "system",
			}
			if err := s.agentRepo.UpsertPrompt(ctx, prompt); err != nil {
				log.Warn().Err(err).Str("prompt_id", meta.ID).Msg("Failed to seed default prompt")
			}
		} else if existing.UpdatedBy == "system" {
			if content, readErr := defaults.ReadDefaultFile(meta.Path); readErr == nil && content != "" {
				existing.Title = meta.Title
				existing.Content = content
				_ = s.agentRepo.UpsertPrompt(ctx, existing)
			}
		}
	}

	for _, meta := range defaults.GetDefaultSkillsMetadata() {
		existing, err := s.skillLoader.GetSkill(ctx, meta.ID)
		if err != nil {
			log.Warn().Err(err).Str("skill_id", meta.ID).Msg("Failed to check existing skill during seeding")
			continue
		}
		if existing == nil {
			content, readErr := defaults.ReadDefaultFile(meta.Path)
			if readErr != nil {
				log.Warn().Err(readErr).Str("path", meta.Path).Msg("Failed to read default skill file")
				continue
			}
			skill := &models.AgentSkill{
				ID:             meta.ID,
				Name:           meta.Name,
				Description:    meta.Description,
				Content:        content,
				ToolDefinition: meta.ToolDef,
				IsEnabled:      true,
				Priority:       meta.Priority,
				UpdatedBy:      "system",
			}
			if err := s.agentRepo.UpsertSkill(ctx, skill); err != nil {
				log.Warn().Err(err).Str("skill_id", meta.ID).Msg("Failed to seed default skill")
			}
		} else if existing.UpdatedBy == "system" {
			if content, readErr := defaults.ReadDefaultFile(meta.Path); readErr == nil && content != "" {
				existing.Name = meta.Name
				existing.Description = meta.Description
				existing.Content = content
				existing.ToolDefinition = meta.ToolDef
				existing.Priority = meta.Priority
				_ = s.agentRepo.UpsertSkill(ctx, existing)
			}
		} else if existing.ToolDefinition == "" && meta.ToolDef != "" {
			existing.ToolDefinition = meta.ToolDef
			_ = s.agentRepo.UpsertSkill(ctx, existing)
		}
	}
	return nil
}

func (s *agentService) GetPrompts(ctx context.Context) ([]*response.AgentPromptResponse, error) {
	prompts, err := s.agentRepo.ListPrompts(ctx)
	if err != nil {
		return nil, err
	}
	return models.AgentPromptsToResponse(prompts), nil
}

func (s *agentService) GetPrompt(ctx context.Context, id string) (*response.AgentPromptResponse, error) {
	prompt, err := s.agentRepo.GetPrompt(ctx, id)
	if err != nil {
		return nil, err
	}
	if prompt == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "prompt not found")
	}
	return prompt.ToResponse(), nil
}

func (s *agentService) UpdatePrompt(ctx context.Context, id string, title, content string, updatedBy string) (*response.AgentPromptResponse, error) {
	trimmedTitle := strings.TrimSpace(title)
	trimmedContent := strings.TrimSpace(content)
	if trimmedTitle == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "prompt title cannot be empty")
	}
	if trimmedContent == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "prompt content cannot be empty")
	}

	prompt := &models.AgentPrompt{
		ID:        id,
		Title:     trimmedTitle,
		Content:   trimmedContent,
		IsActive:  true,
		UpdatedBy: updatedBy,
	}
	if err := s.agentRepo.UpsertPrompt(ctx, prompt); err != nil {
		return nil, err
	}
	res, err := s.agentRepo.GetPrompt(ctx, id)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "prompt not found")
	}
	return res.ToResponse(), nil
}

func (s *agentService) ResetPrompt(ctx context.Context, id string, updatedBy string) (*response.AgentPromptResponse, error) {
	var targetMeta *defaults.DefaultPromptMeta
	for _, meta := range defaults.GetDefaultPromptsMetadata() {
		if meta.ID == id {
			targetMeta = &meta
			break
		}
	}
	if targetMeta == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "no default prompt found for id: "+id)
	}

	content, err := defaults.ReadDefaultFile(targetMeta.Path)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "failed to read embedded default file: "+err.Error())
	}

	prompt := &models.AgentPrompt{
		ID:        targetMeta.ID,
		Title:     targetMeta.Title,
		Content:   content,
		IsActive:  true,
		UpdatedBy: updatedBy,
	}
	if err := s.agentRepo.UpsertPrompt(ctx, prompt); err != nil {
		return nil, err
	}
	res, err := s.agentRepo.GetPrompt(ctx, id)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "prompt not found")
	}
	return res.ToResponse(), nil
}

func (s *agentService) GetSkills(ctx context.Context) ([]*response.AgentSkillResponse, error) {
	skills, err := s.skillLoader.ListSkills(ctx)
	if err != nil {
		return nil, err
	}
	return models.AgentSkillsToResponse(skills), nil
}

func (s *agentService) GetSkill(ctx context.Context, id string) (*response.AgentSkillResponse, error) {
	skill, err := s.skillLoader.GetSkill(ctx, id)
	if err != nil {
		return nil, err
	}
	if skill == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "skill not found")
	}
	return skill.ToResponse(), nil
}

func (s *agentService) CreateSkill(ctx context.Context, req *request.CreateAgentSkillRequest, updatedBy string) (*response.AgentSkillResponse, error) {
	if req == nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "skill cannot be nil")
	}
	id := strings.TrimSpace(req.ID)
	name := strings.TrimSpace(req.Name)
	description := strings.TrimSpace(req.Description)
	content := strings.TrimSpace(req.Content)

	if id == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "skill id cannot be empty")
	}
	if name == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "skill name cannot be empty")
	}
	if content == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "skill content cannot be empty")
	}

	existing, err := s.agentRepo.GetSkill(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperrors.New(apperrors.ErrConflict, "skill with this id already exists")
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}
	priority := 0
	if req.Priority != nil {
		priority = *req.Priority
	}

	skill := &models.AgentSkill{
		ID:             id,
		Name:           name,
		Description:    description,
		Content:        content,
		ToolDefinition: strings.TrimSpace(req.ToolDefinition),
		IsEnabled:      isEnabled,
		Priority:       priority,
		UpdatedBy:      updatedBy,
	}

	if err := s.agentRepo.UpsertSkill(ctx, skill); err != nil {
		return nil, err
	}
	res, err := s.agentRepo.GetSkill(ctx, skill.ID)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "skill not found")
	}
	return res.ToResponse(), nil
}

func (s *agentService) UpdateSkill(ctx context.Context, id string, req *request.UpdateAgentSkillRequest, updatedBy string) (*response.AgentSkillResponse, error) {
	if req == nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "skill cannot be nil")
	}
	existing, err := s.agentRepo.GetSkill(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "skill not found")
	}

	trimmedName := strings.TrimSpace(req.Name)
	trimmedContent := strings.TrimSpace(req.Content)
	if trimmedName == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "skill name cannot be empty")
	}
	if trimmedContent == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "skill content cannot be empty")
	}

	priority := 0
	if req.Priority != nil {
		priority = *req.Priority
	}

	existing.Name = trimmedName
	existing.Description = strings.TrimSpace(req.Description)
	existing.Content = trimmedContent
	existing.ToolDefinition = strings.TrimSpace(req.ToolDefinition)
	existing.Priority = priority
	existing.UpdatedBy = updatedBy

	if err := s.agentRepo.UpsertSkill(ctx, existing); err != nil {
		return nil, err
	}
	res, err := s.agentRepo.GetSkill(ctx, id)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "skill not found")
	}
	return res.ToResponse(), nil
}

func (s *agentService) ToggleSkill(ctx context.Context, id string, isEnabled bool, updatedBy string) error {
	existing, err := s.agentRepo.GetSkill(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return apperrors.New(apperrors.ErrNotFound, "skill not found")
	}
	return s.agentRepo.UpdateSkillEnabled(ctx, id, isEnabled, updatedBy)
}

func (s *agentService) DeleteSkill(ctx context.Context, id string) error {
	existing, err := s.agentRepo.GetSkill(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return apperrors.New(apperrors.ErrNotFound, "skill not found")
	}
	return s.agentRepo.DeleteSkill(ctx, id)
}

func (s *agentService) ResetSkill(ctx context.Context, id string, updatedBy string) (*response.AgentSkillResponse, error) {
	var targetMeta *defaults.DefaultSkillMeta
	for _, meta := range defaults.GetDefaultSkillsMetadata() {
		if meta.ID == id {
			targetMeta = &meta
			break
		}
	}
	if targetMeta == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "no default skill found for id: "+id)
	}

	content, err := defaults.ReadDefaultFile(targetMeta.Path)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "failed to read embedded default file: "+err.Error())
	}

	skill := &models.AgentSkill{
		ID:             targetMeta.ID,
		Name:           targetMeta.Name,
		Description:    targetMeta.Description,
		Content:        content,
		ToolDefinition: targetMeta.ToolDef,
		IsEnabled:      true,
		Priority:       targetMeta.Priority,
		UpdatedBy:      updatedBy,
	}
	if err := s.agentRepo.UpsertSkill(ctx, skill); err != nil {
		return nil, err
	}
	res, err := s.agentRepo.GetSkill(ctx, id)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "skill not found")
	}
	return res.ToResponse(), nil
}

func (s *agentService) BuildSystemPrompt(ctx context.Context, profile *memory.StudentAcademicProfile, ragItems []*response.RAGSearchResultItemResponse, nonce ...string) (string, []string, error) {
	var sb strings.Builder
	var sources []string

	currentNonce := ""
	if len(nonce) > 0 && nonce[0] != "" {
		currentNonce = nonce[0]
	} else {
		currentNonce, _ = crypto.GenerateRandomHex(8)
	}

	soulPrompt, err := s.agentRepo.GetPrompt(ctx, defaults.PromptSoulID)
	if err == nil && soulPrompt != nil && soulPrompt.IsActive && soulPrompt.Content != "" {
		sb.WriteString(soulPrompt.Content)
	} else {
		if defContent, defErr := defaults.ReadDefaultFile("soul.md"); defErr == nil {
			sb.WriteString(defContent)
		}
	}

	sb.WriteString("\n\n---\n\n")

	rulePrompt, err := s.agentRepo.GetPrompt(ctx, defaults.PromptRuleID)
	if err == nil && rulePrompt != nil && rulePrompt.IsActive && rulePrompt.Content != "" {
		sb.WriteString(rulePrompt.Content)
	} else {
		if defContent, defErr := defaults.ReadDefaultFile("rule.md"); defErr == nil {
			sb.WriteString(defContent)
		}
	}

	enabledSkills, err := s.skillLoader.GetActiveSkills(ctx)
	if err == nil && len(enabledSkills) > 0 {
		sb.WriteString("\n\n---\n\n# ACTIVE OPERATIONAL SKILLS\n\n")
		for _, skill := range enabledSkills {
			fmt.Fprintf(&sb, "## Skill: %s (%s)\n%s\n\n", skill.Name, skill.ID, skill.Content)
		}
	}

	if profile != nil {
		sb.WriteString("\n---\n\n# CURRENT STUDENT PROFILE CONTEXT\n\n")
		fmt.Fprintf(&sb, "<student_memory nonce=\"%s\">\n%s\n</student_memory>\n", currentNonce, memory.FormatStudentPromptContext(profile))
	}

	if len(ragItems) > 0 {
		sb.WriteString("\n---\n\n# RETRIEVED REGULATORY DOCUMENTS (RAG KNOWLEDGE):\n\n")
		seenSources := make(map[string]bool)
		for _, item := range ragItems {
			fmt.Fprintf(&sb, "<rag_context nonce=\"%s\">\n- Document: %s\nContent: %s\n</rag_context>\n\n", currentNonce, item.DocTitle, item.Content)
			if item.DocTitle != "" && !seenSources[item.DocTitle] {
				seenSources[item.DocTitle] = true
				sources = append(sources, item.DocTitle)
			}
		}
	}

	return sb.String(), sources, nil
}

func (s *agentService) PreviewSystemPrompt(ctx context.Context, studentCohort string, includeSampleRAG bool) (string, error) {
	var profile *memory.StudentAcademicProfile
	if studentCohort != "" {
		cohortNum := 0
		cleanCohort := strings.TrimPrefix(strings.ToUpper(studentCohort), "K")
		if n, err := strconv.Atoi(cleanCohort); err == nil {
			cohortNum = n
		}
		profile = &memory.StudentAcademicProfile{
			StudentID:  "SV_PREVIEW_USER",
			CohortYear: cohortNum,
		}
	}

	var sampleRAG []*response.RAGSearchResultItemResponse
	if includeSampleRAG && s.ragService != nil {
		if results, err := s.ragService.Search(ctx, models.RAGSearchParams{
			StudentCohort: studentCohort,
			TopK:          2,
		}, nil); err == nil {
			sampleRAG = results
		}
	}

	prompt, _, err := s.BuildSystemPrompt(ctx, profile, sampleRAG)
	return prompt, err
}

func (s *agentService) GetCompactorSettings(ctx context.Context) (*response.CompactorSettingsResponse, error) {
	if s.settingsRepo != nil {
		if raw, err := s.settingsRepo.GetAppSetting(ctx, "compactor_settings"); err == nil && raw != "" {
			var cfg request.UpdateCompactorSettingsRequest
			if err := json.Unmarshal([]byte(raw), &cfg); err == nil {
				return &response.CompactorSettingsResponse{
					MaxContextTokens:      cfg.MaxContextTokens,
					CompactThresholdRatio: cfg.CompactThresholdRatio,
					KeepRecentTurns:       cfg.KeepRecentTurns,
				}, nil
			}
		}
	}
	return &response.CompactorSettingsResponse{
		MaxContextTokens:      250000,
		CompactThresholdRatio: 0.80,
		KeepRecentTurns:       10,
	}, nil
}

func (s *agentService) UpdateCompactorSettings(ctx context.Context, req *request.UpdateCompactorSettingsRequest) (*response.CompactorSettingsResponse, error) {
	if s.settingsRepo == nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Settings repository not configured")
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if err := s.settingsRepo.UpsertAppSetting(ctx, "compactor_settings", string(data)); err != nil {
		return nil, err
	}
	return &response.CompactorSettingsResponse{
		MaxContextTokens:      req.MaxContextTokens,
		CompactThresholdRatio: req.CompactThresholdRatio,
		KeepRecentTurns:       req.KeepRecentTurns,
	}, nil
}

// GetActiveToolDefinitions dynamically resolves active tool definitions from enabled skills.
func (s *agentService) GetActiveToolDefinitions(ctx context.Context) ([]llm.ToolDefinition, error) {
	enabledSkills, err := s.skillLoader.GetActiveSkills(ctx)
	if err != nil || len(enabledSkills) == 0 {
		return agent.GetDefaultToolDefinitions(), nil
	}

	defaultMetaMap := make(map[string]string)
	for _, meta := range defaults.GetDefaultSkillsMetadata() {
		if meta.ToolDef != "" {
			defaultMetaMap[meta.ID] = meta.ToolDef
		}
	}

	var results []llm.ToolDefinition
	seen := make(map[string]bool)

	for _, skill := range enabledSkills {
		rawDef := strings.TrimSpace(skill.ToolDefinition)
		if rawDef == "" {
			rawDef = defaultMetaMap[skill.ID]
		}
		if rawDef == "" {
			continue
		}

		defs, parseErr := agent.ParseToolDefinitions(rawDef)
		if parseErr != nil {
			log.Warn().Err(parseErr).Str("skill_id", skill.ID).Msg("Failed to parse tool definition for skill")
			continue
		}

		for _, d := range defs {
			if d.Function.Name != "" && !seen[d.Function.Name] {
				seen[d.Function.Name] = true
				results = append(results, d)
			}
		}
	}

	if len(results) == 0 {
		return agent.GetDefaultToolDefinitions(), nil
	}

	return results, nil
}


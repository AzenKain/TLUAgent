package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"tluagent-web/internal/agent/defaults"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/memory"
)

// AgentService manages agent personas, rules, and modular skill instructions.
type AgentService interface {
	SetRAGService(ragService RAGService)
	SeedDefaultsIfEmpty(ctx context.Context) error

	GetPrompts(ctx context.Context) ([]*models.AgentPrompt, error)
	GetPrompt(ctx context.Context, id string) (*models.AgentPrompt, error)
	UpdatePrompt(ctx context.Context, id string, title, content string, updatedBy string) (*models.AgentPrompt, error)
	ResetPrompt(ctx context.Context, id string, updatedBy string) (*models.AgentPrompt, error)

	GetSkills(ctx context.Context) ([]*models.AgentSkill, error)
	GetSkill(ctx context.Context, id string) (*models.AgentSkill, error)
	CreateSkill(ctx context.Context, skill *models.AgentSkill) (*models.AgentSkill, error)
	UpdateSkill(ctx context.Context, id string, name, description, content, toolDef string, priority int, updatedBy string) (*models.AgentSkill, error)
	ToggleSkill(ctx context.Context, id string, isEnabled bool, updatedBy string) error
	DeleteSkill(ctx context.Context, id string) error
	ResetSkill(ctx context.Context, id string, updatedBy string) (*models.AgentSkill, error)

	BuildSystemPrompt(ctx context.Context, profile *memory.StudentAcademicProfile, ragItems []*models.RAGSearchResultItem) (string, []string, error)
	PreviewSystemPrompt(ctx context.Context, studentCohort string, includeSampleRAG bool) (string, error)
	GetCompactorSettings(ctx context.Context) (*response.CompactorSettingsResponse, error)
	UpdateCompactorSettings(ctx context.Context, req *request.UpdateCompactorSettingsRequest) (*response.CompactorSettingsResponse, error)
	SetSettingsRepository(settingsRepo repositories.SettingsRepository)
}

type agentService struct {
	agentRepo    repositories.AgentRepository
	ragService   RAGService
	settingsRepo repositories.SettingsRepository
}

// NewAgentService creates a new instance of AgentService.
func NewAgentService(agentRepo repositories.AgentRepository) AgentService {
	return &agentService{
		agentRepo: agentRepo,
	}
}

func (s *agentService) SetSettingsRepository(settingsRepo repositories.SettingsRepository) {
	s.settingsRepo = settingsRepo
}

func (s *agentService) SetRAGService(ragService RAGService) {
	s.ragService = ragService
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
		}
	}

	for _, meta := range defaults.GetDefaultSkillsMetadata() {
		existing, err := s.agentRepo.GetSkill(ctx, meta.ID)
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
		}
	}
	return nil
}

func (s *agentService) GetPrompts(ctx context.Context) ([]*models.AgentPrompt, error) {
	return s.agentRepo.ListPrompts(ctx)
}

func (s *agentService) GetPrompt(ctx context.Context, id string) (*models.AgentPrompt, error) {
	prompt, err := s.agentRepo.GetPrompt(ctx, id)
	if err != nil {
		return nil, err
	}
	if prompt == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "prompt not found")
	}
	return prompt, nil
}

func (s *agentService) UpdatePrompt(ctx context.Context, id string, title, content string, updatedBy string) (*models.AgentPrompt, error) {
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
	return s.agentRepo.GetPrompt(ctx, id)
}

func (s *agentService) ResetPrompt(ctx context.Context, id string, updatedBy string) (*models.AgentPrompt, error) {
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
	return s.agentRepo.GetPrompt(ctx, id)
}

func (s *agentService) GetSkills(ctx context.Context) ([]*models.AgentSkill, error) {
	return s.agentRepo.ListSkills(ctx)
}

func (s *agentService) GetSkill(ctx context.Context, id string) (*models.AgentSkill, error) {
	skill, err := s.agentRepo.GetSkill(ctx, id)
	if err != nil {
		return nil, err
	}
	if skill == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "skill not found")
	}
	return skill, nil
}

func (s *agentService) CreateSkill(ctx context.Context, skill *models.AgentSkill) (*models.AgentSkill, error) {
	if skill == nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "skill cannot be nil")
	}
	skill.ID = strings.TrimSpace(skill.ID)
	skill.Name = strings.TrimSpace(skill.Name)
	skill.Description = strings.TrimSpace(skill.Description)
	skill.Content = strings.TrimSpace(skill.Content)

	if skill.ID == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "skill id cannot be empty")
	}
	if skill.Name == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "skill name cannot be empty")
	}
	if skill.Content == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "skill content cannot be empty")
	}

	existing, err := s.agentRepo.GetSkill(ctx, skill.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperrors.New(apperrors.ErrConflict, "skill with this id already exists")
	}

	if err := s.agentRepo.UpsertSkill(ctx, skill); err != nil {
		return nil, err
	}
	return s.agentRepo.GetSkill(ctx, skill.ID)
}

func (s *agentService) UpdateSkill(ctx context.Context, id string, name, description, content, toolDef string, priority int, updatedBy string) (*models.AgentSkill, error) {
	existing, err := s.agentRepo.GetSkill(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "skill not found")
	}

	trimmedName := strings.TrimSpace(name)
	trimmedContent := strings.TrimSpace(content)
	if trimmedName == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "skill name cannot be empty")
	}
	if trimmedContent == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "skill content cannot be empty")
	}

	existing.Name = trimmedName
	existing.Description = strings.TrimSpace(description)
	existing.Content = trimmedContent
	existing.ToolDefinition = strings.TrimSpace(toolDef)
	existing.Priority = priority
	existing.UpdatedBy = updatedBy

	if err := s.agentRepo.UpsertSkill(ctx, existing); err != nil {
		return nil, err
	}
	return s.agentRepo.GetSkill(ctx, id)
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

func (s *agentService) ResetSkill(ctx context.Context, id string, updatedBy string) (*models.AgentSkill, error) {
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
	return s.agentRepo.GetSkill(ctx, id)
}

func (s *agentService) BuildSystemPrompt(ctx context.Context, profile *memory.StudentAcademicProfile, ragItems []*models.RAGSearchResultItem) (string, []string, error) {
	var sb strings.Builder
	var sources []string

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

	enabledSkills, err := s.agentRepo.ListEnabledSkills(ctx)
	if err == nil && len(enabledSkills) > 0 {
		sb.WriteString("\n\n---\n\n# ACTIVE OPERATIONAL SKILLS\n\n")
		for _, skill := range enabledSkills {
			fmt.Fprintf(&sb, "## Skill: %s (%s)\n%s\n\n", skill.Name, skill.ID, skill.Content)
		}
	}

	if profile != nil {
		sb.WriteString("\n---\n\n# CURRENT STUDENT PROFILE CONTEXT\n\n")
		sb.WriteString(memory.FormatStudentPromptContext(profile))
		sb.WriteString("\n")
	}

	if len(ragItems) > 0 {
		sb.WriteString("\n---\n\n# RETRIEVED REGULATORY DOCUMENTS (RAG KNOWLEDGE):\n\n")
		for _, item := range ragItems {
			fmt.Fprintf(&sb, "- Document: %s\nContent: %s\n\n", item.DocTitle, item.Content)
			if item.DocTitle != "" {
				sources = append(sources, item.DocTitle)
			}
		}
	}

	sb.WriteString("\n---\n\n# MANDATORY CITATION & VERIFICATION DIRECTIVE:\n")
	sb.WriteString("1. CITATION REQUIREMENT: At the very end of EVERY response, you MUST append a 'Nguồn trích dẫn / Căn cứ pháp lý:' section clearly citing the document titles and specific articles used so the student can verify.\n")
	sb.WriteString("2. STRICT ZERO-HALLUCINATION POLICY: Never invent or guess rules, policies, dates, or tuition fees not present in the retrieved context. If an inquiry cannot be answered by the official documents provided, clearly declare that the information is not found in official TLU documents and guide the student to contact the Academic Affairs Department (Phòng Đào tạo / Nhà T).\n")
	sb.WriteString("3. ADVISOR ESCALATION DIRECTIVE: If and ONLY if you DO NOT have sufficient official documents in the provided context to answer the student's question, OR if the question involves special personal requests requiring advisor approval/intervention (e.g. grade appeal, deferred exam, special study plan approval), you MUST append the exact token `[CAN_ESCALATE]` at the very end of your response. If you have enough official documents to answer clearly, DO NOT include `[CAN_ESCALATE]`.\n")

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
			StudentID:  "SV" + cleanCohort + "001",
			CohortYear: cohortNum,
		}
	}

	var sampleRAG []*models.RAGSearchResultItem
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

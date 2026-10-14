package agent

import (
	"context"

	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
)

// SkillLoader defines the contract for dynamically querying and loading skills from the database.
type SkillLoader interface {
	GetActiveSkills(ctx context.Context) ([]*models.AgentSkill, error)
	GetSkill(ctx context.Context, id string) (*models.AgentSkill, error)
	ListSkills(ctx context.Context) ([]*models.AgentSkill, error)
}

type dbSkillLoader struct {
	agentRepo repositories.AgentRepository
}

// NewDBSkillLoader creates a new SkillLoader instance backed by AgentRepository.
func NewDBSkillLoader(agentRepo repositories.AgentRepository) SkillLoader {
	return &dbSkillLoader{agentRepo: agentRepo}
}

func (l *dbSkillLoader) GetActiveSkills(ctx context.Context) ([]*models.AgentSkill, error) {
	return l.agentRepo.ListEnabledSkills(ctx)
}

func (l *dbSkillLoader) GetSkill(ctx context.Context, id string) (*models.AgentSkill, error) {
	return l.agentRepo.GetSkill(ctx, id)
}

func (l *dbSkillLoader) ListSkills(ctx context.Context) ([]*models.AgentSkill, error) {
	return l.agentRepo.ListSkills(ctx)
}

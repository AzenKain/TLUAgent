package repositories

import (
	"context"
	"database/sql"
	"errors"

	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/convert"
)

// AgentRepository defines data access methods for agent prompts and modular skills.
type AgentRepository interface {
	GetPrompt(ctx context.Context, id string) (*models.AgentPrompt, error)
	ListPrompts(ctx context.Context) ([]*models.AgentPrompt, error)
	ListActivePrompts(ctx context.Context) ([]*models.AgentPrompt, error)
	UpsertPrompt(ctx context.Context, prompt *models.AgentPrompt) error

	GetSkill(ctx context.Context, id string) (*models.AgentSkill, error)
	ListSkills(ctx context.Context) ([]*models.AgentSkill, error)
	ListEnabledSkills(ctx context.Context) ([]*models.AgentSkill, error)
	UpsertSkill(ctx context.Context, skill *models.AgentSkill) error
	UpdateSkillEnabled(ctx context.Context, id string, isEnabled bool, updatedBy string) error
	DeleteSkill(ctx context.Context, id string) error

	InvalidatePromptCache(ctx context.Context, id string)
	InvalidateSkillCache(ctx context.Context, id string)
	WithTx(tx *sql.Tx) AgentRepository
}

type agentRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

// NewAgentRepository creates a new instance of AgentRepository.
func NewAgentRepository(db sqlc.DBTX, c cache.Cache) AgentRepository {
	return &agentRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *agentRepository) WithTx(tx *sql.Tx) AgentRepository {
	return &agentRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sfg:  &singleflight.Group{},
	}
}

func (r *agentRepository) GetPrompt(ctx context.Context, id string) (*models.AgentPrompt, error) {
	cacheKey := constants.CacheKeyAgentPromptPrefix + id
	if r.c != nil && !r.inTx {
		var prompt models.AgentPrompt
		if err := r.c.Get(ctx, cacheKey, &prompt); err == nil {
			return &prompt, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		row, err := r.q.GetAgentPrompt(ctx, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, err
		}
		var prompt models.AgentPrompt
		prompt.FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, prompt, constants.NormalCacheDuration)
		}
		return &prompt, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*models.AgentPrompt), nil
}

func (r *agentRepository) ListPrompts(ctx context.Context) ([]*models.AgentPrompt, error) {
	cacheKey := constants.CacheKeyAgentPromptsAll
	if r.c != nil && !r.inTx {
		var prompts []*models.AgentPrompt
		if err := r.c.Get(ctx, cacheKey, &prompts); err == nil {
			return prompts, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		rows, err := r.q.ListAgentPrompts(ctx)
		if err != nil {
			return nil, err
		}
		result := make([]*models.AgentPrompt, 0, len(rows))
		for _, row := range rows {
			var p models.AgentPrompt
			p.FromSqlc(row)
			result = append(result, &p)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.AgentPrompt), nil
}

func (r *agentRepository) ListActivePrompts(ctx context.Context) ([]*models.AgentPrompt, error) {
	cacheKey := constants.CacheKeyAgentPromptsAll + ":active"
	if r.c != nil && !r.inTx {
		var prompts []*models.AgentPrompt
		if err := r.c.Get(ctx, cacheKey, &prompts); err == nil {
			return prompts, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		rows, err := r.q.ListActiveAgentPrompts(ctx)
		if err != nil {
			return nil, err
		}
		result := make([]*models.AgentPrompt, 0, len(rows))
		for _, row := range rows {
			var p models.AgentPrompt
			p.FromSqlc(row)
			result = append(result, &p)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.AgentPrompt), nil
}

func (r *agentRepository) UpsertPrompt(ctx context.Context, prompt *models.AgentPrompt) error {
	isActive := int64(0)
	if prompt.IsActive {
		isActive = 1
	}

	err := r.q.UpsertAgentPrompt(ctx, sqlc.UpsertAgentPromptParams{
		ID:        prompt.ID,
		Title:     prompt.Title,
		Content:   prompt.Content,
		IsActive:  isActive,
		UpdatedBy: convert.StringToNullString(prompt.UpdatedBy),
	})
	if err != nil {
		return err
	}

	r.InvalidatePromptCache(ctx, prompt.ID)
	return nil
}

func (r *agentRepository) GetSkill(ctx context.Context, id string) (*models.AgentSkill, error) {
	cacheKey := constants.CacheKeyAgentSkillPrefix + id
	if r.c != nil && !r.inTx {
		var skill models.AgentSkill
		if err := r.c.Get(ctx, cacheKey, &skill); err == nil {
			return &skill, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		row, err := r.q.GetAgentSkill(ctx, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, err
		}
		var skill models.AgentSkill
		skill.FromSqlc(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, skill, constants.NormalCacheDuration)
		}
		return &skill, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*models.AgentSkill), nil
}

func (r *agentRepository) ListSkills(ctx context.Context) ([]*models.AgentSkill, error) {
	cacheKey := constants.CacheKeyAgentSkillsAll
	if r.c != nil && !r.inTx {
		var skills []*models.AgentSkill
		if err := r.c.Get(ctx, cacheKey, &skills); err == nil {
			return skills, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		rows, err := r.q.ListAgentSkills(ctx)
		if err != nil {
			return nil, err
		}
		result := make([]*models.AgentSkill, 0, len(rows))
		for _, row := range rows {
			var s models.AgentSkill
			s.FromSqlc(row)
			result = append(result, &s)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.AgentSkill), nil
}

func (r *agentRepository) ListEnabledSkills(ctx context.Context) ([]*models.AgentSkill, error) {
	cacheKey := constants.CacheKeyAgentSkillsEnabled
	if r.c != nil && !r.inTx {
		var skills []*models.AgentSkill
		if err := r.c.Get(ctx, cacheKey, &skills); err == nil {
			return skills, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		rows, err := r.q.ListEnabledAgentSkills(ctx)
		if err != nil {
			return nil, err
		}
		result := make([]*models.AgentSkill, 0, len(rows))
		for _, row := range rows {
			var s models.AgentSkill
			s.FromSqlc(row)
			result = append(result, &s)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, result, constants.NormalCacheDuration)
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.AgentSkill), nil
}

func (r *agentRepository) UpsertSkill(ctx context.Context, skill *models.AgentSkill) error {
	isEnabled := int64(0)
	if skill.IsEnabled {
		isEnabled = 1
	}

	err := r.q.UpsertAgentSkill(ctx, sqlc.UpsertAgentSkillParams{
		ID:             skill.ID,
		Name:           skill.Name,
		Description:    skill.Description,
		Content:        skill.Content,
		ToolDefinition: convert.StringToNullString(skill.ToolDefinition),
		IsEnabled:      isEnabled,
		Priority:       int64(skill.Priority),
		UpdatedBy:      convert.StringToNullString(skill.UpdatedBy),
	})
	if err != nil {
		return err
	}

	r.InvalidateSkillCache(ctx, skill.ID)
	return nil
}

func (r *agentRepository) UpdateSkillEnabled(ctx context.Context, id string, isEnabled bool, updatedBy string) error {
	enabledVal := int64(0)
	if isEnabled {
		enabledVal = 1
	}

	err := r.q.UpdateAgentSkillEnabled(ctx, sqlc.UpdateAgentSkillEnabledParams{
		ID:        id,
		IsEnabled: enabledVal,
		UpdatedBy: convert.StringToNullString(updatedBy),
	})
	if err != nil {
		return err
	}

	r.InvalidateSkillCache(ctx, id)
	return nil
}

func (r *agentRepository) DeleteSkill(ctx context.Context, id string) error {
	err := r.q.DeleteAgentSkill(ctx, id)
	if err != nil {
		return err
	}

	r.InvalidateSkillCache(ctx, id)
	return nil
}

func (r *agentRepository) InvalidatePromptCache(ctx context.Context, id string) {
	if r.c == nil {
		return
	}
	_ = r.c.Del(ctx, constants.CacheKeyAgentPromptPrefix+id)
	_ = r.c.Del(ctx, constants.CacheKeyAgentPromptsAll)
	_ = r.c.Del(ctx, constants.CacheKeyAgentPromptsAll+":active")
}

func (r *agentRepository) InvalidateSkillCache(ctx context.Context, id string) {
	if r.c == nil {
		return
	}
	_ = r.c.Del(ctx, constants.CacheKeyAgentSkillPrefix+id)
	_ = r.c.Del(ctx, constants.CacheKeyAgentSkillsAll)
	_ = r.c.Del(ctx, constants.CacheKeyAgentSkillsEnabled)
}

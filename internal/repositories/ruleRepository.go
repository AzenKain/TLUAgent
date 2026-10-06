package repositories

import (
	"context"
	"strings"

	"tluagent-web/internal/models"
)

// RuleRepository provides in-memory fallback retrieval for legacy rule entities.
type RuleRepository interface {
	FindByQuery(ctx context.Context, query string) (*models.AdvisoryRuleEntity, error)
	ListAll(ctx context.Context) ([]*models.AdvisoryRuleEntity, error)
}

type memoryRuleRepository struct {
	rules []*models.AdvisoryRuleEntity
}

// NewMemoryRuleRepository creates an in-memory rule repository.
func NewMemoryRuleRepository() RuleRepository {
	return &memoryRuleRepository{
		rules: []*models.AdvisoryRuleEntity{},
	}
}

func (r *memoryRuleRepository) FindByQuery(ctx context.Context, query string) (*models.AdvisoryRuleEntity, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	q := strings.ToLower(query)
	for _, rule := range r.rules {
		for _, kw := range rule.Keywords {
			if strings.Contains(q, kw) {
				return rule, nil
			}
		}
	}
	return nil, nil
}

func (r *memoryRuleRepository) ListAll(ctx context.Context) ([]*models.AdvisoryRuleEntity, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return r.rules, nil
}

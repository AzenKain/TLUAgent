package repositories

import (
	"context"
	"database/sql"

	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
)

type LLMRepository interface {
	ListProviders(ctx context.Context) ([]sqlc.LlmProvider, error)
	GetProviderByID(ctx context.Context, id string) (sqlc.LlmProvider, error)
	GetDefaultProvider(ctx context.Context) (sqlc.LlmProvider, error)
	CreateProvider(ctx context.Context, p sqlc.CreateLLMProviderParams) (sqlc.LlmProvider, error)
	UpdateProvider(ctx context.Context, p sqlc.UpdateLLMProviderParams) (sqlc.LlmProvider, error)
	SetDefaultProvider(ctx context.Context, id string) error
	DeleteProvider(ctx context.Context, id string) error

	ListModels(ctx context.Context) ([]sqlc.LlmModel, error)
	ListModelsByProviderID(ctx context.Context, providerID string) ([]sqlc.LlmModel, error)
	ListActiveModelsByType(ctx context.Context, modelType string) ([]sqlc.LlmModel, error)
	GetModelByID(ctx context.Context, id string) (sqlc.LlmModel, error)
	GetModelByKey(ctx context.Context, modelKey string) (sqlc.LlmModel, error)
	GetDefaultModelByType(ctx context.Context, modelType string) (sqlc.LlmModel, error)
	CreateModel(ctx context.Context, m sqlc.CreateLLMModelParams) (sqlc.LlmModel, error)
	UpdateModel(ctx context.Context, m sqlc.UpdateLLMModelParams) (sqlc.LlmModel, error)
	UpdateModelVisionCapability(ctx context.Context, p sqlc.UpdateModelVisionCapabilityParams) (sqlc.LlmModel, error)
	SetDefaultModel(ctx context.Context, id string, modelType string) error
	DeleteModel(ctx context.Context, id string) error

	GetModelWithProvider(ctx context.Context, modelID string) (sqlc.GetLLMModelWithProviderRow, error)
	ListActiveChatModelsWithProvider(ctx context.Context) ([]sqlc.ListActiveChatModelsWithProviderRow, error)

	GetChain(ctx context.Context, id string) (sqlc.LlmChain, error)
	GetActiveChainByType(ctx context.Context, chainType string) (sqlc.LlmChain, error)
	ListChains(ctx context.Context) ([]sqlc.LlmChain, error)
	CreateChain(ctx context.Context, p sqlc.CreateChainParams) (sqlc.LlmChain, error)
	UpdateChain(ctx context.Context, p sqlc.UpdateChainParams) (sqlc.LlmChain, error)
	DeleteChain(ctx context.Context, id string) error

	ListChainNodesWithDetails(ctx context.Context, chainID string) ([]sqlc.ListChainNodesWithDetailsRow, error)
	AddChainNode(ctx context.Context, p sqlc.AddChainNodeParams) (sqlc.LlmChainNode, error)
	RemoveChainNode(ctx context.Context, id string) error
	UpdateChainNodePriority(ctx context.Context, p sqlc.UpdateChainNodePriorityParams) error
	ClearChainNodes(ctx context.Context, chainID string) error

	WithTx(tx *sql.Tx) LLMRepository
}

type llmRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

// NewLLMRepository creates a new LLMRepository instance.
func NewLLMRepository(db sqlc.DBTX, c cache.Cache) LLMRepository {
	return &llmRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *llmRepository) WithTx(tx *sql.Tx) LLMRepository {
	return &llmRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sfg:  &singleflight.Group{},
	}
}

func (r *llmRepository) ListProviders(ctx context.Context) ([]sqlc.LlmProvider, error) {
	cacheKey := constants.CacheKeyLLMProvidersAll
	if r.c != nil && !r.inTx {
		var cached []sqlc.LlmProvider
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.ListLLMProviders(ctx)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]sqlc.LlmProvider), nil
}

func (r *llmRepository) GetProviderByID(ctx context.Context, id string) (sqlc.LlmProvider, error) {
	cacheKey := cache.BuildKey("llm_provider", "id", id)
	if r.c != nil && !r.inTx {
		var p sqlc.LlmProvider
		if err := r.c.Get(ctx, cacheKey, &p); err == nil {
			return p, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.GetLLMProviderByID(ctx, id)
		if err != nil {
			return sqlc.LlmProvider{}, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return sqlc.LlmProvider{}, err
	}
	return v.(sqlc.LlmProvider), nil
}

func (r *llmRepository) GetDefaultProvider(ctx context.Context) (sqlc.LlmProvider, error) {
	cacheKey := cache.BuildKey("llm_provider", "default")
	if r.c != nil && !r.inTx {
		var p sqlc.LlmProvider
		if err := r.c.Get(ctx, cacheKey, &p); err == nil {
			return p, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.GetDefaultLLMProvider(ctx)
		if err != nil {
			return sqlc.LlmProvider{}, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return sqlc.LlmProvider{}, err
	}
	return v.(sqlc.LlmProvider), nil
}

func (r *llmRepository) CreateProvider(ctx context.Context, p sqlc.CreateLLMProviderParams) (sqlc.LlmProvider, error) {
	prov, err := r.q.CreateLLMProvider(ctx, p)
	if err != nil {
		return sqlc.LlmProvider{}, err
	}
	r.invalidateCache(ctx, prov.ID, "")
	return prov, nil
}

func (r *llmRepository) UpdateProvider(ctx context.Context, p sqlc.UpdateLLMProviderParams) (sqlc.LlmProvider, error) {
	prov, err := r.q.UpdateLLMProvider(ctx, p)
	if err != nil {
		return sqlc.LlmProvider{}, err
	}
	r.invalidateCache(ctx, prov.ID, "")
	return prov, nil
}

func (r *llmRepository) SetDefaultProvider(ctx context.Context, id string) error {
	if err := r.q.SetDefaultLLMProvider(ctx, id); err != nil {
		return err
	}
	r.invalidateCache(ctx, id, "")
	return nil
}

func (r *llmRepository) DeleteProvider(ctx context.Context, id string) error {
	if err := r.q.DeleteLLMProvider(ctx, id); err != nil {
		return err
	}
	r.invalidateCache(ctx, id, "")
	return nil
}

func (r *llmRepository) ListModels(ctx context.Context) ([]sqlc.LlmModel, error) {
	cacheKey := constants.CacheKeyLLMModelsAll
	if r.c != nil && !r.inTx {
		var cached []sqlc.LlmModel
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.ListLLMModels(ctx)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]sqlc.LlmModel), nil
}

func (r *llmRepository) ListModelsByProviderID(ctx context.Context, providerID string) ([]sqlc.LlmModel, error) {
	cacheKey := cache.BuildKey("llm_models", "provider", providerID)
	if r.c != nil && !r.inTx {
		var cached []sqlc.LlmModel
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.ListLLMModelsByProviderID(ctx, providerID)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]sqlc.LlmModel), nil
}

func (r *llmRepository) ListActiveModelsByType(ctx context.Context, modelType string) ([]sqlc.LlmModel, error) {
	cacheKey := cache.BuildKey("llm_models", "type", modelType)
	if r.c != nil && !r.inTx {
		var cached []sqlc.LlmModel
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.ListActiveLLMModelsByType(ctx, modelType)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]sqlc.LlmModel), nil
}

func (r *llmRepository) GetModelByID(ctx context.Context, id string) (sqlc.LlmModel, error) {
	cacheKey := cache.BuildKey("llm_model", "id", id)
	if r.c != nil && !r.inTx {
		var m sqlc.LlmModel
		if err := r.c.Get(ctx, cacheKey, &m); err == nil {
			return m, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.GetLLMModelByID(ctx, id)
		if err != nil {
			return sqlc.LlmModel{}, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return sqlc.LlmModel{}, err
	}
	return v.(sqlc.LlmModel), nil
}

func (r *llmRepository) GetModelByKey(ctx context.Context, modelKey string) (sqlc.LlmModel, error) {
	cacheKey := cache.BuildKey("llm_model", "key", modelKey)
	if r.c != nil && !r.inTx {
		var m sqlc.LlmModel
		if err := r.c.Get(ctx, cacheKey, &m); err == nil {
			return m, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.GetLLMModelByKey(ctx, modelKey)
		if err != nil {
			return sqlc.LlmModel{}, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return sqlc.LlmModel{}, err
	}
	return v.(sqlc.LlmModel), nil
}

func (r *llmRepository) GetDefaultModelByType(ctx context.Context, modelType string) (sqlc.LlmModel, error) {
	cacheKey := cache.BuildKey("llm_model", "default", modelType)
	if r.c != nil && !r.inTx {
		var m sqlc.LlmModel
		if err := r.c.Get(ctx, cacheKey, &m); err == nil {
			return m, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.GetDefaultLLMModelByType(ctx, modelType)
		if err != nil {
			return sqlc.LlmModel{}, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return sqlc.LlmModel{}, err
	}
	return v.(sqlc.LlmModel), nil
}

func (r *llmRepository) CreateModel(ctx context.Context, m sqlc.CreateLLMModelParams) (sqlc.LlmModel, error) {
	created, err := r.q.CreateLLMModel(ctx, m)
	if err != nil {
		return sqlc.LlmModel{}, err
	}
	r.invalidateCache(ctx, m.ProviderID, created.ID)
	return created, nil
}

func (r *llmRepository) UpdateModel(ctx context.Context, m sqlc.UpdateLLMModelParams) (sqlc.LlmModel, error) {
	updated, err := r.q.UpdateLLMModel(ctx, m)
	if err != nil {
		return sqlc.LlmModel{}, err
	}
	r.invalidateCache(ctx, m.ProviderID, updated.ID)
	return updated, nil
}

func (r *llmRepository) UpdateModelVisionCapability(ctx context.Context, p sqlc.UpdateModelVisionCapabilityParams) (sqlc.LlmModel, error) {
	updated, err := r.q.UpdateModelVisionCapability(ctx, p)
	if err != nil {
		return sqlc.LlmModel{}, err
	}
	r.invalidateCache(ctx, updated.ProviderID, updated.ID)
	return updated, nil
}

func (r *llmRepository) SetDefaultModel(ctx context.Context, id string, modelType string) error {
	if err := r.q.SetDefaultLLMModel(ctx, sqlc.SetDefaultLLMModelParams{ID: id, ModelType: modelType}); err != nil {
		return err
	}
	r.invalidateCache(ctx, "", id)
	return nil
}

func (r *llmRepository) DeleteModel(ctx context.Context, id string) error {
	if err := r.q.DeleteLLMModel(ctx, id); err != nil {
		return err
	}
	r.invalidateCache(ctx, "", id)
	return nil
}

func (r *llmRepository) GetModelWithProvider(ctx context.Context, modelID string) (sqlc.GetLLMModelWithProviderRow, error) {
	cacheKey := cache.BuildKey("llm_model_with_provider", "id", modelID)
	if r.c != nil && !r.inTx {
		var row sqlc.GetLLMModelWithProviderRow
		if err := r.c.Get(ctx, cacheKey, &row); err == nil {
			return row, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.GetLLMModelWithProvider(ctx, modelID)
		if err != nil {
			return sqlc.GetLLMModelWithProviderRow{}, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return sqlc.GetLLMModelWithProviderRow{}, err
	}
	return v.(sqlc.GetLLMModelWithProviderRow), nil
}

func (r *llmRepository) ListActiveChatModelsWithProvider(ctx context.Context) ([]sqlc.ListActiveChatModelsWithProviderRow, error) {
	cacheKey := constants.CacheKeyLLMChatActive
	if r.c != nil && !r.inTx {
		var cached []sqlc.ListActiveChatModelsWithProviderRow
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.ListActiveChatModelsWithProvider(ctx)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]sqlc.ListActiveChatModelsWithProviderRow), nil
}

func (r *llmRepository) GetChain(ctx context.Context, id string) (sqlc.LlmChain, error) {
	cacheKey := cache.BuildKey("llm_chain", "id", id)
	if r.c != nil && !r.inTx {
		var chain sqlc.LlmChain
		if err := r.c.Get(ctx, cacheKey, &chain); err == nil {
			return chain, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.GetChain(ctx, id)
		if err != nil {
			return sqlc.LlmChain{}, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return sqlc.LlmChain{}, err
	}
	return v.(sqlc.LlmChain), nil
}

func (r *llmRepository) GetActiveChainByType(ctx context.Context, chainType string) (sqlc.LlmChain, error) {
	cacheKey := cache.BuildKey("llm_chain", "type", chainType)
	if r.c != nil && !r.inTx {
		var chain sqlc.LlmChain
		if err := r.c.Get(ctx, cacheKey, &chain); err == nil {
			return chain, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.GetActiveChainByType(ctx, chainType)
		if err != nil {
			return sqlc.LlmChain{}, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return sqlc.LlmChain{}, err
	}
	return v.(sqlc.LlmChain), nil
}

func (r *llmRepository) ListChains(ctx context.Context) ([]sqlc.LlmChain, error) {
	cacheKey := cache.BuildKey("llm_chain", "all")
	if r.c != nil && !r.inTx {
		var chains []sqlc.LlmChain
		if err := r.c.Get(ctx, cacheKey, &chains); err == nil {
			return chains, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.ListChains(ctx)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]sqlc.LlmChain), nil
}

func (r *llmRepository) CreateChain(ctx context.Context, p sqlc.CreateChainParams) (sqlc.LlmChain, error) {
	chain, err := r.q.CreateChain(ctx, p)
	if err == nil {
		r.invalidateChainCache(ctx)
	}
	return chain, err
}

func (r *llmRepository) UpdateChain(ctx context.Context, p sqlc.UpdateChainParams) (sqlc.LlmChain, error) {
	chain, err := r.q.UpdateChain(ctx, p)
	if err == nil {
		r.invalidateChainCache(ctx)
	}
	return chain, err
}

func (r *llmRepository) DeleteChain(ctx context.Context, id string) error {
	err := r.q.DeleteChain(ctx, id)
	if err == nil {
		r.invalidateChainCache(ctx)
	}
	return err
}

func (r *llmRepository) ListChainNodesWithDetails(ctx context.Context, chainID string) ([]sqlc.ListChainNodesWithDetailsRow, error) {
	cacheKey := cache.BuildKey("llm_chain_nodes", "chain", chainID)
	if r.c != nil && !r.inTx {
		var rows []sqlc.ListChainNodesWithDetailsRow
		if err := r.c.Get(ctx, cacheKey, &rows); err == nil {
			return rows, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		res, err := r.q.ListChainNodesWithDetails(ctx, chainID)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, res, constants.NormalCacheDuration)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]sqlc.ListChainNodesWithDetailsRow), nil
}

func (r *llmRepository) AddChainNode(ctx context.Context, p sqlc.AddChainNodeParams) (sqlc.LlmChainNode, error) {
	node, err := r.q.AddChainNode(ctx, p)
	if err == nil {
		r.invalidateChainCache(ctx)
	}
	return node, err
}

func (r *llmRepository) RemoveChainNode(ctx context.Context, id string) error {
	err := r.q.RemoveChainNode(ctx, id)
	if err == nil {
		r.invalidateChainCache(ctx)
	}
	return err
}

func (r *llmRepository) UpdateChainNodePriority(ctx context.Context, p sqlc.UpdateChainNodePriorityParams) error {
	err := r.q.UpdateChainNodePriority(ctx, p)
	if err == nil {
		r.invalidateChainCache(ctx)
	}
	return err
}

func (r *llmRepository) ClearChainNodes(ctx context.Context, chainID string) error {
	err := r.q.ClearChainNodes(ctx, chainID)
	if err == nil {
		r.invalidateChainCache(ctx)
	}
	return err
}

func (r *llmRepository) invalidateChainCache(ctx context.Context) {
	if r.c != nil {
		_ = r.c.DelByPattern(context.Background(), "llm_chain*")
	}
}

func (r *llmRepository) invalidateCache(ctx context.Context, providerID, modelID string) {
	if r.c == nil {
		return
	}
	keys := []string{
		constants.CacheKeyLLMProvidersAll,
		constants.CacheKeyLLMModelsAll,
		constants.CacheKeyLLMChatActive,
		cache.BuildKey("llm_provider", "default"),
	}
	if providerID != "" {
		keys = append(keys, cache.BuildKey("llm_provider", "id", providerID))
		keys = append(keys, cache.BuildKey("llm_models", "provider", providerID))
	}
	if modelID != "" {
		keys = append(keys, cache.BuildKey("llm_model", "id", modelID))
		keys = append(keys, cache.BuildKey("llm_model_with_provider", "id", modelID))
	}
	_ = r.c.Del(ctx, keys...)
	_ = r.c.DelByPattern(ctx, "llm_models*")
	_ = r.c.DelByPattern(ctx, "llm_model*")
	r.invalidateChainCache(ctx)
}

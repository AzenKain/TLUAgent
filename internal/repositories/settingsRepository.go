package repositories

import (
	"context"
	"database/sql"

	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
)

type SettingsRepository interface {
	GetSetupState(ctx context.Context, key string) (string, error)
	ClaimInitialSetup(ctx context.Context) (bool, error)
	UpsertSetupState(ctx context.Context, key string, value string) error
	GetAppSetting(ctx context.Context, key string) (string, error)
	UpsertAppSetting(ctx context.Context, key string, valueJSON string) error
	WithTx(tx *sql.Tx) SettingsRepository
}

type settingsRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

func NewSettingsRepository(db sqlc.DBTX, c cache.Cache) SettingsRepository {
	return &settingsRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *settingsRepository) WithTx(tx *sql.Tx) SettingsRepository {
	return &settingsRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sfg:  &singleflight.Group{},
	}
}

func (r *settingsRepository) GetSetupState(ctx context.Context, key string) (string, error) {
	cacheKey := cache.BuildKey("setup_state", "key", key)
	if r.c != nil && !r.inTx {
		var state string
		if err := r.c.Get(ctx, cacheKey, &state); err == nil {
			return state, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		row, err := r.q.GetSetupState(ctx, key)
		if err != nil {
			return "", err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, row.Value, constants.NormalCacheDuration)
		}
		return row.Value, nil
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func (r *settingsRepository) ClaimInitialSetup(ctx context.Context) (bool, error) {
	updated, err := r.q.ClaimInitialSetup(ctx)
	if err != nil {
		return false, err
	}
	if updated > 0 && r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("setup_state", "key", "completed"))
	}
	return updated > 0, nil
}

func (r *settingsRepository) UpsertSetupState(ctx context.Context, key string, value string) error {
	if err := r.q.UpsertSetupState(ctx, sqlc.UpsertSetupStateParams{Key: key, Value: value}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("setup_state", "key", key))
	}
	return nil
}

func (r *settingsRepository) GetAppSetting(ctx context.Context, key string) (string, error) {
	cacheKey := cache.BuildKey("settings", "key", key)
	if r.c != nil && !r.inTx {
		var val string
		if err := r.c.Get(ctx, cacheKey, &val); err == nil {
			return val, nil
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		row, err := r.q.GetAppSetting(ctx, key)
		if err != nil {
			return "", err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, row.ValueJson, constants.NormalCacheDuration)
		}
		return row.ValueJson, nil
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func (r *settingsRepository) UpsertAppSetting(ctx context.Context, key string, valueJSON string) error {
	if err := r.q.UpsertAppSetting(ctx, sqlc.UpsertAppSettingParams{Key: key, ValueJson: valueJSON}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("settings", "key", key), constants.CacheKeySettingsAll)
	}
	return nil
}

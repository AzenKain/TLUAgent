package repositories

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/jsonx"
)

type RoleRepository interface {
	GetByID(ctx context.Context, id string) (*models.RoleEntity, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.RoleEntity, error)
	GetByName(ctx context.Context, name string) (*models.RoleEntity, error)
	All(ctx context.Context) ([]*models.RoleEntity, error)
	ListPermissions(ctx context.Context) ([]*models.PermissionEntity, error)
	ListRolePermissions(ctx context.Context) ([]*models.RolePermissionEntity, error)
	GetRolePermissions(ctx context.Context, roleID string) ([]*models.RolePermissionEntity, error)
	Create(ctx context.Context, params sqlc.CreateRoleParams) (*models.RoleEntity, error)
	Update(ctx context.Context, params sqlc.UpdateRoleParams) (*models.RoleEntity, error)
	UpdateSystemRoleDescription(ctx context.Context, params sqlc.UpdateSystemRoleDescriptionParams) (*models.RoleEntity, error)
	Delete(ctx context.Context, id string) error
	ReplaceRolePermissions(ctx context.Context, roleID string, permissions []*models.RolePermissionEntity) error
	CreateUserRole(ctx context.Context, userID, roleID string) error
	BulkDeleteRolesFromUser(ctx context.Context, userID string) error
	GetAutoAssignRoleIDs(ctx context.Context) ([]string, error)
	CountActiveAdminUsers(ctx context.Context) (int64, error)
	UpdateRolePositions(ctx context.Context, roleIDs []string) error
	InvalidateRoleCache(ctx context.Context, roleIDs []string)
	WithTx(tx *sql.Tx) RoleRepository
}

type roleRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

func NewRoleRepository(db sqlc.DBTX, c cache.Cache) RoleRepository {
	return &roleRepository{q: sqlc.New(db), c: c, sfg: &singleflight.Group{}}
}

func (r *roleRepository) WithTx(tx *sql.Tx) RoleRepository {
	return &roleRepository{q: r.q.WithTx(tx), c: r.c, inTx: true, sfg: &singleflight.Group{}}
}

func (r *roleRepository) GetByID(ctx context.Context, id string) (*models.RoleEntity, error) {
	key := cache.BuildKey("role", "id", id)
	if r.c != nil && !r.inTx {
		var role models.RoleEntity
		if err := r.c.Get(ctx, key, &role); err == nil {
			return &role, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetRoleByID(ctx, id)
		if err != nil {
			return nil, err
		}
		rolePtr := (&models.RoleEntity{}).FromSqlc(row)
		if perms, err := r.GetRolePermissions(ctx, id); err == nil {
			rolePtr.Permissions = perms
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, rolePtr, constants.NormalCacheDuration)
		}
		return rolePtr, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.RoleEntity), nil
}

func (r *roleRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.RoleEntity, error) {
	if len(ids) == 0 {
		return []*models.RoleEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.BuildKey("role", "id", id)
	}

	roles := make([]*models.RoleEntity, 0, len(ids))
	missingIds := []string{}
	missingKeys := []string{}

	if r.c != nil && !r.inTx {
		cachedBytes := r.c.MGet(ctx, keys...)
		for i, bytes := range cachedBytes {
			if len(bytes) > 0 {
				var role models.RoleEntity
				if err := jsonx.Unmarshal(bytes, &role); err == nil {
					roles = append(roles, &role)
					continue
				}
			}
			missingIds = append(missingIds, ids[i])
			missingKeys = append(missingKeys, keys[i])
		}
	} else {
		missingIds = ids
		missingKeys = keys
	}

	if len(missingIds) > 0 {
		sortedIDs := append([]string(nil), missingIds...)
		sort.Strings(sortedIDs)
		sfgKey := "roles:ids:" + strings.Join(sortedIDs, ",")
		v, err, _ := r.sfg.Do(sfgKey, func() (any, error) {
			rows, err := r.q.GetRolesByIDs(ctx, missingIds)
			if err != nil {
				return nil, err
			}
			missingMap := make(map[string]*models.RoleEntity)
			for _, row := range rows {
				result := (&models.RoleEntity{}).FromSqlc(row)
				if perms, err := r.GetRolePermissions(ctx, result.ID); err == nil {
					result.Permissions = perms
				}
				missingMap[result.ID] = result
			}
			return missingMap, nil
		})
		if err != nil {
			return nil, err
		}
		missingMap := v.(map[string]*models.RoleEntity)

		for _, result := range missingMap {
			roles = append(roles, result)
		}

		if r.c != nil {
			missingToCache := make(map[string]any)
			for _, missingId := range missingIds {
				if role, ok := missingMap[missingId]; ok {
					missingToCache[cache.BuildKey("role", "id", missingId)] = role
				}
			}
			if len(missingToCache) > 0 {
				_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
			}
		}
	}

	roleMap := make(map[string]*models.RoleEntity)
	for _, role := range roles {
		roleMap[role.ID] = role
	}
	ordered := make([]*models.RoleEntity, 0, len(ids))
	for _, id := range ids {
		if role, ok := roleMap[id]; ok {
			ordered = append(ordered, role)
		}
	}
	return ordered, nil
}

func (r *roleRepository) GetByName(ctx context.Context, name string) (*models.RoleEntity, error) {
	key := cache.BuildKey("role", "name", name)
	if r.c != nil && !r.inTx {
		var role models.RoleEntity
		if err := r.c.Get(ctx, key, &role); err == nil {
			return &role, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetRoleByName(ctx, name)
		if err != nil {
			return nil, err
		}
		rolePtr := (&models.RoleEntity{}).FromSqlc(row)
		if perms, err := r.GetRolePermissions(ctx, rolePtr.ID); err == nil {
			rolePtr.Permissions = perms
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, rolePtr, constants.NormalCacheDuration)
		}
		return rolePtr, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.RoleEntity), nil
}

func (r *roleRepository) All(ctx context.Context) ([]*models.RoleEntity, error) {
	key := constants.CacheKeyRoleAll
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return r.GetByIDs(ctx, ids)
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		dbIds, err := r.q.GetRoleIDs(ctx)
		if err != nil {
			return nil, err
		}
		return dbIds, nil
	})
	if err != nil {
		return nil, err
	}
	dbIds := v.([]string)

	if r.c != nil && !r.inTx {
		_ = r.c.Set(ctx, key, dbIds, constants.ListCacheDuration)
	}
	return r.GetByIDs(ctx, dbIds)
}

func (r *roleRepository) ListPermissions(ctx context.Context) ([]*models.PermissionEntity, error) {
	key := constants.CacheKeyPermissionAll
	if r.c != nil && !r.inTx {
		var permissions []*models.PermissionEntity
		if err := r.c.Get(ctx, key, &permissions); err == nil {
			return permissions, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.ListPermissions(ctx)
		if err != nil {
			return nil, err
		}
		out := (&models.PermissionEntities{}).FromSqlc(rows)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, out, constants.ListCacheDuration)
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.PermissionEntity), nil
}

func (r *roleRepository) ListRolePermissions(ctx context.Context) ([]*models.RolePermissionEntity, error) {
	key := constants.CacheKeyRolePermAll
	if r.c != nil && !r.inTx {
		var rolePermissions []*models.RolePermissionEntity
		if err := r.c.Get(ctx, key, &rolePermissions); err == nil {
			return rolePermissions, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.ListRolePermissions(ctx)
		if err != nil {
			return nil, err
		}
		out := (&models.RolePermissionEntities{}).FromSqlc(rows)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, out, constants.ListCacheDuration)
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.RolePermissionEntity), nil
}

func (r *roleRepository) GetRolePermissions(ctx context.Context, roleID string) ([]*models.RolePermissionEntity, error) {
	key := cache.BuildKey("role", "permissions", roleID)
	if r.c != nil && !r.inTx {
		var rolePerms []*models.RolePermissionEntity
		if err := r.c.Get(ctx, key, &rolePerms); err == nil {
			return rolePerms, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.GetRolePermissions(ctx, roleID)
		if err != nil {
			return nil, err
		}
		out := (&models.RolePermissionEntities{}).FromSqlc(rows)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, out, constants.ListCacheDuration)
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]*models.RolePermissionEntity), nil
}

func (r *roleRepository) Create(ctx context.Context, params sqlc.CreateRoleParams) (*models.RoleEntity, error) {
	row, err := r.q.CreateRole(ctx, params)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, constants.CacheKeyRoleAll, constants.CacheKeyRoleAutoAssignIDs)
	}
	return (&models.RoleEntity{}).FromSqlc(row), nil
}

func (r *roleRepository) Update(ctx context.Context, params sqlc.UpdateRoleParams) (*models.RoleEntity, error) {
	old, oldErr := r.q.GetRoleByID(ctx, params.ID)
	row, err := r.q.UpdateRole(ctx, params)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		delKeys := []string{constants.CacheKeyRoleAll, cache.BuildKey("role", "id", params.ID), cache.BuildKey("role", "name", row.Name), constants.CacheKeyRoleAutoAssignIDs}
		if oldErr == nil && old.Name != row.Name {
			delKeys = append(delKeys, cache.BuildKey("role", "name", old.Name))
			_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserAllPattern)
		}
		_ = r.c.Del(ctx, delKeys...)
	}
	return (&models.RoleEntity{}).FromSqlc(row), nil
}

func (r *roleRepository) UpdateSystemRoleDescription(ctx context.Context, params sqlc.UpdateSystemRoleDescriptionParams) (*models.RoleEntity, error) {
	row, err := r.q.UpdateSystemRoleDescription(ctx, params)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, constants.CacheKeyRoleAll, cache.BuildKey("role", "id", params.ID), cache.BuildKey("role", "name", row.Name), constants.CacheKeyRoleAutoAssignIDs)
	}
	return (&models.RoleEntity{}).FromSqlc(row), nil
}

func (r *roleRepository) Delete(ctx context.Context, id string) error {
	old, oldErr := r.q.GetRoleByID(ctx, id)
	if err := r.q.DeleteRole(ctx, id); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, constants.CacheKeyRoleAll, cache.BuildKey("role", "id", id), constants.CacheKeyRoleCountActiveAdminUsers, constants.CacheKeyRoleAutoAssignIDs)
		if oldErr == nil {
			_ = r.c.Del(ctx, cache.BuildKey("role", "name", old.Name))
		} else {
			_ = r.c.DelByPattern(context.Background(), constants.CacheKeyRoleNamePattern)
		}
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserAllPattern)
	}
	return nil
}

func (r *roleRepository) ReplaceRolePermissions(ctx context.Context, roleID string, permissions []*models.RolePermissionEntity) error {
	if err := r.q.DeleteRolePermissions(ctx, roleID); err != nil {
		return err
	}
	for _, permission := range permissions {
		if permission == nil {
			continue
		}
		conditions := permission.ConditionsJSON
		if conditions == "" {
			data, err := jsonx.Marshal(permission.Conditions)
			if err != nil {
				return err
			}
			conditions = string(data)
		}
		effect := permission.Effect
		if effect == "" {
			effect = "allow"
		}
		if err := r.q.UpsertRolePermission(ctx, sqlc.UpsertRolePermissionParams{
			ID:             uuid.NewString(),
			RoleID:         roleID,
			PermissionKey:  permission.PermissionKey,
			Effect:         effect,
			ConditionsJson: conditions,
		}); err != nil {
			return err
		}
	}

	if r.c != nil {
		_ = r.c.Del(ctx, constants.CacheKeyRolePermAll, cache.BuildKey("role", "permissions", roleID), cache.BuildKey("role", "id", roleID))
	}
	return nil
}

func (r *roleRepository) CreateUserRole(ctx context.Context, userID, roleID string) error {
	if err := r.q.CreateUserRole(ctx, sqlc.CreateUserRoleParams{UserID: userID, RoleID: roleID}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("user", "id", userID), cache.BuildKey("user", "roles", userID), constants.CacheKeyRoleCountActiveAdminUsers)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserSearch)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserCount)
	}
	return nil
}

func (r *roleRepository) BulkDeleteRolesFromUser(ctx context.Context, userID string) error {
	if err := r.q.BulkDeleteRolesFromUser(ctx, userID); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("user", "id", userID), cache.BuildKey("user", "roles", userID), constants.CacheKeyRoleCountActiveAdminUsers)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserSearch)
		_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserCount)
	}
	return nil
}

func (r *roleRepository) GetAutoAssignRoleIDs(ctx context.Context) ([]string, error) {
	key := constants.CacheKeyRoleAutoAssignIDs
	if r.c != nil && !r.inTx {
		var ids []string
		if err := r.c.Get(ctx, key, &ids); err == nil {
			return ids, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		ids, err := r.q.GetAutoAssignRoleIDs(ctx)
		if err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, ids, constants.ListCacheDuration)
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}

func (r *roleRepository) CountActiveAdminUsers(ctx context.Context) (int64, error) {
	key := constants.CacheKeyRoleCountActiveAdminUsers
	if r.c != nil && !r.inTx {
		var count int64
		if err := r.c.Get(ctx, key, &count); err == nil {
			return count, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		count, err := r.q.CountActiveAdminUsers(ctx)
		if err != nil {
			return int64(0), err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, count, constants.ListCacheDuration)
		}
		return count, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int64), nil
}

func (r *roleRepository) UpdateRolePositions(ctx context.Context, roleIDs []string) error {
	total := len(roleIDs)
	for i, id := range roleIDs {
		pos := int64((total - i) * 10)
		if err := r.q.UpdateRolePosition(ctx, sqlc.UpdateRolePositionParams{
			Position: pos,
			ID:       id,
		}); err != nil {
			return err
		}
	}
	if r.c != nil && !r.inTx {
		r.InvalidateRoleCache(ctx, roleIDs)
	}
	return nil
}

func (r *roleRepository) InvalidateRoleCache(ctx context.Context, roleIDs []string) {
	if r.c == nil {
		return
	}
	delKeys := make([]string, 0, len(roleIDs)+1)
	delKeys = append(delKeys, constants.CacheKeyRoleAll)
	for _, id := range roleIDs {
		delKeys = append(delKeys, cache.BuildKey("role", "id", id))
	}
	_ = r.c.Del(ctx, delKeys...)
}

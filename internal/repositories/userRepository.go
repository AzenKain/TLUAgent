package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
)

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*models.UserEntity, error)
	GetByIDWithoutDeleted(ctx context.Context, id string) (*models.UserEntity, error)
	GetByEmail(ctx context.Context, email string) (*models.UserEntity, error)
	GetAuthByEmail(ctx context.Context, email string) (*models.UserEntity, error)
	GetAuthByID(ctx context.Context, id string) (*models.UserEntity, error)
	CreateUser(ctx context.Context, params sqlc.CreateUserParams) (*models.UserEntity, error)
	UpsertUser(ctx context.Context, params sqlc.UpsertUserParams) (*models.UserEntity, error)
	UpdateProfile(ctx context.Context, params sqlc.UpdateUserProfileParams) (*models.UserEntity, error)
	UpdatePassword(ctx context.Context, id string, passwordHash string) error
	UpdateRefreshToken(ctx context.Context, id string, refreshToken *string) error
	RotateRefreshToken(ctx context.Context, id string, currentRefreshToken, newRefreshToken string) (bool, error)
	GetTokenVersion(ctx context.Context, id string) (int32, error)
	UpdateTokenVersion(ctx context.Context, id string, tokenVersion int64) error
	RevokeSessions(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
	GetByIDs(ctx context.Context, ids []string) ([]*models.UserEntity, error)
	Search(ctx context.Context, params sqlc.SearchUserIDsParams) ([]*models.UserEntity, error)
	Count(ctx context.Context, params sqlc.CountUsersParams) (int64, error)
	InvalidateUserCache(ctx context.Context, id, email string)
	WithTx(tx *sql.Tx) UserRepository
}

type userRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	inTx bool
	sfg  *singleflight.Group
}

func NewUserRepository(db sqlc.DBTX, c cache.Cache) UserRepository {
	return &userRepository{
		q:   sqlc.New(db),
		c:   c,
		sfg: &singleflight.Group{},
	}
}

func (r *userRepository) WithTx(tx *sql.Tx) UserRepository {
	return &userRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		inTx: true,
		sfg:  &singleflight.Group{},
	}
}

func (r *userRepository) hydrateRoles(ctx context.Context, user *models.UserEntity) error {
	key := cache.BuildKey("user", "roles", user.ID)
	if r.c != nil && !r.inTx {
		var roles []*models.RoleSimple
		if err := r.c.Get(ctx, key, &roles); err == nil {
			user.Roles = roles
			return nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		rows, err := r.q.GetUserRoles(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		permRows, err := r.q.GetUserRolePermissions(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		permsByRole := make(map[string][]*models.RolePermissionEntity)
		for _, pr := range permRows {
			permsByRole[pr.RoleID] = append(permsByRole[pr.RoleID], (&models.RolePermissionEntity{}).FromSqlc(pr))
		}
		userRoles := make([]*models.RoleSimple, 0, len(rows))
		for _, row := range rows {
			userRoles = append(userRoles, &models.RoleSimple{
				ID:          row.ID,
				Name:        row.Name,
				IsAdmin:     row.IsAdmin != 0,
				IsBanned:    row.IsBanned != 0,
				Position:    row.Position,
				Permissions: permsByRole[row.ID],
			})
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, userRoles, constants.NormalCacheDuration)
		}
		return userRoles, nil
	})
	if err != nil {
		return err
	}
	user.Roles = v.([]*models.RoleSimple)
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*models.UserEntity, error) {
	key := cache.BuildKey("user", "id", id)
	if r.c != nil && !r.inTx {
		var user models.UserEntity
		if err := r.c.Get(ctx, key, &user); err == nil {
			return &user, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetUserByID(ctx, id)
		if err != nil {
			return nil, err
		}
		userPtr := (&models.UserEntity{}).FromSqlc(row)
		if err := r.hydrateRoles(ctx, userPtr); err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, userPtr, constants.NormalCacheDuration)
		}
		return userPtr, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.UserEntity), nil
}

func (r *userRepository) GetByIDWithoutDeleted(ctx context.Context, id string) (*models.UserEntity, error) {
	key := cache.BuildKey("user", "id_nodelete", id)
	if r.c != nil && !r.inTx {
		var user models.UserEntity
		if err := r.c.Get(ctx, key, &user); err == nil {
			return &user, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetUserByIDWithoutDeleted(ctx, id)
		if err != nil {
			return nil, err
		}
		user := (&models.UserEntity{}).FromSqlc(row)
		if err := r.hydrateRoles(ctx, user); err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, user, constants.NormalCacheDuration)
		}
		return user, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.UserEntity), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.UserEntity, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	key := cache.BuildKey("user", "email", normalized)
	if r.c != nil && !r.inTx {
		var user models.UserEntity
		if err := r.c.Get(ctx, key, &user); err == nil {
			return &user, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		row, err := r.q.GetUserByEmail(ctx, normalized)
		if err != nil {
			return nil, err
		}
		userPtr := (&models.UserEntity{}).FromSqlc(row)
		if err := r.hydrateRoles(ctx, userPtr); err != nil {
			return nil, err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, userPtr, constants.NormalCacheDuration)
			_ = r.c.Set(ctx, cache.BuildKey("user", "id", userPtr.ID), userPtr, constants.NormalCacheDuration)
		}
		return userPtr, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*models.UserEntity), nil
}

func (r *userRepository) GetAuthByEmail(ctx context.Context, email string) (*models.UserEntity, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	row, err := r.q.GetUserByEmail(ctx, normalized)
	if err != nil {
		return nil, err
	}
	user := (&models.UserEntity{}).FromSqlc(row)
	if err := r.hydrateRoles(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepository) GetAuthByID(ctx context.Context, id string) (*models.UserEntity, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	user := (&models.UserEntity{}).FromSqlc(row)
	if err := r.hydrateRoles(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepository) CreateUser(ctx context.Context, params sqlc.CreateUserParams) (*models.UserEntity, error) {
	params.Email = strings.ToLower(strings.TrimSpace(params.Email))
	row, err := r.q.CreateUser(ctx, params)
	if err != nil {
		return nil, err
	}
	user := (&models.UserEntity{}).FromSqlc(row)
	r.InvalidateUserCache(ctx, user.ID, user.Email)
	return user, nil
}

func (r *userRepository) UpsertUser(ctx context.Context, params sqlc.UpsertUserParams) (*models.UserEntity, error) {
	params.Email = strings.ToLower(strings.TrimSpace(params.Email))
	row, err := r.q.UpsertUser(ctx, params)
	if err != nil {
		return nil, err
	}
	user := (&models.UserEntity{}).FromSqlc(row)
	r.InvalidateUserCache(ctx, user.ID, user.Email)
	return user, nil
}

func (r *userRepository) UpdateProfile(ctx context.Context, params sqlc.UpdateUserProfileParams) (*models.UserEntity, error) {
	row, err := r.q.UpdateUserProfile(ctx, params)
	if err != nil {
		return nil, err
	}
	user := (&models.UserEntity{}).FromSqlc(row)
	if err := r.hydrateRoles(ctx, user); err != nil {
		return nil, err
	}
	r.InvalidateUserCache(ctx, user.ID, user.Email)
	return user, nil
}

func (r *userRepository) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := r.q.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		PasswordHash: sql.NullString{String: passwordHash, Valid: true},
		ID:           id,
	}); err != nil {
		return err
	}
	r.InvalidateUserCache(ctx, id, user.Email)
	return nil
}

func (r *userRepository) UpdateRefreshToken(ctx context.Context, id string, refreshToken *string) error {
	var ns sql.NullString
	if refreshToken != nil && *refreshToken != "" {
		ns = sql.NullString{String: *refreshToken, Valid: true}
	}
	return r.q.UpdateUserRefreshToken(ctx, sqlc.UpdateUserRefreshTokenParams{
		RefreshToken: ns,
		ID:           id,
	})
}

func (r *userRepository) RotateRefreshToken(ctx context.Context, id string, currentRefreshToken, newRefreshToken string) (bool, error) {
	rows, err := r.q.RotateUserRefreshToken(ctx, sqlc.RotateUserRefreshTokenParams{
		NewRefreshToken:     sql.NullString{String: newRefreshToken, Valid: true},
		ID:                  id,
		CurrentRefreshToken: sql.NullString{String: currentRefreshToken, Valid: true},
	})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *userRepository) GetTokenVersion(ctx context.Context, id string) (int32, error) {
	key := cache.BuildKey("user", "token", id)
	if r.c != nil && !r.inTx {
		var version int32
		if err := r.c.Get(ctx, key, &version); err == nil {
			return version, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		ver, err := r.q.GetUserTokenVersion(ctx, id)
		if err != nil {
			return int32(0), err
		}
		int32Ver := int32(ver)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, int32Ver, constants.NormalCacheDuration)
		}
		return int32Ver, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int32), nil
}

func (r *userRepository) UpdateTokenVersion(ctx context.Context, id string, tokenVersion int64) error {
	if err := r.q.UpdateUserTokenVersion(ctx, sqlc.UpdateUserTokenVersionParams{
		TokenVersion: tokenVersion,
		ID:           id,
	}); err != nil {
		return err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("user", "token", id), cache.BuildKey("user", "id", id))
	}
	return nil
}

func (r *userRepository) RevokeSessions(ctx context.Context, id string) error {
	user, err := r.GetByID(ctx, id)
	if err != nil && !apperrorsIsNotFound(err) {
		return err
	}
	if err := r.q.RevokeUserSessions(ctx, id); err != nil {
		return err
	}
	email := ""
	if user != nil {
		email = user.Email
	}
	r.InvalidateUserCache(ctx, id, email)
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	user, _ := r.GetByID(ctx, id)
	if err := r.q.DeleteUser(ctx, id); err != nil {
		return err
	}
	email := ""
	if user != nil {
		email = user.Email
	}
	r.InvalidateUserCache(ctx, id, email)
	return nil
}

func (r *userRepository) Restore(ctx context.Context, id string) error {
	if err := r.q.RestoreUser(ctx, id); err != nil {
		return err
	}
	user, _ := r.GetByIDWithoutDeleted(ctx, id)
	email := ""
	if user != nil {
		email = user.Email
	}
	r.InvalidateUserCache(ctx, id, email)
	return nil
}

func (r *userRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.UserEntity, error) {
	if len(ids) == 0 {
		return []*models.UserEntity{}, nil
	}

	resultMap := make(map[string]*models.UserEntity, len(ids))
	var missingIDs []string

	for _, id := range ids {
		key := cache.BuildKey("user", "id", id)
		if r.c != nil && !r.inTx {
			var u models.UserEntity
			if err := r.c.Get(ctx, key, &u); err == nil {
				resultMap[id] = &u
				continue
			}
		}
		missingIDs = append(missingIDs, id)
	}

	if len(missingIDs) > 0 {
		rows, err := r.q.GetUsersByIDs(ctx, missingIDs)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			user := (&models.UserEntity{}).FromSqlc(row)
			_ = r.hydrateRoles(ctx, user)
			resultMap[user.ID] = user
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, cache.BuildKey("user", "id", user.ID), user, constants.NormalCacheDuration)
			}
		}
	}

	out := make([]*models.UserEntity, 0, len(ids))
	for _, id := range ids {
		if user, ok := resultMap[id]; ok {
			out = append(out, user)
		}
	}
	return out, nil
}

func (r *userRepository) Search(ctx context.Context, params sqlc.SearchUserIDsParams) ([]*models.UserEntity, error) {
	ids, err := r.q.SearchUserIDs(ctx, params)
	if err != nil {
		return nil, err
	}
	return r.GetByIDs(ctx, ids)
}

func (r *userRepository) Count(ctx context.Context, params sqlc.CountUsersParams) (int64, error) {
	key := cache.BuildKey("user_count", "role", fmt.Sprintf("%v", params.RoleID), "search", fmt.Sprintf("%v", params.SearchText), "deleted", fmt.Sprintf("%v", params.IsDeleted))
	if r.c != nil && !r.inTx {
		var cnt int64
		if err := r.c.Get(ctx, key, &cnt); err == nil {
			return cnt, nil
		}
	}

	v, err, _ := r.sfg.Do(key, func() (any, error) {
		cnt, err := r.q.CountUsers(ctx, params)
		if err != nil {
			return int64(0), err
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, key, cnt, constants.ListCacheDuration)
		}
		return cnt, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int64), nil
}

func (r *userRepository) InvalidateUserCache(ctx context.Context, id, email string) {
	if r.c == nil {
		return
	}
	keys := []string{
		cache.BuildKey("user", "id", id),
		cache.BuildKey("user", "id_nodelete", id),
		cache.BuildKey("user", "token", id),
		cache.BuildKey("user", "roles", id),
	}
	if email != "" {
		keys = append(keys, cache.BuildKey("user", "email", strings.ToLower(email)))
	}
	_ = r.c.Del(ctx, keys...)
	_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserSearch)
	_ = r.c.DelByPattern(context.Background(), constants.CacheKeyUserCount)
	_ = r.c.DelByPattern(context.Background(), "user_count*")
}

func apperrorsIsNotFound(err error) bool {
	return err == sql.ErrNoRows
}

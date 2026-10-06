package services

import (
	"context"
	"database/sql"
	"slices"
	"strings"

	"github.com/google/uuid"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/convert"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/database"
)

type UserService interface {
	CreateUser(ctx context.Context, claims *response.JWTClaims, dto *request.CreateUserDto) (*response.UserResponse, error)
	GetUserByID(ctx context.Context, userID string) (*response.UserResponse, error)
	GetUserCurrent(ctx context.Context, userID string) (*response.UserResponse, error)
	UpdateProfile(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.UpdateProfileDto) (*response.UserResponse, error)
	ChangePassword(ctx context.Context, userID string, dto *request.ChangePasswordDto) error
	AdminResetPassword(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.ResetPasswordDto) error
	ChangeRoleUser(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.ChangeRoleDto) (*response.UserResponse, error)
	DeleteUser(ctx context.Context, userID string, claims *response.JWTClaims) error
	RestoreUser(ctx context.Context, userID string, claims *response.JWTClaims) (*response.UserResponse, error)
	RevokeUserSessions(ctx context.Context, userID string, claims *response.JWTClaims) error
	SearchUser(ctx context.Context, dto *request.SearchUserDto) (*response.PaginatedResponse, error)
}

type userService struct {
	userRepo     repositories.UserRepository
	roleRepo     repositories.RoleRepository
	settingsRepo repositories.SettingsRepository
	txManager    database.TxManager
}

func NewUserService(
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
	settingsRepo repositories.SettingsRepository,
	txManager database.TxManager,
) UserService {
	return &userService{
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		settingsRepo: settingsRepo,
		txManager:    txManager,
	}
}

func (u *userService) rootAdminID(ctx context.Context) string {
	if u.settingsRepo == nil {
		return ""
	}
	id, err := u.settingsRepo.GetSetupState(ctx, "root_admin_id")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(id)
}

func (u *userService) markOwner(ctx context.Context, users ...*response.UserResponse) {
	rootID := u.rootAdminID(ctx)
	if rootID == "" {
		return
	}
	for _, user := range users {
		if user != nil && user.ID == rootID {
			user.IsOwner = true
		}
	}
}

func (u *userService) resolveRoles(ctx context.Context, roleIDs []string) ([]*models.RoleEntity, error) {
	if len(roleIDs) == 0 {
		autoIDs, err := u.roleRepo.GetAutoAssignRoleIDs(ctx)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get auto roles")
		}
		if len(autoIDs) > 0 {
			return u.roleRepo.GetByIDs(ctx, autoIDs)
		}
		role, err := u.roleRepo.GetByName(ctx, constants.RoleTypeStudent.String())
		if err != nil {
			return []*models.RoleEntity{}, nil
		}
		return []*models.RoleEntity{role}, nil
	}

	roles, err := u.roleRepo.GetByIDs(ctx, roleIDs)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch roles")
	}
	if len(roles) != len(roleIDs) {
		return nil, apperrors.New(apperrors.ErrBadRequest, "One or more roles were not found")
	}
	return roles, nil
}

func (u *userService) CreateUser(ctx context.Context, claims *response.JWTClaims, dto *request.CreateUserDto) (*response.UserResponse, error) {
	email := strings.ToLower(strings.TrimSpace(dto.Email))
	if email == "" || dto.Password == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Email and password are required")
	}

	existing, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil && !apperrors.IsNotFound(err) {
		return nil, apperrors.New(apperrors.ErrInternalError, "Database error")
	}
	if existing != nil {
		return nil, apperrors.New(apperrors.ErrConflict, "User with this email already exists")
	}

	roles, err := u.resolveRoles(ctx, dto.RoleIDs)
	if err != nil {
		return nil, err
	}

	rootID := u.rootAdminID(ctx)
	isRoot := claims != nil && rootID != "" && claims.UId == rootID

	hasAdmin := false
	hasBanned := false
	for _, role := range roles {
		if role.IsAdmin || role.Name == constants.RoleTypeAdmin.String() {
			hasAdmin = true
		}
		if role.IsBanned || role.Name == constants.RoleTypeBanned.String() {
			hasBanned = true
		}
	}

	if hasAdmin && !isRoot {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only the owner can grant the ADMIN role")
	}
	if hasBanned {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Cannot create a user with the BANNED role directly")
	}

	hashedPassword, err := crypto.HashPassword(dto.Password)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to hash password")
	}

	tx, err := u.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()

	txUserRepo := u.userRepo.WithTx(tx)
	txRoleRepo := u.roleRepo.WithTx(tx)

	userID := uuid.NewString()
	_, err = txUserRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           userID,
		Email:        email,
		FullName:     sql.NullString{String: strings.TrimSpace(dto.FullName), Valid: strings.TrimSpace(dto.FullName) != ""},
		StudentCode:  sql.NullString{String: strings.TrimSpace(dto.StudentCode), Valid: strings.TrimSpace(dto.StudentCode) != ""},
		PasswordHash: sql.NullString{String: hashedPassword, Valid: true},
		AuthProvider: "LOCAL",
	})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Failed to create user account")
	}

	for _, role := range roles {
		if err := txRoleRepo.CreateUserRole(ctx, userID, role.ID); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to assign role to user")
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit user creation")
	}

	freshUser, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load created user")
	}

	res := freshUser.ToResponse()
	u.markOwner(ctx, res)
	return res, nil
}

func (u *userService) GetUserByID(ctx context.Context, userID string) (*response.UserResponse, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil && !apperrors.IsNotFound(err) {
		return nil, apperrors.New(apperrors.ErrInternalError, "Database error")
	}
	if user == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "User not found")
	}
	res := user.ToResponse()
	u.markOwner(ctx, res)
	return res, nil
}

func (u *userService) GetUserCurrent(ctx context.Context, userID string) (*response.UserResponse, error) {
	return u.GetUserByID(ctx, userID)
}

func (u *userService) UpdateProfile(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.UpdateProfileDto) (*response.UserResponse, error) {
	rootID := u.rootAdminID(ctx)
	isRoot := rootID != "" && claims != nil && claims.UId == rootID

	if rootID != "" && userID == rootID && !isRoot {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only the owner can modify the owner account")
	}

	userObj, err := u.userRepo.GetByID(ctx, userID)
	if err != nil || userObj == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	if userObj.IsAdmin() && !isRoot && (claims == nil || userID != claims.UId) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only the owner can modify other admin accounts")
	}

	params := sqlc.UpdateUserProfileParams{
		ID:          userID,
		FullName:    convert.StrPtrToNullString(dto.FullName),
		StudentCode: convert.StrPtrToNullString(dto.StudentCode),
		AvatarUrl:   convert.StrPtrToNullString(dto.AvatarUrl),
	}

	updated, err := u.userRepo.UpdateProfile(ctx, params)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to update user profile")
	}

	res := updated.ToResponse()
	u.markOwner(ctx, res)
	return res, nil
}

func (u *userService) ChangePassword(ctx context.Context, userID string, dto *request.ChangePasswordDto) error {
	user, err := u.userRepo.GetAuthByID(ctx, userID)
	if err != nil || user == nil {
		return apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	if !crypto.CheckPasswordHash(dto.OldPassword, user.PasswordHash) {
		return apperrors.New(apperrors.ErrUnauthorized, "Invalid old password")
	}

	hashedPassword, err := crypto.HashPassword(dto.NewPassword)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to hash password")
	}

	tx, err := u.txManager.BeginTx(ctx, nil)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()

	userRepoTx := u.userRepo.WithTx(tx)
	if err := userRepoTx.UpdatePassword(ctx, userID, hashedPassword); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to update password")
	}
	if err := userRepoTx.UpdateTokenVersion(ctx, userID, int64(user.TokenVersion+1)); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to revoke sessions")
	}
	if err := tx.Commit(); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to commit password change")
	}

	u.userRepo.InvalidateUserCache(ctx, userID, user.Email)
	return nil
}

func (u *userService) AdminResetPassword(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.ResetPasswordDto) error {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	rootID := u.rootAdminID(ctx)
	isRoot := rootID != "" && claims != nil && claims.UId == rootID

	if rootID != "" && userID == rootID && !isRoot {
		return apperrors.New(apperrors.ErrForbidden, "Only the owner can modify the owner account")
	}

	if user.IsAdmin() && !isRoot && (claims == nil || userID != claims.UId) {
		return apperrors.New(apperrors.ErrForbidden, "Only the owner can modify other admin accounts")
	}

	hashedPassword, err := crypto.HashPassword(dto.NewPassword)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to hash password")
	}

	tx, err := u.txManager.BeginTx(ctx, nil)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()

	userRepoTx := u.userRepo.WithTx(tx)
	if err := userRepoTx.UpdatePassword(ctx, userID, hashedPassword); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to update password")
	}
	if err := userRepoTx.UpdateTokenVersion(ctx, userID, int64(user.TokenVersion+1)); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to revoke sessions")
	}
	if err := tx.Commit(); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to commit password reset")
	}

	u.userRepo.InvalidateUserCache(ctx, userID, user.Email)
	return nil
}

func (u *userService) ChangeRoleUser(ctx context.Context, userID string, claims *response.JWTClaims, dto *request.ChangeRoleDto) (*response.UserResponse, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	rootID := u.rootAdminID(ctx)
	isRoot := rootID != "" && claims != nil && claims.UId == rootID

	if rootID != "" && userID == rootID && !isRoot {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only the owner can modify the owner account")
	}

	if user.IsAdmin() && !isRoot && (claims == nil || userID != claims.UId) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only the owner can modify other admin accounts")
	}

	roles, err := u.resolveRoles(ctx, dto.Roles)
	if err != nil {
		return nil, err
	}

	hasAdmin := false
	hasBanned := false
	for _, role := range roles {
		if role.IsAdmin || role.Name == constants.RoleTypeAdmin.String() {
			hasAdmin = true
		}
		if role.IsBanned || role.Name == constants.RoleTypeBanned.String() {
			hasBanned = true
		}
	}

	if hasAdmin && !isRoot {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only the owner can grant the ADMIN role")
	}

	isSelf := claims != nil && userID == claims.UId
	if isSelf && hasBanned {
		return nil, apperrors.New(apperrors.ErrForbidden, "You cannot ban yourself")
	}
	if claims != nil && slices.Contains(claims.Roles, constants.RoleTypeAdmin) && isSelf && !hasAdmin {
		return nil, apperrors.New(apperrors.ErrForbidden, "You cannot remove your own ADMIN role")
	}

	tx, err := u.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()

	userRepoTx := u.userRepo.WithTx(tx)
	roleRepoTx := u.roleRepo.WithTx(tx)

	if err := roleRepoTx.BulkDeleteRolesFromUser(ctx, userID); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to clear old roles")
	}
	for _, role := range roles {
		if err := roleRepoTx.CreateUserRole(ctx, userID, role.ID); err != nil {
			return nil, apperrors.New(apperrors.ErrInternalError, "Failed to assign roles")
		}
	}
	if err := userRepoTx.UpdateTokenVersion(ctx, userID, int64(user.TokenVersion+1)); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to revoke sessions")
	}

	if err := tx.Commit(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit role change")
	}

	u.userRepo.InvalidateUserCache(ctx, userID, user.Email)

	freshUser, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch updated user")
	}

	res := freshUser.ToResponse()
	u.markOwner(ctx, res)
	return res, nil
}

func (u *userService) DeleteUser(ctx context.Context, userID string, claims *response.JWTClaims) error {
	if claims != nil && claims.UId == userID {
		return apperrors.New(apperrors.ErrForbidden, "You cannot delete yourself")
	}

	rootID := u.rootAdminID(ctx)
	if rootID != "" && userID == rootID {
		return apperrors.New(apperrors.ErrForbidden, "The owner account cannot be deleted")
	}

	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	isRoot := rootID != "" && claims != nil && claims.UId == rootID
	if user.IsAdmin() && !isRoot {
		return apperrors.New(apperrors.ErrForbidden, "Only the owner can delete other admin accounts")
	}

	tx, err := u.txManager.BeginTx(ctx, nil)
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()

	userRepoTx := u.userRepo.WithTx(tx)
	if err := userRepoTx.UpdateTokenVersion(ctx, userID, int64(user.TokenVersion+1)); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to revoke sessions")
	}
	if err := userRepoTx.UpdateRefreshToken(ctx, userID, nil); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to clear refresh token")
	}
	if err := userRepoTx.Delete(ctx, userID); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to delete user")
	}

	if err := tx.Commit(); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to commit user deletion")
	}

	u.userRepo.InvalidateUserCache(ctx, userID, user.Email)
	return nil
}

func (u *userService) RestoreUser(ctx context.Context, userID string, claims *response.JWTClaims) (*response.UserResponse, error) {
	user, err := u.userRepo.GetByIDWithoutDeleted(ctx, userID)
	if err != nil && !apperrors.IsNotFound(err) {
		return nil, apperrors.New(apperrors.ErrInternalError, "Internal Server Error")
	}
	if user == nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	rootID := u.rootAdminID(ctx)
	isRoot := rootID != "" && claims != nil && claims.UId == rootID
	if user.IsAdmin() && !isRoot {
		return nil, apperrors.New(apperrors.ErrForbidden, "Only the owner can restore other admin accounts")
	}

	tx, err := u.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()

	userRepoTx := u.userRepo.WithTx(tx)
	if err := userRepoTx.Restore(ctx, userID); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to restore user")
	}
	if err := userRepoTx.UpdateTokenVersion(ctx, userID, int64(user.TokenVersion+1)); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to revoke sessions")
	}
	if err := userRepoTx.UpdateRefreshToken(ctx, userID, nil); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to clear refresh token")
	}
	if err := tx.Commit(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit restore")
	}

	u.userRepo.InvalidateUserCache(ctx, userID, user.Email)

	restored, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to fetch restored user")
	}

	res := restored.ToResponse()
	u.markOwner(ctx, res)
	return res, nil
}

func (u *userService) RevokeUserSessions(ctx context.Context, userID string, claims *response.JWTClaims) error {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return apperrors.New(apperrors.ErrNotFound, "User not found")
	}

	rootID := u.rootAdminID(ctx)
	isRoot := rootID != "" && claims != nil && claims.UId == rootID

	if rootID != "" && userID == rootID && !isRoot {
		return apperrors.New(apperrors.ErrForbidden, "Only the owner can revoke sessions of the owner account")
	}

	if user.IsAdmin() && !isRoot && (claims == nil || userID != claims.UId) {
		return apperrors.New(apperrors.ErrForbidden, "Only the owner can revoke sessions of other admin accounts")
	}

	if err := u.userRepo.RevokeSessions(ctx, userID); err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to revoke user sessions")
	}
	u.userRepo.InvalidateUserCache(ctx, userID, user.Email)
	return nil
}

func (u *userService) SearchUser(ctx context.Context, dto *request.SearchUserDto) (*response.PaginatedResponse, error) {
	page := dto.Page
	if page < 1 {
		page = 1
	}
	limit := dto.Limit
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit
	if dto.Offset > 0 {
		offset = dto.Offset
	}

	var isDeleted any
	if dto.IsDeleted != nil {
		if *dto.IsDeleted {
			isDeleted = int64(1)
		} else {
			isDeleted = int64(0)
		}
	}
	var roleID any
	if strings.TrimSpace(dto.RoleID) != "" {
		roleID = strings.TrimSpace(dto.RoleID)
	}
	var searchText any
	if strings.TrimSpace(dto.Search) != "" {
		searchText = strings.TrimSpace(dto.Search)
	}

	total, err := u.userRepo.Count(ctx, sqlc.CountUsersParams{
		IsDeleted:  isDeleted,
		RoleID:     roleID,
		SearchText: searchText,
	})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to count users")
	}

	users, err := u.userRepo.Search(ctx, sqlc.SearchUserIDsParams{
		IsDeleted:  isDeleted,
		RoleID:     roleID,
		SearchText: searchText,
		Offset:     int64(offset),
		Limit:      int64(limit),
	})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to search users")
	}

	resUsers := make([]*response.UserResponse, 0, len(users))
	for _, uEntity := range users {
		resUsers = append(resUsers, uEntity.ToResponse())
	}
	u.markOwner(ctx, resUsers...)

	return response.BuildPaginatedResponse(resUsers, total, page, limit), nil
}

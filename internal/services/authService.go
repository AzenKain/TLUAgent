package services

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/config"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/database"
)

type AuthService interface {
	Signin(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, error)
	SubmitSetup(ctx context.Context, dto *request.SetupDto) (*response.AuthResponse, error)
	SetupRequired(ctx context.Context) bool
	RefreshToken(ctx context.Context, userID string, refreshToken string) (*response.AuthResponse, error)
	Logout(ctx context.Context, userID string) error
	GetMe(ctx context.Context, userID string) (*response.UserResponse, error)
	GenToken(user *models.UserEntity) (*response.AuthResponse, error)
	ValidateCredentials(ctx context.Context, dto *request.SignInDto) (*models.UserEntity, error)
}

type authService struct {
	userRepo     repositories.UserRepository
	roleRepo     repositories.RoleRepository
	settingsRepo repositories.SettingsRepository
	txManager    database.TxManager
	dummyHash    string
}

func NewAuthService(userRepo repositories.UserRepository, roleRepo repositories.RoleRepository, settingsRepo repositories.SettingsRepository, txManager database.TxManager) AuthService {
	dummySecret, err := crypto.GenerateRandomHex(18)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to generate login timing-equalization secret")
	}
	dummyHash, err := crypto.HashPassword(dummySecret)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to hash login timing-equalization secret")
	}
	return &authService{
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		settingsRepo: settingsRepo,
		txManager:    txManager,
		dummyHash:    dummyHash,
	}
}

func refreshTokenDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func parseRefreshTokens(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	tokens := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			tokens = append(tokens, p)
		}
	}
	return tokens
}

func findMatchingRefreshToken(storedTokens []string, token string) int {
	actual := token
	tokenDig := refreshTokenDigest(token)
	for i, stored := range storedTokens {
		stored = strings.TrimSpace(stored)
		expected := tokenDig
		if len(stored) < len("sha256:") || stored[:len("sha256:")] != "sha256:" {
			expected = actual
		}
		if subtle.ConstantTimeCompare([]byte(stored), []byte(expected)) == 1 {
			return i
		}
	}
	return -1
}

const maxActiveRefreshTokens = 10

func addRefreshToken(rawStored string, newDigest string) string {
	tokens := parseRefreshTokens(rawStored)
	if len(tokens) >= maxActiveRefreshTokens {
		tokens = tokens[len(tokens)-maxActiveRefreshTokens+1:]
	}
	tokens = append(tokens, newDigest)
	return strings.Join(tokens, ",")
}

func tokenClaims(user *models.UserEntity, tokenType string, duration time.Duration) *response.JWTClaims {
	now := time.Now()
	uid := user.ID
	return &response.JWTClaims{
		UId:          uid,
		Roles:        models.RolesEntityToRoleConstant(user.Roles),
		RoleIDs:      models.RolesEntityToRoleIDs(user.Roles),
		TokenVersion: user.TokenVersion,
		TokenType:    tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "tluagent",
			Subject:   uid,
			Audience:  jwt.ClaimStrings{"tluagent-" + tokenType},
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}
}

func (a *authService) GenToken(user *models.UserEntity) (*response.AuthResponse, error) {
	jwtSecret := config.GetJWTSecret()
	jwtRefreshSecret := config.GetJWTRefreshSecret()

	claimsAccess := tokenClaims(user, "access", constants.AccessTokenDuration)
	claimsRefresh := tokenClaims(user, "refresh", constants.RefreshTokenDuration)

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsAccess)
	access, err := accessToken.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to sign access token")
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsRefresh)
	refresh, err := refreshToken.SignedString([]byte(jwtRefreshSecret))
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to sign refresh token")
	}

	// Persist hashed refresh token
	digest := refreshTokenDigest(refresh)
	updatedStore := addRefreshToken(user.RefreshToken, digest)
	if err := a.userRepo.UpdateRefreshToken(context.Background(), user.ID, &updatedStore); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to persist refresh token")
	}
	user.RefreshToken = updatedStore

	return &response.AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         user.ToResponse(),
	}, nil
}

func (a *authService) ValidateCredentials(ctx context.Context, dto *request.SignInDto) (*models.UserEntity, error) {
	email := strings.ToLower(strings.TrimSpace(dto.Email))
	if email == "" || dto.Password == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Email and password are required")
	}

	user, err := a.userRepo.GetAuthByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = crypto.CheckPasswordHash(dto.Password, a.dummyHash)
			return nil, apperrors.New(apperrors.ErrUnauthorized, "Invalid email or password")
		}
		return nil, apperrors.New(apperrors.ErrInternalError, "Database error")
	}

	if !crypto.CheckPasswordHash(dto.Password, user.PasswordHash) {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Invalid email or password")
	}

	if slices.Contains(models.RolesEntityToRoleConstant(user.Roles), constants.RoleTypeBanned) {
		return nil, apperrors.New(apperrors.ErrForbidden, "User account is banned")
	}

	return user, nil
}

func (a *authService) Signin(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, error) {
	user, err := a.ValidateCredentials(ctx, dto)
	if err != nil {
		return nil, err
	}
	return a.GenToken(user)
}

func (a *authService) SetupRequired(ctx context.Context) bool {
	completed, err := a.settingsRepo.GetSetupState(ctx, "completed")
	if err != nil && !apperrors.IsNotFound(err) {
		return false
	}
	return completed != "true"
}

func (a *authService) SubmitSetup(ctx context.Context, dto *request.SetupDto) (*response.AuthResponse, error) {
	if !a.SetupRequired(ctx) {
		return nil, apperrors.New(apperrors.ErrForbidden, "Setup has already been completed")
	}

	email := strings.ToLower(strings.TrimSpace(dto.Email))
	if email == "" || dto.Password == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "Email and password are required")
	}

	tx, err := a.txManager.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()

	settingsRepoTx := a.settingsRepo.WithTx(tx)
	userRepoTx := a.userRepo.WithTx(tx)
	roleRepoTx := a.roleRepo.WithTx(tx)

	claimed, err := settingsRepoTx.ClaimInitialSetup(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to claim initial setup")
	}
	if !claimed {
		return nil, apperrors.New(apperrors.ErrForbidden, "Setup has already been completed or is in progress")
	}

	hashedPassword, err := crypto.HashPassword(dto.Password)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to hash password")
	}

	userID := uuid.NewString()
	_, err = userRepoTx.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           userID,
		Email:        email,
		FullName:     sql.NullString{String: strings.TrimSpace(dto.FullName), Valid: strings.TrimSpace(dto.FullName) != ""},
		StudentCode:  sql.NullString{String: strings.TrimSpace(dto.StudentCode), Valid: strings.TrimSpace(dto.StudentCode) != ""},
		PasswordHash: sql.NullString{String: hashedPassword, Valid: true},
		AuthProvider: "LOCAL",
	})
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to create root admin user")
	}

	adminRole, err := roleRepoTx.GetByName(ctx, constants.RoleTypeAdmin.String())
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to get admin role")
	}
	if err := roleRepoTx.CreateUserRole(ctx, userID, adminRole.ID); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to assign admin role")
	}

	if err := settingsRepoTx.UpsertSetupState(ctx, "completed", "true"); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to save setup state")
	}
	if err := settingsRepoTx.UpsertSetupState(ctx, "root_admin_id", userID); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to save root admin id")
	}

	if err := tx.Commit(); err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to commit setup")
	}

	freshUser, err := a.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternalError, "Failed to load root admin user")
	}

	return a.GenToken(freshUser)
}

func (a *authService) RefreshToken(ctx context.Context, userID string, refreshToken string) (*response.AuthResponse, error) {
	if userID == "" || refreshToken == "" {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Missing user ID or refresh token")
	}

	user, err := a.userRepo.GetAuthByID(ctx, userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "User not found")
	}

	storedTokens := parseRefreshTokens(user.RefreshToken)
	idx := findMatchingRefreshToken(storedTokens, refreshToken)
	if idx < 0 {
		// Token rotation reuse detection: revoke all sessions
		_ = a.userRepo.UpdateTokenVersion(ctx, userID, int64(user.TokenVersion+1))
		_ = a.userRepo.UpdateRefreshToken(ctx, userID, nil)
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Invalid refresh token")
	}

	// Rotate token
	storedTokens = append(storedTokens[:idx], storedTokens[idx+1:]...)
	cleanedStored := strings.Join(storedTokens, ",")
	_ = a.userRepo.UpdateRefreshToken(ctx, userID, &cleanedStored)
	user.RefreshToken = cleanedStored

	return a.GenToken(user)
}

func (a *authService) Logout(ctx context.Context, userID string) error {
	user, err := a.userRepo.GetAuthByID(ctx, userID)
	if err == nil && user != nil {
		_ = a.userRepo.UpdateTokenVersion(ctx, userID, int64(user.TokenVersion+1))
	}
	_ = a.userRepo.UpdateRefreshToken(ctx, userID, nil)
	return nil
}

func (a *authService) GetMe(ctx context.Context, userID string) (*response.UserResponse, error) {
	user, err := a.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "User not found")
	}
	return user.ToResponse(), nil
}

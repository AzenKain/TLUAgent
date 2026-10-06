package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/config"
	"tluagent-web/pkg/constants"
)

type contextKey string

const (
	UserClaimsContextKey         contextKey = "user_claims"
	UserIDContextKey             contextKey = "user_id"
	RefreshTokenStringContextKey contextKey = "refresh_token_string"
)

func GetUserClaims(ctx context.Context) *response.JWTClaims {
	if val := ctx.Value(UserClaimsContextKey); val != nil {
		if claims, ok := val.(*response.JWTClaims); ok {
			return claims
		}
	}
	return nil
}

func GetUserID(ctx context.Context) string {
	if val := ctx.Value(UserIDContextKey); val != nil {
		if uid, ok := val.(string); ok {
			return uid
		}
	}
	return ""
}

func GetRawRefreshToken(ctx context.Context) string {
	if val := ctx.Value(RefreshTokenStringContextKey); val != nil {
		if token, ok := val.(string); ok {
			return token
		}
	}
	return ""
}

func GuestClaims() *response.JWTClaims {
	return &response.JWTClaims{
		UId:     "0",
		Roles:   []constants.RoleType{constants.RoleTypeGuest},
		RoleIDs: []string{},
	}
}

func extractAccessToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(authHeader, "Bearer "); ok && after != "" {
		return strings.TrimSpace(after)
	}
	if cookie, err := r.Cookie("access_token"); err == nil && cookie.Value != "" {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

func extractRefreshToken(r *http.Request) string {
	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		return strings.TrimSpace(cookie.Value)
	}
	authHeader := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(authHeader, "Bearer "); ok && after != "" {
		return strings.TrimSpace(after)
	}
	return ""
}

func parseAndValidateToken(tokenString string, secret string, expectedType string, userRepo repositories.UserRepository, ctx context.Context) (*response.JWTClaims, error) {
	if tokenString == "" {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Missing token")
	}

	token, err := jwt.ParseWithClaims(tokenString, &response.JWTClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Invalid or expired JWT")
	}

	claims, ok := token.Claims.(*response.JWTClaims)
	if !ok || claims.TokenType != expectedType || claims.Issuer != "tluagent" || claims.Subject != claims.UId || claims.Subject == "" {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Invalid token claims")
	}

	expectedAudience := "tluagent-" + expectedType
	if !slices.Contains([]string(claims.Audience), expectedAudience) {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Invalid token audience")
	}

	if slices.Contains(claims.Roles, constants.RoleTypeBanned) {
		return nil, apperrors.New(apperrors.ErrForbidden, "User account is banned")
	}

	tokenVersion, err := userRepo.GetTokenVersion(ctx, claims.UId)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "User not found or database error")
	}
	if tokenVersion != claims.TokenVersion {
		return nil, apperrors.New(apperrors.ErrUnauthorized, "Token has been invalidated")
	}

	return claims, nil
}

// JWTAccess verifies short-lived AccessTokens
func JWTAccess(userRepo repositories.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractAccessToken(r)
			jwtSecret := config.GetJWTSecret()

			claims, err := parseAndValidateToken(tokenString, jwtSecret, "access", userRepo, r.Context())
			if err != nil {
				apperrors.HandleError(w, err)
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsContextKey, claims)
			ctx = context.WithValue(ctx, UserIDContextKey, claims.UId)
			ctx = services.WithPermissionContext(ctx, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalJWTAccess falls back to GuestClaims if no token is present, but rejects invalid tokens with 401
func OptionalJWTAccess(userRepo repositories.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractAccessToken(r)
			if tokenString == "" {
				guest := GuestClaims()
				ctx := context.WithValue(r.Context(), UserClaimsContextKey, guest)
				ctx = context.WithValue(ctx, UserIDContextKey, guest.UId)
				ctx = services.WithPermissionContext(ctx, guest)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			jwtSecret := config.GetJWTSecret()
			claims, err := parseAndValidateToken(tokenString, jwtSecret, "access", userRepo, r.Context())
			if err != nil {
				apperrors.HandleError(w, err)
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsContextKey, claims)
			ctx = context.WithValue(ctx, UserIDContextKey, claims.UId)
			ctx = services.WithPermissionContext(ctx, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// JWTRefresh verifies long-lived RefreshTokens using JWT_REFRESH_SECRET
func JWTRefresh(userRepo repositories.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractRefreshToken(r)
			jwtRefreshSecret := config.GetJWTRefreshSecret()

			claims, err := parseAndValidateToken(tokenString, jwtRefreshSecret, "refresh", userRepo, r.Context())
			if err != nil {
				apperrors.HandleError(w, err)
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsContextKey, claims)
			ctx = context.WithValue(ctx, UserIDContextKey, claims.UId)
			ctx = context.WithValue(ctx, RefreshTokenStringContextKey, tokenString)
			ctx = services.WithPermissionContext(ctx, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}


package middlewares

import (
	"net/http"
	"slices"

	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/constants"
)

func RequireRole(roles ...constants.RoleType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserClaims(r.Context())
			if claims == nil {
				apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
				return
			}
			for _, required := range roles {
				if slices.Contains(claims.Roles, required) {
					next.ServeHTTP(w, r)
					return
				}
			}
			apperrors.HandleError(w, apperrors.New(apperrors.ErrForbidden, "Permission denied"))
		})
	}
}

func RequirePermission(permissionCache services.PermissionCache, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if permissionCache == nil {
				apperrors.HandleError(w, apperrors.New(apperrors.ErrForbidden, "Permission check unavailable"))
				return
			}
			claims := GetUserClaims(r.Context())
			if claims == nil {
				apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
				return
			}
			if !permissionCache.Can(r.Context(), claims.UId, permission, nil) {
				apperrors.HandleError(w, apperrors.New(apperrors.ErrForbidden, "Permission denied: requires "+permission))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAnyPermission(permissionCache services.PermissionCache, permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if permissionCache == nil {
				apperrors.HandleError(w, apperrors.New(apperrors.ErrForbidden, "Permission check unavailable"))
				return
			}
			claims := GetUserClaims(r.Context())
			if claims == nil {
				apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
				return
			}
			for _, permission := range permissions {
				if permissionCache.Can(r.Context(), claims.UId, permission, nil) {
					next.ServeHTTP(w, r)
					return
				}
			}
			apperrors.HandleError(w, apperrors.New(apperrors.ErrForbidden, "Permission denied"))
		})
	}
}

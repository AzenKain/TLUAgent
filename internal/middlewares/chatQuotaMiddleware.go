package middlewares

import (
	"net/http"

	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
)

const guestIdentityPrefix = "guest:"

// ChatQuota rejects requests once the caller's daily chat allowance is exhausted.
func ChatQuota(quota services.ChatQuotaService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if quota != nil {
				if err := quota.Allow(r.Context(), chatQuotaIdentity(r)); err != nil {
					apperrors.HandleError(w, err)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func chatQuotaIdentity(r *http.Request) string {
	if uid := GetUserID(r.Context()); uid != "" && uid != "0" {
		return "user:" + uid
	}
	return guestIdentityPrefix + ClientIP(r)
}

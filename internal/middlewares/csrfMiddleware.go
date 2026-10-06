package middlewares

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"tluagent-web/pkg/apperrors"
)

func CSRFProtection() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			method := r.Method
			if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// If request uses Bearer token in header, CSRF via ambient cookie isn't applicable
			if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				next.ServeHTTP(w, r)
				return
			}

			// If no cookie authentication is used, skip CSRF
			cookieToken, err := r.Cookie("access_token")
			if err != nil || cookieToken.Value == "" {
				next.ServeHTTP(w, r)
				return
			}

			csrfCookie, err := r.Cookie("csrf_token")
			headerToken := r.Header.Get("X-CSRF-Token")

			if err != nil || csrfCookie.Value == "" || headerToken == "" || subtle.ConstantTimeCompare([]byte(csrfCookie.Value), []byte(headerToken)) != 1 {
				apperrors.HandleError(w, apperrors.New(apperrors.ErrForbidden, "CSRF token mismatch or missing"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

package middlewares

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"tluagent-web/pkg/apperrors"
)

// CSRFProtection enforces double-submit cookie CSRF validation for mutating requests using cookie authentication.
func CSRFProtection() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			method := r.Method
			if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				next.ServeHTTP(w, r)
				return
			}

			accessCookie, errAccess := r.Cookie("access_token")
			refreshCookie, errRefresh := r.Cookie("refresh_token")
			hasCookieAuth := (errAccess == nil && accessCookie.Value != "") || (errRefresh == nil && refreshCookie.Value != "")
			if !hasCookieAuth {
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

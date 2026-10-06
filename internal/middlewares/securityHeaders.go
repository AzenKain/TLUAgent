package middlewares

import (
	"net/http"

	"tluagent-web/pkg/config"
)

const contentSecurityPolicy = "default-src 'self'; font-src 'self' data:; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'"

const strictTransportSecurity = "max-age=31536000; includeSubDomains"

// SecurityHeaders adds defensive HTTP response headers to every response.
func SecurityHeaders() func(http.Handler) http.Handler {
	secureMode := config.CookieSecureMode()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := w.Header()
			header.Set("X-Content-Type-Options", "nosniff")
			header.Set("X-Frame-Options", "DENY")
			header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			header.Set("Content-Security-Policy", contentSecurityPolicy)
			if secureMode {
				header.Set("Strict-Transport-Security", strictTransportSecurity)
			}

			next.ServeHTTP(w, r)
		})
	}
}

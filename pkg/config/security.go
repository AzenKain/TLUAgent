package config

import "strings"

// CookieSecureMode reports whether auth cookies and HSTS should require HTTPS delivery.
func CookieSecureMode() bool {
	return GetBoolConfigWithDefault("COOKIE_SECURE", GetConfigWithDefault("APP_ENV", "development") == "production")
}

// GetTrustedProxyHops returns the number of trusted reverse-proxy hops in front of the server.
func GetTrustedProxyHops() int {
	return GetIntConfigWithDefault("TRUSTED_PROXY_HOPS", 0)
}

// GetCORSAllowedOrigins returns the parsed allowlist of CORS origins.
func GetCORSAllowedOrigins() []string {
	raw := GetConfigWithDefault("CORS_ALLOWED_ORIGINS", "")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

package config

import "testing"

func TestCookieSecureMode(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SECURE", "")
	if !CookieSecureMode() {
		t.Fatalf("Expected secure mode default true in production")
	}

	t.Setenv("APP_ENV", "development")
	if CookieSecureMode() {
		t.Fatalf("Expected secure mode default false outside production")
	}

	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SECURE", "false")
	if CookieSecureMode() {
		t.Fatalf("Expected COOKIE_SECURE=false to override production default")
	}

	t.Setenv("COOKIE_SECURE", "true")
	if !CookieSecureMode() {
		t.Fatalf("Expected COOKIE_SECURE=true to force secure mode")
	}
}

func TestGetTrustedProxyHops(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_HOPS", "")
	if got := GetTrustedProxyHops(); got != 0 {
		t.Fatalf("Expected default 0 trusted proxy hops, got %d", got)
	}

	t.Setenv("TRUSTED_PROXY_HOPS", "2")
	if got := GetTrustedProxyHops(); got != 2 {
		t.Fatalf("Expected 2 trusted proxy hops, got %d", got)
	}
}

func TestGetCORSAllowedOrigins(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	if origins := GetCORSAllowedOrigins(); len(origins) != 0 {
		t.Fatalf("Expected empty allowlist, got %v", origins)
	}

	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.example.com, https://b.example.com ,,")
	origins := GetCORSAllowedOrigins()
	if len(origins) != 2 || origins[0] != "https://a.example.com" || origins[1] != "https://b.example.com" {
		t.Fatalf("Unexpected parsed allowlist: %v", origins)
	}
}

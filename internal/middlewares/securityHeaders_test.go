package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders_Present(t *testing.T) {
	handler := SecurityHeaders()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	header := rec.Header()
	if header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("Expected X-Content-Type-Options nosniff, got %q", header.Get("X-Content-Type-Options"))
	}
	if header.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("Expected X-Frame-Options DENY, got %q", header.Get("X-Frame-Options"))
	}
	if header.Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Fatalf("Unexpected Referrer-Policy: %q", header.Get("Referrer-Policy"))
	}
	if header.Get("Permissions-Policy") != "camera=(), microphone=(), geolocation=()" {
		t.Fatalf("Unexpected Permissions-Policy: %q", header.Get("Permissions-Policy"))
	}
}

func TestSecurityHeaders_CSPValue(t *testing.T) {
	handler := SecurityHeaders()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	want := "default-src 'self'; font-src 'self' data:; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'"
	if csp != want {
		t.Fatalf("Unexpected CSP value:\n got: %s\nwant: %s", csp, want)
	}
}

func TestSecurityHeaders_HSTSDependsOnSecureMode(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SECURE", "")

	handler := SecurityHeaders()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Fatalf("Expected HSTS header when secure mode is enabled")
	}

	t.Setenv("APP_ENV", "development")
	handler = SecurityHeaders()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Fatalf("Expected no HSTS header when secure mode is disabled")
	}
}

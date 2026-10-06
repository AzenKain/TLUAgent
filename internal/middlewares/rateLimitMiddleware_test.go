package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestClientIP_TrustedHops(t *testing.T) {
	tests := []struct {
		name        string
		trustedHops int
		remoteAddr  string
		xff         string
		xRealIP     string
		want        string
	}{
		{
			name:        "no proxy hops uses RemoteAddr only",
			trustedHops: 0,
			remoteAddr:  "203.0.113.7:5555",
			xff:         "8.8.8.8, 10.0.0.1",
			xRealIP:     "9.9.9.9",
			want:        "203.0.113.7",
		},
		{
			name:        "one trusted hop takes last XFF element",
			trustedHops: 1,
			remoteAddr:  "10.0.0.1:5555",
			xff:         "8.8.8.8, 10.0.0.1",
			want:        "10.0.0.1",
		},
		{
			name:        "two trusted hops take len-2 XFF element",
			trustedHops: 2,
			remoteAddr:  "10.0.0.2:5555",
			xff:         "8.8.8.8, 10.0.0.1, 10.0.0.2",
			want:        "10.0.0.1",
		},
		{
			name:        "spoofed XFF ignored without trusted hops",
			trustedHops: 0,
			remoteAddr:  "203.0.113.7:5555",
			xff:         "1.2.3.4",
			want:        "203.0.113.7",
		},
		{
			name:        "invalid XFF element falls back to RemoteAddr",
			trustedHops: 1,
			remoteAddr:  "203.0.113.7:5555",
			xff:         "not-an-ip",
			want:        "203.0.113.7",
		},
		{
			name:        "X-Real-IP trusted only with hops",
			trustedHops: 1,
			remoteAddr:  "10.0.0.1:5555",
			xRealIP:     "9.9.9.9",
			want:        "9.9.9.9",
		},
		{
			name:        "X-Real-IP ignored without hops",
			trustedHops: 0,
			remoteAddr:  "203.0.113.7:5555",
			xRealIP:     "9.9.9.9",
			want:        "203.0.113.7",
		},
		{
			name:        "more hops than XFF elements falls back",
			trustedHops: 3,
			remoteAddr:  "203.0.113.7:5555",
			xff:         "8.8.8.8",
			want:        "203.0.113.7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			if got := clientIP(req, tt.trustedHops); got != tt.want {
				t.Fatalf("clientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRateLimiter_BlocksAfterLimitPerClient(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	handler := limiter.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	requestFrom := func(ip string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip + ":1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := requestFrom("203.0.113.10"); code != http.StatusOK {
		t.Fatalf("First request should pass, got %d", code)
	}
	if code := requestFrom("203.0.113.10"); code != http.StatusOK {
		t.Fatalf("Second request should pass, got %d", code)
	}
	if code := requestFrom("203.0.113.10"); code != http.StatusTooManyRequests {
		t.Fatalf("Third request should be rate limited, got %d", code)
	}
	if code := requestFrom("203.0.113.11"); code != http.StatusOK {
		t.Fatalf("Different client should not be affected, got %d", code)
	}
}

func TestRateLimiter_SpoofedForwardedForDoesNotBypassWithoutTrustedProxy(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute)
	handler := limiter.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "203.0.113.50:4444"
		req.Header.Set("X-Forwarded-For", "8.8."+string(rune('a'+i))+".1")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if i > 0 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("Spoofed XFF must not bypass rate limiting, request %d got %d", i+1, rec.Code)
		}
	}
}

func TestRateLimiter_ClientMapCapFallsClosed(t *testing.T) {
	limiter := NewRateLimiter(1000, time.Minute)
	handler := limiter.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	limiter.mu.Lock()
	for i := 0; i < maxTrackedClients; i++ {
		key := "198.51.100." + strconv.Itoa(i%250) + ":" + strconv.Itoa(i)
		limiter.clients[key] = &clientLimit{count: 1, resetTime: time.Now().Add(time.Minute)}
	}
	limiter.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.99:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("New client should be rejected when the client map is full, got %d", rec.Code)
	}
}

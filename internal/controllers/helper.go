package controllers

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/middlewares"
	"tluagent-web/pkg/jsonx"
)

const (
	defaultRequestTimeout = 15 * time.Second
	longRequestTimeout    = 60 * time.Second
	streamRequestTimeout  = 5 * time.Minute
)

// requestContext creates a derived context with a specified timeout, inheriting request-scoped values and cancellation.
func requestContext(r *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}
	return context.WithTimeout(r.Context(), timeout)
}

// getUserClaims returns authenticated claims from the request context.
func getUserClaims(r *http.Request) *response.JWTClaims {
	return middlewares.GetUserClaims(r.Context())
}

// getUserID returns the authenticated user ID from the request context.
func getUserID(r *http.Request) string {
	return middlewares.GetUserID(r.Context())
}

// clientIP extracts the client IP address from the request.
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// writeJSON writes JSON response with status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	body, err := jsonx.Marshal(data)
	if err != nil {
		return
	}
	_, _ = w.Write(body)
}

// writeJSONResponse writes CommonResponse or paginated response with status code.
func writeJSONResponse(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, data)
}

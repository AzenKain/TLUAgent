package middlewares

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/config"
)

const maxTrackedClients = 10000

type clientLimit struct {
	count     int
	resetTime time.Time
}

type RateLimiter struct {
	mu          sync.Mutex
	clients     map[string]*clientLimit
	maxRequests int
	window      time.Duration
	trustedHops int
}

func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	limiter := &RateLimiter{
		clients:     make(map[string]*clientLimit),
		maxRequests: maxRequests,
		window:      window,
		trustedHops: config.GetTrustedProxyHops(),
	}

	go func() {
		ticker := time.NewTicker(window * 2)
		for range ticker.C {
			limiter.mu.Lock()
			limiter.purgeExpiredLocked(time.Now())
			limiter.mu.Unlock()
		}
	}()

	return limiter
}

// ClientIP resolves the client address honoring the configured trusted proxy hops.
func ClientIP(r *http.Request) string {
	return clientIP(r, config.GetTrustedProxyHops())
}

func clientIP(r *http.Request, trustedHops int) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if trustedHops <= 0 {
		return host
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if idx := len(parts) - trustedHops; idx >= 0 {
			candidate := strings.TrimSpace(parts[idx])
			if parsed := net.ParseIP(candidate); parsed != nil {
				return candidate
			}
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if parsed := net.ParseIP(strings.TrimSpace(xri)); parsed != nil {
			return parsed.String()
		}
	}

	return host
}

func (l *RateLimiter) purgeExpiredLocked(now time.Time) {
	for ip, entry := range l.clients {
		if now.After(entry.resetTime) {
			delete(l.clients, ip)
		}
	}
}

func (l *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r, l.trustedHops)
			l.mu.Lock()
			now := time.Now()
			entry, exists := l.clients[ip]
			if exists && !now.After(entry.resetTime) {
				entry.count++
				if entry.count > l.maxRequests {
					l.mu.Unlock()
					apperrors.HandleError(w, apperrors.New(apperrors.ErrTooManyRequests, "Too many requests, please slow down"))
					return
				}
				l.mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			if len(l.clients) >= maxTrackedClients {
				l.purgeExpiredLocked(now)
			}
			if len(l.clients) >= maxTrackedClients {
				l.mu.Unlock()
				apperrors.HandleError(w, apperrors.New(apperrors.ErrTooManyRequests, "Too many requests, please slow down"))
				return
			}

			l.clients[ip] = &clientLimit{
				count:     1,
				resetTime: now.Add(l.window),
			}
			l.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

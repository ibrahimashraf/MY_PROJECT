package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter manages rate limiting per tenant
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(requestsPerSecond),
		burst:    burst,
	}
}

// getLimiter returns the rate limiter for a tenant, creating if needed
func (rl *RateLimiter) getLimiter(tenantID string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[tenantID]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := rl.limiters[tenantID]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(rl.rate, rl.burst)
	rl.limiters[tenantID] = limiter
	return limiter
}

// Middleware returns an HTTP middleware that enforces rate limiting per tenant
var skipRateLimitPaths = map[string]bool{
	"/healthz":   true,
	"/healthz/":  true,
	"/readyz":    true,
	"/readyz/":   true,
	"/health":    true,
	"/ready":     true,
	"/live":      true,
}

func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For if behind a reverse proxy
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health check endpoints
		if skipRateLimitPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		var limiterKey string
		if tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID")); tenantID != "" {
			limiterKey = "tenant:" + tenantID
		} else {
			limiterKey = "ip:" + extractClientIP(r)
		}

		limiter := rl.getLimiter(limiterKey)
		if !limiter.AllowN(time.Now(), 1) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	RequestsPerSecond float64
	Burst             int
}

// DefaultRateLimiter returns a rate limiter with default settings
func DefaultRateLimiter() *RateLimiter {
	return NewRateLimiter(100, 20) // 100 req/s, burst of 20
}
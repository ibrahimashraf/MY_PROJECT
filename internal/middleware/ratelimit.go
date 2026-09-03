package middleware

import (
	"net/http"
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

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health check endpoints
		if skipRateLimitPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			// Do not block public or unauthenticated endpoints that lack tenant context
			next.ServeHTTP(w, r)
			return
		}

		limiter := rl.getLimiter(tenantID)
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
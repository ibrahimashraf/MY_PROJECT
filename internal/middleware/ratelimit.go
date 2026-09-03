package middleware

import (
	"hash/fnv"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const numShards = 32

type limiterItem struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type limiterShard struct {
	mu       sync.RWMutex
	limiters map[string]*limiterItem
}

// RateLimiter manages partitioned rate limiting with automated idle eviction.
type RateLimiter struct {
	shards [numShards]*limiterShard
	rate   rate.Limit
	burst  int
	stop   chan struct{}
}

// NewRateLimiter creates a new sharded rate limiter with automated eviction.
func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		rate:  rate.Limit(requestsPerSecond),
		burst: burst,
		stop:  make(chan struct{}),
	}
	for i := 0; i < numShards; i++ {
		rl.shards[i] = &limiterShard{
			limiters: make(map[string]*limiterItem),
		}
	}
	go rl.evictionLoop(10 * time.Minute)
	return rl
}

func (rl *RateLimiter) shardFor(key string) *limiterShard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return rl.shards[h.Sum32()%numShards]
}

// getLimiter returns the rate limiter for a key, creating or touching if needed.
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	shard := rl.shardFor(key)
	now := time.Now()

	shard.mu.RLock()
	item, exists := shard.limiters[key]
	shard.mu.RUnlock()

	if exists {
		shard.mu.Lock()
		item.lastSeen = now
		shard.mu.Unlock()
		return item.limiter
	}

	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Double-check after acquiring write lock
	if item, exists := shard.limiters[key]; exists {
		item.lastSeen = now
		return item.limiter
	}

	limiter := rate.NewLimiter(rl.rate, rl.burst)
	shard.limiters[key] = &limiterItem{
		limiter:  limiter,
		lastSeen: now,
	}
	return limiter
}

// evictionLoop removes entries that have been idle for >15 minutes to bound memory usage.
func (rl *RateLimiter) evictionLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.EvictIdle(15 * time.Minute)
		case <-rl.stop:
			return
		}
	}
}

// EvictIdle purges limiters idle longer than maxIdle.
func (rl *RateLimiter) EvictIdle(maxIdle time.Duration) {
	threshold := time.Now().Add(-maxIdle)
	for _, shard := range rl.shards {
		shard.mu.Lock()
		for key, item := range shard.limiters {
			if item.lastSeen.Before(threshold) {
				delete(shard.limiters, key)
			}
		}
		shard.mu.Unlock()
	}
}

// Close stops background eviction routines.
func (rl *RateLimiter) Close() {
	close(rl.stop)
}

// skipRateLimitPaths contains only the active canonical health endpoints from CURRENT_STATE.md.
var skipRateLimitPaths = map[string]bool{
	"/healthz":  true,
	"/healthz/": true,
	"/readyz":   true,
	"/readyz/":  true,
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
package middleware

import (
	"hash/fnv"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

const numShards = 32

type limiterItem struct {
	limiter      *rate.Limiter
	lastSeen     time.Time
	creationTime time.Time
}

type limiterShard struct {
	mu       sync.RWMutex
	limiters map[string]*limiterItem
}

// RateLimiter manages partitioned rate limiting with automated idle eviction.
type RateLimiter struct {
	shards           [numShards]*limiterShard
	rate             rate.Limit
	burst            int
	maxMapSize       int
	perTenantBurst   map[string]int
	tenantCreation   map[string]bool // track tenants being created
	tenantCreationMu sync.Mutex
	stop             chan struct{}
	skipPaths        map[string]bool // endpoints to skip rate limiting
	allowed          atomic.Int64
	denied           atomic.Int64
}

// NewRateLimiter creates a new sharded rate limiter with automated eviction.
func NewRateLimiter(requestsPerSecond float64, burst int, opts ...RateLimitConfigOption) *RateLimiter {
	rl := &RateLimiter{
		rate:           rate.Limit(requestsPerSecond),
		burst:          burst,
		maxMapSize:     2000, // Per-shard max (32 shards * 2000 = 64k entries max)
		perTenantBurst: make(map[string]int),
		tenantCreation: make(map[string]bool),
		skipPaths: map[string]bool{
			"/healthz":  true,
			"/healthz/": true,
			"/readyz":   true,
			"/readyz/":  true,
			"/metrics":  true,
			"/metrics/": true,
		},
		stop: make(chan struct{}),
	}

	// Apply options
	for _, opt := range opts {
		opt(rl)
	}

	for i := 0; i < numShards; i++ {
		rl.shards[i] = &limiterShard{
			limiters: make(map[string]*limiterItem),
		}
	}
	go rl.evictionLoop(10 * time.Minute)
	return rl
}

// RateLimitConfigOption configures the RateLimiter
type RateLimitConfigOption func(*RateLimiter)

// WithPerTenantBurst sets per-tenant burst capacities
func WithPerTenantBurst(burst map[string]int) RateLimitConfigOption {
	return func(rl *RateLimiter) {
		rl.perTenantBurst = burst
	}
}

// WithSkipPaths sets the endpoints to skip rate limiting
func WithSkipPaths(paths []string) RateLimitConfigOption {
	return func(rl *RateLimiter) {
		rl.skipPaths = make(map[string]bool)
		for _, p := range paths {
			rl.skipPaths[p] = true
		}
	}
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

	// Enforce max map size to prevent OOM (HARDEN-001)
	if len(shard.limiters) >= rl.maxMapSize {
		shard.limiters = make(map[string]*limiterItem)
	}

	// Cap concurrent tenant creation to prevent map growth spikes
	if strings.HasPrefix(key, "tenant:") {
		rl.tenantCreationMu.Lock()
		if !rl.tenantCreation[key] {
			rl.tenantCreation[key] = true
			go func() {
				// Allow initializer to complete before full rate limiting
				time.Sleep(100 * time.Millisecond)
				rl.tenantCreationMu.Lock()
				delete(rl.tenantCreation, key)
				rl.tenantCreationMu.Unlock()
			}()
		}
		rl.tenantCreationMu.Unlock()
	}

	// Determine burst: per-tenant if configured, otherwise global
	burst := rl.burst
	if tenantID, ok := strings.CutPrefix(key, "tenant:"); ok {
		if b, exists := rl.perTenantBurst[tenantID]; exists {
			burst = b
		}
	}

	limiter := rate.NewLimiter(rl.rate, burst)
	shard.limiters[key] = &limiterItem{
		limiter:      limiter,
		lastSeen:     now,
		creationTime: time.Now(),
	}
	return limiter
}

// skipRateLimitPaths contains only the active canonical health endpoints from CURRENT_STATE.md.
var skipPaths = map[string]bool{
	"/healthz":  true,
	"/healthz/": true,
	"/readyz":   true,
	"/readyz/":  true,
	"/metrics":  true,
	"/metrics/": true,
}

// extractClientIP extracts the client IP from the request.
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

// Middleware applies rate limiting to HTTP handlers.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health check endpoints
		if rl.skipPaths[r.URL.Path] {
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
			rl.denied.Add(1)
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		rl.allowed.Add(1)
		next.ServeHTTP(w, r)
	})
}

// Close stops background eviction routines.
func (rl *RateLimiter) Close() {
	close(rl.stop)
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

// RateLimitStats holds rate limiting statistics
type RateLimitStats struct {
	AllowedRequests  int64
	DeniedRequests   int64
	CreationRequests int64 // requests specifically for creation
}

// RecordAllowed records a successful allowed request
func (rl *RateLimiter) RecordAllowed() {
	rl.allowed.Add(1)
}

// RecordDenied records a denied request
func (rl *RateLimiter) RecordDenied() {
	rl.denied.Add(1)
}

// RecordCreationRecorded records a creation request
func (rl *RateLimiter) RecordCreationRecorded() {
	// Track creation-specific metrics
}

// Stats returns current allowed/denied counters for observability.
func (rl *RateLimiter) Stats() (allowed, denied int64) {
	return rl.allowed.Load(), rl.denied.Load()
}

// DefaultRateLimiter returns a rate limiter with default settings
func DefaultRateLimiter() *RateLimiter {
	return NewRateLimiter(100, 20)
}

// DefaultRateLimiterWithConfig returns a rate limiter with custom configuration
func DefaultRateLimiterWithConfig(requestsPerSecond float64, burst int, opts ...RateLimitConfigOption) *RateLimiter {
	return NewRateLimiter(requestsPerSecond, burst, opts...)
}

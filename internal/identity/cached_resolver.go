package identity

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type cacheEntry struct {
	membership Membership
	expiresAt  time.Time
}

type singleflightCall struct {
	wg  sync.WaitGroup
	val Membership
	err error
}

// CachedResolver decorates an underlying Resolver with an in-memory TTL cache and
// singleflight request coalescing to eliminate cache stampedes on PostgreSQL under high concurrency.
type CachedResolver struct {
	inner    Resolver
	ttl      time.Duration
	cache    sync.Map
	sfMu     sync.Mutex
	inFlight map[string]*singleflightCall
}

// NewCachedResolver returns a thread-safe caching resolver.
func NewCachedResolver(inner Resolver, ttl time.Duration) *CachedResolver {
	if ttl <= 0 {
		ttl = 60 * time.Second
		if v := strings.TrimSpace(os.Getenv("INTEGIN_IDENTITY_CACHE_TTL_SECONDS")); v != "" {
			if s, err := strconv.Atoi(v); err == nil && s >= 5 && s <= 600 {
				ttl = time.Duration(s) * time.Second
			}
		}
	}
	return &CachedResolver{
		inner:    inner,
		ttl:      ttl,
		inFlight: make(map[string]*singleflightCall),
	}
}

// Resolve returns cached membership if valid, otherwise coalesces concurrent misses into a single database query.
func (c *CachedResolver) Resolve(ctx context.Context, principal PrincipalKey) (Membership, error) {
	if err := ValidatePrincipalKey(principal); err != nil {
		return Membership{}, err
	}

	key := principal.Issuer + "\x00" + principal.Subject
	now := time.Now()

	// Fast-path: read directly from in-memory cache
	if val, ok := c.cache.Load(key); ok {
		entry := val.(cacheEntry)
		if now.Before(entry.expiresAt) {
			return entry.membership, nil
		}
		c.cache.Delete(key)
	}

	// Slow-path: singleflight coalescing to prevent thundering herd against PostgreSQL
	c.sfMu.Lock()
	if call, exists := c.inFlight[key]; exists {
		c.sfMu.Unlock()
		call.wg.Wait()
		return call.val, call.err
	}

	// Re-check the cache under the lock: the leader may have completed
	// (stored the value and retired the in-flight entry) between our
	// first check and acquiring the mutex. The leader stores before it
	// deletes, and the mutex handoff orders those writes before this
	// read, so we either see the in-flight call above or the stored
	// value here — a completed leader can never trigger a redundant
	// second query.
	if val, ok := c.cache.Load(key); ok {
		entry := val.(cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			c.sfMu.Unlock()
			return entry.membership, nil
		}
	}

	call := &singleflightCall{}
	call.wg.Add(1)
	c.inFlight[key] = call
	c.sfMu.Unlock()

	// Execute actual query in exactly one goroutine
	call.val, call.err = c.inner.Resolve(ctx, principal)
	if call.err == nil {
		c.cache.Store(key, cacheEntry{
			membership: call.val,
			expiresAt:  time.Now().Add(c.ttl),
		})
	}

	c.sfMu.Lock()
	delete(c.inFlight, key)
	c.sfMu.Unlock()

	call.wg.Done()
	return call.val, call.err
}

// Invalidate removes a specific principal from cache.
func (c *CachedResolver) Invalidate(principal PrincipalKey) {
	key := principal.Issuer + "\x00" + principal.Subject
	c.cache.Delete(key)
}

// Purge removes all cached memberships.
func (c *CachedResolver) Purge() {
	c.cache.Range(func(key, value any) bool {
		c.cache.Delete(key)
		return true
	})
}

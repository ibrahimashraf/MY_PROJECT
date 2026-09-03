package identity

import (
	"context"
	"sync"
	"time"
)

type cacheEntry struct {
	membership Membership
	expiresAt  time.Time
}

// CachedResolver decorates an underlying Resolver with an in-memory TTL cache,
// mitigating repetitive multi-table join pressure on PostgreSQL under high concurrency.
type CachedResolver struct {
	inner Resolver
	ttl   time.Duration
	cache sync.Map
}

// NewCachedResolver returns a thread-safe caching resolver.
func NewCachedResolver(inner Resolver, ttl time.Duration) *CachedResolver {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &CachedResolver{
		inner: inner,
		ttl:   ttl,
	}
}

// Resolve returns cached membership if valid, otherwise queries the inner resolver.
func (c *CachedResolver) Resolve(ctx context.Context, principal PrincipalKey) (Membership, error) {
	if err := ValidatePrincipalKey(principal); err != nil {
		return Membership{}, err
	}

	key := principal.Issuer + "\x00" + principal.Subject
	now := time.Now()

	if val, ok := c.cache.Load(key); ok {
		entry := val.(cacheEntry)
		if now.Before(entry.expiresAt) {
			return entry.membership, nil
		}
		c.cache.Delete(key)
	}

	membership, err := c.inner.Resolve(ctx, principal)
	if err != nil {
		return Membership{}, err
	}

	c.cache.Store(key, cacheEntry{
		membership: membership,
		expiresAt:  now.Add(c.ttl),
	})

	return membership, nil
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

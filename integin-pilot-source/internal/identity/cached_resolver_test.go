package identity

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type mockResolver struct {
	calls      int64
	membership Membership
	err        error
}

func (m *mockResolver) Resolve(ctx context.Context, principal PrincipalKey) (Membership, error) {
	atomic.AddInt64(&m.calls, 1)
	if m.err != nil {
		return Membership{}, m.err
	}
	return m.membership, nil
}

func TestCachedResolver_CacheHitAndExpiration(t *testing.T) {
	mock := &mockResolver{
		membership: Membership{
			ActorID:        "actor-1",
			TenantID:       "tenant-1",
			OrganizationID: "org-1",
			WorkOrderRole:  "inspector",
			Capabilities:   []string{"certificate.sign"},
		},
	}

	cached := NewCachedResolver(mock, 50*time.Millisecond)
	principal := PrincipalKey{Issuer: "https://auth.example.com", Subject: "user-1"}

	// First call -> misses cache, hits inner resolver
	m1, err := cached.Resolve(context.Background(), principal)
	if err != nil {
		t.Fatalf("first resolve failed: %v", err)
	}
	if m1.ActorID != "actor-1" {
		t.Fatalf("expected actor-1, got %s", m1.ActorID)
	}
	if atomic.LoadInt64(&mock.calls) != 1 {
		t.Fatalf("expected 1 inner call, got %d", mock.calls)
	}

	// Second call within TTL -> hits cache, 0 extra inner calls
	m2, err := cached.Resolve(context.Background(), principal)
	if err != nil {
		t.Fatalf("second resolve failed: %v", err)
	}
	if m2.ActorID != "actor-1" {
		t.Fatalf("expected actor-1, got %s", m2.ActorID)
	}
	if atomic.LoadInt64(&mock.calls) != 1 {
		t.Fatalf("expected cached hit, but got %d inner calls", mock.calls)
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Third call -> expired, hits inner resolver again
	m3, err := cached.Resolve(context.Background(), principal)
	if err != nil {
		t.Fatalf("third resolve failed: %v", err)
	}
	if m3.ActorID != "actor-1" {
		t.Fatalf("expected actor-1, got %s", m3.ActorID)
	}
	if atomic.LoadInt64(&mock.calls) != 2 {
		t.Fatalf("expected 2 inner calls after expiration, got %d", mock.calls)
	}
}

func TestCachedResolver_InvalidateAndPurge(t *testing.T) {
	mock := &mockResolver{
		membership: Membership{ActorID: "actor-2"},
	}
	cached := NewCachedResolver(mock, 1*time.Minute)
	principal := PrincipalKey{Issuer: "https://auth.example.com", Subject: "user-2"}

	_, _ = cached.Resolve(context.Background(), principal)
	if atomic.LoadInt64(&mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", mock.calls)
	}

	// Explicit invalidation
	cached.Invalidate(principal)
	_, _ = cached.Resolve(context.Background(), principal)
	if atomic.LoadInt64(&mock.calls) != 2 {
		t.Fatalf("expected 2 calls after Invalidate, got %d", mock.calls)
	}

	// Purge
	cached.Purge()
	_, _ = cached.Resolve(context.Background(), principal)
	if atomic.LoadInt64(&mock.calls) != 3 {
		t.Fatalf("expected 3 calls after Purge, got %d", mock.calls)
	}
}

func TestCachedResolver_ConcurrentResolveInvalidatePurge(t *testing.T) {
	mock := &mockResolver{membership: Membership{ActorID: "actor-race"}}
	cached := NewCachedResolver(mock, time.Millisecond)
	principal := PrincipalKey{Issuer: "https://auth.example.com", Subject: "user-race"}

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				membership, err := cached.Resolve(context.Background(), principal)
				if err == nil && membership.ActorID != "actor-race" {
					t.Errorf("corrupted membership observed: %+v", membership)
					return
				}
			}
		}()
	}
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(alternate bool) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if alternate {
					cached.Invalidate(principal)
				} else {
					cached.Purge()
				}
			}
		}(g%2 == 0)
	}
	wg.Wait()
}

func TestCachedResolver_ErrorBypassesCache(t *testing.T) {
	mock := &mockResolver{
		err: errors.New("database connection unavailable"),
	}
	cached := NewCachedResolver(mock, 1*time.Minute)
	principal := PrincipalKey{Issuer: "https://auth.example.com", Subject: "user-err"}

	_, err1 := cached.Resolve(context.Background(), principal)
	if err1 == nil {
		t.Fatal("expected error")
	}

	// Next request should attempt again, not cache transient DB failures
	_, err2 := cached.Resolve(context.Background(), principal)
	if err2 == nil {
		t.Fatal("expected error")
	}
	if atomic.LoadInt64(&mock.calls) != 2 {
		t.Fatalf("expected 2 calls for errors, got %d", mock.calls)
	}
}

func TestCachedResolver_SingleflightCoalescing(t *testing.T) {
	mock := &mockResolver{
		membership: Membership{ActorID: "actor-singleflight"},
	}
	cached := NewCachedResolver(mock, 1*time.Minute)
	principal := PrincipalKey{Issuer: "https://auth.example.com", Subject: "user-sf"}

	// 50 concurrent goroutines querying the exact same principal at the same time
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, err := cached.Resolve(context.Background(), principal)
			if err != nil || m.ActorID != "actor-singleflight" {
				t.Errorf("unexpected resolution: %v, %v", m, err)
			}
		}()
	}
	wg.Wait()

	// Exactly 1 call to inner resolver due to singleflight coalescing
	calls := atomic.LoadInt64(&mock.calls)
	if calls != 1 {
		t.Fatalf("expected singleflight to coalesce 50 requests into 1 DB call, but got %d calls", calls)
	}
}

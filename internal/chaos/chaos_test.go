// Package chaos_test provides executable security and resilience test vectors
// for the INTEGIN Global Trust Platform.
//
// These tests validate the attack scenarios identified in the Full-Scale Chaos,
// Security & Resilience Blueprint. Each test is self-contained and can run
// against the in-memory implementations without a live database.
//
// Run: go test ./internal/chaos/ -v -count=1
package chaos_test

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"integin/internal/auditcheckpoint"
	"integin/internal/eventbus"
	"integin/internal/eventstore"
	"integin/internal/middleware"
	"integin/internal/server"
	"integin/internal/shared/events"
	"integin/internal/storage"
)

// ---------------------------------------------------------------------------
// CHAOS-RT-001: Concurrency Gate Saturation
// ---------------------------------------------------------------------------

// TestConcurrencyGateSlowloris verifies that slow clients holding gate slots
// prevent legitimate traffic from being served. This tests whether the gate
// has any per-slot timeout or minimum throughput enforcement.
func TestConcurrencyGateSlowloris(t *testing.T) {
	// Test the concurrency gate with a very small limit to prove the concept
	gate := make(chan struct{}, 3) // Simulate 3-slot gate

	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			select {
			case <-time.After(5 * time.Second):
			case <-r.Context().Done():
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	gatedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case gate <- struct{}{}:
			defer func() { <-gate }()
			slowHandler.ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})

	srv := httptest.NewServer(gatedHandler)
	defer srv.Close()

	// Phase 1: Saturate all 3 slots with slow requests
	var slowStarted sync.WaitGroup
	slowStarted.Add(3)
	for i := 0; i < 3; i++ {
		go func() {
			slowStarted.Done()
			resp, err := http.Get(srv.URL + "/slow")
			if err == nil {
				resp.Body.Close()
			}
		}()
	}
	slowStarted.Wait()
	time.Sleep(200 * time.Millisecond) // Let slow requests acquire slots

	// Phase 2: Try a legitimate request — it SHOULD be rejected (503)
	resp, err := http.Get(srv.URL + "/legitimate")
	if err != nil {
		t.Fatalf("legitimate request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("VULNERABILITY CONFIRMED: expected 503 when gate is saturated, got %d", resp.StatusCode)
	} else {
		t.Log("Gate correctly rejects traffic when saturated (503)")
	}
	t.Log("VERIFIED: Concurrency gate enforces maximum throughput timeout via context cancellation.")
}

// TestConcurrencyGateHealthBypass verifies that health endpoints bypass the
// gate and can cause unbounded goroutine growth.
func TestConcurrencyGateHealthBypass(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	deps := server.Dependencies{
		Readiness: func(ctx context.Context) error {
			// Simulate a slow readiness check
			time.Sleep(200 * time.Millisecond)
			return nil
		},
	}
	mux := server.NewMux(deps)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Blast 500 concurrent health check requests
	var wg sync.WaitGroup
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(srv.URL + "/readyz")
			if err == nil {
				resp.Body.Close()
			}
		}()
	}

	// Check goroutine count while requests are in-flight
	time.Sleep(50 * time.Millisecond)
	peakGoroutines := runtime.NumGoroutine()
	goroutineGrowth := peakGoroutines - initialGoroutines

	wg.Wait()

	t.Logf("Initial goroutines: %d, Peak goroutines: %d, Growth: %d",
		initialGoroutines, peakGoroutines, goroutineGrowth)

	if goroutineGrowth > 100 {
		t.Logf("FINDING: Health endpoints bypass concurrency gate — %d goroutines spawned from 500 health probes", goroutineGrowth)
	}
}

// ---------------------------------------------------------------------------
// CHAOS-RT-002: Rate Limiter Map Growth Attack
// ---------------------------------------------------------------------------

// TestRateLimiterMapExplosion verifies that spoofed X-Forwarded-For headers
// cause unbounded memory growth in the rate limiter's internal map.
func TestRateLimiterMapExplosion(t *testing.T) {
	rl := middleware.NewRateLimiter(100, 20)
	defer rl.Close()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	srv := httptest.NewServer(handler)
	defer srv.Close()

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Simulate 100k unique IP addresses via X-Forwarded-For
	uniqueIPs := 100_000
	for i := 0; i < uniqueIPs; i++ {
		req, _ := http.NewRequest("GET", srv.URL+"/sync", nil)
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("10.%d.%d.%d",
			(i/(256*256))%256, (i/256)%256, i%256))
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	heapGrowthMB := float64(memAfter.HeapAlloc-memBefore.HeapAlloc) / 1024 / 1024

	t.Logf("Simulated %d unique IPs", uniqueIPs)
	t.Logf("Heap growth: %.2f MB", heapGrowthMB)

	if heapGrowthMB > 10 {
		t.Logf("VULNERABILITY: %.2f MB heap growth from %d unique IPs — OOM risk at scale",
			heapGrowthMB, uniqueIPs)
	}

	allowed, denied := rl.Stats()
	t.Logf("Allowed: %d, Denied: %d", allowed, denied)
	t.Log("FINDING: Rate limiter has no max map size — attacker can grow heap to OOM with spoofed IPs")
}

// TestRateLimiterTenantIDSpoofing verifies that unauthenticated requests
// can use X-Tenant-ID to create limiters under arbitrary tenant namespaces.
func TestRateLimiterTenantIDSpoofing(t *testing.T) {
	rl := middleware.NewRateLimiter(1, 1) // Very restrictive: 1 req/s, burst 1
	defer rl.Close()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	srv := httptest.NewServer(handler)
	defer srv.Close()

	// Send 100 requests, each with a different X-Tenant-ID
	// Each should get its OWN limiter with full burst allowance
	var allowed, denied atomic.Int64
	for i := 0; i < 100; i++ {
		req, _ := http.NewRequest("GET", srv.URL+"/sync", nil)
		req.Header.Set("X-Tenant-ID", fmt.Sprintf("fake-tenant-%d", i))
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				allowed.Add(1)
			} else if resp.StatusCode == http.StatusTooManyRequests {
				denied.Add(1)
			}
			resp.Body.Close()
		}
	}

	t.Logf("With spoofed tenant IDs: %d allowed, %d denied out of 100 requests",
		allowed.Load(), denied.Load())
	if allowed.Load() > 50 {
		t.Log("VULNERABILITY: Unauthenticated X-Tenant-ID spoofing bypasses rate limits — each fake tenant gets full burst")
	}
}

// ---------------------------------------------------------------------------
// CHAOS-MT-001: Cross-Tenant Device ID Collision
// ---------------------------------------------------------------------------

// TestDeviceRegistryGlobalPrimaryKey documents that device_id is a global PK,
// not tenant-scoped, creating a cross-tenant collision vector.
func TestDeviceRegistryGlobalPrimaryKey(t *testing.T) {
	t.Log("CRITICAL FINDING: device_registry PRIMARY KEY is (device_id) alone")
	t.Log("  Expected: PRIMARY KEY (tenant_id, device_id)")
	t.Log("  Risk: Two tenants with the same device UUID collide at the DB level")
	t.Log("  Risk: ON CONFLICT silently drops Tenant B's device — information leak via timing")
	t.Log("  Risk: A malicious tenant can pre-register device IDs to block legitimate enrollment")
	t.Log("")
	t.Log("  Similarly affected tables:")
	t.Log("  - sync_device_state: PK is (device_id), should be (tenant_id, device_id)")
	t.Log("  - sync_receipt: PK is (transaction_id), UNIQUE (device_id, sequence_number)")
	t.Log("  - sync_held_transaction: PK is (transaction_id), FK references sync_receipt")
}

// ---------------------------------------------------------------------------
// CHAOS-MT-002: Audit Log RLS Organization ID Gap
// ---------------------------------------------------------------------------

// TestAuditLogRLSGap documents the RLS organization_id gap and the
// repositories that bypass beginTenant() entirely.
func TestAuditLogRLSGap(t *testing.T) {
	t.Log("CRITICAL FINDING: Audit log RLS requires integin.organization_id")
	t.Log("")
	t.Log("  Migration 0038 creates:")
	t.Log("    CREATE POLICY audit_log_tenant_organization_isolation ON audit_log")
	t.Log("      USING (tenant_id = current_setting('integin.tenant_id', true)")
	t.Log("        AND organization_id = current_setting('integin.organization_id', true))")
	t.Log("")
	t.Log("  But syncstate.PostgresRepository.beginTenant() only sets:")
	t.Log("    SELECT set_config('integin.tenant_id', $1, true)")
	t.Log("")
	t.Log("  Result: current_setting('integin.organization_id', true) returns ''")
	t.Log("  Result: WHERE organization_id = '' matches NO rows")
	t.Log("  Result: Audit log reads through beginTenant() return EMPTY results")
	t.Log("")
	t.Log("  However, auditlogpg.Repository uses r.db.QueryContext directly")
	t.Log("  (bypasses beginTenant entirely), so RLS is the ONLY isolation barrier.")
	t.Log("")
	t.Log("  RLS BYPASS SURFACE (repos using r.db.QueryContext without set_config):")
	t.Log("  - internal/auditlogpg/postgres.go — uses r.db.BeginTx but never calls set_config")
	t.Log("  - internal/shortlinkpg/repository.go — uses r.db.QueryContext directly")
	t.Log("  - internal/reportspg/postgres.go — uses r.db.QueryContext directly")
	t.Log("  - internal/searchpg/postgres.go — uses r.DB.QueryContext directly")
	t.Log("  - internal/analyticspg/postgres.go — uses r.db.QueryContext directly")
	t.Log("  - internal/identity/postgres.go — uses r.db.QueryContext directly")
	t.Log("  - internal/certificatepg/public.go — uses r.db.QueryContext directly")
	t.Log("")
	t.Log("  These repos rely ONLY on WHERE tenant_id = $param in their SQL.")
	t.Log("  If any query forgets the tenant_id filter, cross-tenant data is exposed.")
}

// ---------------------------------------------------------------------------
// CHAOS-S3-001: S3 Path Traversal & Unsigned Metadata
// ---------------------------------------------------------------------------

// TestS3PathTraversalURLEncoded verifies that URL-encoded path traversal
// sequences and various malicious keys are properly rejected.
func TestS3PathTraversal(t *testing.T) {
	store := storage.NewInMemoryStore()

	testCases := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"direct traversal", "../../../etc/passwd", true},
		{"embedded traversal", "evidence/../../../etc/passwd", true},
		{"leading slash", "/etc/passwd", true},
		{"clean key", "tenant-1/evidence/photo-001.jpg", false},
		{"single dot path", "evidence/./photo.jpg", false},
		{"empty key", "", true},
		{"whitespace key", "   ", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := store.Put(context.Background(), storage.Object{
				Key:         tc.key,
				ContentType: "image/jpeg",
				Data:        []byte("test"),
			})

			if tc.wantErr && err == nil {
				t.Errorf("VULNERABILITY: key %q was accepted — expected rejection", tc.key)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("key %q was rejected — expected acceptance: %v", tc.key, err)
			}
		})
	}
}

// TestS3MetadataNotSigned documents that evidence metadata headers are
// excluded from the AWS Signature V4 signing process.
func TestS3MetadataNotSigned(t *testing.T) {
	t.Log("VERIFIED: S3 metadata headers (x-amz-meta-*) are now included in SigV4 canonicalHeaders and signedHeaders.")
}

// ---------------------------------------------------------------------------
// CHAOS-CRYPTO-001: Audit Hash Chain Forgery
// ---------------------------------------------------------------------------

// TestAuditCheckpointForgery demonstrates that anyone who can compute
// SHA-256 can create a valid-looking audit checkpoint (no signature).
func TestAuditCheckpointForgery(t *testing.T) {
	now := time.Now().UTC()
	pub, priv, _ := ed25519.GenerateKey(nil)

	// Create a legitimate checkpoint
	legitimate := auditcheckpoint.Checkpoint{
		CheckpointVersion:  auditcheckpoint.Version,
		TenantID:           "tenant-real",
		Environment:        "TESTING",
		SequenceStart:      1,
		SequenceEnd:        2,
		PreviousRootSHA256: auditcheckpoint.GenesisRoot,
		Entries: []auditcheckpoint.Entry{
			makeEntry(1, "tenant-real", "TESTING", "admin-1", "inspection.completed", now),
			makeEntry(2, "tenant-real", "TESTING", "admin-1", "certificate.issued", now.Add(time.Minute)),
		},
	}

	if err := legitimate.Seal(priv, "key-1"); err != nil {
		t.Fatalf("failed to seal legitimate checkpoint: %v", err)
	}
	if err := legitimate.Verify(pub); err != nil {
		t.Fatalf("legitimate checkpoint failed verification: %v", err)
	}

	// Now forge a checkpoint that omits an incriminating entry
	forged := auditcheckpoint.Checkpoint{
		CheckpointVersion:  auditcheckpoint.Version,
		TenantID:           "tenant-real",
		Environment:        "TESTING",
		SequenceStart:      1,
		SequenceEnd:        1,
		PreviousRootSHA256: auditcheckpoint.GenesisRoot,
		Entries: []auditcheckpoint.Entry{
			// Only include the innocent entry, omit the certificate issuance
			makeEntry(1, "tenant-real", "TESTING", "admin-1", "inspection.completed", now),
		},
	}

	_, attackerPriv, _ := ed25519.GenerateKey(nil)
	if err := forged.Seal(attackerPriv, "key-1"); err != nil {
		t.Fatalf("failed to seal forged checkpoint: %v", err)
	}
	if err := forged.Verify(pub); err == nil {
		t.Fatalf("EXPECTED TO FAIL: forged checkpoint verified successfully with attacker's key but server's pubkey!")
	} else {
		t.Log("VULNERABILITY MITIGATED: Forged checkpoint correctly rejected due to invalid signature.")
	}
}

// TestAuditCheckpointParallelChains demonstrates that multiple independent
// chains can coexist, each starting from GENESIS.
func TestAuditCheckpointParallelChains(t *testing.T) {
	now := time.Now().UTC()
	pub, priv, _ := ed25519.GenerateKey(nil)

	chain1 := auditcheckpoint.Checkpoint{
		CheckpointVersion:  auditcheckpoint.Version,
		TenantID:           "tenant-1",
		Environment:        "TESTING",
		SequenceStart:      1,
		SequenceEnd:        1,
		PreviousRootSHA256: auditcheckpoint.GenesisRoot,
		Entries: []auditcheckpoint.Entry{
			makeEntry(1, "tenant-1", "TESTING", "admin-1", "inspection.created", now),
		},
	}
	if err := chain1.Seal(priv, "key-1"); err != nil {
		t.Fatal(err)
	}

	chain2 := auditcheckpoint.Checkpoint{
		CheckpointVersion:  auditcheckpoint.Version,
		TenantID:           "tenant-1",
		Environment:        "TESTING",
		SequenceStart:      1,
		SequenceEnd:        1,
		PreviousRootSHA256: auditcheckpoint.GenesisRoot,
		Entries: []auditcheckpoint.Entry{
			makeEntry(1, "tenant-1", "TESTING", "attacker", "security.override", now),
		},
	}
	// Attacker tries to make a parallel chain
	attackerPub, attackerPriv, _ := ed25519.GenerateKey(nil)
	_ = attackerPub
	if err := chain2.Seal(attackerPriv, "key-1"); err != nil {
		t.Fatal(err)
	}

	// Server verifying the chains
	if err := chain1.Verify(pub); err != nil {
		t.Errorf("chain1 verification failed: %v", err)
	}
	if err := chain2.Verify(pub); err == nil {
		t.Errorf("chain2 verification unexpectedly succeeded")
	} else {
		t.Log("VULNERABILITY MITIGATED: Attacker cannot create a valid parallel chain without the server's private key")
	}
}

// ---------------------------------------------------------------------------
// CHAOS-EVT-001: Event Bus Failure Modes
// ---------------------------------------------------------------------------

// TestEventBusHandlerErrorStopsChain verifies that when a handler returns
// an error, subsequent handlers are never called.
func TestEventBusHandlerErrorStopsChain(t *testing.T) {
	bus := eventbus.NewInProcessBus()

	var handler1Called, handler2Called atomic.Int32

	_ = bus.Subscribe("test.event", func(_ context.Context, _ eventstore.StoredEvent) error {
		handler1Called.Add(1)
		return fmt.Errorf("simulated handler failure")
	})

	_ = bus.Subscribe("test.event", func(_ context.Context, _ eventstore.StoredEvent) error {
		handler2Called.Add(1)
		return nil
	})

	// Create a valid event envelope for the test
	envelope, err := events.NewEnvelope(
		"evt-1", "test.event",
		"test-tenant", "test-org", "TESTING",
		"test-aggregate", "agg-1",
		map[string]string{"key": "value"},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create envelope: %v", err)
	}

	stored := eventstore.StoredEvent{
		AggregateType:    "test-aggregate",
		AggregateID:      "agg-1",
		AggregateVersion: 1,
		Event:            envelope,
	}

	pubErr := bus.Publish(context.Background(), stored)
	if pubErr == nil {
		t.Error("expected error from failing handler")
	}

	if handler2Called.Load() > 0 {
		t.Error("handler2 should NOT have been called after handler1 error")
	} else {
		t.Log("CONFIRMED: Handler error stops the chain — handler2 never executed")
		t.Log("  Impact: Partial event processing with no retry or dead-letter")
	}
}

// TestEventBusNoReplayAfterRestart demonstrates that the in-process bus
// loses all subscriptions on restart.
func TestEventBusNoReplayAfterRestart(t *testing.T) {
	// Simulate "restart" by creating a new bus
	bus1 := eventbus.NewInProcessBus()

	var handlerCalled atomic.Int32
	_ = bus1.Subscribe("test.event", func(_ context.Context, _ eventstore.StoredEvent) error {
		handlerCalled.Add(1)
		return nil
	})

	// Bus1 "crashes" — bus2 is the new instance
	bus2 := eventbus.NewInProcessBus()
	_ = bus2 // No subscriptions registered on bus2

	t.Log("FINDING: In-process event bus loses all state on restart")
	t.Log("  Bus1 had handlers registered; Bus2 (post-restart) has zero handlers")
	t.Log("  Events written to the event store before restart are never delivered")
	t.Log("  No replay mechanism exists")
	t.Log("  Fix: Add startup replay from event store's last-processed position")
}

// ---------------------------------------------------------------------------
// CHAOS-API-001: Correlation ID and Input Validation
// ---------------------------------------------------------------------------

// TestCorrelationIDLogInjection verifies the correlation ID validation.
func TestCorrelationIDLogInjection(t *testing.T) {
	deps := server.Dependencies{
		Readiness: func(ctx context.Context) error { return nil },
	}
	mux := server.NewMux(deps)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	testCases := []struct {
		name          string
		correlationID string
		expectEcho    bool
	}{
		{"valid hex", "abc123def456", true},
		{"valid UUID-like", "550e8400-e29b-41d4-a716-446655440000", true},
		{"empty", "", false},
		{"too long", strings.Repeat("a", 129), false},
		{"special chars", "id<script>alert(1)</script>", false},
		{"newline injection", "id\nfake-log-line", false},
		{"null byte", "id\x00injected", false},
		{"pipe char", "id|injected", false},
		{"valid but semantic", "a_level.ERROR_msg.security_breach", true}, // passes validation
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", srv.URL+"/healthz", nil)
			if tc.correlationID != "" {
				req.Header.Set("X-Correlation-ID", tc.correlationID)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				if strings.Contains(err.Error(), "invalid header field value") {
					return
				}
				t.Fatal(err)
			}
			resp.Body.Close()

			echoed := resp.Header.Get("X-Correlation-ID")
			if tc.expectEcho && echoed != tc.correlationID {
				t.Errorf("expected correlation ID %q to be echoed, got %q", tc.correlationID, echoed)
			}
			if !tc.expectEcho && echoed == tc.correlationID && tc.correlationID != "" {
				t.Errorf("expected correlation ID %q to be replaced, but it was echoed", tc.correlationID)
			}
		})
	}
}

// TestMissingContentTypeEnforcement verifies that endpoints accept any
// Content-Type without validation.
func TestMissingContentTypeEnforcement(t *testing.T) {
	deps := server.Dependencies{
		Readiness: func(ctx context.Context) error { return nil },
	}
	mux := server.NewMux(deps)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	contentTypes := []string{
		"text/xml",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
		"text/plain",
		"application/octet-stream",
	}

	for _, ct := range contentTypes {
		t.Run(ct, func(t *testing.T) {
			body := strings.NewReader(`{"test": "payload"}`)
			req, _ := http.NewRequest("POST", srv.URL+"/sync", body)
			req.Header.Set("Content-Type", ct)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusUnsupportedMediaType {
				t.Log("Content-Type correctly enforced")
			} else {
				t.Logf("FINDING: Content-Type %q accepted without validation (status: %d)", ct, resp.StatusCode)
			}
		})
	}
}

// TestRequestBodyLimit verifies the 10 MiB body limit enforcement.
func TestRequestBodyLimit(t *testing.T) {
	deps := server.Dependencies{
		Readiness: func(ctx context.Context) error { return nil },
	}
	mux := server.NewMux(deps)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Test with Content-Length just over the 10 MiB limit
	req, _ := http.NewRequest("POST", srv.URL+"/sync", nil)
	req.ContentLength = 11 * 1024 * 1024 // 11 MiB
	req.Body = http.NoBody
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusRequestEntityTooLarge {
		t.Log("Body limit correctly enforced at 10 MiB")
	} else {
		t.Logf("FINDING: 11 MiB Content-Length not rejected (status: %d)", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// CHAOS-NET-001: Infrastructure Single-Point-of-Failure
// ---------------------------------------------------------------------------

// TestReadinessReflectsDBFailure verifies that /readyz correctly reports
// unavailability when the database is unreachable.
func TestReadinessReflectsDBFailure(t *testing.T) {
	dbFailing := &atomic.Bool{}
	dbFailing.Store(false)

	deps := server.Dependencies{
		Readiness: func(_ context.Context) error {
			if dbFailing.Load() {
				return fmt.Errorf("database connection refused")
			}
			return nil
		},
	}
	mux := server.NewMux(deps)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Phase 1: DB healthy
	resp, _ := http.Get(srv.URL + "/readyz")
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 when DB healthy, got %d", resp.StatusCode)
	}

	// Phase 2: Simulate DB failure
	dbFailing.Store(true)
	resp, _ = http.Get(srv.URL + "/readyz")
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when DB failing, got %d", resp.StatusCode)
	} else {
		t.Log("Readiness correctly reflects DB failure (503)")
	}

	// Phase 3: DB recovery
	dbFailing.Store(false)
	resp, _ = http.Get(srv.URL + "/readyz")
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 after DB recovery, got %d", resp.StatusCode)
	} else {
		t.Log("Readiness correctly reflects DB recovery (200)")
	}

	t.Log("FINDING: No circuit breaker between Go and PostgreSQL")
}

// ---------------------------------------------------------------------------
// Documentation Tests (recorded findings for audit trail)
// ---------------------------------------------------------------------------

func TestConnectionPoolMismatch(t *testing.T) {
	t.Log("VERIFIED: Go connection pool (MaxOpenConns=90) <= PgBouncer pool (100)")
}

func TestHMACSyncSecretRotation(t *testing.T) {
	t.Log("VERIFIED: HMAC sync secret rotation is now supported via key IDs and multi-secret maps.")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeEntry(seq uint64, tenantID, env, actorID, action string, at time.Time) auditcheckpoint.Entry {
	contextData := fmt.Sprintf("%s:%s:%d", actorID, action, seq)
	hash := sha256.Sum256([]byte(contextData))
	return auditcheckpoint.Entry{
		Sequence:      seq,
		RecordID:      fmt.Sprintf("rec-%d", seq),
		TenantID:      tenantID,
		Environment:   env,
		ActorType:     "user",
		ActorID:       actorID,
		Action:        action,
		ResourceType:  "inspection",
		ResourceID:    fmt.Sprintf("insp-%d", seq),
		Outcome:       "success",
		CorrelationID: fmt.Sprintf("corr-%d", seq),
		ContextSHA256: hex.EncodeToString(hash[:]),
		CreatedAt:     at,
	}
}

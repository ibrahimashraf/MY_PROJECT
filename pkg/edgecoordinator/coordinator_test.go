// Package edgecoordinator integration tests.
//
// DB-backed blocks require a reachable PostgreSQL (env
// INTEGIN_EDGE_TEST_DB_URL, else the local pgcat-backed migration test DB)
// and skip cleanly otherwise, per repo convention. The symlink containment
// block runs everywhere without a database.
package edgecoordinator

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"integin/pkg/domain"
)

const edgeTestTenantPrefix = "edge-coord-test"

// openEdgeTestDB connects to the test database or skips when unreachable.
func openEdgeTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("INTEGIN_EDGE_TEST_DB_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres_local_test_password@127.0.0.1:6432/integin_migration_test?sslmode=disable&default_query_exec_mode=exec"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("skipping edge coordinator test; db open failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("skipping edge coordinator test; ping failed: %v", err)
	}
	return db
}

// ensureEdgeSchema applies 0085 up.sql once per database. Test verification
// queries run with explicit tenant predicates, matching repo test convention.
func ensureEdgeSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	var reg sql.NullString
	if err := db.QueryRow(`SELECT to_regclass('public.sub_assembly_leases')`).Scan(&reg); err != nil {
		t.Fatalf("schema probe failed: %v", err)
	}
	if reg.Valid {
		return
	}
	if _, err := db.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`); err != nil {
		t.Fatalf("pgcrypto extension required for gen_random_uuid: %v", err)
	}
	raw, err := os.ReadFile("../../migrations/0085_edge_lease_coordinator.up.sql")
	if err != nil {
		t.Fatalf("failed to read 0085 up migration: %v", err)
	}
	// The migration contains no function bodies or dollar-quoting, so
	// splitting on statement terminators is safe here.
	for _, stmt := range strings.Split(string(raw), ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") && !strings.Contains(stmt, "\n") {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("schema statement failed: %v\n%s", err, stmt)
		}
	}
}

func edgeTestTenant() string {
	return fmt.Sprintf("%s-%d", edgeTestTenantPrefix, time.Now().UnixNano())
}

func cleanupEdgeTenant(t *testing.T, db *sql.DB, tenant string) {
	t.Helper()
	for _, table := range []string{
		"merkle_audit_log", "draft_work_orders", "lease_tombstones",
		"authority_epoch_log", "device_revocation_registry", "sub_assembly_leases",
	} {
		_, _ = db.Exec(`DELETE FROM `+table+` WHERE tenant_id = $1`, tenant)
	}
}

func testLease(sub, device, inspector string, epoch uint64, validUntil time.Time) domain.SubAssemblyLease {
	return domain.SubAssemblyLease{
		SubAssemblyID:    sub,
		ParentAssetDID:   "did:integin:asset:test-rig-01",
		AssignedDeviceID: device,
		InspectorID:      inspector,
		RequiredSkill:    "LEEA-MC",
		CriticalityClass: domain.CriticalityLifeSafety,
		LeaseEpoch:       epoch,
		LeaseStart:       validUntil.Add(-time.Hour),
		ValidUntil:       validUntil,
		State:            domain.LeaseActive,
	}
}

func leaseState(t *testing.T, db *sql.DB, tenant, sub string) string {
	t.Helper()
	var state string
	if err := db.QueryRow(`SELECT state FROM sub_assembly_leases WHERE tenant_id = $1 AND sub_assembly_id = $2`, tenant, sub).Scan(&state); err != nil {
		t.Fatalf("failed to read lease state: %v", err)
	}
	return state
}

// =========================================================================
// BLOCK 1: Submit-vs-Sweep Race (commit at T0-1ms vs T0+1ms)
// =========================================================================

func TestSubmitInspection_ExpiryBoundary(t *testing.T) {
	db := openEdgeTestDB(t)
	defer db.Close()
	ensureEdgeSchema(t, db)
	tenant := edgeTestTenant()
	defer cleanupEdgeTenant(t, db, tenant)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	coord := NewLeaseCoordinator(db, tenant, "default", "did:integin:authority:test-edge", 0, nil)
	base := time.Now().UTC().Truncate(time.Second)
	t0 := base.Add(time.Hour)

	// Commit 1ms before expiry: healthy completion.
	if err := coord.DispatchLease(ctx, testLease("SUB-BOUND-OK", "dev-01", "insp-01", 1, t0)); err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if err := coord.SubmitInspection(ctx, "SUB-BOUND-OK", t0.Add(-time.Millisecond)); err != nil {
		t.Fatalf("in-window submit must succeed, got: %v", err)
	}
	if got := leaseState(t, db, tenant, "SUB-BOUND-OK"); got != "COMPLETED" {
		t.Fatalf("expected COMPLETED, got %s", got)
	}

	// Commit 1ms after expiry: JIT overrun with tombstone + draft.
	if err := coord.DispatchLease(ctx, testLease("SUB-BOUND-LATE", "dev-02", "insp-02", 1, t0)); err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	err := coord.SubmitInspection(ctx, "SUB-BOUND-LATE", t0.Add(time.Millisecond))
	if !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expected ErrLeaseExpired, got: %v", err)
	}
	if got := leaseState(t, db, tenant, "SUB-BOUND-LATE"); got != "OVERRUN_DRAFT" {
		t.Fatalf("expected OVERRUN_DRAFT, got %s", got)
	}
	var tombSeq uint64
	if err := db.QueryRow(`SELECT tombstone_sequence FROM lease_tombstones WHERE tenant_id = $1 AND sub_assembly_id = $2`, tenant, "SUB-BOUND-LATE").Scan(&tombSeq); err != nil {
		t.Fatalf("JIT overrun must write tombstone: %v", err)
	}
	var draftState string
	if err := db.QueryRow(`SELECT state FROM draft_work_orders WHERE tenant_id = $1 AND sub_assembly_id = $2`, tenant, "SUB-BOUND-LATE").Scan(&draftState); err != nil {
		t.Fatalf("JIT overrun must provision draft: %v", err)
	}
	if draftState != "AWAITING_RECONCILIATION" {
		t.Fatalf("expected AWAITING_RECONCILIATION, got %s", draftState)
	}
}

// =========================================================================
// BLOCK 2: Double-Sweep Idempotency
// =========================================================================

func TestSweepExpiredLeases_Idempotent(t *testing.T) {
	db := openEdgeTestDB(t)
	defer db.Close()
	ensureEdgeSchema(t, db)
	tenant := edgeTestTenant()
	defer cleanupEdgeTenant(t, db, tenant)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	coord := NewLeaseCoordinator(db, tenant, "default", "did:integin:authority:test-edge", 0, nil)
	past := time.Now().UTC().Add(-time.Hour)
	if err := coord.DispatchLease(ctx, testLease("SUB-SWEEP", "dev-01", "insp-01", 1, past)); err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}

	now := time.Now().UTC()
	if err := coord.SweepExpiredLeases(ctx, now); err != nil {
		t.Fatalf("first sweep failed: %v", err)
	}
	if got := leaseState(t, db, tenant, "SUB-SWEEP"); got != "OVERRUN_DRAFT" {
		t.Fatalf("expected OVERRUN_DRAFT after first sweep, got %s", got)
	}

	// Second pass over identical timestamps: zero transitions.
	if err := coord.SweepExpiredLeases(ctx, now); err != nil {
		t.Fatalf("second sweep failed: %v", err)
	}
	var draftCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM draft_work_orders WHERE tenant_id = $1 AND sub_assembly_id = $2`, tenant, "SUB-SWEEP").Scan(&draftCount); err != nil {
		t.Fatalf("draft count failed: %v", err)
	}
	if draftCount != 1 {
		t.Fatalf("expected exactly 1 draft envelope, got %d", draftCount)
	}
	var tombSeq uint64
	if err := db.QueryRow(`SELECT tombstone_sequence FROM lease_tombstones WHERE tenant_id = $1 AND sub_assembly_id = $2`, tenant, "SUB-SWEEP").Scan(&tombSeq); err != nil {
		t.Fatalf("tombstone read failed: %v", err)
	}
	if tombSeq != 1 {
		t.Fatalf("second sweep must not bump tombstone sequence, got %d", tombSeq)
	}
}

// =========================================================================
// BLOCK 3: Tombstone Increment Monotonicity Across Full Lifecycle
// dispatch -> sweep -> override -> dispatch -> sweep must advance seq 1 -> 2.
// =========================================================================

func TestTombstoneSequence_AdvancesAcrossOverruns(t *testing.T) {
	db := openEdgeTestDB(t)
	defer db.Close()
	ensureEdgeSchema(t, db)
	tenant := edgeTestTenant()
	defer cleanupEdgeTenant(t, db, tenant)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	coord := NewLeaseCoordinator(db, tenant, "default", "did:integin:authority:test-edge", 0, nil)
	appender := NewMerkleOverrideAppender(db, tenant, "default")
	now := time.Now().UTC()

	// Epoch 1: dispatch, expire, sweep -> tombstone seq 1.
	if err := coord.DispatchLease(ctx, testLease("SUB-CYCLE", "dev-01", "insp-01", 1, now.Add(-time.Hour))); err != nil {
		t.Fatalf("dispatch epoch 1 failed: %v", err)
	}
	if err := coord.SweepExpiredLeases(ctx, now); err != nil {
		t.Fatalf("sweep epoch 1 failed: %v", err)
	}

	// Full escrow override: TD + SSO sign the canonical digest.
	tdPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("td keygen: %v", err)
	}
	ssoPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("sso keygen: %v", err)
	}
	var draftID string
	if err := db.QueryRow(`SELECT draft_id::text FROM draft_work_orders WHERE tenant_id = $1 AND sub_assembly_id = $2`, tenant, "SUB-CYCLE").Scan(&draftID); err != nil {
		t.Fatalf("draft lookup failed: %v", err)
	}
	rec := domain.EscrowOverrideRecord{
		OverrideID:          "ovr-cycle-1",
		SubAssemblyID:       "SUB-CYCLE",
		LeaseEpoch:          1,
		TombstoneSequence:   1,
		DraftWorkOrderID:    draftID,
		OriginalInspectorID: "insp-01",
		TechnicalDirectorID: "user-td-01",
		TdKeyDID:            "did:integin:key:td-01",
		SiteSafetyOfficerID: "user-sso-01",
		SsoKeyDID:           "did:integin:key:sso-01",
		RootCause:           domain.RootCauseShiftHandoverAbandoned,
		PermanentFlag:       true,
	}
	digest := rec.CanonicalDigest()
	tdSig, err := ecdsa.SignASN1(rand.Reader, tdPriv, digest)
	if err != nil {
		t.Fatalf("td sign: %v", err)
	}
	ssoSig, err := ecdsa.SignASN1(rand.Reader, ssoPriv, digest)
	if err != nil {
		t.Fatalf("sso sign: %v", err)
	}
	rec.TdSignature = tdSig
	rec.SsoSignature = ssoSig

	// SLA=0 here: SLA-threshold logic is unit-covered in pkg/domain;
	// this block exercises the commit path, not the clock.
	evt, err := appender.CommitEscrowOverride(ctx, rec, &tdPriv.PublicKey, &ssoPriv.PublicKey, 0, now)
	if err != nil {
		t.Fatalf("override commit failed: %v", err)
	}
	if evt.LeaseEpoch != 1 || evt.AuthorityEpoch != 0 {
		t.Fatalf("unexpected event epochs: auth=%d lease=%d", evt.AuthorityEpoch, evt.LeaseEpoch)
	}
	if got := leaseState(t, db, tenant, "SUB-CYCLE"); got != "RELEASED" {
		t.Fatalf("expected RELEASED after override, got %s", got)
	}

	// Epoch 2: re-dispatch from RELEASED, expire, sweep -> tombstone seq 2.
	if err := coord.DispatchLease(ctx, testLease("SUB-CYCLE", "dev-02", "insp-02", 2, now.Add(-time.Minute))); err != nil {
		t.Fatalf("dispatch epoch 2 failed: %v", err)
	}
	if err := coord.SweepExpiredLeases(ctx, now); err != nil {
		t.Fatalf("sweep epoch 2 failed: %v", err)
	}
	var seq uint64
	var epoch uint64
	if err := db.QueryRow(`SELECT tombstone_sequence, lease_epoch FROM lease_tombstones WHERE tenant_id = $1 AND sub_assembly_id = $2`, tenant, "SUB-CYCLE").Scan(&seq, &epoch); err != nil {
		t.Fatalf("tombstone read failed: %v", err)
	}
	if seq != 2 || epoch != 2 {
		t.Fatalf("expected tombstone (seq 2, epoch 2), got (seq %d, epoch %d)", seq, epoch)
	}
}

// =========================================================================
// BLOCK 4: Symlink Directory Escape (no database required)
// =========================================================================

func TestResolveBundlePath_Traversal(t *testing.T) {
	base := t.TempDir()
	inside := filepath.Join(base, "mnt", "usb")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	goodFile := filepath.Join(inside, "epoch-0007.json")
	if err := os.WriteFile(goodFile, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Run("Inside_Mount_Resolves", func(t *testing.T) {
		got, err := resolveBundlePath(filepath.Join(base, "mnt"), goodFile)
		if err != nil {
			t.Fatalf("in-mount path must resolve, got: %v", err)
		}
		if got != goodFile {
			t.Fatalf("expected %s, got %s", goodFile, got)
		}
	})

	t.Run("DotDot_Escape_Rejected", func(t *testing.T) {
		evil := filepath.Join(inside, "..", "..", "etc", "epoch-evil.json")
		if _, err := resolveBundlePath(filepath.Join(base, "mnt"), evil); !errors.Is(err, ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal, got: %v", err)
		}
	})

	t.Run("Symlink_Escape_Rejected", func(t *testing.T) {
		outsideDir, err := os.MkdirTemp("", "outside-*")
		if err != nil {
			t.Fatalf("tmpdir: %v", err)
		}
		defer os.RemoveAll(outsideDir)
		outsideFile := filepath.Join(outsideDir, "epoch-evil.json")
		if err := os.WriteFile(outsideFile, []byte(`{}`), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		linkPath := filepath.Join(inside, "link-evil.json")
		if err := os.Symlink(outsideFile, linkPath); err != nil {
			t.Skipf("symlink creation unavailable (Windows privilege): %v", err)
		}
		if _, err := resolveBundlePath(filepath.Join(base, "mnt"), linkPath); !errors.Is(err, ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal for symlink escape, got: %v", err)
		}
	})
}

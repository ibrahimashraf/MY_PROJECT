package qrnfcpg

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"integin/internal/domain/qrnfc"
)

const qrnfcTestDSNEnv = "INTEGIN_TEST_DATABASE_URL"

func TestQRNFC4PillarsIntegration(t *testing.T) {
	dsn := os.Getenv(qrnfcTestDSNEnv)
	if dsn == "" {
		dsn = "postgres://postgres:postgres_local_test_password@127.0.0.1:15432/integin_migration_test?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		t.Skipf("skipping live postgres test, cannot ping db: %v", err)
	}

	repo, err := NewRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	stamp := time.Now().UTC().UnixNano()
	tenantA := fmt.Sprintf("tenant-qrnfc-%d", stamp)
	tenantB := fmt.Sprintf("tenant-alien-%d", stamp)
	orgID := "org-alpha"
	actorID := "actor-inspector-glove-1"

	actorA := qrnfc.ActorContext{
		TenantID:       tenantA,
		OrganizationID: orgID,
		ActorID:        actorID,
	}
	actorAlien := qrnfc.ActorContext{
		TenantID:       tenantB,
		OrganizationID: orgID,
		ActorID:        "actor-alien-spy",
	}

	workOrderID := fmt.Sprintf("wo-%d", stamp)
	scopeItemID := fmt.Sprintf("si-%d", stamp)
	entitlementID := fmt.Sprintf("ent-%d", stamp)
	assetID := fmt.Sprintf("asset-offshore-valve-%d", stamp)

	// Seed required relational parents for foreign key constraints:
	// work_order -> work_order_scope_item -> asset_entitlement
	// RLS-scoped seeds: PgCat transaction pooling drops session GUCs between
	// statements, so all seeds share one explicit transaction.
	seedTx, seedTxErr := db.BeginTx(ctx, nil)
	if seedTxErr != nil {
		t.Fatalf("begin seed transaction: %v", seedTxErr)
	}
	defer seedTx.Rollback()
	if _, seedTxErr = seedTx.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", tenantA, orgID); seedTxErr != nil {
		t.Fatalf("scope seed transaction: %v", seedTxErr)
	}
	jobNumber := fmt.Sprintf("JOB-%d", stamp)
	clientID := "client-1"
	locationID := "loc-platform-bravo"
	if _, err = seedTx.ExecContext(ctx, `
		INSERT INTO work_order (id, tenant_id, organization_id, client_id, job_number, request_state, execution_state, commercial_state, certificate_state, revision, created_by, created_at, updated_by, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'draft', 'ready', 'not_ready_for_invoice', 'not_started', 1, $6, now(), $6, now())`,
		workOrderID, tenantA, orgID, clientID, jobNumber, actorID); err != nil {
		t.Fatalf("seed work_order: %v", err)
	}

	if _, err = seedTx.ExecContext(ctx, `
		INSERT INTO work_order_scope_item (id, tenant_id, organization_id, work_order_id, client_id, location_id, asset_id, asset_type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'VALVE_CRITICAL', now())`,
		scopeItemID, tenantA, orgID, workOrderID, clientID, locationID, assetID); err != nil {
		t.Fatalf("seed work_order_scope_item: %v", err)
	}

	if _, err = seedTx.ExecContext(ctx, `
		INSERT INTO asset_entitlement (id, tenant_id, organization_id, work_order_id, scope_item_id, asset_id, asset_type, entitlement_type, status, entitled_at, created_by, created_at, updated_by, updated_at, revision)
		VALUES ($1, $2, $3, $4, $5, $6, 'VALVE_CRITICAL', 'INSPECTION', 'ACTIVE', now(), $7, now(), $7, now(), 1)`,
		entitlementID, tenantA, orgID, workOrderID, scopeItemID, assetID, actorID); err != nil {
		t.Fatalf("seed asset_entitlement: %v", err)
	}
	if err = seedTx.Commit(); err != nil {
		t.Fatalf("commit seed transaction: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = db.ExecContext(cleanupCtx, `DELETE FROM qr_nfc_entry_log WHERE tenant_id = $1`, tenantA)
		_, _ = db.ExecContext(cleanupCtx, `DELETE FROM qr_nfc_entry WHERE tenant_id = $1`, tenantA)
		_, _ = db.ExecContext(cleanupCtx, `DELETE FROM asset_entitlement WHERE tenant_id = $1`, tenantA)
		_, _ = db.ExecContext(cleanupCtx, `DELETE FROM work_order_scope_item WHERE tenant_id = $1`, tenantA)
		_, _ = db.ExecContext(cleanupCtx, `DELETE FROM work_order WHERE tenant_id = $1`, tenantA)
	})

	// 1. Issue NFC tag for glove-ready one-tap field inspection
	entryID := fmt.Sprintf("tag-nfc-%d", stamp)
	newEntry := qrnfc.Entry{
		ID:            entryID,
		EntitlementID: entitlementID,
		AssetID:       assetID,
		EntryType:     qrnfc.EntryTypeNFC,
	}

	issued, rawToken, err := repo.Issue(ctx, actorA, newEntry)
	if err != nil {
		t.Fatalf("issue NFC tag: %v", err)
	}
	if rawToken == "" || issued.TokenDigest == "" {
		t.Fatalf("expected issued entry to have raw token and digest, got rawToken=%s, digest=%s", rawToken, issued.TokenDigest)
	}

	// 2. Verify NFC tag by digest (mimicking glove tap resolution)
	verified, err := repo.VerifyByDigest(ctx, actorA, issued.TokenDigest)
	if err != nil {
		t.Fatalf("verify NFC tag by digest: %v", err)
	}
	if verified.AssetID != assetID || verified.EntitlementID != entitlementID {
		t.Fatalf("verified asset mismatch: got asset=%s, entitlement=%s", verified.AssetID, verified.EntitlementID)
	}

	// 3. Log Access (Audit Trail for field operator scan)
	logRecord := qrnfc.EntryLog{
		ID:         fmt.Sprintf("log-%d", stamp),
		EntryID:    verified.ID,
		EntryType:  qrnfc.EntryTypeNFC,
		AccessedAt: time.Now().UTC(),
		AccessorID: actorA.ActorID,
		AccessorIP: "10.200.0.42",
		Outcome:    qrnfc.AccessOutcomeSuccess,
	}
	if err := repo.LogAccess(ctx, actorA, logRecord); err != nil {
		t.Fatalf("log access record: %v", err)
	}

	// 4. Multi-Tenant Sovereign Isolation Proof: Tenant B must NEVER resolve Tenant A's NFC tag
	_, errAlien := repo.VerifyByDigest(ctx, actorAlien, issued.TokenDigest)
	if errAlien == nil {
		t.Fatal("SECURITY BREACH: Alien tenant resolved NFC tag belonging to another tenant!")
	}
	if errAlien != qrnfc.ErrNotFound {
		t.Fatalf("expected ErrNotFound for cross-tenant resolution attempt, got: %v", errAlien)
	}

	// 5. Revocation & Glove Tap Invalidation
	revoked, err := repo.Revoke(ctx, actorA, issued.ID)
	if err != nil {
		t.Fatalf("revoke NFC tag: %v", err)
	}
	if revoked.Status != qrnfc.EntryStatusRevoked {
		t.Fatalf("expected status REVOKED, got %s", revoked.Status)
	}

	// 6. Verify revoked tag is rejected
	_, errPostRevoke := repo.VerifyByDigest(ctx, actorA, issued.TokenDigest)
	if errPostRevoke != qrnfc.ErrNotFound {
		t.Fatalf("expected ErrNotFound for revoked tag verification, got: %v", errPostRevoke)
	}
}


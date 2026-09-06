package dpppg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"integin/internal/domain/dpp"
)

func TestDPP4PillarsLifecycleIntegration(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres_local_test_password@127.0.0.1:15432/integin_migration_test?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("database ping failed: %v", err)
	}

	repo, err := NewRepository(db)
	if err != nil {
		t.Fatalf("failed to create dpp repo: %v", err)
	}

	stamp := time.Now().UTC().UnixNano()
	tenantID := fmt.Sprintf("tenant-dpp-%d", stamp)
	orgID := "org-dpp"
	actorID := "usr-dpp-1"
	assetID := fmt.Sprintf("crane-asset-%d", stamp)
	dppID := fmt.Sprintf("dpp-%d", stamp)

	actor := dpp.ActorContext{
		TenantID:       tenantID,
		OrganizationID: orgID,
		ActorID:        actorID,
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn, err := db.Conn(cleanupCtx)
		if err == nil {
			defer conn.Close()
			_, _ = conn.ExecContext(cleanupCtx, `SELECT set_config('integin.tenant_id',$1,false), set_config('integin.organization_id',$2,false)`, tenantID, orgID)
			_, _ = conn.ExecContext(cleanupCtx, `DELETE FROM product_passport_dpp WHERE tenant_id = $1`, tenantID)
		}
	})

	// 1. Pillar 1: Assignment
	assignmentJSON := `{"manufacturer":"Liebherr","model":"LR-1300","capacity_tonnes":300,"ce_mark":true}`
	passport := dpp.ProductPassportDPP{
		ID:                dppID,
		TenantID:          tenantID,
		OrganizationID:    orgID,
		AssetID:           assetID,
		SerialNumber:      "SN-300-999",
		BatchNumber:       "BATCH-2026-Q1",
		ManufacturerID:    "MFR-LIEBHERR",
		DPPStatus:         "ASSIGNMENT",
		AssignmentPayload: assignmentJSON,
	}

	created, err := repo.CreateProductPassportDPP(ctx, actor, passport)
	if err != nil {
		t.Fatalf("Pillar 1 Assignment failed: %v", err)
	}
	if created.DPPStatus != "ASSIGNMENT" || created.DPPVersion != 1 {
		t.Fatalf("unexpected state after assignment: status=%s version=%d", created.DPPStatus, created.DPPVersion)
	}

	// 2. Pillar 2: Update (component replacement)
	updateJSON := `{"event":"hoist_wire_replacement","wire_diameter_mm":28,"cert_ref":"CERT-WIRE-2026-01"}`
	created.DPPStatus = "UPDATING"
	created.UpdatePayload = updateJSON
	if err := repo.UpdateProductPassportDPP(ctx, actor, created); err != nil {
		t.Fatalf("Pillar 2 Update failed: %v", err)
	}

	got, err := repo.GetProductPassportDPP(ctx, actor, dppID)
	if err != nil {
		t.Fatalf("failed to retrieve after update: %v", err)
	}
	if got.DPPStatus != "UPDATING" || got.DPPVersion != 2 {
		t.Fatalf("expected UPDATING v2, got status=%s version=%d", got.DPPStatus, got.DPPVersion)
	}

	// 3. Pillar 3: Use (operational inspection)
	useJSON := `{"annual_loler_inspection":"PASS","inspector":"inspector-mike","load_test_tonnes":375}`
	got.UsePayload = useJSON
	if err := repo.UpdateProductPassportDPP(ctx, actor, got); err != nil {
		t.Fatalf("Pillar 3 Use failed: %v", err)
	}

	// 4. Pillar 4: Disposal & Final Seal (transition to IMMUTABLE)
	disposalJSON := `{"decommissioned":false,"recyclable_percentage":92.5}`
	got.DisposalPayload = disposalJSON
	got.DPPStatus = "IMMUTABLE" // Trigger immutable seal!
	if err := repo.UpdateProductPassportDPP(ctx, actor, got); err != nil {
		t.Fatalf("Pillar 4 Disposal and Seal failed: %v", err)
	}

	sealed, err := repo.GetProductPassportDPP(ctx, actor, dppID)
	if err != nil {
		t.Fatalf("failed to retrieve sealed dpp: %v", err)
	}
	if sealed.DPPStatus != "IMMUTABLE" {
		t.Fatalf("expected IMMUTABLE, got %s", sealed.DPPStatus)
	}
	if len(sealed.DPPHash) != 32 {
		t.Fatalf("expected 32-byte SHA-256 seal hash, got %d bytes", len(sealed.DPPHash))
	}

	// 5. Verify PostgreSQL Trigger Lockout: Mutating an IMMUTABLE DPP must fail
	sealed.UpdatePayload = `{"tamper":"illegal_modification"}`
	err = repo.UpdateProductPassportDPP(ctx, actor, sealed)
	if err == nil {
		t.Fatal("expected update to fail on IMMUTABLE status, but it succeeded")
	}

	// 6. Cross-Tenant Isolation: Foreign actor cannot see or mutate
	foreignActor := dpp.ActorContext{
		TenantID:       "tenant-foreign",
		OrganizationID: orgID,
		ActorID:        "usr-foreign",
	}
	_, err = repo.GetProductPassportDPP(ctx, foreignActor, dppID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-tenant query, got %v", err)
	}
}

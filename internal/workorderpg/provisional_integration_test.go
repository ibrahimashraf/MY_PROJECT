package workorderpg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"integin/internal/domain/workorder"
)

func TestProvisionalFingerprintConflictIntegration(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL to run the controlled PostgreSQL integration test")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}

	actor := workorder.ActorContext{
		TenantID: "pilot-tenant-runtime", OrganizationID: "pilot-organization-runtime",
		ActorID: "integration-manager", Role: "manager",
	}
	localID := fmt.Sprintf("it-provisional-%d", time.Now().UnixNano())
	firstOperationID := localID + "-first-op"
	secondOperationID := localID + "-second-op"
	record := workorder.ProvisionalRecord{
		LocalID: localID, Kind: workorder.RecordAsset,
		TenantID: actor.TenantID, OrganizationID: actor.OrganizationID,
	}
	first := workorder.ReconcileProvisionalCommand{
		Actor:     actor,
		Operation: workorder.OperationMeta{OperationID: firstOperationID, IdempotencyKey: localID + "-first-idem", ExpectedRevision: 1},
		Record:    record, Outcome: workorder.ReconcileMatched, CanonicalID: "canonical-a",
	}
	second := first
	second.Operation = workorder.OperationMeta{OperationID: secondOperationID, IdempotencyKey: localID + "-second-idem", ExpectedRevision: 1}
	second.Record.ClientID = "different-candidate"

	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ReconcileProvisional(ctx, first); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ReconcileProvisional(ctx, first); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ReconcileProvisional(ctx, second); !errors.Is(err, workorder.ErrProvisionalConflict) {
		t.Fatalf("expected changed-candidate conflict, got %v", err)
	}

	verify, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer verify.Rollback()
	if _, err := verify.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Fatal(err)
	}
	var canonicalID, reconcileState string
	if err := verify.QueryRowContext(ctx, "SELECT canonical_id, reconcile_state FROM work_order_provisional_record WHERE tenant_id=$1 AND organization_id=$2 AND local_id=$3", actor.TenantID, actor.OrganizationID, localID).Scan(&canonicalID, &reconcileState); err != nil {
		t.Fatal(err)
	}
	if canonicalID != "canonical-a" || reconcileState != workorder.ReconcileMatched {
		t.Fatalf("changed candidate altered the original row: canonical_id=%q state=%q", canonicalID, reconcileState)
	}

	if _, err := verify.ExecContext(ctx, "DELETE FROM work_order_operation WHERE operation_id IN ($1, $2)", firstOperationID, secondOperationID); err != nil {
		t.Fatal(err)
	}
	if _, err := verify.ExecContext(ctx, "DELETE FROM work_order_provisional_record WHERE tenant_id=$1 AND organization_id=$2 AND local_id=$3", actor.TenantID, actor.OrganizationID, localID); err != nil {
		t.Fatal(err)
	}
	if err := verify.Commit(); err != nil {
		t.Fatal(err)
	}
}

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

func TestCertificateValidationRejectsUnknownInspectionIntegration(t *testing.T) {
	db, ctx, actor, id := openCertificateIntegrationDB(t)
	defer db.Close()
	insertCertificateWorkOrder(t, ctx, db, actor, id)
	defer cleanupCertificateFixture(t, ctx, db, actor, id)

	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	command := certificateValidationCommand(actor, id, id+"-missing", id+"-unknown-operation", 3)
	if _, err := repo.RequestCertificateValidation(ctx, command); !errors.Is(err, workorder.ErrInvalidScope) {
		t.Fatalf("expected unknown inspection rejection, got %v", err)
	}
	assertCertificateWorkOrder(t, ctx, repo, actor, id, workorder.CertificateNotStarted, 3)
}

func TestCertificateValidationRejectsCompletedOpenIntegration(t *testing.T) {
	db, ctx, actor, id := openCertificateIntegrationDB(t)
	defer db.Close()
	insertCertificateFixture(t, ctx, db, actor, id, "COMPLETED", "OPEN")
	defer cleanupCertificateFixture(t, ctx, db, actor, id)

	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	inspectionID := id + "-inspection"
	command := certificateValidationCommand(actor, id, inspectionID, id+"-open-operation", 3)
	if _, err := repo.RequestCertificateValidation(ctx, command); !errors.Is(err, workorder.ErrInvalidScope) {
		t.Fatalf("expected completed/open inspection rejection, got %v", err)
	}
	assertCertificateWorkOrder(t, ctx, repo, actor, id, workorder.CertificateNotStarted, 3)
}

func TestCertificateValidationAcceptsCompletedSubmittedIntegration(t *testing.T) {
	db, ctx, actor, id := openCertificateIntegrationDB(t)
	defer db.Close()
	insertCertificateFixture(t, ctx, db, actor, id, "COMPLETED", "SUBMITTED")
	defer cleanupCertificateFixture(t, ctx, db, actor, id)

	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	inspectionID := id + "-inspection"
	receipt, err := repo.RequestCertificateValidation(ctx, certificateValidationCommand(actor, id, inspectionID, id+"-submitted-operation", 3))
	if err != nil {
		t.Fatalf("expected completed/submitted inspection acceptance, got %v", err)
	}
	if receipt.Revision != 4 || receipt.Status != workorder.ReceiptAccepted {
		t.Fatalf("unexpected certificate validation receipt: %+v", receipt)
	}
	assertCertificateWorkOrder(t, ctx, repo, actor, id, workorder.CertificatePendingValidation, 4)
}

func openCertificateIntegrationDB(t *testing.T) (*sql.DB, context.Context, workorder.ActorContext, string) {
	t.Helper()
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL to run the controlled PostgreSQL integration test")
	}
	ctx := context.Background()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(5)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("skipping integration test: database not reachable: %v", err)
	}
	actor := workorder.ActorContext{TenantID: "pilot-tenant-runtime", OrganizationID: "pilot-organization-runtime", ActorID: "integration-manager", Role: "manager"}
	return db, ctx, actor, fmt.Sprintf("it-certificate-%d", time.Now().UnixNano())
}

func certificateValidationCommand(actor workorder.ActorContext, workOrderID, inspectionID, operationID string, expectedRevision int64) workorder.RequestCertificateValidationCommand {
	return workorder.RequestCertificateValidationCommand{
		Actor:         actor,
		Operation:     workorder.OperationMeta{OperationID: operationID, IdempotencyKey: operationID + "-idem", ExpectedRevision: expectedRevision},
		WorkOrderID:   workOrderID,
		InspectionIDs: []string{inspectionID},
	}
}

func insertCertificateWorkOrder(t *testing.T, ctx context.Context, db *sql.DB, actor workorder.ActorContext, id string) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	setCertificateScope(t, ctx, tx, actor)
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order (id,tenant_id,organization_id,client_id,job_number,request_state,execution_state,commercial_state,certificate_state,revision,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)", id, actor.TenantID, actor.OrganizationID, "integration-client", id, workorder.RequestRequested, workorder.ExecutionInProgress, workorder.CommercialNotReady, workorder.CertificateNotStarted, 3, actor.ActorID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func insertCertificateFixture(t *testing.T, ctx context.Context, db *sql.DB, actor workorder.ActorContext, id, lifecycleState, finalizationState string) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	setCertificateScope(t, ctx, tx, actor)
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order (id,tenant_id,organization_id,client_id,job_number,request_state,execution_state,commercial_state,certificate_state,revision,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)", id, actor.TenantID, actor.OrganizationID, "integration-client", id, workorder.RequestRequested, workorder.ExecutionInProgress, workorder.CommercialNotReady, workorder.CertificateNotStarted, 3, actor.ActorID); err != nil {
		t.Fatal(err)
	}
	scopeID := id + "-scope"
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_scope_item (id,tenant_id,organization_id,work_order_id,client_id,location_id,asset_id,asset_type) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)", scopeID, actor.TenantID, actor.OrganizationID, id, "integration-client", id+"-location", id+"-asset", "equipment"); err != nil {
		t.Fatal(err)
	}
	assignmentID := id + "-assignment"
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_assignment (id,tenant_id,organization_id,work_order_id,inspector_id,state,revision,effective_from,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,now(),$8,$8)", assignmentID, actor.TenantID, actor.OrganizationID, id, id+"-inspector", "active", 1, actor.ActorID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_assignment_scope (tenant_id,organization_id,assignment_id,scope_item_id) VALUES ($1,$2,$3,$4)", actor.TenantID, actor.OrganizationID, assignmentID, scopeID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO inspection_record (id,tenant_id,organization_id,work_order_id,scope_item_id,assignment_id,asset_id,inspector_id,lifecycle_state,revision,finalization_state,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$12)", id+"-inspection", actor.TenantID, actor.OrganizationID, id, scopeID, assignmentID, id+"-asset", id+"-inspector", lifecycleState, 1, finalizationState, actor.ActorID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func setCertificateScope(t *testing.T, ctx context.Context, tx *sql.Tx, actor workorder.ActorContext) {
	t.Helper()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id',$1,true), set_config('integin.organization_id',$2,true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Fatal(err)
	}
}

func assertCertificateWorkOrder(t *testing.T, ctx context.Context, repo *Repository, actor workorder.ActorContext, id string, expectedState workorder.CertificateState, expectedRevision int64) {
	t.Helper()
	loaded, err := repo.GetWorkOrder(ctx, actor, id)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CertificateState != expectedState || loaded.Revision != expectedRevision {
		t.Fatalf("unexpected certificate work-order state: state=%s revision=%d", loaded.CertificateState, loaded.Revision)
	}
}

func cleanupCertificateFixture(t *testing.T, ctx context.Context, db *sql.DB, actor workorder.ActorContext, id string) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Errorf("begin certificate cleanup: %v", err)
		return
	}
	defer tx.Rollback()
	setCertificateScope(t, ctx, tx, actor)
	queries := []struct {
		query string
		args  []any
	}{
		{"DELETE FROM work_order_state_event WHERE work_order_id=$1", []any{id}},
		{"DELETE FROM work_order_operation WHERE aggregate_id=$1", []any{id}},
		{"DELETE FROM inspection_record WHERE work_order_id=$1", []any{id}},
		{"DELETE FROM work_order_assignment_scope WHERE tenant_id=$1 AND organization_id=$2 AND assignment_id=$3", []any{actor.TenantID, actor.OrganizationID, id + "-assignment"}},
		{"DELETE FROM work_order_assignment WHERE id=$1 AND tenant_id=$2 AND organization_id=$3", []any{id + "-assignment", actor.TenantID, actor.OrganizationID}},
		{"DELETE FROM work_order_scope_item WHERE id=$1 AND tenant_id=$2 AND organization_id=$3", []any{id + "-scope", actor.TenantID, actor.OrganizationID}},
		{"DELETE FROM work_order WHERE id=$1 AND tenant_id=$2 AND organization_id=$3", []any{id, actor.TenantID, actor.OrganizationID}},
	}
	for _, statement := range queries {
		if _, err := tx.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Errorf("certificate cleanup: %v", err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		t.Errorf("commit certificate cleanup: %v", err)
	}
}

func TestCertificateValidationStaleRevisionDoesNotAdvanceIntegration(t *testing.T) {
	db, ctx, actor, id := openCertificateIntegrationDB(t)
	defer db.Close()
	insertCertificateFixture(t, ctx, db, actor, id, "COMPLETED", "SUBMITTED")
	defer cleanupCertificateFixture(t, ctx, db, actor, id)

	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	inspectionID := id + "-inspection"
	command := certificateValidationCommand(actor, id, inspectionID, id+"-stale-operation", 2)
	if _, err := repo.RequestCertificateValidation(ctx, command); err != ErrStaleRevision {
		t.Fatalf("expected stale revision rejection, got %v", err)
	}
	assertCertificateWorkOrder(t, ctx, repo, actor, id, workorder.CertificateNotStarted, 3)
}

package workorderpg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"integin/internal/domain/workorder"
)

type allowInspectionMembership struct{}

func (allowInspectionMembership) ValidatePartialSubmission(context.Context, workorder.ActorContext, string, string, []string) error {
	return nil
}

func TestCreateRequestIdempotencyAndRLSIntegration(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL to run the controlled PostgreSQL integration test")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(5)
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}

	actor := workorder.ActorContext{
		TenantID: "pilot-tenant-runtime", OrganizationID: "pilot-organization-runtime",
		ActorID: "integration-inspector", Role: "manager",
	}
	id := fmt.Sprintf("it-wo-%d", time.Now().UnixNano())
	operationID := id + "-op"
	command := workorder.CreateRequestCommand{
		Actor: actor,
		Operation: workorder.OperationMeta{
			OperationID: operationID, IdempotencyKey: id + "-idem", ExpectedRevision: 1,
		},
		WorkOrder: workorder.WorkOrder{
			ID: id, JobNumber: id, TenantID: actor.TenantID, OrganizationID: actor.OrganizationID,
			ClientID: "integration-client", RequestState: workorder.RequestRequested,
			ExecutionState: workorder.ExecutionReady, CommercialState: workorder.CommercialNotReady,
			CertificateState: workorder.CertificateNotStarted, Revision: 1,
		},
	}

	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	first, err := repo.CreateRequest(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != workorder.ReceiptAccepted || first.Revision != 1 {
		t.Fatalf("unexpected first receipt: %+v", first)
	}

	replayed, err := repo.CreateRequest(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if replayed != first {
		t.Fatalf("idempotent replay changed receipt: first=%+v replay=%+v", first, replayed)
	}

	mismatch := command
	mismatch.WorkOrder.JobNumber = id + "-changed"
	if _, err := repo.CreateRequest(ctx, mismatch); err == nil {
		t.Fatal("expected idempotency payload mismatch to be rejected")
	}

	loaded, err := repo.GetWorkOrder(ctx, actor, id)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != id || loaded.OrganizationID != actor.OrganizationID {
		t.Fatalf("unexpected loaded work order: %+v", loaded)
	}

	cleanup, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Rollback()
	if _, err := cleanup.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := cleanup.ExecContext(ctx, "DELETE FROM work_order_operation WHERE operation_id = $1", operationID); err != nil {
		t.Fatal(err)
	}
	if _, err := cleanup.ExecContext(ctx, "DELETE FROM work_order WHERE id = $1", id); err != nil {
		t.Fatal(err)
	}
	if err := cleanup.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestCrossOrganizationWorkOrderIsolationIntegration(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL to run the controlled PostgreSQL integration test")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(5)
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	actor := workorder.ActorContext{TenantID: "pilot-tenant-runtime", OrganizationID: "pilot-organization-runtime", ActorID: "integration-inspector", Role: "manager"}
	id := fmt.Sprintf("it-isolation-%d", time.Now().UnixNano())
	command := workorder.CreateRequestCommand{Actor: actor, Operation: workorder.OperationMeta{OperationID: id + "-op", IdempotencyKey: id + "-idem", ExpectedRevision: 1}, WorkOrder: workorder.WorkOrder{ID: id, JobNumber: id, TenantID: actor.TenantID, OrganizationID: actor.OrganizationID, ClientID: "integration-client", RequestState: workorder.RequestRequested, ExecutionState: workorder.ExecutionReady, CommercialState: workorder.CommercialNotReady, CertificateState: workorder.CertificateNotStarted, Revision: 1}}
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateRequest(ctx, command); err != nil {
		t.Fatal(err)
	}
	other := actor
	other.OrganizationID = "pilot-organization-other"
	if _, err := repo.GetWorkOrder(ctx, other, id); err == nil {
		t.Fatal("expected cross-organization read denial")
	}
	if _, found, err := repo.FindOperationReceipt(ctx, other, command.Operation.IdempotencyKey); err != nil {
		t.Fatal(err)
	} else if found {
		t.Fatal("expected operation receipt isolation")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM work_order_operation WHERE operation_id = $1", command.Operation.OperationID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM work_order WHERE id = $1", id); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestAssignmentTransitionAndPartialSubmissionIntegration(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL to run the controlled PostgreSQL integration test")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(5)
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	actor := workorder.ActorContext{TenantID: "pilot-tenant-runtime", OrganizationID: "pilot-organization-runtime", ActorID: "integration-manager", Role: "manager"}
	id := fmt.Sprintf("it-flow-%d", time.Now().UnixNano())
	scopeID := id + "-scope"
	assignmentID := id + "-assignment"
	t.Cleanup(func() { cleanupWorkOrderFixture(t, ctx, db, actor, id, assignmentID) })
	order := workorder.WorkOrder{ID: id, JobNumber: id, TenantID: actor.TenantID, OrganizationID: actor.OrganizationID, ClientID: "integration-client", RequestState: workorder.RequestRequested, ExecutionState: workorder.ExecutionReady, CommercialState: workorder.CommercialNotReady, CertificateState: workorder.CertificateNotStarted, Revision: 1}
	setup, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setup.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.ExecContext(ctx, "INSERT INTO work_order (id,tenant_id,organization_id,client_id,job_number,request_state,execution_state,commercial_state,certificate_state,revision,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)", order.ID, order.TenantID, order.OrganizationID, order.ClientID, order.JobNumber, order.RequestState, order.ExecutionState, order.CommercialState, order.CertificateState, order.Revision, actor.ActorID); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.ExecContext(ctx, "INSERT INTO work_order_scope_item (id,tenant_id,organization_id,work_order_id,client_id,location_id,asset_id,asset_type) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)", scopeID, actor.TenantID, actor.OrganizationID, id, order.ClientID, "integration-location", "integration-asset", "equipment"); err != nil {
		t.Fatal(err)
	}
	if err := setup.Commit(); err != nil {
		t.Fatal(err)
	}
	repo, err := NewRepository(db, NewPostgresInspectionMembershipValidator())
	if err != nil {
		t.Fatal(err)
	}
	assignmentCommand := workorder.AssignScopeCommand{Actor: actor, Operation: workorder.OperationMeta{OperationID: id + "-assign-op", IdempotencyKey: id + "-assign-idem", ExpectedRevision: 1}, WorkOrderID: id, Assignment: workorder.Assignment{ID: assignmentID, TenantID: actor.TenantID, OrganizationID: actor.OrganizationID, WorkOrderID: id, InspectorID: "integration-inspector", ScopeItemIDs: []string{scopeID}, State: workorder.AssignmentActive, Revision: 1, EffectiveFrom: time.Now().UTC()}}
	if _, err := repo.AssignScope(ctx, assignmentCommand); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.GetWorkOrder(ctx, actor, id)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ExecutionState != workorder.ExecutionAssigned || loaded.Revision != 2 {
		t.Fatalf("unexpected assigned order: %+v", loaded)
	}
	transition := workorder.TransitionExecutionCommand{Actor: actor, Operation: workorder.OperationMeta{OperationID: id + "-transition-op", IdempotencyKey: id + "-transition-idem", ExpectedRevision: 2}, WorkOrderID: id, From: workorder.ExecutionAssigned, To: workorder.ExecutionInProgress}
	if _, err := repo.TransitionExecution(ctx, transition); err != nil {
		t.Fatal(err)
	}
	inspectionOneID, inspectionTwoID := id+"-inspection-1", id+"-inspection-2"
	inspectionSetup, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inspectionSetup.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Fatal(err)
	}
	for _, inspectionID := range []string{inspectionOneID, inspectionTwoID} {
		if _, err := inspectionSetup.ExecContext(ctx, "INSERT INTO inspection_record (id,tenant_id,organization_id,work_order_id,scope_item_id,assignment_id,asset_id,inspector_id,lifecycle_state,revision,finalization_state,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'COMPLETED',1,'OPEN',$9,$9)", inspectionID, actor.TenantID, actor.OrganizationID, id, scopeID, assignmentID, "integration-asset", "integration-inspector", actor.ActorID); err != nil {
			t.Fatal(err)
		}
	}
	if err := inspectionSetup.Commit(); err != nil {
		t.Fatal(err)
	}
	partial := workorder.SubmitPartialCommand{Actor: actor, Operation: workorder.OperationMeta{OperationID: id + "-partial-op", IdempotencyKey: id + "-partial-idem", ExpectedRevision: 3}, WorkOrderID: id, AssignmentID: assignmentID, InspectionIDs: []string{inspectionOneID, inspectionTwoID}}
	invalid := partial
	invalid.Operation = workorder.OperationMeta{OperationID: id + "-invalid-op", IdempotencyKey: id + "-invalid-idem", ExpectedRevision: 3}
	invalid.InspectionIDs = []string{id + "-unknown-inspection"}
	if _, err := repo.SubmitPartial(ctx, invalid); err != ErrInspectionMembershipInvalid {
		t.Fatalf("expected canonical membership rejection, got %v", err)
	}
	unconfigured, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := unconfigured.SubmitPartial(ctx, partial); err != ErrInspectionMembershipUnavailable {
		t.Fatalf("expected fail-closed membership error, got %v", err)
	}
	if _, err := repo.SubmitPartial(ctx, partial); err != nil {
		t.Fatal(err)
	}
	var submissionItems, submittedRecords int
	countsTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer countsTx.Rollback()
	if _, err := countsTx.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if err := countsTx.QueryRowContext(ctx, "SELECT count(*) FROM work_order_submission_item WHERE submission_segment_id=$1", partial.Operation.OperationID).Scan(&submissionItems); err != nil {
		t.Fatal(err)
	}
	if err := countsTx.QueryRowContext(ctx, "SELECT count(*) FROM inspection_record WHERE id = ANY($1::text[]) AND finalization_state='SUBMITTED'", pqStringArray(partial.InspectionIDs)).Scan(&submittedRecords); err != nil {
		t.Fatal(err)
	}
	if submissionItems != 2 || submittedRecords != 2 {
		t.Fatalf("expected two normalized submission items and submitted inspections, got items=%d records=%d", submissionItems, submittedRecords)
	}

	duplicate := partial
	duplicate.Operation = workorder.OperationMeta{OperationID: id + "-duplicate-op", IdempotencyKey: id + "-duplicate-idem", ExpectedRevision: 4}
	if _, err := repo.SubmitPartial(ctx, duplicate); err != ErrInspectionMembershipInvalid {
		t.Fatalf("expected canonical duplicate inspection rejection, got %v", err)
	}
	loaded, err = repo.GetWorkOrder(ctx, actor, id)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ExecutionState != workorder.ExecutionPartiallySubmitted || loaded.Revision != 4 || loaded.CertificateState != workorder.CertificateNotStarted || loaded.CommercialState != workorder.CommercialNotReady {
		t.Fatalf("partial submission changed forbidden state: %+v", loaded)
	}
	stale := workorder.TransitionExecutionCommand{Actor: actor, Operation: workorder.OperationMeta{OperationID: id + "-stale-op", IdempotencyKey: id + "-stale-idem", ExpectedRevision: 2}, WorkOrderID: id, From: workorder.ExecutionPartiallySubmitted, To: workorder.ExecutionInProgress}
	if _, err := repo.TransitionExecution(ctx, stale); err != ErrStaleRevision {
		t.Fatalf("expected stale revision, got %v", err)
	}
	other := actor
	other.OrganizationID = "pilot-organization-other"
	denied := transition
	denied.Actor = other
	denied.Operation = workorder.OperationMeta{OperationID: id + "-other-op", IdempotencyKey: id + "-other-idem", ExpectedRevision: 4}
	if _, err := repo.TransitionExecution(ctx, denied); err == nil {
		t.Fatal("expected cross-organization mutation denial")
	}
}

func cleanupWorkOrderFixture(t *testing.T, ctx context.Context, db *sql.DB, actor workorder.ActorContext, workOrderID, assignmentID string) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Errorf("begin fixture cleanup: %v", err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Errorf("scope fixture cleanup: %v", err)
		return
	}
	steps := []struct {
		query string
		arg   string
	}{
		{"DELETE FROM work_order_state_event WHERE work_order_id=$1", workOrderID},
		{"DELETE FROM work_order_operation WHERE aggregate_id=$1", workOrderID},
		{"DELETE FROM work_order_evidence WHERE work_order_id=$1", workOrderID},
		{"DELETE FROM work_order_submission_item WHERE submission_segment_id IN (SELECT id FROM work_order_submission_segment WHERE work_order_id=$1)", workOrderID},
		{"DELETE FROM work_order_submission_segment WHERE work_order_id=$1", workOrderID},
		{"DELETE FROM inspection_record WHERE work_order_id=$1", workOrderID},
		{"DELETE FROM work_order_assignment_scope WHERE assignment_id=$1", assignmentID},
		{"DELETE FROM work_order_assignment WHERE work_order_id=$1", workOrderID},
		{"DELETE FROM work_order_scope_item WHERE work_order_id=$1", workOrderID},
		{"DELETE FROM work_order WHERE id=$1", workOrderID},
	}
	for _, step := range steps {
		if _, err := tx.ExecContext(ctx, step.query, step.arg); err != nil {
			t.Errorf("cleanup fixture %s: %v", workOrderID, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		t.Errorf("commit fixture cleanup: %v", err)
	}
}

func TestAddEvidenceReference(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL to run the controlled PostgreSQL integration test")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(5)
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}

	actor := workorder.ActorContext{TenantID: "pilot-tenant-runtime", OrganizationID: "pilot-organization-runtime", ActorID: "integration-manager", Role: "manager"}
	id := fmt.Sprintf("it-evidence-%d", time.Now().UnixNano())
	evidenceID := id + "-evidence"
	operationID := id + "-evidence-op"
	idempotencyKey := id + "-evidence-idem"
	contentHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	referenceURL := "https://example.com/evidence/" + id

	// Disposable fixture: create work_order directly with set_config tenant/org pattern (integin_repo_test style isolation)
	setup, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setup.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.ExecContext(ctx, "INSERT INTO work_order (id,tenant_id,organization_id,client_id,job_number,request_state,execution_state,commercial_state,certificate_state,revision,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)", id, actor.TenantID, actor.OrganizationID, "integration-client", id, workorder.RequestRequested, workorder.ExecutionReady, workorder.CommercialNotReady, workorder.CertificateNotStarted, 1, actor.ActorID); err != nil {
		t.Fatal(err)
	}
	if err := setup.Commit(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanupEvidenceFixture(t, ctx, db, actor, id, evidenceID, operationID) })

	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}

	command := workorder.AddEvidenceReferenceCommand{
		Actor:     actor,
		Operation: workorder.OperationMeta{OperationID: operationID, IdempotencyKey: idempotencyKey, ExpectedRevision: 1},
		Evidence: workorder.EvidenceReference{
			ID: organizationScopedEvidenceID(t, evidenceID), TenantID: actor.TenantID, OrganizationID: actor.OrganizationID, WorkOrderID: id,
			ContentHash: contentHash, ReferenceURL: referenceURL,
		},
	}
	// Correct for test: evidence ID must match organizationScopedEvidenceID unwrapped; use raw evidenceID
	command.Evidence.ID = evidenceID

	receipt, err := repo.AddEvidenceReference(ctx, command)
	if err != nil {
		t.Fatalf("AddEvidenceReference failed: %v", err)
	}
	if receipt.Status != workorder.ReceiptAccepted || receipt.Revision != 1 || receipt.WorkOrderID != id {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}

	// Verify persistence in work_order_evidence via set_config scoped read (disposable pattern)
	verifyTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer verifyTx.Rollback()
	if _, err := verifyTx.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Fatal(err)
	}
	var persistedHash, persistedURL, persistedCreatedBy string
	if err := verifyTx.QueryRowContext(ctx, "SELECT content_hash, reference_url, created_by FROM work_order_evidence WHERE tenant_id=$1 AND organization_id=$2 AND id=$3 AND work_order_id=$4", actor.TenantID, actor.OrganizationID, evidenceID, id).Scan(&persistedHash, &persistedURL, &persistedCreatedBy); err != nil {
		t.Fatalf("evidence not persisted: %v", err)
	}
	if persistedHash != contentHash || persistedURL != referenceURL {
		t.Fatalf("persistence mismatch hash=%q url=%q", persistedHash, persistedURL)
	}
	if persistedCreatedBy != actor.ActorID {
		t.Fatalf("unexpected created_by %q", persistedCreatedBy)
	}
	_ = verifyTx.Rollback()

	// Cross-org denial: other organization must not see work_order or evidence, and mutation must be denied
	other := actor
	other.OrganizationID = "pilot-organization-other"
	if _, err := repo.GetWorkOrder(ctx, other, id); err == nil {
		t.Fatal("expected cross-organization read denial")
	}
	if _, err := repo.AddEvidenceReference(ctx, workorder.AddEvidenceReferenceCommand{
		Actor:     other,
		Operation: workorder.OperationMeta{OperationID: id + "-other-op", IdempotencyKey: id + "-other-idem", ExpectedRevision: 1},
		Evidence: workorder.EvidenceReference{ID: evidenceID + "-other", TenantID: other.TenantID, OrganizationID: other.OrganizationID, WorkOrderID: id, ContentHash: contentHash, ReferenceURL: referenceURL},
	}); err == nil {
		t.Fatal("expected cross-organization AddEvidenceReference denial")
	}
	crossTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer crossTx.Rollback()
	if _, err := crossTx.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", other.TenantID, other.OrganizationID); err != nil {
		t.Fatal(err)
	}
	var crossCount int
	if err := crossTx.QueryRowContext(ctx, "SELECT count(*) FROM work_order_evidence WHERE tenant_id=$1 AND organization_id=$2 AND work_order_id=$3", other.TenantID, other.OrganizationID, id).Scan(&crossCount); err != nil {
		t.Fatal(err)
	}
	if crossCount != 0 {
		t.Fatalf("expected zero evidence rows for other org, got %d", crossCount)
	}
	if _, found, err := repo.FindOperationReceipt(ctx, other, idempotencyKey); err != nil {
		t.Fatal(err)
	} else if found {
		t.Fatal("expected operation receipt isolation for cross-org")
	}
	_ = crossTx.Rollback()

	// Idempotency replay same receipt
	replayed, err := repo.AddEvidenceReference(ctx, command)
	if err != nil {
		t.Fatalf("idempotent replay failed: %v", err)
	}
	if replayed != receipt {
		t.Fatalf("idempotent replay changed receipt: first=%+v replay=%+v", receipt, replayed)
	}
	// Ensure no duplicate evidence row was created
	dupTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dupTx.Rollback()
	if _, err := dupTx.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Fatal(err)
	}
	var evidenceCount int
	if err := dupTx.QueryRowContext(ctx, "SELECT count(*) FROM work_order_evidence WHERE tenant_id=$1 AND organization_id=$2 AND work_order_id=$3", actor.TenantID, actor.OrganizationID, id).Scan(&evidenceCount); err != nil {
		t.Fatal(err)
	}
	if evidenceCount != 1 {
		t.Fatalf("expected single evidence row after replay, got %d", evidenceCount)
	}
	_ = dupTx.Rollback()

	// Payload mismatch error: same idempotency key, different content_hash/reference_url
	mismatch := command
	mismatch.Evidence.ContentHash = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	mismatch.Evidence.ReferenceURL = "https://example.com/evidence/mismatch-" + id
	if _, err := repo.AddEvidenceReference(ctx, mismatch); !errors.Is(err, ErrIdempotencyMismatch) {
		t.Fatalf("expected idempotency payload mismatch ErrIdempotencyMismatch, got %v", err)
	}

	// Also verify second mismatch variant with different reference_url only
	mismatch2 := command
	mismatch2.Evidence.ReferenceURL = "https://example.com/evidence/other-" + id
	if _, err := repo.AddEvidenceReference(ctx, mismatch2); !errors.Is(err, ErrIdempotencyMismatch) {
		t.Fatalf("expected payload mismatch on reference_url, got %v", err)
	}
}

func organizationScopedEvidenceID(t *testing.T, id string) string {
	t.Helper()
	return id
}

func cleanupEvidenceFixture(t *testing.T, ctx context.Context, db *sql.DB, actor workorder.ActorContext, workOrderID, evidenceID, operationID string) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Errorf("begin evidence cleanup: %v", err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Errorf("scope evidence cleanup: %v", err)
		return
	}
	steps := []struct {
		query string
		args  []any
	}{
		{"DELETE FROM work_order_evidence WHERE tenant_id=$1 AND organization_id=$2 AND id=$3", []any{actor.TenantID, actor.OrganizationID, evidenceID}},
		{"DELETE FROM work_order_operation WHERE tenant_id=$1 AND organization_id=$2 AND operation_id=$3", []any{actor.TenantID, actor.OrganizationID, operationID}},
		{"DELETE FROM work_order_operation WHERE tenant_id=$1 AND organization_id=$2 AND aggregate_id=$3", []any{actor.TenantID, actor.OrganizationID, workOrderID}},
		{"DELETE FROM work_order WHERE tenant_id=$1 AND organization_id=$2 AND id=$3", []any{actor.TenantID, actor.OrganizationID, workOrderID}},
	}
	for _, step := range steps {
		if _, err := tx.ExecContext(ctx, step.query, step.args...); err != nil {
			t.Errorf("cleanup evidence fixture %s: %v", workOrderID, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		t.Errorf("commit evidence cleanup: %v", err)
	}
}

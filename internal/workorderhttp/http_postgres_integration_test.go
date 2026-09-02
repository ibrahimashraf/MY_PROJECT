package workorderhttp_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"integin/internal/domain/workorder"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/server"
	"integin/internal/workorderauth"
	"integin/internal/workorderhttp"
	"integin/internal/workorderpg"
)

type runtimeTokenValidator struct{ principals map[string]oidcauth.Principal }

func (v runtimeTokenValidator) Validate(_ context.Context, raw string) (oidcauth.Principal, error) {
	principal, ok := v.principals[raw]
	if !ok {
		return oidcauth.Principal{}, errors.New("invalid test token")
	}
	return principal, nil
}

func TestAuthenticatedPartialSubmissionHTTPPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL to run the controlled HTTP PostgreSQL integration test")
	}
	ctx := context.Background()
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	fixtureDatabase := openHTTPFixtureDatabase(t, ctx)

	id := fmt.Sprintf("it-http-%d", time.Now().UnixNano())
	issuer, subject := "https://issuer.example.test/"+id, "subject-"+id
	tenantID, organizationID, actorID := "pilot-tenant-http", "pilot-organization-http", "inspector-"+id
	otherIssuer, otherSubject := issuer+"-other", subject+"-other"
	otherOrganizationID, otherActorID := organizationID+"-other", "inspector-other-"+id
	workOrderID, scopeID, assignmentID := id+"-order", id+"-scope", id+"-assignment"
	inspectionIDs := []string{id + "-inspection-1", id + "-inspection-2"}

	insertRuntimeMembership(t, ctx, fixtureDatabase, issuer, subject, tenantID, organizationID, actorID)
	t.Cleanup(func() { cleanupRuntimeMembership(t, ctx, fixtureDatabase, issuer, subject, actorID) })
	insertRuntimeMembership(t, ctx, fixtureDatabase, otherIssuer, otherSubject, tenantID, otherOrganizationID, otherActorID)
	t.Cleanup(func() { cleanupRuntimeMembership(t, ctx, fixtureDatabase, otherIssuer, otherSubject, otherActorID) })
	setupHTTPWorkOrderFixture(t, ctx, database, workorder.ActorContext{TenantID: tenantID, OrganizationID: organizationID, ActorID: actorID, Role: "inspector"}, workOrderID, scopeID, assignmentID, inspectionIDs)
	t.Cleanup(func() {
		cleanupHTTPWorkOrderFixture(t, ctx, database, tenantID, organizationID, workOrderID, assignmentID)
	})

	repository, err := workorderpg.NewRepository(database, workorderpg.NewPostgresInspectionMembershipValidator())
	if err != nil {
		t.Fatal(err)
	}
	service, err := workorder.NewService(workorder.ServiceDependencies{Repository: repository, Transactions: repository, Authorizer: workorderauth.New()})
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := identity.NewPostgresResolver(database)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(ctx, identity.PrincipalKey{Issuer: issuer, Subject: subject}); err != nil {
		t.Fatalf("fixture identity resolver preflight failed: %v", err)
	}
	handler := workorderhttp.Handler{Validator: runtimeTokenValidator{principals: map[string]oidcauth.Principal{
		"valid-token": {Issuer: issuer, Subject: subject},
		"other-token": {Issuer: otherIssuer, Subject: otherSubject},
	}}, Resolver: resolver, Service: service}
	runtime := httptest.NewServer(server.NewMux(server.Dependencies{WorkOrderHandler: handler}))
	t.Cleanup(runtime.Close)

	body := map[string]any{
		"operation_id": id + "-operation", "idempotency_key": id + "-idempotency", "expected_revision": 1,
		"work_order_id": workOrderID, "assignment_id": assignmentID, "inspection_ids": inspectionIDs,
		"tenant_id": "attacker-controlled-and-ignored", "organization_id": "attacker-controlled-and-ignored", "actor_id": "attacker-controlled-and-ignored",
	}
	rejectedAuthority := postPartialSubmission(t, runtime.URL, "valid-token", body)
	if rejectedAuthority.Code != http.StatusBadRequest {
		t.Fatalf("authority-shaped request status = %d body=%s, want %d", rejectedAuthority.Code, rejectedAuthority.Body.String(), http.StatusBadRequest)
	}
	delete(body, "tenant_id")
	delete(body, "organization_id")
	delete(body, "actor_id")
	first := postPartialSubmission(t, runtime.URL, "valid-token", body)
	if first.Code != http.StatusOK {
		t.Fatalf("valid submission status = %d body=%s", first.Code, first.Body.String())
	}
	var firstReceipt workorder.MutationReceipt
	if err := json.NewDecoder(first.Body).Decode(&firstReceipt); err != nil {
		t.Fatal(err)
	}
	if firstReceipt.WorkOrderID != workOrderID || firstReceipt.TenantID != tenantID || firstReceipt.Revision != 2 {
		t.Fatalf("unexpected derived receipt: %+v", firstReceipt)
	}

	replay := postPartialSubmission(t, runtime.URL, "valid-token", body)
	if replay.Code != http.StatusOK {
		t.Fatalf("idempotent replay status = %d body=%s", replay.Code, replay.Body.String())
	}
	var replayReceipt workorder.MutationReceipt
	if err := json.NewDecoder(replay.Body).Decode(&replayReceipt); err != nil {
		t.Fatal(err)
	}
	if replayReceipt != firstReceipt {
		t.Fatalf("idempotent replay receipt changed: first=%+v replay=%+v", firstReceipt, replayReceipt)
	}

	assertHTTPSubmissionPersistence(t, ctx, fixtureDatabase, workOrderID, inspectionIDs)

	crossOrganization := postPartialSubmission(t, runtime.URL, "other-token", body)
	if crossOrganization.Code != http.StatusForbidden {
		t.Fatalf("cross-organization status = %d body=%s, want %d without order disclosure", crossOrganization.Code, crossOrganization.Body.String(), http.StatusForbidden)
	}
	if bytes.Contains(crossOrganization.Body.Bytes(), []byte(workOrderID)) {
		t.Fatalf("cross-organization response disclosed work-order identifier: %s", crossOrganization.Body.String())
	}

	cleanupHTTPWorkOrderFixture(t, ctx, database, tenantID, organizationID, workOrderID, assignmentID)
	cleanupRuntimeMembership(t, ctx, fixtureDatabase, otherIssuer, otherSubject, otherActorID)
	cleanupRuntimeMembership(t, ctx, fixtureDatabase, issuer, subject, actorID)
	assertHTTPRuntimeFixtureCleanup(t, ctx, database, fixtureDatabase, workOrderID, issuer, subject, actorID, otherIssuer, otherSubject, otherActorID)
}

func openHTTPFixtureDatabase(t *testing.T, ctx context.Context) *sql.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("INTEGIN_TEST_FIXTURE_DATABASE_URL"))
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_FIXTURE_DATABASE_URL to run HTTP PostgreSQL fixture setup")
	}
	fixtureDatabase, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fixtureDatabase.Close() })
	if err := fixtureDatabase.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	return fixtureDatabase
}

func insertRuntimeMembership(t *testing.T, ctx context.Context, database *sql.DB, issuer, subject, tenantID, organizationID, actorID string) {
	t.Helper()
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var subjectID, membershipID int64
	if _, err := tx.ExecContext(ctx, "INSERT INTO identity_actor (actor_id, tenant_id, organization_id) VALUES ($1,$2,$3)", actorID, tenantID, organizationID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRowContext(ctx, "INSERT INTO identity_subject (issuer, subject) VALUES ($1,$2) RETURNING subject_id", issuer, subject).Scan(&subjectID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRowContext(ctx, "INSERT INTO identity_membership (subject_id, tenant_id, organization_id, actor_id, work_order_role) VALUES ($1,$2,$3,$4,'inspector') RETURNING membership_id", subjectID, tenantID, organizationID, actorID).Scan(&membershipID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO identity_membership_capability (membership_id, capability) VALUES ($1,$2)", membershipID, workorderauth.CapabilitySubmitPartial); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO identity_membership_capability (membership_id, capability) VALUES ($1,$2)", membershipID, workorderauth.CapabilityAddEvidenceReference); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func cleanupRuntimeMembership(t *testing.T, ctx context.Context, database *sql.DB, issuer, subject, actorID string) {
	t.Helper()
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Errorf("begin identity cleanup %s: %v", subject, err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM identity_membership_capability WHERE membership_id IN (SELECT membership_id FROM identity_membership WHERE subject_id IN (SELECT subject_id FROM identity_subject WHERE issuer=$1 AND subject=$2))", issuer, subject); err != nil {
		t.Errorf("delete identity membership capability fixture %s: %v", subject, err)
		return
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM identity_membership WHERE subject_id IN (SELECT subject_id FROM identity_subject WHERE issuer=$1 AND subject=$2)", issuer, subject); err != nil {
		t.Errorf("delete identity membership fixture %s: %v", subject, err)
		return
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM identity_subject WHERE issuer=$1 AND subject=$2", issuer, subject); err != nil {
		t.Errorf("delete identity subject fixture %s: %v", subject, err)
		return
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM identity_actor WHERE actor_id=$1", actorID); err != nil {
		t.Errorf("delete identity actor fixture %s: %v", actorID, err)
		return
	}
	if err := tx.Commit(); err != nil {
		t.Errorf("commit identity cleanup %s: %v", subject, err)
	}
}

func setupHTTPWorkOrderFixture(t *testing.T, ctx context.Context, database *sql.DB, actor workorder.ActorContext, workOrderID, scopeID, assignmentID string, inspectionIDs []string) {
	t.Helper()
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order (id,tenant_id,organization_id,client_id,job_number,request_state,execution_state,commercial_state,certificate_state,revision,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'requested','in_progress','not_ready_for_invoice','not_started',1,$6,$6)", workOrderID, actor.TenantID, actor.OrganizationID, "http-client", workOrderID, actor.ActorID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_scope_item (id,tenant_id,organization_id,work_order_id,client_id,location_id,asset_id,asset_type) VALUES ($1,$2,$3,$4,'http-client','http-location','http-asset','equipment')", scopeID, actor.TenantID, actor.OrganizationID, workOrderID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_assignment (id,tenant_id,organization_id,work_order_id,inspector_id,state,revision,effective_from,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'active',1,now(),$5,$5)", assignmentID, actor.TenantID, actor.OrganizationID, workOrderID, actor.ActorID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_assignment_scope (tenant_id,organization_id,assignment_id,scope_item_id) VALUES ($1,$2,$3,$4)", actor.TenantID, actor.OrganizationID, assignmentID, scopeID); err != nil {
		t.Fatal(err)
	}
	for _, inspectionID := range inspectionIDs {
		if _, err := tx.ExecContext(ctx, "INSERT INTO inspection_record (id,tenant_id,organization_id,work_order_id,scope_item_id,assignment_id,asset_id,inspector_id,lifecycle_state,revision,finalization_state,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,'http-asset',$7,'COMPLETED',1,'OPEN',$7,$7)", inspectionID, actor.TenantID, actor.OrganizationID, workOrderID, scopeID, assignmentID, actor.ActorID); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func cleanupHTTPWorkOrderFixture(t *testing.T, ctx context.Context, database *sql.DB, tenantID, organizationID, workOrderID, assignmentID string) {
	t.Helper()
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Errorf("begin work-order cleanup: %v", err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", tenantID, organizationID); err != nil {
		t.Errorf("scope work-order cleanup: %v", err)
		return
	}
	for _, step := range []struct {
		statement string
		argument  string
	}{
		{"DELETE FROM work_order_state_event WHERE work_order_id=$1", workOrderID},
		{"DELETE FROM work_order_operation WHERE aggregate_id=$1", workOrderID},
		{"DELETE FROM work_order_submission_item WHERE submission_segment_id IN (SELECT id FROM work_order_submission_segment WHERE work_order_id=$1)", workOrderID},
		{"DELETE FROM work_order_submission_segment WHERE work_order_id=$1", workOrderID},
		{"DELETE FROM inspection_record WHERE work_order_id=$1", workOrderID},
		{"DELETE FROM work_order_assignment_scope WHERE assignment_id=$1", assignmentID},
		{"DELETE FROM work_order_assignment WHERE work_order_id=$1", workOrderID},
		{"DELETE FROM work_order_scope_item WHERE work_order_id=$1", workOrderID},
		{"DELETE FROM work_order WHERE id=$1", workOrderID},
	} {
		if _, err := tx.ExecContext(ctx, step.statement, step.argument); err != nil {
			t.Errorf("cleanup work-order fixture %s: %v", workOrderID, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		t.Errorf("commit work-order cleanup: %v", err)
	}
}

func postPartialSubmission(t *testing.T, baseURL, token string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, baseURL+"/work-orders/partial-submissions", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	recorder := httptest.NewRecorder()
	recorder.Code = response.StatusCode
	if _, err := recorder.Body.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}
	return recorder
}

func assertHTTPSubmissionPersistence(t *testing.T, ctx context.Context, database *sql.DB, workOrderID string, inspectionIDs []string) {
	t.Helper()
	var submittedInspections, submissionItems int
	if err := database.QueryRowContext(ctx, "SELECT count(*) FROM inspection_record WHERE work_order_id=$1 AND finalization_state='SUBMITTED'", workOrderID).Scan(&submittedInspections); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, "SELECT count(*) FROM work_order_submission_item WHERE submission_segment_id IN (SELECT id FROM work_order_submission_segment WHERE work_order_id=$1)", workOrderID).Scan(&submissionItems); err != nil {
		t.Fatal(err)
	}
	if submittedInspections != len(inspectionIDs) || submissionItems != len(inspectionIDs) {
		t.Fatalf("canonical submission persistence = inspections:%d items:%d, want %d each", submittedInspections, submissionItems, len(inspectionIDs))
	}
	var executionState string
	if err := database.QueryRowContext(ctx, "SELECT execution_state FROM work_order WHERE id=$1", workOrderID).Scan(&executionState); err != nil {
		t.Fatal(err)
	}
	if executionState != string(workorder.ExecutionPartiallySubmitted) {
		t.Fatalf("execution state = %q, want %q", executionState, workorder.ExecutionPartiallySubmitted)
	}
}

func assertHTTPRuntimeFixtureCleanup(t *testing.T, ctx context.Context, database, fixtureDatabase *sql.DB, workOrderID, issuer, subject, actorID, otherIssuer, otherSubject, otherActorID string) {
	t.Helper()
	var remainingWorkOrders, remainingInspections, remainingSubjects, remainingActors int
	if err := database.QueryRowContext(ctx, "SELECT count(*) FROM work_order WHERE id=$1", workOrderID).Scan(&remainingWorkOrders); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, "SELECT count(*) FROM inspection_record WHERE work_order_id=$1", workOrderID).Scan(&remainingInspections); err != nil {
		t.Fatal(err)
	}
	if err := fixtureDatabase.QueryRowContext(ctx, "SELECT count(*) FROM identity_subject WHERE (issuer=$1 AND subject=$2) OR (issuer=$3 AND subject=$4)", issuer, subject, otherIssuer, otherSubject).Scan(&remainingSubjects); err != nil {
		t.Fatal(err)
	}
	if err := fixtureDatabase.QueryRowContext(ctx, "SELECT count(*) FROM identity_actor WHERE actor_id=$1 OR actor_id=$2", actorID, otherActorID).Scan(&remainingActors); err != nil {
		t.Fatal(err)
	}
	if remainingWorkOrders != 0 || remainingInspections != 0 || remainingSubjects != 0 || remainingActors != 0 {
		t.Fatalf("runtime fixture cleanup left rows: work_orders=%d inspections=%d identity_subjects=%d identity_actors=%d", remainingWorkOrders, remainingInspections, remainingSubjects, remainingActors)
	}
}

func TestAuthenticatedEvidenceReferenceHTTPPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL to run the controlled HTTP PostgreSQL integration test")
	}
	ctx := context.Background()
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	fixtureDatabase := openHTTPFixtureDatabase(t, ctx)

	id := fmt.Sprintf("it-http-evidence-%d", time.Now().UnixNano())
	issuer, subject := "https://issuer.example.test/"+id, "subject-"+id
	tenantID, organizationID, actorID := "pilot-tenant-http", "pilot-organization-http", "inspector-"+id
	otherIssuer, otherSubject := issuer+"-other", subject+"-other"
	otherOrganizationID, otherActorID := organizationID+"-other", "inspector-other-"+id
	workOrderID, scopeID, assignmentID := id+"-order", id+"-scope", id+"-assignment"
	inspectionIDs := []string{id + "-inspection-1", id + "-inspection-2"}

	insertRuntimeMembership(t, ctx, fixtureDatabase, issuer, subject, tenantID, organizationID, actorID)
	t.Cleanup(func() { cleanupRuntimeMembership(t, ctx, fixtureDatabase, issuer, subject, actorID) })
	insertRuntimeMembership(t, ctx, fixtureDatabase, otherIssuer, otherSubject, tenantID, otherOrganizationID, otherActorID)
	t.Cleanup(func() { cleanupRuntimeMembership(t, ctx, fixtureDatabase, otherIssuer, otherSubject, otherActorID) })
	setupHTTPWorkOrderFixture(t, ctx, database, workorder.ActorContext{TenantID: tenantID, OrganizationID: organizationID, ActorID: actorID, Role: "inspector", Capabilities: []string{workorderauth.CapabilityAddEvidenceReference}}, workOrderID, scopeID, assignmentID, inspectionIDs)
	t.Cleanup(func() {
		cleanupHTTPWorkOrderFixture(t, ctx, database, tenantID, organizationID, workOrderID, assignmentID)
	})

	repository, err := workorderpg.NewRepository(database, workorderpg.NewPostgresInspectionMembershipValidator())
	if err != nil {
		t.Fatal(err)
	}
	service, err := workorder.NewService(workorder.ServiceDependencies{Repository: repository, Transactions: repository, Authorizer: workorderauth.New()})
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := identity.NewPostgresResolver(database)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(ctx, identity.PrincipalKey{Issuer: issuer, Subject: subject}); err != nil {
		t.Fatalf("fixture identity resolver preflight failed: %v", err)
	}
	handler := workorderhttp.EvidenceHandler{Validator: runtimeTokenValidator{principals: map[string]oidcauth.Principal{
		"valid-token": {Issuer: issuer, Subject: subject},
		"other-token": {Issuer: otherIssuer, Subject: otherSubject},
	}}, Resolver: resolver, Service: service}
	runtime := httptest.NewServer(server.NewMux(server.Dependencies{WorkOrderEvidenceHandler: handler}))
	t.Cleanup(runtime.Close)

	evidenceID := id + "-evidence-1"
	contentHash := "a" + strings.Repeat("f", 63)
	referenceURL := "https://evidence.example.com/" + evidenceID
	body := map[string]any{
		"operation_id":       id + "-operation",
		"idempotency_key":    id + "-idempotency",
		"expected_revision":  1,
		"evidence_id":        evidenceID,
		"content_hash":       contentHash,
		"reference_url":      referenceURL,
		"tenant_id":          "attacker-controlled-and-ignored",
		"organization_id":    "attacker-controlled-and-ignored",
		"actor_id":           "attacker-controlled-and-ignored",
	}
	rejectedAuthority := postEvidenceReference(t, runtime.URL, "valid-token", workOrderID, body)
	if rejectedAuthority.Code != http.StatusBadRequest {
		t.Fatalf("authority-shaped request status = %d body=%s, want %d", rejectedAuthority.Code, rejectedAuthority.Body.String(), http.StatusBadRequest)
	}
	delete(body, "tenant_id")
	delete(body, "organization_id")
	delete(body, "actor_id")
	first := postEvidenceReference(t, runtime.URL, "valid-token", workOrderID, body)
	if first.Code != http.StatusOK {
		t.Fatalf("valid evidence reference status = %d body=%s", first.Code, first.Body.String())
	}
	var firstReceipt workorder.MutationReceipt
	if err := json.NewDecoder(first.Body).Decode(&firstReceipt); err != nil {
		t.Fatal(err)
	}
	if firstReceipt.WorkOrderID != workOrderID || firstReceipt.TenantID != tenantID || firstReceipt.Revision != 1 {
		t.Fatalf("unexpected derived receipt: %+v", firstReceipt)
	}

	replay := postEvidenceReference(t, runtime.URL, "valid-token", workOrderID, body)
	if replay.Code != http.StatusOK {
		t.Fatalf("idempotent replay status = %d body=%s", replay.Code, replay.Body.String())
	}
	var replayReceipt workorder.MutationReceipt
	if err := json.NewDecoder(replay.Body).Decode(&replayReceipt); err != nil {
		t.Fatal(err)
	}
	if replayReceipt != firstReceipt {
		t.Fatalf("idempotent replay receipt changed: first=%+v replay=%+v", firstReceipt, replayReceipt)
	}

	assertEvidenceReferencePersistence(t, ctx, database, workOrderID, evidenceID, contentHash, referenceURL, actorID)

	crossOrganization := postEvidenceReference(t, runtime.URL, "other-token", workOrderID, body)
	if crossOrganization.Code != http.StatusForbidden {
		t.Fatalf("cross-organization status = %d body=%s, want %d without order disclosure", crossOrganization.Code, crossOrganization.Body.String(), http.StatusForbidden)
	}
	if bytes.Contains(crossOrganization.Body.Bytes(), []byte(workOrderID)) {
		t.Fatalf("cross-organization response disclosed work-order identifier: %s", crossOrganization.Body.String())
	}

	cleanupHTTPWorkOrderFixture(t, ctx, database, tenantID, organizationID, workOrderID, assignmentID)
	cleanupRuntimeMembership(t, ctx, fixtureDatabase, otherIssuer, otherSubject, otherActorID)
	cleanupRuntimeMembership(t, ctx, fixtureDatabase, issuer, subject, actorID)
	assertHTTPRuntimeFixtureCleanup(t, ctx, database, fixtureDatabase, workOrderID, issuer, subject, actorID, otherIssuer, otherSubject, otherActorID)
}

func postEvidenceReference(t *testing.T, baseURL, token, workOrderID string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, baseURL+"/work-orders/"+workOrderID+"/evidence", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	recorder := httptest.NewRecorder()
	recorder.Code = response.StatusCode
	if _, err := recorder.Body.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}
	return recorder
}

func assertEvidenceReferencePersistence(t *testing.T, ctx context.Context, database *sql.DB, workOrderID, evidenceID, contentHash, referenceURL, createdBy string) {
	t.Helper()
	var storedEvidenceID, storedContentHash, storedReferenceURL, storedCreatedBy string
	if err := database.QueryRowContext(ctx, "SELECT id, content_hash, reference_url, created_by FROM work_order_evidence WHERE tenant_id=$1 AND organization_id=$2 AND work_order_id=$3 AND id=$4", "pilot-tenant-http", "pilot-organization-http", workOrderID, evidenceID).Scan(&storedEvidenceID, &storedContentHash, &storedReferenceURL, &storedCreatedBy); err != nil {
		t.Fatal(err)
	}
	if storedEvidenceID != evidenceID || storedContentHash != contentHash || storedReferenceURL != referenceURL || storedCreatedBy != createdBy {
		t.Fatalf("evidence reference persistence mismatch: id=%s hash=%s url=%s by=%s", storedEvidenceID, storedContentHash, storedReferenceURL, storedCreatedBy)
	}
}

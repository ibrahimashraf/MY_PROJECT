package evidencehttp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"integin/internal/evidencepg"
	"integin/internal/evidenceregistration"
	"integin/internal/identity"
	"integin/internal/storage"
	"integin/internal/workorderauth"
)

type integrationResolver struct{ membership identity.Membership }

func (r integrationResolver) Resolve(context.Context, identity.PrincipalKey) (identity.Membership, error) {
	return r.membership, nil
}

func TestAuthenticatedEvidenceMetadataRegistrationPostgresIntegration(t *testing.T) {
	ctx := context.Background()
	dsn := strings.TrimSpace(os.Getenv("INTEGIN_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL and INTEGIN_S3_* to run the controlled evidence registration integration test")
	}
	store := integrationRegistrationStore(t)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}

	id := fmt.Sprintf("it-evidence-registration-%d", time.Now().UnixNano())
	tenant, organization, actorID := "pilot-tenant-runtime", "pilot-organization-runtime", "integration-inspector"
	workOrderID, scopeID, assignmentID, inspectionID := id+"-order", id+"-scope", id+"-assignment", id+"-inspection"
	key := tenant + "/" + organization + "/evidence/" + id
	t.Cleanup(func() {
		_ = store.Delete(ctx, key)
		_ = store.Delete(ctx, tenant+"/pilot-organization-other/evidence/"+id)
		cleanupRegistrationFixture(t, ctx, db, tenant, organization, id, inspectionID, assignmentID, scopeID, workOrderID)
		if _, err := store.Get(ctx, key); err == nil {
			t.Errorf("primary object fixture remains after cleanup")
		}
	})
	setupRegistrationInspection(t, ctx, db, tenant, organization, actorID, workOrderID, scopeID, assignmentID, inspectionID)
	data := []byte("ciphertext-registration-payload")
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	if err := store.Put(ctx, storage.Object{Key: key, ContentType: "application/octet-stream", Data: data}); err != nil {
		t.Fatal(err)
	}
	repository, err := evidencepg.NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	service, err := evidenceregistration.NewService(evidenceregistration.Dependencies{Repository: repository, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	membership := identity.Membership{ActorID: actorID, TenantID: tenant, OrganizationID: organization, WorkOrderRole: "inspector", Capabilities: []string{workorderauth.CapabilitySubmitPartial}}
	handler := Handler{Validator: validatorStub{}, Resolver: integrationResolver{membership: membership}, Service: service}
	body := registrationIntegrationBody(t, id, inspectionID, digest, int64(len(data)), nil)

	forged := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/evidence/metadata-registrations", bytes.NewReader(registrationIntegrationBody(t, id, inspectionID, digest, int64(len(data)), map[string]any{"tenant_id": "forged"})))
	request.Header.Set("Authorization", "Bearer token")
	handler.ServeHTTP(forged, request)
	if forged.Code != http.StatusBadRequest || registrationCount(t, ctx, db, id) != 0 {
		t.Fatalf("authority-field rejection status=%d rows=%d", forged.Code, registrationCount(t, ctx, db, id))
	}

	response := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/evidence/metadata-registrations", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer token")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"APPLIED"`) || registrationCount(t, ctx, db, id) != 1 {
		t.Fatalf("registration status=%d body=%s rows=%d", response.Code, response.Body.String(), registrationCount(t, ctx, db, id))
	}

	duplicate := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/evidence/metadata-registrations", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer token")
	handler.ServeHTTP(duplicate, request)
	if duplicate.Code != http.StatusOK || !strings.Contains(duplicate.Body.String(), `"DUPLICATE"`) || registrationCount(t, ctx, db, id) != 1 {
		t.Fatalf("duplicate status=%d body=%s rows=%d", duplicate.Code, duplicate.Body.String(), registrationCount(t, ctx, db, id))
	}

	missing := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/evidence/metadata-registrations", bytes.NewReader(registrationIntegrationBody(t, id+"-missing", inspectionID, digest, int64(len(data)), nil)))
	request.Header.Set("Authorization", "Bearer token")
	handler.ServeHTTP(missing, request)
	if missing.Code != http.StatusConflict || registrationCount(t, ctx, db, id+"-missing") != 0 {
		t.Fatalf("missing object status=%d rows=%d", missing.Code, registrationCount(t, ctx, db, id+"-missing"))
	}

	otherOrganization := "pilot-organization-other"
	otherKey := tenant + "/" + otherOrganization + "/evidence/" + id
	if err := store.Put(ctx, storage.Object{Key: otherKey, ContentType: "application/octet-stream", Data: data}); err != nil {
		t.Fatal(err)
	}
	otherMembership := membership
	otherMembership.OrganizationID = otherOrganization
	otherHandler := Handler{Validator: validatorStub{}, Resolver: integrationResolver{membership: otherMembership}, Service: service}
	crossOrganization := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/evidence/metadata-registrations", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer token")
	otherHandler.ServeHTTP(crossOrganization, request)
	if crossOrganization.Code != http.StatusForbidden || registrationCount(t, ctx, db, id) != 1 {
		t.Fatalf("cross organization status=%d original rows=%d", crossOrganization.Code, registrationCount(t, ctx, db, id))
	}
}

func integrationRegistrationStore(t *testing.T) storage.Store {
	t.Helper()
	endpoint, bucket := strings.TrimSpace(os.Getenv("INTEGIN_S3_ENDPOINT")), strings.TrimSpace(os.Getenv("INTEGIN_S3_BUCKET"))
	accessKey, secretKey, region := strings.TrimSpace(os.Getenv("INTEGIN_S3_ACCESS_KEY")), strings.TrimSpace(os.Getenv("INTEGIN_S3_SECRET_KEY")), strings.TrimSpace(os.Getenv("INTEGIN_S3_REGION"))
	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" || region == "" {
		t.Skip("set INTEGIN_S3_ENDPOINT, INTEGIN_S3_BUCKET, INTEGIN_S3_ACCESS_KEY, INTEGIN_S3_SECRET_KEY, and INTEGIN_S3_REGION")
	}
	store, err := storage.NewS3Store(endpoint, bucket, nil, storage.NewAWSSigV4Signer(accessKey, secretKey, region, "s3"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func registrationIntegrationBody(t *testing.T, evidenceID, inspectionID, digest string, size int64, extra map[string]any) []byte {
	t.Helper()
	payload := map[string]any{"evidence_id": evidenceID, "inspection_id": inspectionID, "content_type": "application/octet-stream", "ciphertext_bytes": size, "plaintext_sha256": strings.Repeat("a", 64), "ciphertext_sha256": digest, "captured_at": time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC), "device_id": evidenceID + "-device", "authority_id": evidenceID + "-authority", "authority_epoch": 1, "transaction_id": evidenceID + "-transaction", "receipt_id": evidenceID + "-receipt", "signature_algorithm": "Ed25519", "key_id": evidenceID + "-key", "encryption_algorithm": "AES-256-GCM", "encryption_key_reference": evidenceID + "-storage-key", "classification": "CONFIDENTIAL", "retention_reference": "retention-v1", "hold_state": "NONE", "redaction_policy_reference": "redaction-v1"}
	for key, value := range extra {
		payload[key] = value
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func setupRegistrationInspection(t *testing.T, ctx context.Context, db *sql.DB, tenant, organization, actorID, workOrderID, scopeID, assignmentID, inspectionID string) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id',$1,true),set_config('integin.organization_id',$2,true)", tenant, organization); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order (id,tenant_id,organization_id,client_id,job_number,request_state,execution_state,commercial_state,certificate_state,revision,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'requested','ready','not_ready','not_started',1,$6,$6)", workOrderID, tenant, organization, "it-evidence-registration-client", workOrderID, actorID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_scope_item (id,tenant_id,organization_id,work_order_id,client_id,location_id,asset_id,asset_type) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)", scopeID, tenant, organization, workOrderID, "it-evidence-registration-client", "it-evidence-registration-location", "it-evidence-registration-asset", "equipment"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_assignment (id,tenant_id,organization_id,work_order_id,inspector_id,state,revision,effective_from,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'active',1,now(),$6,$6)", assignmentID, tenant, organization, workOrderID, actorID, actorID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_assignment_scope (tenant_id,organization_id,assignment_id,scope_item_id) VALUES ($1,$2,$3,$4)", tenant, organization, assignmentID, scopeID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO inspection_record (id,tenant_id,organization_id,work_order_id,scope_item_id,assignment_id,asset_id,inspector_id,lifecycle_state,revision,finalization_state,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'COMPLETED',1,'OPEN',$8,$8)", inspectionID, tenant, organization, workOrderID, scopeID, assignmentID, "it-evidence-registration-asset", actorID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func cleanupRegistrationFixture(t *testing.T, ctx context.Context, db *sql.DB, tenant, organization, evidenceID, inspectionID, assignmentID, scopeID, workOrderID string) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Errorf("begin registration cleanup: %v", err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id',$1,true),set_config('integin.organization_id',$2,true)", tenant, organization); err != nil {
		t.Errorf("scope registration cleanup: %v", err)
		return
	}
	for _, entry := range []struct{ query, id string }{
		{"DELETE FROM evidence_metadata WHERE id=$1", evidenceID},
		{"DELETE FROM inspection_record WHERE id=$1", inspectionID},
		{"DELETE FROM work_order_assignment_scope WHERE assignment_id=$1", assignmentID},
		{"DELETE FROM work_order_assignment WHERE id=$1", assignmentID},
		{"DELETE FROM work_order_scope_item WHERE id=$1", scopeID},
		{"DELETE FROM work_order WHERE id=$1", workOrderID},
	} {
		if _, err := tx.ExecContext(ctx, entry.query, entry.id); err != nil {
			t.Errorf("registration cleanup: %v", err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		t.Errorf("commit registration cleanup: %v", err)
	}
}

func registrationCount(t *testing.T, ctx context.Context, db *sql.DB, id string) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM evidence_metadata WHERE id=$1", id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

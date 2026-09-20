package evidenceexport

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"integin/internal/domain/evidence"
	"integin/internal/evidencepg"
	"integin/internal/exportmanifest"
	"integin/internal/storage"
)

func TestPostgresRustFSVerifiedExportIntegration(t *testing.T) {
	ctx := context.Background()
	dsn := strings.TrimSpace(os.Getenv("INTEGIN_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL and INTEGIN_S3_* to run the controlled evidence export integration test")
	}
	store := integrationStore(t)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(5)
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}

	id := fmt.Sprintf("it-evidence-%d", time.Now().UnixNano())
	actor := evidence.ActorContext{TenantID: "pilot-tenant-runtime", OrganizationID: "pilot-organization-runtime", ActorID: "integration-inspector"}
	workOrderID, scopeID, assignmentID, inspectionID := id+"-order", id+"-scope", id+"-assignment", id+"-inspection"
	objectKey := actor.TenantID + "/" + actor.OrganizationID + "/evidence/" + id
	t.Cleanup(func() {
		if err := store.Delete(ctx, objectKey); err != nil && !strings.Contains(err.Error(), "status 404") && !strings.Contains(err.Error(), "not found") {
			t.Errorf("delete object fixture: %v", err)
		}
		if _, err := store.Get(ctx, objectKey); err == nil {
			t.Errorf("object fixture %s remains after cleanup", id)
		}
		cleanupEvidenceFixture(t, ctx, db, actor, id, inspectionID, assignmentID, scopeID, workOrderID)
	})

	setupEvidenceInspection(t, ctx, db, actor, workOrderID, scopeID, assignmentID, inspectionID)
	repo, err := evidencepg.NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("ciphertext-evidence-payload-v1")
	metadata := integrationMetadata(actor, id, inspectionID, data)
	if metadata.ObjectKey != objectKey {
		t.Fatalf("unexpected contained object key: %s", metadata.ObjectKey)
	}
	if err := store.Put(ctx, storage.Object{Key: objectKey, ContentType: metadata.ContentType, Data: data, Metadata: map[string]string{"ciphertext-sha256": metadata.CiphertextSHA256}}); err != nil {
		t.Fatal(err)
	}
	registered, inserted, err := repo.Register(ctx, actor, metadata)
	if err != nil || !inserted || registered.ID != id {
		t.Fatalf("register = %#v inserted=%v err=%v", registered, inserted, err)
	}
	if _, inserted, err := repo.Register(ctx, actor, metadata); err != nil || inserted {
		t.Fatalf("idempotent register inserted=%v err=%v", inserted, err)
	}
	assertEvidenceRuntimeRLS(t, ctx, db, actor, id)
	rows, err := repo.ListByInspection(ctx, actor, inspectionID)
	if err != nil || len(rows) != 1 || rows[0].ID != id {
		t.Fatalf("same-tenant inspection list rows=%#v err=%v", rows, err)
	}
	other := actor
	other.OrganizationID = "pilot-organization-other"
	otherRows, err := repo.ListByTenantOrganization(ctx, other)
	if err != nil || len(otherRows) != 0 {
		t.Fatalf("cross-organization list rows=%#v err=%v", otherRows, err)
	}

	projection, err := New(repo, store, fixedClock)
	if err != nil {
		t.Fatal(err)
	}
	request := Request{ExportID: id + "-export", Exporter: exportmanifest.ExporterIdentity{SubjectReference: "subject-integration", MembershipReference: "membership-integration"}, ApprovalReference: "approval-integration"}
	manifest, err := projection.Export(ctx, actor, request)
	if err != nil || len(manifest.Evidence) != 1 || manifest.Evidence[0].EvidenceID != id || manifest.ManifestChecksum == "" {
		t.Fatalf("verified export manifest=%#v err=%v", manifest, err)
	}
	if err := manifest.Verify(); err != nil {
		t.Fatalf("sealed manifest verify: %v", err)
	}

	if err := store.Delete(ctx, objectKey); err != nil {
		t.Fatal(err)
	}
	if _, err := projection.Export(ctx, actor, request); !errors.Is(err, ErrMissingObject) {
		t.Fatalf("missing object error = %v", err)
	}
	if err := store.Put(ctx, storage.Object{Key: objectKey, ContentType: metadata.ContentType, Data: data}); err != nil {
		t.Fatal(err)
	}
	mutated := append([]byte(nil), data...)
	mutated[0] ^= 0xff
	if err := store.Put(ctx, storage.Object{Key: objectKey, ContentType: metadata.ContentType, Data: mutated}); err != nil {
		t.Fatal(err)
	}
	if _, err := projection.Export(ctx, actor, request); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("same-length digest mismatch error = %v", err)
	}
}

func integrationStore(t *testing.T) storage.Store {
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

func integrationMetadata(actor evidence.ActorContext, id, inspectionID string, data []byte) evidence.Metadata {
	sum := sha256.Sum256(data)
	return evidence.Metadata{ID: id, TenantID: actor.TenantID, OrganizationID: actor.OrganizationID, InspectionID: inspectionID, ObjectKey: actor.TenantID + "/" + actor.OrganizationID + "/evidence/" + id, ContentType: "application/octet-stream", CiphertextBytes: int64(len(data)), PlaintextSHA256: strings.Repeat("a", 64), CiphertextSHA256: hex.EncodeToString(sum[:]), CapturedAt: fixedClock(), DeviceID: id + "-device", AuthorityID: id + "-authority", AuthorityEpoch: 1, TransactionID: id + "-transaction", ReceiptID: id + "-receipt", SignatureAlgorithm: "Ed25519", KeyID: id + "-signing-key", EncryptionAlgorithm: "AES-256-GCM", EncryptionKeyRef: id + "-storage-key", Classification: "CONFIDENTIAL", RetentionReference: "retention-v1", HoldState: "NONE", RedactionPolicyRef: "redaction-v1", RegisteredBy: actor.ActorID}
}

func setupEvidenceInspection(t *testing.T, ctx context.Context, db *sql.DB, actor evidence.ActorContext, workOrderID, scopeID, assignmentID, inspectionID string) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id',$1,true),set_config('integin.organization_id',$2,true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order (id,tenant_id,organization_id,client_id,job_number,request_state,execution_state,commercial_state,certificate_state,revision,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'requested','ready','not_ready','not_started',1,$6,$6)", workOrderID, actor.TenantID, actor.OrganizationID, "it-evidence-client", workOrderID, actor.ActorID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_scope_item (id,tenant_id,organization_id,work_order_id,client_id,location_id,asset_id,asset_type) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)", scopeID, actor.TenantID, actor.OrganizationID, workOrderID, "it-evidence-client", "it-evidence-location", "it-evidence-asset", "equipment"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_assignment (id,tenant_id,organization_id,work_order_id,inspector_id,state,revision,effective_from,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'active',1,now(),$6,$6)", assignmentID, actor.TenantID, actor.OrganizationID, workOrderID, actor.ActorID, actor.ActorID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO work_order_assignment_scope (tenant_id,organization_id,assignment_id,scope_item_id) VALUES ($1,$2,$3,$4)", actor.TenantID, actor.OrganizationID, assignmentID, scopeID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO inspection_record (id,tenant_id,organization_id,work_order_id,scope_item_id,assignment_id,asset_id,inspector_id,lifecycle_state,revision,finalization_state,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'COMPLETED',1,'OPEN',$8,$8)", inspectionID, actor.TenantID, actor.OrganizationID, workOrderID, scopeID, assignmentID, "it-evidence-asset", actor.ActorID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func cleanupEvidenceFixture(t *testing.T, ctx context.Context, db *sql.DB, actor evidence.ActorContext, evidenceID, inspectionID, assignmentID, scopeID, workOrderID string) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Errorf("begin fixture cleanup: %v", err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id',$1,true),set_config('integin.organization_id',$2,true)", actor.TenantID, actor.OrganizationID); err != nil {
		t.Errorf("scope fixture cleanup: %v", err)
		return
	}
	for _, statement := range []struct {
		query string
		id    string
	}{
		{"DELETE FROM evidence_metadata WHERE id=$1", evidenceID},
		{"DELETE FROM inspection_record WHERE id=$1", inspectionID},
		{"DELETE FROM work_order_assignment_scope WHERE assignment_id=$1", assignmentID},
		{"DELETE FROM work_order_assignment WHERE id=$1", assignmentID},
		{"DELETE FROM work_order_scope_item WHERE id=$1", scopeID},
		{"DELETE FROM work_order WHERE id=$1", workOrderID},
	} {
		if _, err := tx.ExecContext(ctx, statement.query, statement.id); err != nil {
			t.Errorf("cleanup fixture %s: %v", evidenceID, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		t.Errorf("commit fixture cleanup: %v", err)
	}
}

func assertEvidenceRuntimeRLS(t *testing.T, ctx context.Context, db *sql.DB, actor evidence.ActorContext, evidenceID string) {
	t.Helper()
	for _, proof := range []struct {
		organizationID string
		want           int
	}{
		{organizationID: actor.OrganizationID, want: 1},
		{organizationID: "pilot-organization-other", want: 0},
	} {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, "SET LOCAL ROLE integin_test_runtime"); err != nil {
			_ = tx.Rollback()
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id',$1,true),set_config('integin.organization_id',$2,true)", actor.TenantID, proof.organizationID); err != nil {
			_ = tx.Rollback()
			t.Fatal(err)
		}
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM evidence_metadata WHERE id=$1", evidenceID).Scan(&count); err != nil {
			_ = tx.Rollback()
			t.Fatal(err)
		}
		if err := tx.Rollback(); err != nil {
			t.Fatal(err)
		}
		if count != proof.want {
			t.Fatalf("runtime RLS organization %q count=%d want=%d", proof.organizationID, count, proof.want)
		}
	}
}


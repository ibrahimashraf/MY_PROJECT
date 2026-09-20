package certificaterender

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river"

	"integin/internal/certificatepg"
	"integin/internal/domain/certificate"
	"integin/internal/domain/certificateauthority"
	domainrender "integin/internal/domain/certificaterender"
	"integin/internal/queue"
	"integin/internal/storage"
)

func TestCertificateRenderRustFSIntegration(t *testing.T) {
	if os.Getenv("INTEGIN_RUSTFS_INTEGRATION") != "1" {
		t.Skip("skipping RustFS integration test; set INTEGIN_RUSTFS_INTEGRATION=1 to run")
	}

	endpoint := os.Getenv("INTEGIN_S3_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://127.0.0.1:19000"
	}
	bucket := os.Getenv("INTEGIN_S3_BUCKET")
	if bucket == "" {
		bucket = "integin-pilot-evidence"
	}
	accessKey := os.Getenv("INTEGIN_S3_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "pilot-zXk5NviVcn8obpN1Mll1VyMJ"
	}
	secretKey := os.Getenv("INTEGIN_S3_SECRET_KEY")
	if secretKey == "" {
		secretKey = "ZFsOePCin0ceJfDK3Bpar3x4diAYJeyD8jR6Q6mftzofrZSwZNQOUzkK8cqCob6asrhr4qqmKP6hwKHKzzSUaYCSzeQ7URCc"
	}
	region := os.Getenv("INTEGIN_S3_REGION")
	if region == "" {
		region = "us-east-1"
	}

	signer := storage.NewAWSSigV4Signer(accessKey, secretKey, region, "s3")
	store, err := storage.NewS3Store(endpoint, bucket, http.DefaultClient, signer)
	if err != nil {
		t.Fatalf("failed to create S3 store: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := store.Health(ctx); err != nil {
		t.Fatalf("store health check failed: %v", err)
	}

	// Connect to PostgreSQL database
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres_local_test_password@127.0.0.1:15432/integin_migration_test?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to connect to DB: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("database ping failed: %v", err)
	}

	repo, err := certificatepg.NewRepository(db)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	renderer := domainrender.NewDeterministicPDFRenderer()
	renderSvc, err := NewRenderService(renderer, store, repo)
	if err != nil {
		t.Fatalf("failed to create render service: %v", err)
	}

	worker := NewCertificateRenderWorker(renderSvc, repo)

	stamp := time.Now().UTC().UnixNano()
	tenantID := fmt.Sprintf("it-render-%d", stamp)
	orgID := "org-render"
	certID := fmt.Sprintf("cert-%d", stamp)
	actorID := "usr_pilot_actor"
	tmplID := fmt.Sprintf("tmpl-%d", stamp)
	workOrderID := fmt.Sprintf("wo-%d", stamp)
	scopeID := fmt.Sprintf("sc-%d", stamp)
	assignmentID := fmt.Sprintf("as-%d", stamp)
	inspectionID := fmt.Sprintf("in-%d", stamp)
	policyID := fmt.Sprintf("po-%d", stamp)
	assetID := "asset-render"

	actor := certificateauthority.ActorContext{
		TenantID:       tenantID,
		OrganizationID: orgID,
		ActorID:        actorID,
	}

	tmplCode := "lifting"
	tmplVersion := 3

	// Set GUC scope for initial seeding
	if _, err := db.ExecContext(ctx, `SELECT set_config('integin.tenant_id',$1,false), set_config('integin.organization_id',$2,false)`, tenantID, orgID); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn, err := db.Conn(cleanupCtx)
		if err == nil {
			defer conn.Close()
			_, _ = conn.ExecContext(cleanupCtx, `SELECT set_config('integin.tenant_id',$1,false), set_config('integin.organization_id',$2,false)`, tenantID, orgID)
			for _, statement := range []string{
				`DELETE FROM certificate_artifact WHERE tenant_id = $1`,
				`DELETE FROM certificate_snapshot WHERE tenant_id = $1`,
				`DELETE FROM certificate_record WHERE tenant_id = $1`,
				`DELETE FROM certificate_policy WHERE tenant_id = $1`,
				`DELETE FROM certificate_template_cell WHERE tenant_id = $1`,
				`DELETE FROM certificate_template WHERE tenant_id = $1`,
				`DELETE FROM inspection_record WHERE tenant_id = $1`,
				`DELETE FROM work_order_assignment_scope WHERE tenant_id = $1`,
				`DELETE FROM work_order_assignment WHERE tenant_id = $1`,
				`DELETE FROM work_order_scope_item WHERE tenant_id = $1`,
				`DELETE FROM work_order WHERE tenant_id = $1`,
			} {
				_, _ = conn.ExecContext(cleanupCtx, statement, tenantID)
			}
		}
	})

	now := time.Now().UTC()

	// Seed work order and inspection
	if _, err := db.ExecContext(ctx, `INSERT INTO work_order (id,tenant_id,organization_id,client_id,job_number,request_state,execution_state,commercial_state,certificate_state,revision,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'accepted','in_progress','ready_for_office_confirmation','pending_validation',1,$6,$6)`, workOrderID, tenantID, orgID, "client-a", "job-a", actorID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO work_order_scope_item (id,tenant_id,organization_id,work_order_id,client_id,location_id,asset_id,asset_type) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, scopeID, tenantID, orgID, workOrderID, "client-a", "location-a", assetID, "lifting"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO work_order_assignment (id,tenant_id,organization_id,work_order_id,inspector_id,state,revision,effective_from,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'active',1,$6,$5,$5)`, assignmentID, tenantID, orgID, workOrderID, actorID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO work_order_assignment_scope (tenant_id,organization_id,assignment_id,scope_item_id) VALUES ($1,$2,$3,$4)`, tenantID, orgID, assignmentID, scopeID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO inspection_record (id,tenant_id,organization_id,work_order_id,scope_item_id,assignment_id,asset_id,inspector_id,lifecycle_state,revision,finalization_state,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'APPROVED',7,'FINALIZED',$8,$8)`, inspectionID, tenantID, orgID, workOrderID, scopeID, assignmentID, assetID, actorID); err != nil {
		t.Fatal(err)
	}
	evidenceID := fmt.Sprintf("ev-sig-%d", stamp)
	if _, err := db.ExecContext(ctx, `INSERT INTO evidence_metadata (id,tenant_id,organization_id,inspection_id,object_key,content_type,ciphertext_bytes,plaintext_sha256,ciphertext_sha256,captured_at,device_id,authority_id,authority_epoch,transaction_id,receipt_id,signature_algorithm,key_id,encryption_algorithm,encryption_key_reference,classification,retention_reference,hold_state,redaction_policy_reference,registered_by) VALUES ($1,$2,$3,$4,$2||'/'||$3||'/evidence/'||$1,'image/png',48210,'9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08','5feceb66ffc86f38d952786c6d696c79c2dbc239dd4e91b46729d73a27fb57e9',$5,'device-a','authority-a',1,'transaction-a','receipt-a','Ed25519','key-a','AES-256-GCM','storage-key-a','CONFIDENTIAL','retention-v1','NONE','redaction-v1',$6)`, evidenceID, tenantID, orgID, inspectionID, now, actorID); err != nil {
		t.Fatal(err)
	}

	// Seed template
	if _, err := db.ExecContext(ctx, `INSERT INTO certificate_template (id,tenant_id,organization_id,template_code,version,title,asset_type,status,catalog_version,page_count,page_width_points,page_height_points,created_by,approved_by,approved_at) VALUES ($1,$2,$3,$4,$5,'Lifting certificate','lifting','APPROVED',1,1,612,792,$6,$6,$7)`, tmplID, tenantID, orgID, tmplCode, tmplVersion, actorID, now); err != nil {
		t.Fatal(err)
	}

	// Seed policy
	if _, err := db.ExecContext(ctx, `INSERT INTO certificate_policy (id,tenant_id,organization_id,template_code,template_version,policy_version,validity_days,self_issue_allowed,status,created_by,approved_by,approved_at) VALUES ($1,$2,$3,$4,$5,1,365,false,'APPROVED',$6,$6,$7)`, policyID, tenantID, orgID, tmplCode, tmplVersion, actorID, now); err != nil {
		t.Fatal(err)
	}

	// Seed cell
	if _, err := db.ExecContext(ctx, `
		INSERT INTO certificate_template_cell (
			tenant_id, organization_id, template_id, cell_id, page_number, x_points, y_points, width_points, height_points, kind, binding_key, static_text, label, fit_policy, max_lines, required, repeat_source, max_items
		) VALUES (
			$1, $2, $3, 'cell-render-1', 1, 50, 100, 200, 25, 'TEXT', 'inspection.asset_id', NULL, 'Asset ID', 'SINGLE_LINE_REQUIRED', 1, true, NULL, NULL
		)`, tenantID, orgID, tmplID); err != nil {
		t.Fatalf("failed to seed cell: %v", err)
	}

	// Prepare actors
	inspector := actor
	inspector.Capabilities = map[string]bool{"certificate.prepare": true}

	reviewer := actor
	reviewer.ActorID = "reviewer-a"
	reviewer.Capabilities = map[string]bool{"certificate.review": true}

	signerActor := actor
	signerActor.ActorID = "signer-a"
	signerActor.Capabilities = map[string]bool{"certificate.sign": true}

	issuer := actor
	issuer.ActorID = "issuer-a"
	issuer.Capabilities = map[string]bool{"certificate.issue": true}

	// Create draft, submit, review, sign, and issue certificate
	_, err = repo.CreateDraft(ctx, inspector, certificateauthority.CreateDraftRequest{
		CertificateID:   certID,
		InspectionID:    inspectionID,
		TemplateCode:    tmplCode,
		TemplateVersion: int64(tmplVersion),
		Profile:         certificateauthority.IndependentReview,
	}, now)
	if err != nil {
		t.Fatalf("failed to create draft: %v", err)
	}

	if err := repo.Submit(ctx, inspector, certID, now); err != nil {
		t.Fatalf("failed to submit: %v", err)
	}
	if err := repo.Review(ctx, reviewer, certID, now); err != nil {
		t.Fatalf("failed to review: %v", err)
	}
	if err := repo.Sign(ctx, signerActor, certID, certificate.SignatureEvent{SignerID: "signer-a", SignerName: "Test Signer", Capacity: certificate.CapacityClient, StatementVersion: "client_ack_v1", ImageSHA256Hex: "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", ImageBytes: 48210, ImageEvidenceID: evidenceID, SnapshotSHA256: "5feceb66ffc86f38d952786c6d696c79c2dbc239dd4e91b46729d73a27fb57e9"}, now); err != nil {
		t.Fatalf("failed to sign: %v", err)
	}
	issued, err := repo.Issue(ctx, issuer, certID, now)
	if err != nil {
		t.Fatalf("failed to issue: %v", err)
	}
	if issued.CertificateNumber == "" {
		t.Fatalf("empty certificate number on issuance")
	}

	// Reset GUCs so worker can establish its own transaction scope
	if _, err := db.ExecContext(ctx, `SELECT set_config('integin.tenant_id','',false), set_config('integin.organization_id','',false)`); err != nil {
		t.Fatal(err)
	}

	// Now run the River Worker
	riverJob := &river.Job[queue.CertificateRenderJobArgs]{
		Args: queue.CertificateRenderJobArgs{
			TenantID:       tenantID,
			OrganizationID: orgID,
			CertificateID:  certID,
			InspectionID:   inspectionID,
			ActorUserID:    actorID,
		},
	}

	if err := worker.Work(ctx, riverJob); err != nil {
		t.Fatalf("worker.Work failed: %v", err)
	}

	// Verify artifact in repository
	record, err := repo.GetArtifact(ctx, actor, certID, "CERTIFICATE_PDF")
	if err != nil {
		t.Fatalf("failed to get artifact record: %v", err)
	}

	if record.CertificateID != certID {
		t.Fatalf("mismatched certificate id in record: %s vs %s", record.CertificateID, certID)
	}

	expectedKey := "tenants/" + tenantID + "/certs/" + certID + "/certificate.pdf"
	if record.ObjectKey != expectedKey {
		t.Fatalf("expected object key %s, got %s", expectedKey, record.ObjectKey)
	}

	// Verify artifact in RustFS S3 store
	s3Obj, err := store.Get(ctx, record.ObjectKey)
	if err != nil {
		t.Fatalf("failed to retrieve PDF from RustFS: %v", err)
	}
	defer func() {
		_ = store.Delete(context.Background(), record.ObjectKey)
	}()

	if s3Obj.ContentType != "application/pdf" {
		t.Fatalf("expected application/pdf, got %s", s3Obj.ContentType)
	}
	if len(s3Obj.Data) == 0 {
		t.Fatal("retrieved PDF data is empty")
	}

	// Verify PDF magic bytes
	if len(s3Obj.Data) < 4 || string(s3Obj.Data[:4]) != "%PDF" {
		t.Fatalf("PDF magic header mismatch, got %q", string(s3Obj.Data[:min(10, len(s3Obj.Data))]))
	}

	// Verify SHA256 integrity
	calcSHA := sha256.Sum256(s3Obj.Data)
	if hex.EncodeToString(calcSHA[:]) != hex.EncodeToString(record.ArtifactSHA256) {
		t.Fatalf("SHA256 digest mismatch between S3 object and DB record")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}


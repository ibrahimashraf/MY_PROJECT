package certificatepg

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/certificateauthority"

	_ "github.com/lib/pq"
)

func TestPostgresCreateCertificateDraftIntegration(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("INTEGIN_TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().UTC().UnixNano()
	tenantID := fmt.Sprintf("it-certificate-lifecycle-%d", stamp)
	organizationID := "org-a"
	actor := certificateauthority.ActorContext{TenantID: tenantID, OrganizationID: organizationID, ActorID: "inspector-a", Capabilities: map[string]bool{"certificate.prepare": true}}
	workOrderID := fmt.Sprintf("work-order-%d", stamp)
	scopeID := fmt.Sprintf("scope-%d", stamp)
	assignmentID := fmt.Sprintf("assignment-%d", stamp)
	inspectionID := fmt.Sprintf("inspection-%d", stamp)
	templateID := fmt.Sprintf("template-%d", stamp)
	policyID := fmt.Sprintf("policy-%d", stamp)
	assetRegistryID := fmt.Sprintf("asset-registry-%d", stamp)
	publicScopeID := fmt.Sprintf("public-scope-%d", stamp)
	certificateID := fmt.Sprintf("certificate-%d", stamp)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, statement := range []string{
			`DELETE FROM certificate_audit_event WHERE tenant_id = $1`,
			`DELETE FROM certificate_snapshot WHERE tenant_id = $1`,
			`DELETE FROM certificate_record WHERE tenant_id = $1`,
			`DELETE FROM certificate_number_sequence WHERE tenant_id = $1`,
			`DELETE FROM certificate_policy_public_binding WHERE tenant_id = $1`,
			`DELETE FROM certificate_policy WHERE tenant_id = $1`,
			`DELETE FROM certificate_template_cell WHERE tenant_id = $1`,
			`DELETE FROM certificate_template WHERE tenant_id = $1`,
			`DELETE FROM inspection_public_scope_item WHERE tenant_id = $1`,
			`DELETE FROM inspection_public_scope WHERE tenant_id = $1`,
			`DELETE FROM asset_registry WHERE tenant_id = $1`,
			`DELETE FROM inspection_record WHERE tenant_id = $1`,
			`DELETE FROM work_order_assignment_scope WHERE tenant_id = $1`,
			`DELETE FROM work_order_assignment WHERE tenant_id = $1`,
			`DELETE FROM work_order_scope_item WHERE tenant_id = $1`,
			`DELETE FROM work_order WHERE tenant_id = $1`,
		} {
			if _, err := db.ExecContext(cleanupCtx, statement, tenantID); err != nil {
				t.Errorf("cleanup %q: %v", statement, err)
			}
		}
	})
	now := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `SELECT set_config('integin.tenant_id',$1,true), set_config('integin.organization_id',$2,true)`, tenantID, organizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO work_order (id,tenant_id,organization_id,client_id,job_number,request_state,execution_state,commercial_state,certificate_state,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'OPEN','ASSIGNED','OPEN','OPEN',$6,$6)`, workOrderID, tenantID, organizationID, "client-a", "job-a", actor.ActorID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO work_order_scope_item (id,tenant_id,organization_id,work_order_id,client_id,location_id,asset_id,asset_type) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, scopeID, tenantID, organizationID, workOrderID, "client-a", "location-a", "asset-a", "lifting"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO work_order_assignment (id,tenant_id,organization_id,work_order_id,inspector_id,state,effective_from,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'ACTIVE',$6,$5,$5)`, assignmentID, tenantID, organizationID, workOrderID, actor.ActorID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO work_order_assignment_scope (tenant_id,organization_id,assignment_id,scope_item_id) VALUES ($1,$2,$3,$4)`, tenantID, organizationID, assignmentID, scopeID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO inspection_record (id,tenant_id,organization_id,work_order_id,scope_item_id,assignment_id,asset_id,inspector_id,lifecycle_state,revision,finalization_state,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'APPROVED',7,'FINALIZED',$8,$8)`, inspectionID, tenantID, organizationID, workOrderID, scopeID, assignmentID, "asset-a", actor.ActorID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO certificate_template (id,tenant_id,organization_id,template_code,version,title,asset_type,status,catalog_version,page_count,page_width_points,page_height_points,created_by,approved_by,approved_at) VALUES ($1,$2,$3,'lifting',3,'Lifting certificate','lifting','APPROVED',1,1,612,792,$4,$4,$5)`, templateID, tenantID, organizationID, actor.ActorID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO certificate_policy (id,tenant_id,organization_id,template_code,template_version,policy_version,validity_days,self_issue_allowed,status,created_by,approved_by,approved_at) VALUES ($1,$2,$3,'lifting',3,1,365,false,'APPROVED',$4,$4,$5)`, policyID, tenantID, organizationID, actor.ActorID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO asset_registry (id,tenant_id,organization_id,asset_id,asset_type,serial_number,description,lifecycle_state,created_by,updated_by) VALUES ($1,$2,$3,'asset-a','lifting','SERIAL-A','Controlled lifting asset','ACTIVE',$4,$4)`, assetRegistryID, tenantID, organizationID, actor.ActorID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO inspection_public_scope (id,tenant_id,organization_id,inspection_id,inspection_revision,inspection_type,taxonomy_version,result_state,created_by) VALUES ($1,$2,$3,$4,7,'periodic_lifting',1,'PASS',$5)`, publicScopeID, tenantID, organizationID, inspectionID, actor.ActorID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO inspection_public_scope_item (id,tenant_id,organization_id,scope_id,scope_code,display_label,outcome,display_order) VALUES ($1,$2,$3,$4,'visual','Visual examination','PASS',1)`, publicScopeID+"-item", tenantID, organizationID, publicScopeID); err != nil {
		t.Fatal(err)
	}
	for order, key := range []string{"asset.serial_number", "asset.description", "asset.type", "inspection.type", "inspection.public_scope"} {
		if _, err := db.ExecContext(ctx, `INSERT INTO certificate_policy_public_binding (id,tenant_id,organization_id,policy_id,binding_key,required,display_order,created_by) VALUES ($1,$2,$3,$4,$5,true,$6,$7)`, fmt.Sprintf("binding-%d-%d", stamp, order), tenantID, organizationID, policyID, key, order+1, actor.ActorID); err != nil {
			t.Fatal(err)
		}
	}
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := repository.CreateDraft(ctx, actor, certificateauthority.CreateDraftRequest{CertificateID: certificateID, InspectionID: inspectionID, TemplateCode: "lifting", TemplateVersion: 3, Profile: certificateauthority.IndependentReview}, now)
	if err != nil {
		t.Fatal(err)
	}
	if certificate.Status() != certificateauthority.Draft || certificate.PolicyID() != policyID {
		t.Fatalf("unexpected draft: %#v", certificate)
	}
	var records, audits int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM certificate_record WHERE tenant_id = $1 AND id = $2`, tenantID, certificateID).Scan(&records); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM certificate_audit_event WHERE tenant_id = $1 AND certificate_id = $2 AND action = 'DRAFT_CREATED'`, tenantID, certificateID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if records != 1 || audits != 1 {
		t.Fatalf("expected one draft and audit, got records=%d audits=%d", records, audits)
	}
	if err := repository.Submit(ctx, actor, certificateID, now); err != nil {
		t.Fatal(err)
	}
	reviewer := actor
	reviewer.ActorID = "reviewer-a"
	reviewer.Capabilities = map[string]bool{"certificate.review": true}
	if err := repository.Review(ctx, reviewer, certificateID, now); err != nil {
		t.Fatal(err)
	}
	signer := actor
	signer.ActorID = "signer-a"
	signer.Capabilities = map[string]bool{"certificate.sign": true}
	if err := repository.Sign(ctx, signer, certificateID, now); err != nil {
		t.Fatal(err)
	}
	issuer := actor
	issuer.ActorID = "issuer-a"
	issuer.Capabilities = map[string]bool{"certificate.issue": true}
	issued, err := repository.Issue(ctx, issuer, certificateID, now)
	if err != nil {
		t.Fatal(err)
	}
	if issued.CertificateNumber == "" || issued.PublicToken == "" || issued.ExpiresAt.IsZero() {
		t.Fatalf("incomplete issuance result: %#v", issued)
	}
	projection, found, err := repository.VerifyPublic(ctx, issued.PublicToken)
	if err != nil || !found {
		t.Fatalf("expected public projection, found=%v err=%v", found, err)
	}
	if projection.CertificateNumber != issued.CertificateNumber || projection.Status != "ISSUED" || projection.AssetID != "asset-a" {
		t.Fatalf("unexpected public projection: %#v", projection)
	}
	if projection.AssetSerialNumber != "SERIAL-A" || projection.AssetDescription != "Controlled lifting asset" || projection.AssetType != "lifting" || projection.InspectionType != "periodic_lifting" || !strings.Contains(string(projection.TestScope), "Visual examination") {
		t.Fatalf("unexpected immutable public bindings: %#v", projection)
	}
	if _, err := db.ExecContext(ctx, `UPDATE asset_registry SET serial_number='MUTATED-SERIAL',description='Mutated description',updated_by=$1 WHERE tenant_id=$2 AND id=$3`, actor.ActorID, tenantID, assetRegistryID); err != nil {
		t.Fatal(err)
	}
	var immutableSnapshot string
	if err := db.QueryRowContext(ctx, `SELECT public_binding_snapshot::text FROM certificate_snapshot WHERE tenant_id=$1 AND certificate_id=$2`, tenantID, certificateID).Scan(&immutableSnapshot); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(immutableSnapshot, "SERIAL-A") || strings.Contains(immutableSnapshot, "MUTATED-SERIAL") {
		t.Fatalf("issued snapshot was rewritten by mutable asset source: %s", immutableSnapshot)
	}
	if _, found, err := repository.VerifyPublic(ctx, issued.PublicToken+"x"); err != nil || found {
		t.Fatalf("wrong token result found=%v err=%v", found, err)
	}
	authority := actor
	authority.ActorID = "authority-a"
	authority.Capabilities = map[string]bool{"certificate.revoke": true}
	if err := repository.Revoke(ctx, authority, certificateID, "unsafe asset condition", now); err != nil {
		t.Fatal(err)
	}
	revoked, found, err := repository.VerifyPublic(ctx, issued.PublicToken)
	if err != nil || !found || revoked.Status != "REVOKED" {
		t.Fatalf("expected truthful revoked projection found=%v value=%#v err=%v", found, revoked, err)
	}
	if err := repository.Revoke(ctx, authority, certificateID, "repeat", now); err == nil {
		t.Fatal("expected duplicate revocation rejection")
	}
	if _, err := db.ExecContext(ctx, `UPDATE certificate_policy SET self_issue_allowed=true WHERE tenant_id=$1 AND id=$2`, tenantID, policyID); err != nil {
		t.Fatal(err)
	}
	selfActor := actor
	selfActor.Capabilities = map[string]bool{"certificate.prepare": true, "certificate.review": true, "certificate.sign": true, "certificate.issue": true, "certificate.self_issue": true}
	selfCertificateID := certificateID + "-self"
	selfDraft, err := repository.CreateDraft(ctx, selfActor, certificateauthority.CreateDraftRequest{CertificateID: selfCertificateID, InspectionID: inspectionID, TemplateCode: "lifting", TemplateVersion: 3, Profile: certificateauthority.SeniorSelfIssue, SelfIssueReason: "remote emergency assignment"}, now)
	if err != nil || selfDraft.Profile() != certificateauthority.SeniorSelfIssue {
		t.Fatalf("senior self-issue draft failed value=%#v err=%v", selfDraft, err)
	}
	if err := repository.Submit(ctx, selfActor, selfCertificateID, now); err != nil {
		t.Fatal(err)
	}
	if err := repository.Review(ctx, selfActor, selfCertificateID, now); err != nil {
		t.Fatal(err)
	}
	if err := repository.Sign(ctx, selfActor, selfCertificateID, now); err != nil {
		t.Fatal(err)
	}
	selfIssued, err := repository.Issue(ctx, selfActor, selfCertificateID, now)
	if err != nil || selfIssued.CertificateNumber == "" {
		t.Fatalf("senior self-issue issuance failed value=%#v err=%v", selfIssued, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE inspection_record SET revision=8,updated_by=$1 WHERE tenant_id=$2 AND id=$3`, selfActor.ActorID, tenantID, inspectionID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO inspection_public_scope (id,tenant_id,organization_id,inspection_id,inspection_revision,inspection_type,taxonomy_version,result_state,created_by) VALUES ($1,$2,$3,$4,8,'periodic_lifting',1,'PASS',$5)`, publicScopeID+"-r8", tenantID, organizationID, inspectionID, selfActor.ActorID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO inspection_public_scope_item (id,tenant_id,organization_id,scope_id,scope_code,display_label,outcome,display_order) VALUES ($1,$2,$3,$4,'visual','Visual examination','PASS',1)`, publicScopeID+"-r8-item", tenantID, organizationID, publicScopeID+"-r8"); err != nil {
		t.Fatal(err)
	}
	replacementID := selfCertificateID + "-replacement"
	if _, err := repository.CreateDraft(ctx, selfActor, certificateauthority.CreateDraftRequest{CertificateID: replacementID, InspectionID: inspectionID, TemplateCode: "lifting", TemplateVersion: 3, Profile: certificateauthority.SeniorSelfIssue, SelfIssueReason: "corrected inspection revision"}, now); err != nil {
		t.Fatal(err)
	}
	if err := repository.Submit(ctx, selfActor, replacementID, now); err != nil {
		t.Fatal(err)
	}
	if err := repository.Review(ctx, selfActor, replacementID, now); err != nil {
		t.Fatal(err)
	}
	if err := repository.Sign(ctx, selfActor, replacementID, now); err != nil {
		t.Fatal(err)
	}
	replacement, err := repository.Issue(ctx, selfActor, replacementID, now)
	if err != nil || replacement.CertificateNumber == "" {
		t.Fatalf("replacement issuance failed value=%#v err=%v", replacement, err)
	}
	superseder := actor
	superseder.ActorID = "authority-a"
	superseder.Capabilities = map[string]bool{"certificate.supersede": true}
	if err := repository.Supersede(ctx, superseder, selfCertificateID, replacementID, now); err != nil {
		t.Fatal(err)
	}
	superseded, found, err := repository.VerifyPublic(ctx, selfIssued.PublicToken)
	if err != nil || !found || superseded.Status != "SUPERSEDED" {
		t.Fatalf("expected truthful superseded projection found=%v value=%#v err=%v", found, superseded, err)
	}
	expirer := actor
	expirer.ActorID = "authority-a"
	expirer.Capabilities = map[string]bool{"certificate.expire": true}
	expiryCheck := now.AddDate(0, 0, 366)
	if err := repository.Expire(ctx, expirer, replacementID, expiryCheck); err != nil {
		t.Fatal(err)
	}
	expired, found, err := repository.VerifyPublic(ctx, replacement.PublicToken)
	if err != nil || !found || expired.Status != "EXPIRED" {
		t.Fatalf("expected truthful expired projection found=%v value=%#v err=%v", found, expired, err)
	}
	if err := repository.Expire(ctx, expirer, replacementID, expiryCheck); err == nil {
		t.Fatal("expected duplicate expiry rejection")
	}
	if _, err := repository.Issue(ctx, issuer, certificateID, now); err == nil {
		t.Fatal("expected duplicate issuance rejection")
	}
	var snapshots, issuedAudits, tokenDigestBytes int
	var publicBindingSnapshot string
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM certificate_snapshot WHERE tenant_id=$1 AND certificate_id=$2`, tenantID, certificateID).Scan(&snapshots); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM certificate_audit_event WHERE tenant_id=$1 AND certificate_id=$2 AND action='ISSUED'`, tenantID, certificateID).Scan(&issuedAudits); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT octet_length(public_token_digest) FROM certificate_record WHERE tenant_id=$1 AND id=$2 AND status IN ('ISSUED','REVOKED')`, tenantID, certificateID).Scan(&tokenDigestBytes); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT public_binding_snapshot::text FROM certificate_snapshot WHERE tenant_id=$1 AND certificate_id=$2`, tenantID, certificateID).Scan(&publicBindingSnapshot); err != nil {
		t.Fatal(err)
	}
	if snapshots != 1 || issuedAudits != 1 || tokenDigestBytes != 32 {
		t.Fatalf("unexpected issuance persistence snapshots=%d audits=%d tokenBytes=%d", snapshots, issuedAudits, tokenDigestBytes)
	}
	for _, expected := range []string{"SERIAL-A", "Controlled lifting asset", "periodic_lifting", "Visual examination"} {
		if !strings.Contains(publicBindingSnapshot, expected) {
			t.Fatalf("public binding snapshot lacks %q: %s", expected, publicBindingSnapshot)
		}
	}
	denied := actor
	denied.OrganizationID = "org-b"
	if _, err := repository.CreateDraft(ctx, denied, certificateauthority.CreateDraftRequest{CertificateID: certificateID + "-denied", InspectionID: inspectionID, TemplateCode: "lifting", TemplateVersion: 3, Profile: certificateauthority.IndependentReview}, now); err == nil {
		t.Fatal("expected cross-organization denial")
	}
	var deniedRecords int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM certificate_record WHERE tenant_id=$1 AND id=$2`, tenantID, certificateID+"-denied").Scan(&deniedRecords); err != nil {
		t.Fatal(err)
	}
	if deniedRecords != 0 {
		t.Fatalf("cross-organization attempt persisted %d records", deniedRecords)
	}
}

package certificatepublichttp

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"

	"integin/internal/certificatepg"
	"integin/internal/domain/certificateauthority"
	"integin/internal/server"

	_ "github.com/lib/pq"
)

func TestPostgresPublicVerifierHTTPIntegration(t *testing.T) {
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
	tenantID := fmt.Sprintf("it-certificate-http-perf-%d", stamp)
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
	for _, seed := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO work_order (id,tenant_id,organization_id,client_id,job_number,request_state,execution_state,commercial_state,certificate_state,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'OPEN','ASSIGNED','OPEN','OPEN',$6,$6)`, []any{workOrderID, tenantID, organizationID, "client-a", "job-a", actor.ActorID}},
		{`INSERT INTO work_order_scope_item (id,tenant_id,organization_id,work_order_id,client_id,location_id,asset_id,asset_type) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, []any{scopeID, tenantID, organizationID, workOrderID, "client-a", "location-a", "asset-a", "lifting"}},
		{`INSERT INTO work_order_assignment (id,tenant_id,organization_id,work_order_id,inspector_id,state,effective_from,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,'ACTIVE',$6,$5,$5)`, []any{assignmentID, tenantID, organizationID, workOrderID, actor.ActorID, now}},
		{`INSERT INTO work_order_assignment_scope (tenant_id,organization_id,assignment_id,scope_item_id) VALUES ($1,$2,$3,$4)`, []any{tenantID, organizationID, assignmentID, scopeID}},
		{`INSERT INTO inspection_record (id,tenant_id,organization_id,work_order_id,scope_item_id,assignment_id,asset_id,inspector_id,lifecycle_state,revision,finalization_state,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'APPROVED',7,'FINALIZED',$8,$8)`, []any{inspectionID, tenantID, organizationID, workOrderID, scopeID, assignmentID, "asset-a", actor.ActorID}},
		{`INSERT INTO certificate_template (id,tenant_id,organization_id,template_code,version,title,asset_type,status,catalog_version,page_count,page_width_points,page_height_points,created_by,approved_by,approved_at) VALUES ($1,$2,$3,'lifting',3,'Lifting certificate','lifting','APPROVED',1,1,612,792,$4,$4,$5)`, []any{templateID, tenantID, organizationID, actor.ActorID, now}},
		{`INSERT INTO certificate_policy (id,tenant_id,organization_id,template_code,template_version,policy_version,validity_days,self_issue_allowed,status,created_by,approved_by,approved_at) VALUES ($1,$2,$3,'lifting',3,1,365,false,'APPROVED',$4,$4,$5)`, []any{policyID, tenantID, organizationID, actor.ActorID, now}},
		{`INSERT INTO asset_registry (id,tenant_id,organization_id,asset_id,asset_type,serial_number,description,lifecycle_state,created_by,updated_by) VALUES ($1,$2,$3,'asset-a','lifting','SERIAL-A','Controlled lifting asset','ACTIVE',$4,$4)`, []any{assetRegistryID, tenantID, organizationID, actor.ActorID}},
		{`INSERT INTO inspection_public_scope (id,tenant_id,organization_id,inspection_id,inspection_revision,inspection_type,taxonomy_version,result_state,created_by) VALUES ($1,$2,$3,$4,7,'periodic_lifting',1,'PASS',$5)`, []any{publicScopeID, tenantID, organizationID, inspectionID, actor.ActorID}},
		{`INSERT INTO inspection_public_scope_item (id,tenant_id,organization_id,scope_id,scope_code,display_label,outcome,display_order) VALUES ($1,$2,$3,$4,'visual','Visual examination','PASS',1)`, []any{publicScopeID + "-item", tenantID, organizationID, publicScopeID}},
	} {
		if _, err := db.ExecContext(ctx, seed.query, seed.args...); err != nil {
			t.Fatal(err)
		}
	}
	for order, bindingKey := range []string{"asset.serial_number", "asset.description", "asset.type", "inspection.type", "inspection.public_scope"} {
		if _, err := db.ExecContext(ctx, `INSERT INTO certificate_policy_public_binding (id,tenant_id,organization_id,policy_id,binding_key,required,display_order,created_by) VALUES ($1,$2,$3,$4,$5,true,$6,$7)`, fmt.Sprintf("binding-%d-%d", stamp, order), tenantID, organizationID, policyID, bindingKey, order+1, actor.ActorID); err != nil {
			t.Fatal(err)
		}
	}

	repository, err := certificatepg.NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateDraft(ctx, actor, certificateauthority.CreateDraftRequest{CertificateID: certificateID, InspectionID: inspectionID, TemplateCode: "lifting", TemplateVersion: 3, Profile: certificateauthority.IndependentReview}, now); err != nil {
		t.Fatal(err)
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
	issuer := actor
	issuer.ActorID = "issuer-a"
	issuer.Capabilities = map[string]bool{"certificate.sign": true, "certificate.issue": true}
	if err := repository.Sign(ctx, issuer, certificateID, now); err != nil {
		t.Fatal(err)
	}
	issued, err := repository.Issue(ctx, issuer, certificateID, now)
	if err != nil {
		t.Fatal(err)
	}

	runtime := httptest.NewServer(server.NewMux(server.Dependencies{CertificatePublicHandler: &Handler{Verifier: repository, Limit: 64}}))
	t.Cleanup(runtime.Close)
	url := runtime.URL + "/verify/certificates/" + issued.PublicToken
	client := runtime.Client()
	measurements := make([]time.Duration, 0, 20)
	for range 20 {
		started := time.Now()
		response, err := client.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, response.Body)
		response.Body.Close()
		if response.StatusCode != http.StatusOK || response.Header.Get("Cache-Control") != "no-store" {
			t.Fatalf("status=%d cache=%q", response.StatusCode, response.Header.Get("Cache-Control"))
		}
		measurements = append(measurements, time.Since(started))
	}
	sort.Slice(measurements, func(i, j int) bool { return measurements[i] < measurements[j] })
	var total time.Duration
	for _, measurement := range measurements {
		total += measurement
	}
	t.Logf("db_http_public_verifier samples=%d mean=%s min=%s p95=%s max=%s", len(measurements), total/time.Duration(len(measurements)), measurements[0], measurements[18], measurements[len(measurements)-1])
}

package certtemplatepg

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"integin/internal/domain/certificatetemplate"

	_ "github.com/lib/pq"
)

func TestPostgresResolveCanonicalInspectionTemplateIntegration(t *testing.T) {
	dsn := os.Getenv(certificateTemplateIntegrationDSNEnv)
	if dsn == "" {
		t.Skipf("set %s to run certificate template PostgreSQL integration tests", certificateTemplateIntegrationDSNEnv)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}
	stamp := time.Now().UTC().UnixNano()
	actor := certificatetemplate.ActorContext{TenantID: fmt.Sprintf("it-certificate-template-resolver-%d", stamp), OrganizationID: "org-1", ActorID: "template-admin-1"}
	workOrderID := fmt.Sprintf("order-%d", stamp)
	scopeID := fmt.Sprintf("scope-%d", stamp)
	assignmentID := fmt.Sprintf("assignment-%d", stamp)
	inspectionID := fmt.Sprintf("inspection-%d", stamp)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		for _, statement := range []string{
			`DELETE FROM certificate_template_cell WHERE tenant_id = $1`,
			`DELETE FROM certificate_template WHERE tenant_id = $1`,
			`DELETE FROM inspection_record WHERE tenant_id = $1`,
			`DELETE FROM work_order_assignment_scope WHERE tenant_id = $1`,
			`DELETE FROM work_order_assignment WHERE tenant_id = $1`,
			`DELETE FROM work_order_scope_item WHERE tenant_id = $1`,
			`DELETE FROM work_order WHERE tenant_id = $1`,
		} {
			if _, err := db.ExecContext(cleanupCtx, statement, actor.TenantID); err != nil {
				t.Errorf("cleanup certificate resolver fixtures: %v", err)
				return
			}
		}
	})
	now := time.Date(2026, time.August, 21, 22, 30, 0, 0, time.UTC)
	if _, err := db.ExecContext(ctx, `SELECT set_config('integin.tenant_id',$1,false), set_config('integin.organization_id',$2,false)`, actor.TenantID, actor.OrganizationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `SELECT set_config('integin.tenant_id','',false), set_config('integin.organization_id','',false)`)
	})
	if _, err := db.ExecContext(ctx, `INSERT INTO work_order (id,tenant_id,organization_id,client_id,job_number,request_state,execution_state,commercial_state,certificate_state,revision,created_by,updated_by) VALUES ($1,$2,$3,'client-1',$4,'accepted','in_progress','ready_for_office_confirmation','pending_validation',1,$5,$5)`, workOrderID, actor.TenantID, actor.OrganizationID, "JOB-"+fmt.Sprint(stamp), actor.ActorID); err != nil {
		t.Fatalf("seed work order: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO work_order_scope_item (id,tenant_id,organization_id,work_order_id,client_id,location_id,asset_id,asset_type) VALUES ($1,$2,$3,$4,'client-1','location-1','asset-1','lifting')`, scopeID, actor.TenantID, actor.OrganizationID, workOrderID); err != nil {
		t.Fatalf("seed scope item: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO work_order_assignment (id,tenant_id,organization_id,work_order_id,inspector_id,state,revision,effective_from,created_by,updated_by) VALUES ($1,$2,$3,$4,'inspector-1','active',1,$5,$6,$6)`, assignmentID, actor.TenantID, actor.OrganizationID, workOrderID, now, actor.ActorID); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO work_order_assignment_scope (tenant_id,organization_id,assignment_id,scope_item_id) VALUES ($1,$2,$3,$4)`, actor.TenantID, actor.OrganizationID, assignmentID, scopeID); err != nil {
		t.Fatalf("seed assignment scope: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO inspection_record (id,tenant_id,organization_id,work_order_id,scope_item_id,assignment_id,asset_id,inspector_id,lifecycle_state,revision,finalization_state,created_by,updated_by,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,'asset-1','inspector-1','COMPLETED',2,'SUBMITTED',$7,$7,$8,$8)`, inspectionID, actor.TenantID, actor.OrganizationID, workOrderID, scopeID, assignmentID, actor.ActorID, now); err != nil {
		t.Fatalf("seed inspection: %v", err)
	}
	repository, err := NewRepository(db, certificatetemplate.DefaultCatalog())
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	definition := certificatetemplate.Definition{ID: fmt.Sprintf("template-%d", stamp), TemplateCode: "lifting_certificate", Version: 1, Title: "Lifting certificate", AssetType: "lifting", Status: certificatetemplate.Draft, CatalogVersion: 1, PageCount: 1, PageWidth: 595, PageHeight: 842, Cells: []certificatetemplate.Cell{
		{ID: "asset_id", PageNumber: 1, Rectangle: certificatetemplate.Rectangle{X: 20, Y: 20, Width: 180, Height: 30}, Kind: certificatetemplate.TextCell, BindingKey: certificatetemplate.InspectionAssetID, FitPolicy: certificatetemplate.SingleLineRequired, MaxLines: 1, Required: true},
		{ID: "inspection_state", PageNumber: 1, Rectangle: certificatetemplate.Rectangle{X: 20, Y: 60, Width: 180, Height: 30}, Kind: certificatetemplate.TextCell, BindingKey: certificatetemplate.InspectionLifecycleState, FitPolicy: certificatetemplate.SingleLineRequired, MaxLines: 1, Required: true},
	}}
	if _, _, err := repository.RegisterDraft(ctx, actor, definition); err != nil {
		t.Fatalf("register template: %v", err)
	}
	if _, err := repository.Approve(ctx, actor, definition.TemplateCode, definition.Version, now); err != nil {
		t.Fatalf("approve template: %v", err)
	}
	resolved, err := repository.ResolveInspectionTemplate(ctx, actor, definition.TemplateCode, definition.Version, inspectionID)
	if err != nil || len(resolved.Cells) != 2 {
		t.Fatalf("resolve template: %#v err=%v", resolved, err)
	}
	values := map[string]string{}
	for _, cell := range resolved.Cells {
		values[cell.CellID] = cell.RenderedText
	}
	if values["asset_id"] != "asset-1" || values["inspection_state"] != "COMPLETED" {
		t.Fatalf("unexpected resolved canonical values: %#v", values)
	}
	if _, err := repository.ResolveInspectionTemplate(ctx, certificatetemplate.ActorContext{TenantID: actor.TenantID, OrganizationID: "org-2", ActorID: "template-admin-2"}, definition.TemplateCode, definition.Version, inspectionID); err != ErrNotFound {
		t.Fatalf("expected cross-organization not found, got %v", err)
	}
}

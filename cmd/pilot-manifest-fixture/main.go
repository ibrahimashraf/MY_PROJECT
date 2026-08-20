package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"integin/internal/domain/device_trust"
	"integin/internal/domain/workpackage"
	"integin/internal/fixturesignals"
)

type fixture struct {
	DeviceID        string `json:"device_id"`
	AuthorityID     string `json:"authority_id"`
	TenantID        string `json:"tenant_id"`
	OrganizationID  string `json:"organization_id"`
	InspectionID    string `json:"inspection_id"`
	UserID          string `json:"user_id"`
	DeviceKeyID     string `json:"device_key_id"`
	DevicePublicKey string `json:"device_public_key_base64url"`
}

const pilotFieldAssetIDsJSON = `{"condition":"pilot-manifest-demo-asset"}`

func q(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func main() {
	if len(os.Args) != 2 {
		panic("fixture path required")
	}
	var f fixture
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(raw, &f); err != nil {
		panic(err)
	}
	if f.DeviceID == "" || f.AuthorityID == "" || f.TenantID == "" || f.OrganizationID == "" || f.InspectionID == "" || f.UserID == "" || f.DeviceKeyID == "" || f.DevicePublicKey == "" {
		panic("public pilot fixture is incomplete")
	}
	authoritySecret := strings.TrimSpace(os.Getenv("INTEGIN_PILOT_FIXTURE_AUTHORITY_SECRET"))
	if authoritySecret == "" {
		panic("isolated pilot fixture authority secret is unavailable")
	}
	p := workpackage.Package{ID: "pilot-manifest-demo-package", TenantID: f.TenantID, OrganizationID: f.OrganizationID, TemplateCode: "pilot-manifest-demo", TemplateVersion: 1, PackageVersion: 1, SchemaVersion: 1, State: workpackage.PublicationApproved, Sections: []workpackage.Section{{ID: "pilot-section", Title: "Pilot manifest binding", Fields: []workpackage.FieldDefinition{{ID: "condition", Prompt: "Pilot condition", Type: workpackage.FieldText, Required: true}}}}}
	p, err = p.WithComputedHash()
	if err != nil {
		panic(err)
	}
	definition, err := json.Marshal(p.Sections)
	if err != nil {
		panic(err)
	}
	pub, err := base64.RawURLEncoding.DecodeString(f.DevicePublicKey)
	if err != nil {
		panic(err)
	}
	// PostgreSQL timestamptz persists microsecond precision; sign precisely the
	// values that the candidate will reload and verify from the isolated database.
	now := time.Now().UTC().Truncate(time.Microsecond)
	device, err := device_trust.NewDevice(f.DeviceID, f.TenantID, f.OrganizationID, f.UserID, f.DevicePublicKey)
	if err != nil {
		panic(err)
	}
	if err := device.Trust(); err != nil {
		panic(err)
	}
	authority, err := device_trust.IssueAuthorityPackage(device, f.AuthorityID, authoritySecret, []string{"work_package.read"}, now, 2*time.Hour)
	if err != nil {
		panic(err)
	}
	if authority.IssuedAt.Nanosecond()%int(time.Microsecond) != 0 || authority.ExpiresAt.Nanosecond()%int(time.Microsecond) != 0 {
		panic("isolated pilot authority timestamps must be database-precision aligned")
	}
	fmt.Printf("BEGIN;\n")
	fmt.Printf("INSERT INTO device_registry (device_id,tenant_id,organization_id,user_id,key_id,public_key,state,authority_epoch,enrolled_at,updated_at) VALUES (%s,%s,%s,%s,%s,decode(%s,'base64'),'TRUSTED',1,now(),now()) ON CONFLICT (device_id) DO UPDATE SET key_id=EXCLUDED.key_id,public_key=EXCLUDED.public_key,state=EXCLUDED.state,authority_epoch=EXCLUDED.authority_epoch,updated_at=now() WHERE device_registry.tenant_id=EXCLUDED.tenant_id AND device_registry.organization_id=EXCLUDED.organization_id AND device_registry.user_id=EXCLUDED.user_id;\n", q(f.DeviceID), q(f.TenantID), q(f.OrganizationID), q(f.UserID), q(f.DeviceKeyID), q(base64.StdEncoding.EncodeToString(pub)))
	fmt.Printf("INSERT INTO authority_package (authority_id,tenant_id,organization_id,device_id,user_id,authority_epoch,scopes,procedure_version,issued_at,expires_at,signature) VALUES (%s,%s,%s,%s,%s,1,'[\"work_package.read\"]'::jsonb,'pilot-v1',%s::timestamptz,%s::timestamptz,convert_to(%s,'UTF8')) ON CONFLICT (authority_id) DO UPDATE SET scopes=EXCLUDED.scopes,procedure_version=EXCLUDED.procedure_version,issued_at=EXCLUDED.issued_at,expires_at=EXCLUDED.expires_at,signature=EXCLUDED.signature WHERE authority_package.tenant_id=EXCLUDED.tenant_id AND authority_package.organization_id=EXCLUDED.organization_id AND authority_package.device_id=EXCLUDED.device_id AND authority_package.user_id=EXCLUDED.user_id AND authority_package.authority_epoch=EXCLUDED.authority_epoch;\n", q(f.AuthorityID), q(f.TenantID), q(f.OrganizationID), q(f.DeviceID), q(f.UserID), q(authority.IssuedAt.Format(time.RFC3339Nano)), q(authority.ExpiresAt.Format(time.RFC3339Nano)), q(authority.Signature))
	fmt.Printf("INSERT INTO work_package (tenant_id,organization_id,package_id,package_version,template_code,template_version,schema_version,publication_state,package_hash,definition,created_at,approved_at) VALUES (%s,%s,%s,1,'pilot-manifest-demo',1,1,'approved',%s,%s::jsonb,now(),now()) ON CONFLICT (tenant_id,organization_id,package_id,package_version) DO UPDATE SET template_code=EXCLUDED.template_code,template_version=EXCLUDED.template_version,schema_version=EXCLUDED.schema_version,publication_state=EXCLUDED.publication_state,package_hash=EXCLUDED.package_hash,definition=EXCLUDED.definition,approved_at=EXCLUDED.approved_at;\n", q(f.TenantID), q(f.OrganizationID), q(p.ID), q(p.PackageHash), q(string(definition)))
	fmt.Printf("INSERT INTO work_package_assignment (tenant_id,organization_id,inspection_id,device_id,package_id,package_version,authority_epoch,expires_at,assigned_at) VALUES (%s,%s,%s,%s,%s,1,1,%s::timestamptz,now()) ON CONFLICT (tenant_id,organization_id,inspection_id) DO UPDATE SET expires_at=EXCLUDED.expires_at WHERE work_package_assignment.device_id=EXCLUDED.device_id AND work_package_assignment.package_id=EXCLUDED.package_id AND work_package_assignment.package_version=EXCLUDED.package_version AND work_package_assignment.authority_epoch=EXCLUDED.authority_epoch;\n", q(f.TenantID), q(f.OrganizationID), q(f.InspectionID), q(f.DeviceID), q(p.ID), q(authority.ExpiresAt.Format(time.RFC3339Nano)))
	fmt.Printf("INSERT INTO work_package_assignment_context (tenant_id,organization_id,inspection_id,root_asset_id,inspection_type,procedure_version,scheduled_at,field_asset_ids) VALUES (%s,%s,%s,'pilot-manifest-demo-asset','pilot-manifest-demo','pilot-v1',now(),%s::jsonb) ON CONFLICT (tenant_id,organization_id,inspection_id) DO UPDATE SET root_asset_id=EXCLUDED.root_asset_id,inspection_type=EXCLUDED.inspection_type,procedure_version=EXCLUDED.procedure_version,scheduled_at=EXCLUDED.scheduled_at,field_asset_ids=EXCLUDED.field_asset_ids WHERE work_package_assignment_context.tenant_id=EXCLUDED.tenant_id AND work_package_assignment_context.organization_id=EXCLUDED.organization_id;\nCOMMIT;\n", q(f.TenantID), q(f.OrganizationID), q(f.InspectionID), q(pilotFieldAssetIDsJSON))
	if _, err := fixturesignals.EmitFromEnvironment(time.Now); err != nil {
		panic(err)
	}
}

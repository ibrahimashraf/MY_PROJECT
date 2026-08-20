package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

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
	now := time.Now().UTC()
	expires := now.Add(2 * time.Hour)
	fmt.Printf("BEGIN;\n")
	fmt.Printf("INSERT INTO device_registry (device_id,tenant_id,organization_id,user_id,key_id,public_key,state,authority_epoch,enrolled_at,updated_at) VALUES (%s,%s,%s,%s,%s,decode(%s,'base64'),'TRUSTED',1,now(),now()) ON CONFLICT (device_id) DO NOTHING;\n", q(f.DeviceID), q(f.TenantID), q(f.OrganizationID), q(f.UserID), q(f.DeviceKeyID), q(base64.StdEncoding.EncodeToString(pub)))
	fmt.Printf("INSERT INTO authority_package (authority_id,tenant_id,organization_id,device_id,user_id,authority_epoch,scopes,procedure_version,issued_at,expires_at,signature) VALUES (%s,%s,%s,%s,%s,1,'[\"work_package_manifest.read\"]'::jsonb,'pilot-v1',now(),%s,''::bytea) ON CONFLICT (authority_id) DO NOTHING;\n", q(f.AuthorityID), q(f.TenantID), q(f.OrganizationID), q(f.DeviceID), q(f.UserID), q(expires.Format(time.RFC3339)))
	fmt.Printf("INSERT INTO work_package (tenant_id,organization_id,package_id,package_version,template_code,template_version,schema_version,publication_state,package_hash,definition,created_at,approved_at) VALUES (%s,%s,%s,1,'pilot-manifest-demo',1,1,'approved',%s,%s::jsonb,now(),now()) ON CONFLICT DO NOTHING;\n", q(f.TenantID), q(f.OrganizationID), q(p.ID), q(p.PackageHash), q(string(definition)))
	fmt.Printf("INSERT INTO work_package_assignment (tenant_id,organization_id,inspection_id,device_id,package_id,package_version,authority_epoch,expires_at,assigned_at) VALUES (%s,%s,%s,%s,%s,1,1,%s,now()) ON CONFLICT (tenant_id,organization_id,inspection_id) DO NOTHING;\n", q(f.TenantID), q(f.OrganizationID), q(f.InspectionID), q(f.DeviceID), q(p.ID), q(expires.Format(time.RFC3339)))
	fmt.Printf("INSERT INTO work_package_assignment_context (tenant_id,organization_id,inspection_id,root_asset_id,inspection_type,procedure_version,scheduled_at,field_asset_ids) VALUES (%s,%s,%s,'pilot-manifest-demo-asset','pilot-manifest-demo','pilot-v1',now(),'{\"condition\":[\"pilot-manifest-demo-asset\"]}'::jsonb) ON CONFLICT (tenant_id,organization_id,inspection_id) DO NOTHING;\nCOMMIT;\n", q(f.TenantID), q(f.OrganizationID), q(f.InspectionID))
	if _, err := fixturesignals.EmitFromEnvironment(time.Now); err != nil {
		panic(err)
	}
}

package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetentionLegalHoldPolicyV1IsScopedNonEnforcingAndAuditable(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("retention_legal_hold_redaction_export_access_policy_v1.json"))
	if err != nil {
		t.Fatalf("read retention policy: %v", err)
	}

	var policy struct {
		PolicyVersion string `json:"policy_version"`
		Status        string `json:"status"`
		Scope         struct {
			TenantScopeRequirement      string `json:"tenant_scope_requirement"`
			PolicyActivationRequirement string `json:"policy_activation_requirement"`
		} `json:"scope"`
		Retention struct {
			DispositionState string `json:"disposition_state"`
		} `json:"retention"`
		LegalHold struct {
			PrecedenceRule string `json:"precedence_rule"`
		} `json:"legal_hold"`
		Redaction struct {
			SourceImmutabilityRule string `json:"source_immutability_rule"`
			TransformationState    string `json:"transformation_state"`
		} `json:"redaction"`
		ExportAccess struct {
			AccessState         string `json:"access_state"`
			ManifestRequirement string `json:"manifest_requirement"`
		} `json:"export_access"`
		AuditAndRecovery struct {
			AuditEventRequirements []string `json:"audit_event_requirements"`
			RecoveryRequirement    string   `json:"recovery_requirement"`
		} `json:"audit_and_recovery"`
		ConflictResolution struct {
			AutomaticResolution string `json:"automatic_resolution"`
		} `json:"conflict_resolution"`
		ImplementationDeferrals []string `json:"implementation_deferrals"`
	}
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatalf("policy must be valid JSON: %v", err)
	}
	if policy.PolicyVersion != "retention-legal-hold-redaction-export-access/v1" {
		t.Fatalf("policy version = %q", policy.PolicyVersion)
	}
	if policy.Status != "design-baseline-not-enforced" {
		t.Fatalf("policy status = %q", policy.Status)
	}
	if policy.Scope.TenantScopeRequirement != "exactly-one-tenant-per-policy-instance" || policy.Scope.PolicyActivationRequirement == "" {
		t.Fatal("policy must require tenant scope and accountable activation")
	}
	if policy.Retention.DispositionState != "not-implemented" || policy.Redaction.TransformationState != "not-implemented" || policy.ExportAccess.AccessState != "not-implemented" {
		t.Fatal("policy must not claim enforcement implementation")
	}
	if !strings.Contains(strings.ToLower(policy.LegalHold.PrecedenceRule), "blocks disposition") {
		t.Fatal("legal hold must block disposition")
	}
	if !strings.Contains(strings.ToLower(policy.Redaction.SourceImmutabilityRule), "immutable") {
		t.Fatal("redaction must preserve immutable source evidence")
	}
	if policy.ExportAccess.ManifestRequirement == "" || len(policy.AuditAndRecovery.AuditEventRequirements) < 6 || policy.AuditAndRecovery.RecoveryRequirement == "" {
		t.Fatal("policy requires export-manifest and auditable recovery controls")
	}
	if policy.ConflictResolution.AutomaticResolution != "prohibited" {
		t.Fatal("policy must prohibit automatic conflict resolution")
	}
	if len(policy.ImplementationDeferrals) < 3 {
		t.Fatal("policy must declare implementation deferrals")
	}
}

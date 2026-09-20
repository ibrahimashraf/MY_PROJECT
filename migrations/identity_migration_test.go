// INTEGIN OIDC foundation migration contract: runtime resolves only a local issuer-subject membership function.
package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestIdentitySubjectMembershipMigrationContract(t *testing.T) {
	sql, err := os.ReadFile("0004_identity_subject_membership.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	required := []string{
		"CREATE TABLE IF NOT EXISTS identity_subject",
		"UNIQUE (issuer, subject)",
		"CREATE TABLE IF NOT EXISTS identity_membership",
		"identity_membership_one_active_per_subject_idx",
		"CREATE TABLE IF NOT EXISTS identity_membership_capability",
		"CREATE OR REPLACE FUNCTION integin_resolve_identity_membership",
		"SECURITY DEFINER",
		"REVOKE ALL ON TABLE identity_subject",
		"REVOKE ALL ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM PUBLIC",
		"GRANT EXECUTE ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) TO integin_test_runtime",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
	if _, err := os.Stat("0004_identity_subject_membership.down.sql"); err != nil {
		t.Fatal(err)
	}
}

func TestIdentityRuntimeRoleIsStandardized(t *testing.T) {
	sql, err := os.ReadFile("0004_identity_subject_membership.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	if !strings.Contains(text, "TO integin_test_runtime") {
		t.Fatal("identity migration must grant resolver execution to integin_test_runtime")
	}
	if strings.Contains(text, "integin_pilot_runtime") {
		t.Fatal("identity migration must not retain the obsolete integin_pilot_runtime contract")
	}
}

func TestIdentityActorAlignmentMigrationContract(t *testing.T) {
	sql, err := os.ReadFile("0008_identity_actor_alignment.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	required := []string{
		"CREATE TABLE IF NOT EXISTS identity_actor",
		"actor_id TEXT PRIMARY KEY",
		"tenant_id TEXT NOT NULL",
		"organization_id TEXT NOT NULL",
		"status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED'))",
		"ADD COLUMN actor_id TEXT",
		"ADD COLUMN work_order_role TEXT",
		"identity_membership_actor_tenant_org_fk",
		"identity_membership_work_order_role_ck",
		"RAISE EXCEPTION",
		"RETURNS TABLE (",
		"actor_id TEXT",
		"tenant_id TEXT",
		"organization_id TEXT",
		"work_order_role TEXT",
		"capabilities TEXT[]",

		"SECURITY DEFINER",
		"SET search_path = pg_catalog, public",
		"GRANT EXECUTE ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) TO integin_test_runtime",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("actor alignment migration missing %q", fragment)
		}
	}
	if strings.Contains(text, "UPDATE identity_membership SET actor_id") {
		t.Fatal("actor alignment migration must not guess existing actor mappings")
	}
	if _, err := os.Stat("0008_identity_actor_alignment.down.sql"); err != nil {
		t.Fatal(err)
	}
}

func TestIdentityActorAlignmentRollbackContract(t *testing.T) {
	sql, err := os.ReadFile("0008_identity_actor_alignment.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	required := []string{
		"REVOKE EXECUTE ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM integin_test_runtime",
		"DROP FUNCTION IF EXISTS integin_resolve_identity_membership(TEXT, TEXT)",
		"DROP CONSTRAINT IF EXISTS identity_membership_actor_tenant_org_fk",
		"DROP TABLE IF EXISTS identity_actor",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("actor alignment rollback missing %q", fragment)
		}
	}
}

func TestIdentityActorAlignmentMigrationRejectsUnmappedMemberships(t *testing.T) {
	sql, err := os.ReadFile("0008_identity_actor_alignment.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	if !strings.Contains(text, "WHERE actor_id IS NULL OR work_order_role IS NULL") {
		t.Fatal("actor alignment migration must fail closed when existing memberships are unmapped")
	}
	if !strings.Contains(text, "actor_id IS NULL OR tenant_id IS NULL OR organization_id IS NULL") {
		t.Fatal("actor alignment migration must fail closed when actor identity fields are incomplete")
	}
}


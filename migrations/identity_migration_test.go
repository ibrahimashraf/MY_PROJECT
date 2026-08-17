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
		"GRANT EXECUTE ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) TO integin_pilot_runtime",
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

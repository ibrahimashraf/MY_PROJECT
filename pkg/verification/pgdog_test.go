package verification

import (
	"net/url"
	"os"
	"strings"
	"testing"
)

const (
	pgdogApplianceUser     = "integin_owner"
	pgdogApplianceDatabase = "integin_appliance"
	pgdogApplianceDSN      = "postgres://integin_owner:integin_sovereign_db_secret@appliance-pgdog:6432/integin_appliance?sslmode=disable&default_query_exec_mode=exec"
)

func TestPgDogApplianceConfigReconciled(t *testing.T) {
	pgdogRaw, err := os.ReadFile("../../deployments/pgdog.toml")
	if err != nil {
		t.Fatalf("read pgdog.toml: %v", err)
	}
	pgdogContent := string(pgdogRaw)

	usersRaw, err := os.ReadFile("../../deployments/users.toml")
	if err != nil {
		t.Fatalf("read users.toml: %v", err)
	}
	usersContent := string(usersRaw)

	// Validate pgdog.toml general and database definitions
	requiredDbSettings := []string{
		`port = 6432`,
		`name = "integin_appliance"`,
		`host = "appliance-postgres"`,
		`pool_mode = "transaction"`,
		`name = "integin_migration_test"`,
	}
	for _, want := range requiredDbSettings {
		if !strings.Contains(pgdogContent, want) {
			t.Errorf("pgdog.toml missing %q", want)
		}
	}

	// Validate users.toml credentials mapping
	requiredUserSettings := []string{
		`name = "` + pgdogApplianceUser + `"`,
		`database = "` + pgdogApplianceDatabase + `"`,
		`name = "postgres"`,
		`database = "integin_migration_test"`,
	}
	for _, want := range requiredUserSettings {
		if !strings.Contains(usersContent, want) {
			t.Errorf("users.toml missing %q", want)
		}
	}
}

func TestPgDogApplianceDSNParsesExecMode(t *testing.T) {
	u, err := url.Parse(pgdogApplianceDSN)
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}
	if u.Scheme != "postgres" {
		t.Errorf("scheme = %q", u.Scheme)
	}
	if username := u.User.Username(); username != pgdogApplianceUser {
		t.Errorf("user = %q, want %q", username, pgdogApplianceUser)
	}
	if u.Hostname() != "appliance-pgdog" {
		t.Errorf("host = %q, want appliance-pgdog", u.Hostname())
	}
	if u.Port() != "6432" {
		t.Errorf("port = %q, want 6432 (pgdog listener)", u.Port())
	}
	if !strings.EqualFold(u.Path, "/"+pgdogApplianceDatabase) {
		t.Errorf("database path = %q, want /integin_appliance", u.Path)
	}
	q := u.Query()
	if got := q.Get("default_query_exec_mode"); got != "exec" {
		t.Errorf("default_query_exec_mode = %q, want exec", got)
	}
	if got := q.Get("sslmode"); got != "disable" {
		t.Errorf("sslmode = %q, want disable", got)
	}
}

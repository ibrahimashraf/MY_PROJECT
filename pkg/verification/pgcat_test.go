package verification

import (
	"net/url"
	"os"
	"strings"
	"testing"
)

const (
	pgcatApplianceUser     = "integin_owner"
	pgcatApplianceDatabase = "integin_appliance"
	pgcatApplianceDSN      = "postgres://integin_owner:integin_sovereign_db_secret@appliance-pgcat:6432/integin_appliance?sslmode=disable&default_query_exec_mode=exec"
)

// sectionBlocks splits a TOML file into [header] blocks and returns the raw
// text of every block whose header starts with the given prefix.
func sectionBlocks(t *testing.T, path, prefix string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var blocks []string
	var cur string
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && !strings.Contains(trimmed, `"`) {
			if strings.HasPrefix(trimmed, prefix) {
				blocks = append(blocks, cur)
			}
			cur = line
			continue
		}
		if strings.TrimSpace(cur) != "" {
			cur += "\n" + line
		}
	}
	if strings.HasPrefix(strings.TrimSpace(cur), prefix) {
		blocks = append(blocks, cur)
	}
	return strings.Join(blocks, "\n")
}

func TestPgCatApplianceConfigReconciled(t *testing.T) {
	appliance := sectionBlocks(t, "../../deployments/pgcat.toml", "[pools.integin_appliance")
	required := []string{
		`[pools.integin_appliance]`,
		`pool_mode = "transaction"`,
		`query_parser_enabled = false`,
		`[pools.integin_appliance.users.0]`,
		`username = "` + pgcatApplianceUser + `"`,
		`pool_size = 80`,
		`[pools.integin_appliance.shards.0]`,
		`"appliance-postgres"`,
		`database = "` + pgcatApplianceDatabase + `"`,
	}
	for _, want := range required {
		if !strings.Contains(appliance, want) {
			t.Errorf("pgcat.toml appliance pool missing %q", want)
		}
	}
	// Sibling consumers (internal/chaos, docker-compose.pgcat.yml) resolve the
	// local dev pool by name; it must survive the appliance reconciliation.
	if !strings.Contains(sectionBlocks(t, "../../deployments/pgcat.toml", "[pools.integin_migration_test"), `[pools.integin_migration_test]`) {
		t.Error("pgcat.toml must keep the integin_migration_test pool for local dev tooling")
	}
}

func TestPgCatApplianceDSNParsesExecMode(t *testing.T) {
	u, err := url.Parse(pgcatApplianceDSN)
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}
	if u.Scheme != "postgres" {
		t.Errorf("scheme = %q", u.Scheme)
	}
	if username := u.User.Username(); username != pgcatApplianceUser {
		t.Errorf("user = %q, want %q", username, pgcatApplianceUser)
	}
	if u.Hostname() != "appliance-pgcat" {
		t.Errorf("host = %q, want appliance-pgcat", u.Hostname())
	}
	if u.Port() != "6432" {
		t.Errorf("port = %q, want 6432 (pgcat listener)", u.Port())
	}
if !strings.EqualFold(u.Path, "/"+pgcatApplianceDatabase) {
		t.Errorf("database path = %q, want /integin_appliance", u.Path)
	}
	q := u.Query()
	if got := q.Get("default_query_exec_mode"); got != "exec" {
		t.Errorf("default_query_exec_mode = %q, want exec (simple-protocol fallback under transaction pooling)", got)
	}
	if got := q.Get("sslmode"); got != "disable" {
		t.Errorf("sslmode = %q, want disable (appliance edge)", got)
	}
}


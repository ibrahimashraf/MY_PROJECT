// Command package drdrill hosts the CI-executable Disaster Recovery drill
// harness verification. It runs under `go test ./...` on any agent (Linux
// runners or a developer workstation) without requiring physical dual-NVMe
// hardware: the drill's dry-run path is exercised against temporary paths and
// the <60s RTO SLA is asserted.
package drdrill

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const drSLA = 60 * time.Second

// pgBackRestConf is the parsed shape of pgbackrest.conf sections.
type pgBackRestConf map[string]map[string]string

// parsePgBackRestConf is a minimal INI/stanza parser sufficient for the
// pgbackrest.conf contract (sections in [brackets], key=value pairs).
// Duplicate sections merge; duplicates keys keep the last value.
func parsePgBackRestConf(t *testing.T, path string) pgBackRestConf {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	conf := pgBackRestConf{}
	current := ""
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = line[1 : len(line)-1]
			if conf[current] == nil {
				conf[current] = map[string]string{}
			}
			continue
		}
		if key, value, ok := strings.Cut(line, "="); ok {
			conf[current][strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return conf
}

func TestPgBackRestConfStanzaParsing(t *testing.T) {
	conf := parsePgBackRestConf(t, "pgbackrest.conf")

	global, ok := conf["global"]
	if !ok {
		t.Fatal("missing [global] section")
	}
	for key, want := range map[string]string{
		"repo1-path":           "/mnt/nvme-backup/pgbackrest",
		"repo1-retention-full": "2",
		"compress-type":        "lz4",
		"process-max":          "4",
	} {
		if got := global[key]; got != want {
			t.Errorf("global.%s = %q, want %q", key, got, want)
		}
	}

	stanza, ok := conf["integin-appliance"]
	if !ok {
		t.Fatal("missing [integin-appliance] stanza")
	}
	for key, want := range map[string]string{
		"pg1-path": "/var/lib/postgresql/data",
		"pg1-user": "postgres",
		"pg1-port": "5432",
	} {
		if got := stanza[key]; got != want {
			t.Errorf("stanza.%s = %q, want %q", key, got, want)
		}
	}
}

func TestRestoreApplianceScriptPathAssertions(t *testing.T) {
	script, err := os.ReadFile("restore-appliance.sh")
	if err != nil {
		t.Fatalf("read drill script: %v", err)
	}
	body := string(script)
	for _, fragment := range []string{
		`[ ! -d "${BACKUP_REPO}" ]`, // NVMe repo mount guard
		`[ ! -f "${CONFIG_FILE}" ]`, // pgbackrest config guard (dry-run path)
		"60.00s",                    // SLA threshold
		"# Target SLA: <60 seconds", // documented RTO
	} {
		if !strings.Contains(body, fragment) {
			t.Errorf("restore-appliance.sh missing guard/text %q", fragment)
		}
	}
}

// TestRestoreDrillDryRunUnderSLATarget executes the drill in dry-run mode:
// no pgbackrest binary and no physical NVMe — only temporary paths — and
// asserts it completes within the <60s SLA with a PASS verdict.
func TestRestoreDrillDryRunUnderSLATarget(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skipf("bash not available on this host: %v", err)
	}
	if probeOut, probeErr := exec.Command(bash, "-c", "printf ok").CombinedOutput(); probeErr != nil || strings.TrimSpace(string(probeOut)) != "ok" {
		t.Skipf("bash is not functional on this host (probe: %v %q)", probeErr, probeOut)
	}
	if pg, perr := exec.LookPath("pgbackrest"); perr == nil {
		t.Skipf("pgbackrest installed at %s; real dry-run not enforceable", pg)
	}

	dir := t.TempDir()
	repoDir := filepath.Join(dir, "nvme-backup")
	configPath := filepath.Join(dir, "pgbackrest.conf")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config, err := os.ReadFile("pgbackrest.conf")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, config, 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bash, "restore-appliance.sh")
	cmd.Env = append(os.Environ(),
		"PGBACKREST_CONFIG="+configPath,
		"PGBACKREST_REPO="+repoDir,
		"PGBACKREST_STANZA=integin-appliance",
		"PGDATA="+filepath.Join(dir, "pgdata"),
	)

	start := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)

	output := string(out)
	if !strings.Contains(output, "RTO METRIC") {
		t.Errorf("drill did not report an RTO metric; output:\n%s", output)
	}
	if !strings.Contains(output, "STATUS: PASS") {
		t.Errorf("drill did not pass the SLA verdict; output:\n%s", output)
	}
	if err != nil {
		t.Fatalf("dry-run drill failed: %v\noutput:\n%s", err, output)
	}
	if elapsed >= drSLA {
		t.Errorf("dry-run drill took %s, SLA is <60s", elapsed)
	}
}

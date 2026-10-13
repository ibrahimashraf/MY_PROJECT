package migrations

import (
	"strings"
	"testing"
)

func TestDiscoverUpMigrationsContract(t *testing.T) {
	files, err := DiscoverUpMigrations()
	if err != nil {
		t.Fatalf("DiscoverUpMigrations failed: %v", err)
	}

	if len(files) < 80 {
		t.Fatalf("expected at least 80 up migrations, discovered %d", len(files))
	}

	// Verify sequential sorting and exclusion of draft/down files
	for i, f := range files {
		if strings.Contains(f.Name, ".down.") {
			t.Errorf("discovered up migration %s contains .down.", f.Name)
		}
		if strings.Contains(f.Name, ".draft.") {
			t.Errorf("discovered up migration %s contains .draft.", f.Name)
		}
		if f.Checksum == "" {
			t.Errorf("migration %s missing SHA256 checksum", f.Name)
		}
		if i > 0 && files[i-1].Name >= f.Name {
			t.Errorf("migrations not strictly sorted: %s came after %s", f.Name, files[i-1].Name)
		}
	}

	t.Logf("Successfully verified %d embedded up migrations.", len(files))
}

func TestDiscoverDownFilePairing(t *testing.T) {
	down, err := DiscoverDownFile("0001_event_log.sql")
	if err != nil {
		t.Fatalf("DiscoverDownFile for 0001_event_log.sql failed: %v", err)
	}

	if down.Name != "0001_event_log.down.sql" {
		t.Errorf("expected down file 0001_event_log.down.sql, got %s", down.Name)
	}
	if !strings.Contains(down.Content, "DROP TABLE") {
		t.Errorf("expected DROP TABLE in rollback, got: %s", down.Content)
	}
}

func TestNonTransactionalDetection(t *testing.T) {
	files, err := DiscoverUpMigrations()
	if err != nil {
		t.Fatalf("DiscoverUpMigrations failed: %v", err)
	}

	var foundNonTx bool
	for _, f := range files {
		if strings.Contains(f.Content, "CREATE INDEX CONCURRENTLY") {
			if !f.NonTransactional {
				t.Errorf("migration %s contains CREATE INDEX CONCURRENTLY but NonTransactional is false", f.Name)
			}
			foundNonTx = true
		}
	}

	if !foundNonTx {
		t.Log("Note: No CREATE INDEX CONCURRENTLY detected in current up migrations suite.")
	}
}

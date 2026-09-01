package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestEventLogMigrationContract(t *testing.T) {
	sql, err := os.ReadFile("0001_event_log.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	required := []string{"CREATE TABLE IF NOT EXISTS event_log", "tenant_id", "aggregate_version", "PARTITION BY RANGE", "event_log_aggregate_replay_idx", "event_log_tenant_time_idx"}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
	if _, err := os.Stat("0001_event_log.down.sql"); err != nil {
		t.Fatal(err)
	}
}

func TestDeviceTrustSyncMigrationContract(t *testing.T) {
	sql, err := os.ReadFile("0002_device_trust_sync.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	required := []string{
		"CREATE TABLE IF NOT EXISTS device_registry",
		"CREATE TABLE IF NOT EXISTS authority_package",
		"CREATE TABLE IF NOT EXISTS sync_device_state",
		"CREATE TABLE IF NOT EXISTS sync_receipt",
		"CREATE TABLE IF NOT EXISTS sync_held_transaction",
		"last_accepted_sequence",
		"UNIQUE (device_id, sequence_number)",
		"ENABLE ROW LEVEL SECURITY",
		"current_setting('integin.tenant_id', true)",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
	if _, err := os.Stat("0002_device_trust_sync.down.sql"); err != nil {
		t.Fatal(err)
	}
}

func TestEventLogTenantRLSMigrationContract(t *testing.T) {
	sql, err := os.ReadFile("0003_event_log_tenant_rls.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	required := []string{
		"ALTER TABLE event_log ENABLE ROW LEVEL SECURITY",
		"ALTER TABLE event_log FORCE ROW LEVEL SECURITY",
		"ALTER TABLE event_log_default ENABLE ROW LEVEL SECURITY",
		"event_log_tenant_isolation",
		"event_log_default_tenant_isolation",
		"current_setting('integin.tenant_id', true)",
		"WITH CHECK",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
	if _, err := os.Stat("0003_event_log_tenant_rls.down.sql"); err != nil {
		t.Fatal(err)
	}
}

func TestAnomalyDetectionMigrationContract(t *testing.T) {
	sql, err := os.ReadFile("0048_anomaly_detection.candidate.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	required := []string{
		"CREATE TABLE anomaly_rules",
		"CREATE TABLE anomaly_alerts",
		"tenant_id",
		"CHECK (type IN ('geo', 'frequency', 'pattern'))",
		"CHECK (status IN ('firing', 'acknowledged', 'resolved'))",
		"idx_anomaly_rules_tenant",
		"idx_anomaly_alerts_dedup",
		"REFERENCES anomaly_rules(id) ON DELETE CASCADE",
		"REFERENCES short_links(code) ON DELETE CASCADE",
		"update_anomaly_rule_updated_at",
		"update_anomaly_alert_updated_at",
		"GRANT SELECT, INSERT, UPDATE, DELETE ON anomaly_rules TO integin_runtime",
		"GRANT SELECT, INSERT, UPDATE, DELETE ON anomaly_alerts TO integin_runtime",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
	if _, err := os.Stat("0048_anomaly_detection.down.candidate.sql"); err != nil {
		t.Fatal(err)
	}
}

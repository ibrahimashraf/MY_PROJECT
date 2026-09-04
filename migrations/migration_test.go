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
	sql, err := os.ReadFile("0048_anomaly_detection.sql")
	if err != nil {
		sql, err = os.ReadFile("0048_anomaly_detection.candidate.sql")
		if err != nil {
			t.Fatal(err)
		}
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
	if _, err := os.Stat("0048_anomaly_detection.down.sql"); err != nil {
		if _, err := os.Stat("0048_anomaly_detection.down.candidate.sql"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCertificateArtifactMetadataMigrationContract(t *testing.T) {
	sql, err := os.ReadFile("0015_certificate_artifact_metadata.sql")
	if err != nil {
		sql, err = os.ReadFile("0015_certificate_artifact_metadata.candidate.sql")
		if err != nil {
			t.Fatal(err)
		}
	}
	text := string(sql)
	required := []string{
		"CREATE TABLE certificate_artifact",
		"id TEXT PRIMARY KEY",
		"tenant_id TEXT NOT NULL",
		"organization_id TEXT NOT NULL",
		"certificate_id TEXT NOT NULL",
		"CHECK (artifact_type IN ('CERTIFICATE_PDF'))",
		"CHECK (content_type = 'application/pdf')",
		"CHECK (byte_size > 0)",
		"artifact_sha256 BYTEA NOT NULL",
		"snapshot_sha256 BYTEA NOT NULL",
		"renderer_version TEXT NOT NULL",
		"UNIQUE (tenant_id, organization_id, id)",
		"UNIQUE (tenant_id, organization_id, certificate_id, artifact_type)",
		"UNIQUE (tenant_id, organization_id, object_key)",
		"REFERENCES certificate_record(id)",
		"ENABLE ROW LEVEL SECURITY",
		"FORCE ROW LEVEL SECURITY",
		"certificate_artifact_tenant_isolation",
		"current_setting('integin.tenant_id', true)",
		"current_setting('integin.organization_id', true)",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
	downSQL, err := os.ReadFile("0015_certificate_artifact_metadata.down.sql")
	if err != nil {
		downSQL, err = os.ReadFile("0015_certificate_artifact_metadata.down.candidate.sql")
		if err != nil {
			t.Fatal(err)
		}
	}
	if !strings.Contains(string(downSQL), "DROP TABLE certificate_artifact") {
		t.Fatal("0015 down migration missing DROP TABLE certificate_artifact")
	}
}

func TestFieldPackageAndRiverCanonicalMigrationsContract(t *testing.T) {
	migrations := []struct {
		upFile   string
		downFile string
		required []string
	}{
		{
			upFile:   "0052_work_order_form_versioning.sql",
			downFile: "0052_work_order_form_versioning.down.sql",
			required: []string{"CREATE TABLE form_version", "CREATE TABLE form_field", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0053_asset_entitlement_offline_package.sql",
			downFile: "0053_asset_entitlement_offline_package.down.sql",
			required: []string{"CREATE TABLE asset_entitlement", "package_hash", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0054_qr_nfc_entry.sql",
			downFile: "0054_qr_nfc_entry.down.sql",
			required: []string{"CREATE TABLE qr_nfc_entry", "token_digest", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0055_assurance_projection_corrective_work.sql",
			downFile: "0055_assurance_projection_corrective_work.down.sql",
			required: []string{"CREATE TABLE assurance_projection", "CREATE TABLE corrective_work", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0056_evidence_release_pack.sql",
			downFile: "0056_evidence_release_pack.down.sql",
			required: []string{"CREATE TABLE evidence_pack", "CREATE TABLE release_pack", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0057_device_registry_composite_pk.sql",
			downFile: "0057_device_registry_composite_pk.down.sql",
			required: []string{"PRIMARY KEY (tenant_id, device_id)"},
		},
		{
			upFile:   "0058_custody_site_handover.sql",
			downFile: "0058_custody_site_handover.down.sql",
			required: []string{"CREATE TABLE custody_chain", "CREATE TABLE work_order_site_handover", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0059_river_job_queue.sql",
			downFile: "0059_river_job_queue.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS river_job", "CREATE TABLE IF NOT EXISTS river_leader"},
		},
	}

	for _, m := range migrations {
		sql, err := os.ReadFile(m.upFile)
		if err != nil {
			t.Fatalf("failed reading %s: %v", m.upFile, err)
		}
		text := string(sql)
		for _, fragment := range m.required {
			if !strings.Contains(text, fragment) {
				t.Fatalf("migration %s missing required fragment %q", m.upFile, fragment)
			}
		}
		if _, err := os.Stat(m.downFile); err != nil {
			t.Fatalf("missing matching down migration %s: %v", m.downFile, err)
		}
	}
}



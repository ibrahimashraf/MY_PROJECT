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
		if strings.HasPrefix(fragment, "CREATE TABLE ") && !strings.Contains(fragment, "IF NOT EXISTS") {
			tableName := strings.TrimPrefix(fragment, "CREATE TABLE ")
			if !strings.Contains(text, fragment) && !strings.Contains(text, "CREATE TABLE IF NOT EXISTS "+tableName) {
				t.Fatalf("migration missing %q", fragment)
			}
		} else {
			if !strings.Contains(text, fragment) {
				t.Fatalf("migration missing %q", fragment)
			}
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
	if !strings.Contains(text, "CREATE TABLE certificate_artifact") && !strings.Contains(text, "CREATE TABLE IF NOT EXISTS certificate_artifact") {
		t.Fatalf("migration missing CREATE TABLE [IF NOT EXISTS] certificate_artifact")
	}
	required := []string{
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
			required: []string{"CREATE TABLE asset_entitlement", "CREATE TABLE asset_tag", "package_hash", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
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
		{
			upFile:   "0034_license_entitlement.sql",
			downFile: "0034_license_entitlement.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS tenant_license", "CREATE TABLE IF NOT EXISTS license_audit", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0035_feature_flag_overrides.sql",
			downFile: "0035_feature_flag_overrides.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS feature_flag_override", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0019_hierarchical_register.sql",
			downFile: "0019_hierarchical_register.down.sql",
			required: []string{"CREATE TABLE location_branch", "CREATE TABLE location_area", "CREATE TABLE location_zone", "CREATE TABLE equipment_type", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0021_scheduling_calendar.sql",
			downFile: "0021_scheduling_calendar.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS schedule_calendar_entry", "CREATE TABLE IF NOT EXISTS technician_competency", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0022_multi_inspect.sql",
			downFile: "0022_multi_inspect.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS multi_inspect_batch", "CREATE TABLE IF NOT EXISTS multi_inspect_item", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0023_comments_traffic_light.sql",
			downFile: "0023_comments_traffic_light.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS comment_library", "CREATE TABLE IF NOT EXISTS inspection_comment", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0024_escalation_overdue.sql",
			downFile: "0024_escalation_overdue.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS notification_template", "CREATE TABLE IF NOT EXISTS escalation_rule", "CREATE TABLE IF NOT EXISTS escalation_event", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0026_custom_docx_templates.sql",
			downFile: "0026_custom_docx_templates.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS certificate_template_docx", "CREATE TABLE IF NOT EXISTS certificate_pack", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0027_client_portal_domains_acls.sql",
			downFile: "0027_client_portal_domains_acls.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS client_portal_domain", "CREATE TABLE IF NOT EXISTS client_portal_acl", "CREATE TABLE IF NOT EXISTS quick_link", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0030_hse_notification_csv_export.sql",
			downFile: "0030_hse_notification_csv_export.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS hse_notification", "CREATE TABLE IF NOT EXISTS csv_export_job", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0032_full_dpp_regulatory_monitor.sql",
			downFile: "0032_full_dpp_regulatory_monitor.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS product_passport_dpp", "CREATE TABLE IF NOT EXISTS regulatory_monitor", "CREATE TABLE IF NOT EXISTS compliance_action", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0020_bulk_import_export.sql",
			downFile: "0020_bulk_import_export.down.sql",
			required: []string{"CREATE TABLE bulk_import_job", "CREATE TABLE bulk_import_row", "CREATE TABLE bulk_export_job", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0025_job_linkage_failed_queue.sql",
			downFile: "0025_job_linkage_failed_queue.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS job_linkage_config", "CREATE TABLE IF NOT EXISTS failed_inspection_queue", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0028_integrations_xero_m365_api.sql",
			downFile: "0028_integrations_xero_m365_api.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS integration_config", "CREATE TABLE IF NOT EXISTS data_api_token", "CREATE TABLE IF NOT EXISTS sync_job", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0029_parts_charges_timesheet_auto.sql",
			downFile: "0029_parts_charges_timesheet_auto.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS parts_catalog", "CREATE TABLE IF NOT EXISTS service_charge", "CREATE TABLE IF NOT EXISTS timesheet_auto_capture", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0031_nfc_rfid_qr_tagging_photo_markup.sql",
			downFile: "0031_nfc_rfid_qr_tagging_photo_markup.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS photo_markup", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0033_configurable_settings_audit_export.sql",
			downFile: "0033_configurable_settings_audit_export.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS tenant_setting", "CREATE TABLE IF NOT EXISTS audit_trail_export", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0036_full_text_search.sql",
			downFile: "0036_full_text_search.down.sql",
			required: []string{"search_vector tsvector", "GIN (search_vector)"},
		},
		{
			upFile:   "0037_search_backfill.sql",
			downFile: "0037_search_backfill.down.sql",
			required: []string{"UPDATE asset_registry", "UPDATE work_order"},
		},
		{
			upFile:   "0038_immutable_audit_log.sql",
			downFile: "0038_immutable_audit_log.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS audit_log", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY", "audit_log_no_update"},
		},
		{
			upFile:   "0046_webhook_delivery.sql",
			downFile: "0046_webhook_delivery.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS webhook_deliveries", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0049_analytics_dashboard.sql",
			downFile: "0049_analytics_dashboard.down.sql",
			required: []string{"idx_scan_events_tenant_code_timestamp", "idx_scan_events_tenant_geo"},
		},
		{
			upFile:   "0050_multi_tenant_scale_optimizations.sql",
			downFile: "0050_multi_tenant_scale_optimizations.down.sql",
			required: []string{"btree_gin", "asset_registry_tenant_search_idx", "asset_registry_tenant_created_idx"},
		},
		{
			upFile:   "0051_runtime_scale_safeties.sql",
			downFile: "0051_runtime_scale_safeties.down.sql",
			required: []string{"statement_timeout", "idle_in_transaction_session_timeout", "deadlock_timeout"},
		},
		{
			upFile:   "0039_short_links.sql",
			downFile: "0039_short_links.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS short_links", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0040_short_link_scan_events.sql",
			downFile: "0040_short_link_scan_events.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS short_link_scan_events", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0041_short_link_webhook.sql",
			downFile: "0041_short_link_webhook.down.sql",
			required: []string{"webhook_url TEXT"},
		},
		{
			upFile:   "0042_short_link_custom_domain.sql",
			downFile: "0042_short_link_custom_domain.down.sql",
			required: []string{"custom_domain TEXT", "idx_short_links_custom_domain"},
		},
		{
			upFile:   "0045_short_link_bulk_jobs.sql",
			downFile: "0045_short_link_bulk_jobs.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS short_link_bulk_jobs", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0047_short_link_hmac.sql",
			downFile: "0047_short_link_hmac.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS short_link_hmac_secrets", "hmac_secret_ref TEXT", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0071_sync_idempotency_cache.sql",
			downFile: "0071_sync_idempotency_cache.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS sync_idempotency_cache", "PRIMARY KEY (tenant_id, organization_id, key_hash)", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY", "sync_idempotency_cache_evict_expired", "expires_at > created_at"},
		},
		{
			upFile:   "0072_river_poison_quarantine.sql",
			downFile: "0072_river_poison_quarantine.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS river_poison_quarantine", "job_id         BIGINT      PRIMARY KEY", "river_poison_quarantine_kind_idx", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY"},
		},
		{
			upFile:   "0073_tool_calibration_registry.sql",
			downFile: "0073_tool_calibration_registry.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS tool_calibration_registry", "PRIMARY KEY (tenant_id, organization_id, id)", "tool_calibration_registry_equipment_due_idx", "tool_calibration_registry_status_idx", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY", "tool_calibration_registry_tenant_isolation", "NULLIF(current_setting('integin.tenant_id', true), '')"},
		},
		{
			upFile:   "0074_partitioning_and_cqrs.sql",
			downFile: "0074_partitioning_and_cqrs.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS sensor_telemetry_stream", "PARTITION BY RANGE (reading_timestamp)", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY", "CREATE TABLE IF NOT EXISTS cqrs_replication_outbox", "fillfactor = 85"},
		},
		{
			upFile:   "0075_device_enrollment_requests.sql",
			downFile: "0075_device_enrollment_requests.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS device_enrollment_requests", "PRIMARY KEY (tenant_id, organization_id, request_id)", "device_enrollment_requests_device_idx", "device_enrollment_requests_status_idx", "CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED'))", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY", "device_enrollment_requests_tenant_isolation", "NULLIF(current_setting('integin.tenant_id', true), '')", "GRANT SELECT, INSERT, UPDATE ON device_enrollment_requests TO integin_runtime"},
		},
		{
			upFile:   "0076_oidc_session_store.sql",
			downFile: "0076_oidc_session_store.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS oidc_session_store", "PRIMARY KEY (session_id)", "subject        TEXT        NOT NULL", "revoked_at     TIMESTAMPTZ", "oidc_session_store_expiry_idx", "expires_at > issued_at", "GRANT SELECT, INSERT, UPDATE, DELETE ON oidc_session_store TO integin_runtime"},
		},
		{
			upFile:   "0081_audit_log_valid_time.sql",
			downFile: "0081_audit_log_valid_time.down.sql",
			required: []string{"ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS valid_time TIMESTAMPTZ", "audit_log_valid_time_idx"},
		},
		{
			upFile:   "0077_retention_and_legal_hold.sql",
			downFile: "0077_retention_and_legal_hold.down.sql",
			required: []string{"CREATE TABLE IF NOT EXISTS tenant_retention_policy", "retention_days", "NOT NULL DEFAULT 365", "CHECK (action IN ('ARCHIVE', 'PURGE'))", "PRIMARY KEY (tenant_id, entity_type)", "CREATE TABLE IF NOT EXISTS legal_hold_registry", "CHECK (status IN ('ACTIVE', 'RELEASED'))", "UNIQUE (tenant_id, entity_type, entity_id, id)", "CREATE TABLE IF NOT EXISTS export_approval_registry", "CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED'))", "CREATE TABLE IF NOT EXISTS deletion_evidence_receipt", "tombstone_hash", "ENABLE ROW LEVEL SECURITY", "FORCE ROW LEVEL SECURITY", "NULLIF(current_setting('integin.tenant_id', true), '')", "GRANT SELECT, INSERT ON deletion_evidence_receipt TO integin_runtime", "deletion_evidence_receipt_immutable", "BEFORE UPDATE OR DELETE ON deletion_evidence_receipt"},
		},
	}

	for _, m := range migrations {
		sql, err := os.ReadFile(m.upFile)
		if err != nil {
			t.Fatalf("failed reading %s: %v", m.upFile, err)
		}
		text := string(sql)
		for _, fragment := range m.required {
			if strings.HasPrefix(fragment, "CREATE TABLE ") && !strings.Contains(fragment, "IF NOT EXISTS") {
				tableName := strings.TrimPrefix(fragment, "CREATE TABLE ")
				if !strings.Contains(text, fragment) && !strings.Contains(text, "CREATE TABLE IF NOT EXISTS "+tableName) {
					t.Fatalf("migration %s missing required fragment %q", m.upFile, fragment)
				}
			} else {
				if !strings.Contains(text, fragment) {
					t.Fatalf("migration %s missing required fragment %q", m.upFile, fragment)
				}
			}
		}
		if _, err := os.Stat(m.downFile); err != nil {
			t.Fatalf("missing matching down migration %s: %v", m.downFile, err)
		}
	}
}

func TestRetentionAndLegalHoldMigrationContract(t *testing.T) {
	sql, err := os.ReadFile("0077_retention_and_legal_hold.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	tables := []string{
		"CREATE TABLE IF NOT EXISTS tenant_retention_policy",
		"CREATE TABLE IF NOT EXISTS legal_hold_registry",
		"CREATE TABLE IF NOT EXISTS export_approval_registry",
		"CREATE TABLE IF NOT EXISTS deletion_evidence_receipt",
	}
	for _, table := range tables {
		if !strings.Contains(text, table) {
			t.Fatalf("migration missing %q", table)
		}
	}
	required := []string{
		"PRIMARY KEY (tenant_id, entity_type)",
		"UNIQUE (tenant_id, entity_type, entity_id, id)",
		"CHECK (action IN ('ARCHIVE', 'PURGE'))",
		"CHECK (status IN ('ACTIVE', 'RELEASED'))",
		"CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED'))",
		"ENABLE ROW LEVEL SECURITY",
		"FORCE ROW LEVEL SECURITY",
		"NULLIF(current_setting('integin.tenant_id', true), '')",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
	for _, table := range []string{"tenant_retention_policy", "legal_hold_registry", "export_approval_registry", "deletion_evidence_receipt"} {
		if !strings.Contains(text, table+"_tenant_isolation") {
			t.Fatalf("migration missing RLS policy for %s", table)
		}
	}

	down, err := os.ReadFile("0077_retention_and_legal_hold.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	downText := string(down)
	for _, table := range []string{"tenant_retention_policy", "legal_hold_registry", "export_approval_registry", "deletion_evidence_receipt"} {
		if !strings.Contains(downText, "DROP TABLE IF EXISTS "+table) {
			t.Fatalf("down migration missing DROP TABLE IF EXISTS %s", table)
		}
	}
	if !strings.Contains(downText, "DROP FUNCTION IF EXISTS deletion_evidence_receipt_immutable()") {
		t.Fatal("down migration missing immutable trigger function drop")
	}
}

func TestAdvisoryGovernanceRegisterMigrationContract(t *testing.T) {
	sql, err := os.ReadFile("0078_advisory_governance_register.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	tables := []string{
		"CREATE TABLE IF NOT EXISTS advisory_model_registry",
		"CREATE TABLE IF NOT EXISTS advisory_prompt_registry",
		"CREATE TABLE IF NOT EXISTS advisory_audit_trail",
		"CREATE TABLE IF NOT EXISTS advisory_feedback",
	}
	for _, table := range tables {
		if !strings.Contains(text, table) {
			t.Fatalf("migration missing %q", table)
		}
	}
	required := []string{
		"CHECK (status IN ('APPROVED', 'DEPRECATED'))",
		"allowed_zones TEXT[]      NOT NULL",
		"NOT NULL DEFAULT 4096",
		"CHECK (status IN ('ACTIVE', 'SUPERSEDED'))",
		"system_prompt_hash TEXT        NOT NULL",
		"input_schema_hash  TEXT        NOT NULL",
		"evidence_refs  TEXT[]           NOT NULL DEFAULT '{}'",
		"confidence     DOUBLE PRECISION NOT NULL",
		"CHECK (blocking = FALSE)",
		"CHECK (disposition IN ('ACCEPTED', 'REJECTED', 'IGNORED', 'CORRECTED'))",
		"ENABLE ROW LEVEL SECURITY",
		"FORCE ROW LEVEL SECURITY",
		"NULLIF(current_setting('integin.tenant_id', true), '')",
		"GRANT SELECT ON advisory_model_registry TO integin_runtime",
		"GRANT SELECT ON advisory_prompt_registry TO integin_runtime",
		"GRANT SELECT, INSERT ON advisory_audit_trail TO integin_runtime",
		"GRANT SELECT, INSERT ON advisory_feedback TO integin_runtime",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
	for _, table := range []string{"advisory_audit_trail", "advisory_feedback"} {
		if !strings.Contains(text, table+"_tenant_isolation") {
			t.Fatalf("migration missing RLS policy for %s", table)
		}
	}

	down, err := os.ReadFile("0078_advisory_governance_register.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	downText := string(down)
	for _, table := range []string{"advisory_feedback", "advisory_audit_trail", "advisory_prompt_registry", "advisory_model_registry"} {
		if !strings.Contains(downText, "DROP TABLE IF EXISTS "+table) {
			t.Fatalf("down migration missing DROP TABLE IF EXISTS %s", table)
		}
	}
	if !strings.Contains(downText, "DISABLE ROW LEVEL SECURITY") {
		t.Fatal("down migration missing RLS disable")
	}
}

func TestSignedAuditCheckpointsMigrationContract(t *testing.T) {
	sql, err := os.ReadFile("0079_signed_audit_checkpoints.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	required := []string{
		"CREATE TABLE IF NOT EXISTS audit_checkpoint_registry",
		"id                   TEXT        PRIMARY KEY",
		"sequence_start       BIGINT      NOT NULL",
		"sequence_end         BIGINT      NOT NULL",
		"previous_root_sha256 TEXT        NOT NULL",
		"root_sha256          TEXT        NOT NULL",
		"signature            TEXT        NOT NULL",
		"key_id               TEXT        NOT NULL",
		"object_key           TEXT        NOT NULL",
		"UNIQUE (tenant_id, id)",
		"UNIQUE (tenant_id, root_sha256)",
		"CHECK (sequence_end >= sequence_start)",
		"btrim(tenant_id) <> ''",
		"btrim(environment) <> ''",
		"btrim(root_sha256) <> ''",
		"ENABLE ROW LEVEL SECURITY",
		"FORCE ROW LEVEL SECURITY",
		"audit_checkpoint_registry_tenant_isolation",
		"NULLIF(current_setting('integin.tenant_id', true), '')",
		"audit_checkpoint_no_update",
		"BEFORE UPDATE OR DELETE ON audit_checkpoint_registry",
		"GRANT SELECT, INSERT ON TABLE audit_checkpoint_registry TO integin_runtime",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
	if !strings.Contains(text, "CREATE POLICY ") {
		t.Fatalf("migration missing CREATE POLICY")
	}

	down, err := os.ReadFile("0079_signed_audit_checkpoints.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	downText := string(down)
	for _, fragment := range []string{
		"DISABLE ROW LEVEL SECURITY",
		"DROP TRIGGER IF EXISTS audit_checkpoint_immutable ON audit_checkpoint_registry",
		"DROP FUNCTION IF EXISTS audit_checkpoint_no_update()",
		"DROP POLICY IF EXISTS audit_checkpoint_registry_tenant_isolation ON audit_checkpoint_registry",
		"DROP TABLE IF EXISTS audit_checkpoint_registry",
	} {
		if !strings.Contains(downText, fragment) {
			t.Fatalf("down migration missing %q", fragment)
		}
	}
}

func TestTenantGUCRenameMigrationContract(t *testing.T) {
	sql, err := os.ReadFile("0085_rename_tenant_gucs_to_integin.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	for _, fragment := range []string{
		"pg_policy",
		"integin.tenant_id",
		"integin.organization_id",
		"integin.tenant_id",
		"integin.organization_id",
		"DROP POLICY",
		"CREATE POLICY",
		"RAISE EXCEPTION",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
	down, err := os.ReadFile("0085_rename_tenant_gucs_to_integin.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"pg_policy", "DROP POLICY", "CREATE POLICY"} {
		if !strings.Contains(string(down), fragment) {
			t.Fatalf("down migration missing %q", fragment)
		}
	}
}

-- INTEGIN 150+ configurable settings per tenant + complete audit trail export.
-- Core: "150+ configurable settings per tenant", "Complete audit trail exportable"
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.
BEGIN;

CREATE TABLE IF NOT EXISTS tenant_setting (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    setting_key TEXT NOT NULL CHECK (char_length(setting_key) BETWEEN 1 AND 128),
    setting_value JSONB NOT NULL DEFAULT '{}'::jsonb,
    scope TEXT NOT NULL CHECK (scope IN ('GLOBAL','ORGANIZATION','BRANCH')) DEFAULT 'ORGANIZATION',
    is_editable BOOLEAN NOT NULL DEFAULT TRUE,
    description TEXT,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, setting_key, scope)
);

CREATE TABLE IF NOT EXISTS audit_trail_export (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    export_type TEXT NOT NULL CHECK (export_type IN ('FULL','INCREMENTAL','DATE_RANGE')),
    date_from TIMESTAMPTZ,
    date_to TIMESTAMPTZ,
    filters JSONB DEFAULT '{}'::jsonb,
    status TEXT NOT NULL CHECK (status IN ('PENDING','GENERATING','READY','FAILED','EXPIRED')) DEFAULT 'PENDING',
    file_key TEXT,
    row_count BIGINT,
    checksum_sha256 BYTEA CHECK (checksum_sha256 IS NULL OR octet_length(checksum_sha256) = 32),
    requested_by TEXT NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS tenant_setting_key_scope_idx ON tenant_setting (tenant_id, organization_id, setting_key, scope);
CREATE INDEX IF NOT EXISTS audit_trail_export_status_requested_idx ON audit_trail_export (tenant_id, organization_id, status, requested_at DESC);

ALTER TABLE tenant_setting ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_setting FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_trail_export ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_trail_export FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_setting_tenant_isolation ON tenant_setting USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
CREATE POLICY audit_trail_export_tenant_isolation ON audit_trail_export USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

-- Seed 150+ default settings (representative subset)
INSERT INTO tenant_setting (id, tenant_id, organization_id, setting_key, setting_value, scope, is_editable, description, created_by, created_at) VALUES
('setting-001', 'default', 'default', 'email_template_certificate_issued', '{"subject": "Certificate {{cert_number}} issued", "body": "Your certificate is ready."}', 'GLOBAL', TRUE, 'Certificate issued email template', 'system', now()),
('setting-002', 'default', 'default', 'email_template_inspection_due', '{"subject": "Inspection due for {{asset_id}}", "body": "Inspection due in {{days}} days."}', 'GLOBAL', TRUE, 'Inspection due reminder template', 'system', now()),
('setting-003', 'default', 'default', 'dispatch_field_labels', '{"field1": "Site Contact", "field2": "Access Instructions", "field3": "Safety Requirements", "field4": "Special Equipment"}', 'GLOBAL', TRUE, 'Dispatch template field labels', 'system', now()),
('setting-004', 'default', 'default', 'approval_threshold_days', '{"pre_due": 7, "due": 0, "overdue": 3}', 'GLOBAL', TRUE, 'Escalation thresholds in days', 'system', now()),
('setting-005', 'default', 'default', 'label_terminology', '{"work_order": "Job", "inspection": "Inspection", "certificate": "Certificate", "technician": "Inspector"}', 'GLOBAL', TRUE, 'Custom label terminology', 'system', now())
ON CONFLICT DO NOTHING;

COMMIT;
-- Migration 0030: HSE Notification + CSV Export
-- Compatible with existing schema (TEXT IDs, integin.* settings, FORCE RLS)
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.

BEGIN;

-- hse_notification table
CREATE TABLE IF NOT EXISTS hse_notification (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    inspection_id TEXT NOT NULL,
    defect_code TEXT NOT NULL CHECK (char_length(defect_code) BETWEEN 1 AND 100),
    defect_severity TEXT NOT NULL CHECK (defect_severity IN ('IMMEDIATE','MAJOR','MINOR')),
    hse_reference TEXT,
    report_payload JSONB,
    status TEXT NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','SUBMITTED','ACKNOWLEDGED','REJECTED')),
    submitted_by TEXT,
    submitted_at TIMESTAMPTZ,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by TEXT,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- csv_export_job table
CREATE TABLE IF NOT EXISTS csv_export_job (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    export_type TEXT NOT NULL CHECK (export_type IN ('EQUIPMENT','INSPECTION','CERTIFICATE','AUDIT','ALL')),
    filters JSONB,
    status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','GENERATING','READY','FAILED','EXPIRED')),
    file_key TEXT,
    row_count BIGINT,
    expires_at TIMESTAMPTZ,
    requested_by TEXT NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

-- export_template table
CREATE TABLE IF NOT EXISTS export_template (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    export_type TEXT NOT NULL CHECK (export_type IN ('EQUIPMENT','INSPECTION','CERTIFICATE','AUDIT','ALL')),
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    columns JSONB NOT NULL,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_hse_notification_inspection ON hse_notification (tenant_id, organization_id, inspection_id);
CREATE INDEX IF NOT EXISTS idx_hse_notification_status ON hse_notification (tenant_id, organization_id, status);
CREATE INDEX IF NOT EXISTS idx_csv_export_job_status_requested ON csv_export_job (tenant_id, organization_id, status, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_export_template_type ON export_template (tenant_id, organization_id, export_type);

-- RLS
ALTER TABLE hse_notification ENABLE ROW LEVEL SECURITY;
ALTER TABLE hse_notification FORCE ROW LEVEL SECURITY;
ALTER TABLE csv_export_job ENABLE ROW LEVEL SECURITY;
ALTER TABLE csv_export_job FORCE ROW LEVEL SECURITY;
ALTER TABLE export_template ENABLE ROW LEVEL SECURITY;
ALTER TABLE export_template FORCE ROW LEVEL SECURITY;

CREATE POLICY hse_notification_tenant_isolation ON hse_notification
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE POLICY csv_export_job_tenant_isolation ON csv_export_job
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE POLICY export_template_tenant_isolation ON export_template
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;
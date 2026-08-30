-- 0030_hse_notification_csv_export.candidate.sql
-- HSE Notification + CSV Export tables

-- hse_notification table
CREATE TABLE hse_notification (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    inspection_id UUID NOT NULL,
    defect_code VARCHAR(100) NOT NULL,
    defect_severity VARCHAR(20) NOT NULL CHECK (defect_severity IN ('IMMEDIATE', 'MAJOR', 'MINOR')),
    hse_reference VARCHAR(100),
    report_payload JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'SUBMITTED', 'ACKNOWLEDGED', 'REJECTED')),
    submitted_by UUID,
    submitted_at TIMESTAMPTZ,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by UUID,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Enable RLS
ALTER TABLE hse_notification ENABLE ROW LEVEL SECURITY;

-- Index on inspection_id
CREATE INDEX idx_hse_notification_inspection_id ON hse_notification(inspection_id);

-- csv_export_job table
CREATE TABLE csv_export_job (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    export_type VARCHAR(20) NOT NULL CHECK (export_type IN ('EQUIPMENT', 'INSPECTION', 'CERTIFICATE', 'AUDIT', 'ALL')),
    filters JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'GENERATING', 'READY', 'FAILED', 'EXPIRED')),
    file_key VARCHAR(500),
    row_count INTEGER,
    expires_at TIMESTAMPTZ,
    requested_by UUID NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Enable RLS
ALTER TABLE csv_export_job ENABLE ROW LEVEL SECURITY;

-- Index on status + requested_by
CREATE INDEX idx_csv_export_job_status_requested_by ON csv_export_job(status, requested_by);

-- export_template table
CREATE TABLE export_template (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    export_type VARCHAR(20) NOT NULL CHECK (export_type IN ('EQUIPMENT', 'INSPECTION', 'CERTIFICATE', 'AUDIT', 'ALL')),
    name VARCHAR(200) NOT NULL,
    columns JSONB NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Enable RLS
ALTER TABLE export_template ENABLE ROW LEVEL SECURITY;

-- Index on export_type for faster lookups
CREATE INDEX idx_export_template_export_type ON export_template(export_type);
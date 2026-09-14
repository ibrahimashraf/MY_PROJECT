-- Migration 0083: Add report_config and generated_report tables with strict tenant RLS
BEGIN;

CREATE TABLE IF NOT EXISTS report_config (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    format TEXT NOT NULL,
    schedule JSONB,
    filters JSONB,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_report_config_tenant_org ON report_config (tenant_id, organization_id, created_at DESC);

CREATE TABLE IF NOT EXISTS generated_report (
    id TEXT PRIMARY KEY,
    config_id TEXT NOT NULL REFERENCES report_config(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    status TEXT NOT NULL,
    file_path TEXT NOT NULL,
    row_count BIGINT NOT NULL DEFAULT 0,
    error_msg TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_generated_report_tenant_org ON generated_report (tenant_id, organization_id, created_at DESC);

ALTER TABLE report_config ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_config FORCE ROW LEVEL SECURITY;

ALTER TABLE generated_report ENABLE ROW LEVEL SECURITY;
ALTER TABLE generated_report FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS report_config_tenant_isolation ON report_config;
CREATE POLICY report_config_tenant_isolation ON report_config
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );

DROP POLICY IF EXISTS generated_report_tenant_isolation ON generated_report;
CREATE POLICY generated_report_tenant_isolation ON generated_report
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );

COMMIT;

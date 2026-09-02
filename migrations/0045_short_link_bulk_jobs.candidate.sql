-- Migration 0045: Short Link Bulk Jobs
-- TEXT IDs, combined RLS, FORCE RLS

BEGIN;

CREATE TABLE IF NOT EXISTS short_link_bulk_jobs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    job_type TEXT NOT NULL CHECK (job_type IN ('bulk_create', 'bulk_import', 'export')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed')) DEFAULT 'pending',
    total_items INT NOT NULL DEFAULT 0,
    completed_items INT NOT NULL DEFAULT 0,
    failed_items INT NOT NULL DEFAULT 0,
    created_by TEXT,
    error_message TEXT,
    result_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_bulk_jobs_tenant_org ON short_link_bulk_jobs (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_bulk_jobs_status ON short_link_bulk_jobs (tenant_id, organization_id, status);
CREATE INDEX IF NOT EXISTS idx_bulk_jobs_created_at ON short_link_bulk_jobs (tenant_id, organization_id, created_at DESC);

ALTER TABLE short_link_bulk_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE short_link_bulk_jobs FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS short_link_bulk_jobs_isolation ON short_link_bulk_jobs;
CREATE POLICY short_link_bulk_jobs_isolation ON short_link_bulk_jobs
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

GRANT SELECT, INSERT, UPDATE, DELETE ON short_link_bulk_jobs TO integin_runtime;

COMMIT;

-- integin bulk import/export staging.
-- CANDIDATE ONLY: do not apply without a verified pre-apply backup and isolated up/down review.
BEGIN;

CREATE TABLE IF NOT EXISTS bulk_import_job (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'VALIDATING', 'IMPORTED', 'FAILED')),
    source_type TEXT NOT NULL CHECK (source_type IN ('CSV', 'XLSX')),
    target TEXT NOT NULL CHECK (target IN ('WORK_ORDER', 'ASSET', 'INSPECTION')),
    total_rows BIGINT NOT NULL CHECK (total_rows >= 0),
    imported_rows BIGINT NOT NULL DEFAULT 0 CHECK (imported_rows >= 0),
    error_log JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS bulk_import_row (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES bulk_import_job(id) ON DELETE CASCADE,
    row_num BIGINT NOT NULL CHECK (row_num > 0),
    raw_data JSONB NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'VALID', 'INVALID', 'IMPORTED', 'FAILED')),
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (job_id, row_num)
);

CREATE TABLE IF NOT EXISTS bulk_export_job (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    target TEXT NOT NULL CHECK (target IN ('WORK_ORDER', 'ASSET', 'INSPECTION', 'CERTIFICATE')),
    filters JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'GENERATING', 'COMPLETED', 'FAILED')),
    file_key TEXT,
    row_count BIGINT NOT NULL DEFAULT 0 CHECK (row_count >= 0),
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX bulk_import_job_scope_status_idx ON bulk_import_job (tenant_id, organization_id, status, created_at DESC);
CREATE INDEX bulk_import_row_job_idx ON bulk_import_row (job_id, row_num);
CREATE INDEX bulk_export_job_scope_status_idx ON bulk_export_job (tenant_id, organization_id, status, created_at DESC);

ALTER TABLE bulk_import_job ENABLE ROW LEVEL SECURITY;
ALTER TABLE bulk_import_job FORCE ROW LEVEL SECURITY;
ALTER TABLE bulk_import_row ENABLE ROW LEVEL SECURITY;
ALTER TABLE bulk_import_row FORCE ROW LEVEL SECURITY;
ALTER TABLE bulk_export_job ENABLE ROW LEVEL SECURITY;
ALTER TABLE bulk_export_job FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS bulk_import_job_tenant_isolation ON bulk_import_job;
CREATE POLICY bulk_import_job_tenant_isolation ON bulk_import_job USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
DROP POLICY IF EXISTS bulk_import_row_tenant_isolation ON bulk_import_row;
CREATE POLICY bulk_import_row_tenant_isolation ON bulk_import_row USING (job_id IN (SELECT id FROM bulk_import_job WHERE tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))) WITH CHECK (job_id IN (SELECT id FROM bulk_import_job WHERE tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)));
DROP POLICY IF EXISTS bulk_export_job_tenant_isolation ON bulk_export_job;
CREATE POLICY bulk_export_job_tenant_isolation ON bulk_export_job USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;
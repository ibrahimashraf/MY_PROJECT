-- Migration 0025: Job Linkage Config + Failed Inspection Queue
-- Compatible with existing schema (TEXT IDs, integin.* settings, FORCE RLS)
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.

BEGIN;

-- job_linkage_config table
CREATE TABLE IF NOT EXISTS job_linkage_config (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    require_job_for_inspection BOOLEAN NOT NULL DEFAULT TRUE,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id)
);

-- failed_inspection_queue table
CREATE TABLE IF NOT EXISTS failed_inspection_queue (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    inspection_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    assignment_id TEXT NOT NULL,
    failure_reason TEXT NOT NULL,
    suppression_reason TEXT NOT NULL CHECK (suppression_reason IN ('AUTO_SUPPRESSED','MANUAL_REVIEW')),
    review_status TEXT NOT NULL DEFAULT 'PENDING' CHECK (review_status IN ('PENDING','REVIEWED','RELEASED','REJECTED')),
    reviewed_by TEXT,
    reviewed_at TIMESTAMPTZ,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index
CREATE INDEX IF NOT EXISTS idx_failed_inspection_queue_review_status ON failed_inspection_queue (tenant_id, organization_id, review_status) WHERE review_status IN ('PENDING','REVIEWED','RELEASED','REJECTED');

-- RLS
ALTER TABLE job_linkage_config ENABLE ROW LEVEL SECURITY;
ALTER TABLE job_linkage_config FORCE ROW LEVEL SECURITY;
ALTER TABLE failed_inspection_queue ENABLE ROW LEVEL SECURITY;
ALTER TABLE failed_inspection_queue FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS job_linkage_config_tenant_isolation ON job_linkage_config;
CREATE POLICY job_linkage_config_tenant_isolation ON job_linkage_config
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

DROP POLICY IF EXISTS failed_inspection_queue_tenant_isolation ON failed_inspection_queue;
CREATE POLICY failed_inspection_queue_tenant_isolation ON failed_inspection_queue
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;
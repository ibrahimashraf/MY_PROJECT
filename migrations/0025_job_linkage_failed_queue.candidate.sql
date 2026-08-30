-- 0025_job_linkage_failed_queue.candidate.sql
-- Mandatory job linkage + failed suppression queue

CREATE TABLE job_linkage_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    org_id UUID NOT NULL REFERENCES organizations(id),
    require_job_for_inspection BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, org_id)
);

ALTER TABLE job_linkage_config ENABLE ROW LEVEL SECURITY;

CREATE POLICY job_linkage_config_tenant_isolation ON job_linkage_config
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE TABLE failed_inspection_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    org_id UUID NOT NULL REFERENCES organizations(id),
    inspection_id UUID NOT NULL REFERENCES inspections(id),
    work_order_id UUID NOT NULL REFERENCES work_orders(id),
    assignment_id UUID NOT NULL REFERENCES assignments(id),
    failure_reason TEXT NOT NULL,
    suppression_reason VARCHAR(32) NOT NULL CHECK (suppression_reason IN ('AUTO_SUPPRESSED', 'MANUAL_REVIEW')),
    review_status VARCHAR(16) NOT NULL DEFAULT 'PENDING' CHECK (review_status IN ('PENDING', 'REVIEWED', 'RELEASED', 'REJECTED')),
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE failed_inspection_queue ENABLE ROW LEVEL SECURITY;

CREATE POLICY failed_inspection_queue_tenant_isolation ON failed_inspection_queue
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE INDEX idx_failed_inspection_queue_review_status
    ON failed_inspection_queue (review_status)
    WHERE review_status IN ('PENDING', 'REVIEWED', 'RELEASED', 'REJECTED');
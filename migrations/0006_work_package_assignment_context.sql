-- INTEGIN assignment context: immutable inspection-specific attributes required
-- to render an approved work package in INTEGIN Field. This migration is
-- additive and source-only until separately reviewed for isolated pilot use.
CREATE TABLE IF NOT EXISTS work_package_assignment_context (
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    inspection_id TEXT NOT NULL,
    root_asset_id TEXT NOT NULL,
    inspection_type TEXT NOT NULL,
    procedure_version TEXT NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    field_asset_ids JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, inspection_id),
    FOREIGN KEY (tenant_id, organization_id, inspection_id)
        REFERENCES work_package_assignment (tenant_id, organization_id, inspection_id)
        ON DELETE RESTRICT
);

ALTER TABLE work_package_assignment_context ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS work_package_assignment_context_tenant_isolation ON work_package_assignment_context;
CREATE POLICY work_package_assignment_context_tenant_isolation ON work_package_assignment_context
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

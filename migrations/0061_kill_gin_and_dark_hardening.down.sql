-- INTEGIN Hardening: Rollback for 0061_kill_gin_and_dark_hardening.sql
BEGIN;

-- Restore GIN index if needed
CREATE INDEX IF NOT EXISTS river_job_args_index ON river_job USING GIN (args);

-- Restore original RLS policies
DROP POLICY IF EXISTS work_order_tenant_organization_isolation ON work_order;
CREATE POLICY work_order_tenant_organization_isolation ON work_order
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

DROP POLICY IF EXISTS work_order_scope_item_tenant_organization_isolation ON work_order_scope_item;
CREATE POLICY work_order_scope_item_tenant_organization_isolation ON work_order_scope_item
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

DROP POLICY IF EXISTS work_order_assignment_tenant_organization_isolation ON work_order_assignment;
CREATE POLICY work_order_assignment_tenant_organization_isolation ON work_order_assignment
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

COMMIT;

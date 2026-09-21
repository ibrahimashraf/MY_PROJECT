-- integin Hardening: Kill GIN Write Bottleneck & Harden RLS Against Empty String Poisoning
-- Builds on 0060_river_scale_10k.sql.

BEGIN;

-- 1. KILL THE GIN WRITE BOTTLENECK
-- GIN indexes take exclusive page locks during high-throughput inserts.
-- River worker pollers never query inside arbitrary JSON args. Dropping this eliminates LWLock serialization.
DROP INDEX IF EXISTS river_job_args_index;

-- 2. HARDEN RLS POLICIES ON CORE WORK ORDER TABLES AGAINST EMPTY-STRING/POISONED SESSION REUSE
-- Explicitly require non-null and non-empty tenant context so dirty pooler connections cannot leak rows.

DROP POLICY IF EXISTS work_order_tenant_organization_isolation ON work_order;
CREATE POLICY work_order_tenant_organization_isolation ON work_order
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

DROP POLICY IF EXISTS work_order_scope_item_tenant_organization_isolation ON work_order_scope_item;
CREATE POLICY work_order_scope_item_tenant_organization_isolation ON work_order_scope_item
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

DROP POLICY IF EXISTS work_order_assignment_tenant_organization_isolation ON work_order_assignment;
CREATE POLICY work_order_assignment_tenant_organization_isolation ON work_order_assignment
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

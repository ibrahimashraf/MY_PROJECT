-- INTEGIN Ticket 02-03: tenant/organization RLS and least-privilege runtime access.
-- The runtime role must be provisioned separately as a non-owner, non-superuser
-- role without BYPASSRLS. This migration does not create or alter roles.

GRANT USAGE ON SCHEMA public TO integin_runtime;

REVOKE ALL ON TABLE work_order, work_order_scope_item, work_order_assignment,
    work_order_assignment_scope, inspection_record, work_order_submission_segment,
    work_order_submission_item, work_order_operation, work_order_state_event,
    work_order_provisional_record FROM PUBLIC, integin_runtime;

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE work_order, work_order_scope_item,
    work_order_assignment, work_order_assignment_scope, inspection_record,
    work_order_submission_segment, work_order_submission_item, work_order_operation,
    work_order_state_event, work_order_provisional_record TO integin_runtime;

ALTER TABLE work_order ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order FORCE ROW LEVEL SECURITY;
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

ALTER TABLE work_order_scope_item ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_scope_item FORCE ROW LEVEL SECURITY;
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

ALTER TABLE work_order_assignment ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_assignment FORCE ROW LEVEL SECURITY;
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

ALTER TABLE work_order_assignment_scope ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_assignment_scope FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS work_order_assignment_scope_tenant_organization_isolation ON work_order_assignment_scope;
CREATE POLICY work_order_assignment_scope_tenant_organization_isolation ON work_order_assignment_scope
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

ALTER TABLE inspection_record ENABLE ROW LEVEL SECURITY;
ALTER TABLE inspection_record FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS inspection_record_tenant_organization_isolation ON inspection_record;
CREATE POLICY inspection_record_tenant_organization_isolation ON inspection_record
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

ALTER TABLE work_order_submission_segment ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_submission_segment FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS work_order_submission_segment_tenant_organization_isolation ON work_order_submission_segment;
CREATE POLICY work_order_submission_segment_tenant_organization_isolation ON work_order_submission_segment
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

ALTER TABLE work_order_submission_item ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_submission_item FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS work_order_submission_item_tenant_organization_isolation ON work_order_submission_item;
CREATE POLICY work_order_submission_item_tenant_organization_isolation ON work_order_submission_item
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

ALTER TABLE work_order_operation ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_operation FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS work_order_operation_tenant_organization_isolation ON work_order_operation;
CREATE POLICY work_order_operation_tenant_organization_isolation ON work_order_operation
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

ALTER TABLE work_order_state_event ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_state_event FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS work_order_state_event_tenant_organization_isolation ON work_order_state_event;
CREATE POLICY work_order_state_event_tenant_organization_isolation ON work_order_state_event
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

ALTER TABLE work_order_provisional_record ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_provisional_record FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS work_order_provisional_record_tenant_organization_isolation ON work_order_provisional_record;
CREATE POLICY work_order_provisional_record_tenant_organization_isolation ON work_order_provisional_record
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

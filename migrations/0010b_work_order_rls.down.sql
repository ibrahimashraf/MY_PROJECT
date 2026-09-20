DROP POLICY IF EXISTS work_order_tenant_organization_isolation ON work_order;
DROP POLICY IF EXISTS work_order_scope_item_tenant_organization_isolation ON work_order_scope_item;
DROP POLICY IF EXISTS work_order_assignment_tenant_organization_isolation ON work_order_assignment;
DROP POLICY IF EXISTS work_order_assignment_scope_tenant_organization_isolation ON work_order_assignment_scope;
DROP POLICY IF EXISTS inspection_record_tenant_organization_isolation ON inspection_record;
DROP POLICY IF EXISTS work_order_submission_segment_tenant_organization_isolation ON work_order_submission_segment;
DROP POLICY IF EXISTS work_order_submission_item_tenant_organization_isolation ON work_order_submission_item;
DROP POLICY IF EXISTS work_order_operation_tenant_organization_isolation ON work_order_operation;
DROP POLICY IF EXISTS work_order_state_event_tenant_organization_isolation ON work_order_state_event;
DROP POLICY IF EXISTS work_order_provisional_record_tenant_organization_isolation ON work_order_provisional_record;

ALTER TABLE work_order NO FORCE ROW LEVEL SECURITY;
ALTER TABLE work_order_scope_item NO FORCE ROW LEVEL SECURITY;
ALTER TABLE work_order_assignment NO FORCE ROW LEVEL SECURITY;
ALTER TABLE work_order_assignment_scope NO FORCE ROW LEVEL SECURITY;
ALTER TABLE inspection_record NO FORCE ROW LEVEL SECURITY;
ALTER TABLE work_order_submission_segment NO FORCE ROW LEVEL SECURITY;
ALTER TABLE work_order_submission_item NO FORCE ROW LEVEL SECURITY;
ALTER TABLE work_order_operation NO FORCE ROW LEVEL SECURITY;
ALTER TABLE work_order_state_event NO FORCE ROW LEVEL SECURITY;
ALTER TABLE work_order_provisional_record NO FORCE ROW LEVEL SECURITY;

ALTER TABLE work_order DISABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_scope_item DISABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_assignment DISABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_assignment_scope DISABLE ROW LEVEL SECURITY;
ALTER TABLE inspection_record DISABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_submission_segment DISABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_submission_item DISABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_operation DISABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_state_event DISABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_provisional_record DISABLE ROW LEVEL SECURITY;

REVOKE ALL ON TABLE work_order, work_order_scope_item, work_order_assignment,
    work_order_assignment_scope, inspection_record, work_order_submission_segment,
    work_order_submission_item, work_order_operation, work_order_state_event,
    work_order_provisional_record FROM integin_test_runtime;

-- INTEGIN Hardening: Fix Unindexed Foreign Keys to Prevent Deadlocks and Table Scans
-- Addresses Trap #3 discovered during deep architectural audit.

BEGIN;

-- 1. identity_membership
CREATE INDEX IF NOT EXISTS idx_identity_membership_actor_tenant_org
    ON identity_membership (actor_id, tenant_id, organization_id);

-- 2. work_package_assignment
CREATE INDEX IF NOT EXISTS idx_work_package_assignment_pkg_ref
    ON work_package_assignment (tenant_id, organization_id, package_id, package_version);

-- 3, 4, 5. work_order_submission_item
CREATE INDEX IF NOT EXISTS idx_work_order_submission_item_segment_ref
    ON work_order_submission_item (tenant_id, organization_id, submission_segment_id, work_order_id, assignment_id);

CREATE INDEX IF NOT EXISTS idx_work_order_submission_item_inspection_ref
    ON work_order_submission_item (tenant_id, organization_id, inspection_id, work_order_id, assignment_id, scope_item_id);

CREATE INDEX IF NOT EXISTS idx_work_order_submission_item_assignment_ref
    ON work_order_submission_item (tenant_id, organization_id, assignment_id, scope_item_id);

-- 6, 7. work_order_handover
CREATE INDEX IF NOT EXISTS idx_work_order_handover_to_assignment
    ON work_order_handover (tenant_id, organization_id, to_assignment_id);

CREATE INDEX IF NOT EXISTS idx_work_order_handover_from_assignment
    ON work_order_handover (tenant_id, organization_id, from_assignment_id);

-- 8, 9, 10. certificate_record
CREATE INDEX IF NOT EXISTS idx_certificate_record_policy_id
    ON certificate_record (policy_id);

CREATE INDEX IF NOT EXISTS idx_certificate_record_supersedes_id
    ON certificate_record (supersedes_id);

CREATE INDEX IF NOT EXISTS idx_certificate_record_superseded_by_id
    ON certificate_record (superseded_by_id);

-- 11. time_sheet
CREATE INDEX IF NOT EXISTS idx_time_sheet_work_order_ref
    ON time_sheet (tenant_id, organization_id, work_order_id);

-- 12. work_order_site_handover
CREATE INDEX IF NOT EXISTS idx_work_order_site_handover_from_assignment
    ON work_order_site_handover (tenant_id, organization_id, from_assignment_id);

COMMIT;

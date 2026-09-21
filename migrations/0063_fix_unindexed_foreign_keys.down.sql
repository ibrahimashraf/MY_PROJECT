-- Revert integin Hardening: Drop covering foreign key indexes added in 0063

BEGIN;

DROP INDEX IF EXISTS idx_identity_membership_actor_tenant_org;
DROP INDEX IF EXISTS idx_work_package_assignment_pkg_ref;
DROP INDEX IF EXISTS idx_work_order_submission_item_segment_ref;
DROP INDEX IF EXISTS idx_work_order_submission_item_inspection_ref;
DROP INDEX IF EXISTS idx_work_order_submission_item_assignment_ref;
DROP INDEX IF EXISTS idx_work_order_handover_to_assignment;
DROP INDEX IF EXISTS idx_work_order_handover_from_assignment;
DROP INDEX IF EXISTS idx_certificate_record_policy_id;
DROP INDEX IF EXISTS idx_certificate_record_supersedes_id;
DROP INDEX IF EXISTS idx_certificate_record_superseded_by_id;
DROP INDEX IF EXISTS idx_time_sheet_work_order_ref;
DROP INDEX IF EXISTS idx_work_order_site_handover_from_assignment;

COMMIT;

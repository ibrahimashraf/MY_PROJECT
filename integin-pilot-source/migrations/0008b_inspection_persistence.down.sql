-- Rollback for 0008b_inspection_persistence.
DROP INDEX IF EXISTS inspection_record_membership_lookup;
DROP TABLE IF EXISTS work_order_submission_item;
DROP TABLE IF EXISTS inspection_record;
ALTER TABLE IF EXISTS work_order_submission_segment DROP CONSTRAINT IF EXISTS work_order_submission_segment_identity_unique;
ALTER TABLE IF EXISTS work_order_assignment DROP CONSTRAINT IF EXISTS work_order_assignment_order_identity_unique;
ALTER TABLE IF EXISTS work_order_scope_item DROP CONSTRAINT IF EXISTS work_order_scope_item_order_identity_unique;

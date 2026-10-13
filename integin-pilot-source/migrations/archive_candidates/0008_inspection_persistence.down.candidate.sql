-- CANDIDATE ROLLBACK ONLY. Review dependency order before use.

BEGIN;

DROP TABLE IF EXISTS work_order_submission_item;
DROP TABLE IF EXISTS inspection_record;

ALTER TABLE work_order_submission_segment
    DROP CONSTRAINT IF EXISTS work_order_submission_segment_identity_unique;

ALTER TABLE work_order_assignment
    DROP CONSTRAINT IF EXISTS work_order_assignment_order_identity_unique;

ALTER TABLE work_order_scope_item
    DROP CONSTRAINT IF EXISTS work_order_scope_item_order_identity_unique;

COMMIT;

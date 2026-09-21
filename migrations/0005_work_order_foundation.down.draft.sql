-- integin Stage 0 Work-Order Foundation
-- DRAFT ONLY: do not apply without explicit migration authorization.
-- Rollback order follows foreign-key dependencies from leaves to roots.

BEGIN;

DROP TABLE IF EXISTS work_order_state_event;
DROP TABLE IF EXISTS work_order_provisional_record;
DROP TABLE IF EXISTS work_order_submission_segment;
DROP TABLE IF EXISTS work_order_operation;
DROP TABLE IF EXISTS work_order_assignment_scope;
DROP TABLE IF EXISTS work_order_assignment;
DROP TABLE IF EXISTS work_order_scope_item;
DROP TABLE IF EXISTS work_order;

COMMIT;

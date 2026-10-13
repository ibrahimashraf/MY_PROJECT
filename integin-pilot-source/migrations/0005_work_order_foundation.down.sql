-- Rollback for 0005_work_order_foundation: removes the work-order foundation tables.
DROP INDEX IF EXISTS work_order_scope_lookup;
DROP INDEX IF EXISTS work_order_assignment_lookup;
DROP INDEX IF EXISTS work_order_operation_lookup;
DROP INDEX IF EXISTS work_order_event_lookup;
DROP TABLE IF EXISTS work_order_assignment_scope;
DROP TABLE IF EXISTS work_order_submission_segment;
DROP TABLE IF EXISTS work_order_operation;
DROP TABLE IF EXISTS work_order_state_event;
DROP TABLE IF EXISTS work_order_assignment;
DROP TABLE IF EXISTS work_order_scope_item;
DROP TABLE IF EXISTS work_order_provisional_record;
DROP TABLE IF EXISTS work_order;

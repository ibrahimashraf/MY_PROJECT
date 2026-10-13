-- Rollback for 0069_state_machine_and_sequence_bounds.sql

BEGIN;

ALTER TABLE work_order DROP CONSTRAINT IF EXISTS work_order_request_state_check;
ALTER TABLE work_order DROP CONSTRAINT IF EXISTS work_order_execution_state_check;
ALTER TABLE work_order DROP CONSTRAINT IF EXISTS work_order_commercial_state_check;
ALTER TABLE work_order DROP CONSTRAINT IF EXISTS work_order_certificate_state_check;
ALTER TABLE work_order_assignment DROP CONSTRAINT IF EXISTS work_order_assignment_state_check;
ALTER TABLE sync_device_state DROP CONSTRAINT IF EXISTS sync_device_state_sequence_bounds_check;

COMMIT;

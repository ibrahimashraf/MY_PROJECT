-- Down migration for 0081_audit_log_valid_time
BEGIN;
DROP INDEX IF EXISTS audit_log_valid_time_idx;
ALTER TABLE audit_log DROP COLUMN IF EXISTS valid_time;
COMMIT;
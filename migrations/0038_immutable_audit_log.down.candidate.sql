-- Down migration for 0038_immutable_audit_log
BEGIN;
DROP TABLE IF EXISTS immutable_audit_log CASCADE;
DROP TABLE IF EXISTS audit_log_hash_chain CASCADE;
COMMIT;
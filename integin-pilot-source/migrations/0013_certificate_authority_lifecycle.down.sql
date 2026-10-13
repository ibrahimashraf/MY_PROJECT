-- Rollback for 0013_certificate_authority_lifecycle.
DROP INDEX IF EXISTS certificate_record_scope_status_idx;
DROP INDEX IF EXISTS certificate_audit_event_certificate_idx;
DROP INDEX IF EXISTS certificate_record_one_active_revision;
DROP TABLE IF EXISTS certificate_audit_event;
DROP TABLE IF EXISTS certificate_snapshot;
DROP TABLE IF EXISTS certificate_record;
DROP TABLE IF EXISTS certificate_number_sequence;
DROP TABLE IF EXISTS certificate_policy;

-- CANDIDATE ROLLBACK ONLY: review against an isolated disposable copy before use.
BEGIN;
DROP TABLE IF EXISTS certificate_audit_event;
DROP TABLE IF EXISTS certificate_snapshot;
DROP TABLE IF EXISTS certificate_record;
DROP TABLE IF EXISTS certificate_number_sequence;
DROP TABLE IF EXISTS certificate_policy;
COMMIT;

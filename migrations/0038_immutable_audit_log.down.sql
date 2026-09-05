-- Down migration for 0038_immutable_audit_log
BEGIN;
DROP POLICY IF EXISTS audit_log_tenant_organization_isolation ON audit_log;
DROP TRIGGER IF EXISTS audit_log_immutable ON audit_log;
DROP FUNCTION IF EXISTS audit_log_no_update() CASCADE;
DROP TABLE IF EXISTS audit_log CASCADE;
COMMIT;

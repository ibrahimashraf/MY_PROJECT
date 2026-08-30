BEGIN;
DROP POLICY IF EXISTS audit_trail_export_tenant_isolation ON audit_trail_export;
DROP POLICY IF EXISTS tenant_setting_tenant_isolation ON tenant_setting;
DROP INDEX IF EXISTS audit_trail_export_status_requested_idx;
DROP INDEX IF EXISTS tenant_setting_key_scope_idx;
DROP TABLE IF EXISTS audit_trail_export;
DROP TABLE IF EXISTS tenant_setting;
COMMIT;
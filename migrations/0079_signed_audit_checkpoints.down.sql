-- Migration 0079 down: remove the signed audit checkpoint registry, its
-- immutability guard, RLS policy, and runtime privileges.

BEGIN;

ALTER TABLE audit_checkpoint_registry DISABLE ROW LEVEL SECURITY;
DROP TRIGGER IF EXISTS audit_checkpoint_immutable ON audit_checkpoint_registry;
DROP FUNCTION IF EXISTS audit_checkpoint_no_update();
DROP POLICY IF EXISTS audit_checkpoint_registry_tenant_isolation ON audit_checkpoint_registry;
DROP TABLE IF EXISTS audit_checkpoint_registry;
REVOKE ALL ON TABLE audit_checkpoint_registry FROM integin_runtime;

COMMIT;
-- Down migration for 0085_edge_lease_coordinator
-- Drops policies, disables RLS, and removes tables in reverse-dependency order.

ALTER TABLE IF EXISTS merkle_audit_log DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS merkle_audit_log_tenant_isolation ON merkle_audit_log;
DROP TABLE IF EXISTS merkle_audit_log;

ALTER TABLE IF EXISTS device_revocation_registry DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS device_revocation_registry_tenant_isolation ON device_revocation_registry;
DROP TABLE IF EXISTS device_revocation_registry;

ALTER TABLE IF EXISTS authority_epoch_log DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS authority_epoch_log_tenant_isolation ON authority_epoch_log;
DROP TABLE IF EXISTS authority_epoch_log;

ALTER TABLE IF EXISTS lease_tombstones DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS lease_tombstones_tenant_isolation ON lease_tombstones;
DROP TABLE IF EXISTS lease_tombstones;

ALTER TABLE IF EXISTS draft_work_orders DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS draft_work_orders_tenant_isolation ON draft_work_orders;
DROP TABLE IF EXISTS draft_work_orders;

ALTER TABLE IF EXISTS sub_assembly_leases DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sub_assembly_leases_tenant_isolation ON sub_assembly_leases;
DROP TABLE IF EXISTS sub_assembly_leases;

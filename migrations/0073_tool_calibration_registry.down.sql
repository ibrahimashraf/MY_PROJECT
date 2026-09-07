BEGIN;

DROP POLICY IF EXISTS tool_calibration_registry_tenant_isolation ON tool_calibration_registry;
ALTER TABLE tool_calibration_registry DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS tool_calibration_registry;

COMMIT;
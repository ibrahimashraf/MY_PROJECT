-- Rollback integin assignment context: work_package_assignment_context
BEGIN;

DROP POLICY IF EXISTS work_package_assignment_context_tenant_isolation ON work_package_assignment_context;
ALTER TABLE work_package_assignment_context NO FORCE ROW LEVEL SECURITY;
ALTER TABLE work_package_assignment_context DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS work_package_assignment_context;

COMMIT;

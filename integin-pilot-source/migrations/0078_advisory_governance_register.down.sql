-- Migration 0078 down: reverses 0078_advisory_governance_register.sql.
-- Drop policies, disable RLS, and drop tables in reverse dependency order.

BEGIN;

DROP POLICY IF EXISTS advisory_feedback_tenant_isolation ON advisory_feedback;
ALTER TABLE advisory_feedback DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS advisory_audit_trail_tenant_isolation ON advisory_audit_trail;
ALTER TABLE advisory_audit_trail DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS advisory_feedback;
DROP TABLE IF EXISTS advisory_audit_trail;
DROP TABLE IF EXISTS advisory_prompt_registry;
DROP TABLE IF EXISTS advisory_model_registry;

COMMIT;
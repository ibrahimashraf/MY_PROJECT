-- Rollback INTEGIN Work-Order Field Package: versioned form definitions, field catalog, and evidence policy.
-- CANDIDATE ONLY: do not apply without a verified backup and isolated SQL review.

BEGIN;

DROP POLICY IF EXISTS evidence_policy_tenant_organization_isolation ON evidence_policy;
DROP POLICY IF EXISTS form_field_tenant_organization_isolation ON form_field;
DROP POLICY IF EXISTS form_version_tenant_organization_isolation ON form_version;

ALTER TABLE evidence_policy NO FORCE ROW LEVEL SECURITY;
ALTER TABLE evidence_policy DISABLE ROW LEVEL SECURITY;
ALTER TABLE form_field NO FORCE ROW LEVEL SECURITY;
ALTER TABLE form_field DISABLE ROW LEVEL SECURITY;
ALTER TABLE form_version NO FORCE ROW LEVEL SECURITY;
ALTER TABLE form_version DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS evidence_policy;
DROP TABLE IF EXISTS form_field;
DROP TABLE IF EXISTS form_version;

COMMIT;

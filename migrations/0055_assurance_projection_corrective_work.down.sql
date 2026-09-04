-- Rollback INTEGIN Work-Order Field Package: assurance projection and corrective-work links.
-- CANDIDATE ONLY: do not apply without a verified backup and isolated SQL review.

BEGIN;

DROP POLICY IF EXISTS corrective_work_tenant_organization_isolation ON corrective_work;
DROP POLICY IF EXISTS assurance_projection_tenant_organization_isolation ON assurance_projection;

ALTER TABLE corrective_work NO FORCE ROW LEVEL SECURITY;
ALTER TABLE corrective_work DISABLE ROW LEVEL SECURITY;
ALTER TABLE assurance_projection NO FORCE ROW LEVEL SECURITY;
ALTER TABLE assurance_projection DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS corrective_work;
DROP TABLE IF EXISTS assurance_projection;

COMMIT;

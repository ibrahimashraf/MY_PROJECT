-- Rollback INTEGIN Work-Order Field Package: evidence pack and release-pack composition.
-- CANDIDATE ONLY: do not apply without a verified backup and isolated SQL review.

BEGIN;

DROP POLICY IF EXISTS release_pack_tenant_organization_isolation ON release_pack;
DROP POLICY IF EXISTS evidence_pack_tenant_organization_isolation ON evidence_pack;

ALTER TABLE release_pack NO FORCE ROW LEVEL SECURITY;
ALTER TABLE release_pack DISABLE ROW LEVEL SECURITY;
ALTER TABLE evidence_pack NO FORCE ROW LEVEL SECURITY;
ALTER TABLE evidence_pack DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS release_pack;
DROP TABLE IF EXISTS evidence_pack;

COMMIT;

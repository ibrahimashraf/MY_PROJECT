-- Rollback INTEGIN Work-Order Field Package: asset entitlement and offline package.
-- CANDIDATE ONLY: do not apply without a verified backup and isolated SQL review.

BEGIN;

DROP POLICY IF EXISTS offline_package_tenant_organization_isolation ON offline_package;
DROP POLICY IF EXISTS asset_tag_tenant_organization_isolation ON asset_tag;
DROP POLICY IF EXISTS asset_entitlement_tenant_organization_isolation ON asset_entitlement;

ALTER TABLE offline_package NO FORCE ROW LEVEL SECURITY;
ALTER TABLE offline_package DISABLE ROW LEVEL SECURITY;
ALTER TABLE asset_tag NO FORCE ROW LEVEL SECURITY;
ALTER TABLE asset_tag DISABLE ROW LEVEL SECURITY;
ALTER TABLE asset_entitlement NO FORCE ROW LEVEL SECURITY;
ALTER TABLE asset_entitlement DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS offline_package;
DROP TABLE IF EXISTS asset_tag;
DROP TABLE IF EXISTS asset_entitlement;

COMMIT;

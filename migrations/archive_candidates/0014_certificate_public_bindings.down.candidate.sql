-- INTEGIN certificate public-binding rollback candidate.
-- Review only; apply only to the isolated disposable database that received 0014.
BEGIN;

ALTER TABLE certificate_snapshot DROP COLUMN public_binding_snapshot;
DROP TABLE certificate_policy_public_binding;
DROP TABLE inspection_public_scope_item;
DROP TABLE inspection_public_scope;
DROP TABLE asset_registry;
ALTER TABLE certificate_policy DROP CONSTRAINT certificate_policy_tenant_org_id_unique;

COMMIT;


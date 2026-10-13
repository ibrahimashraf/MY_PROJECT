-- Rollback for 0014_certificate_public_bindings.
DROP INDEX IF EXISTS asset_registry_lookup_idx;
DROP INDEX IF EXISTS inspection_public_scope_lookup_idx;
DROP INDEX IF EXISTS certificate_policy_public_binding_lookup_idx;
ALTER TABLE IF EXISTS certificate_snapshot DROP COLUMN IF EXISTS public_binding_snapshot;
DROP TABLE IF EXISTS certificate_policy_public_binding;
DROP TABLE IF EXISTS inspection_public_scope_item;
DROP TABLE IF EXISTS inspection_public_scope;
DROP TABLE IF EXISTS asset_registry;
ALTER TABLE IF EXISTS certificate_policy DROP CONSTRAINT IF EXISTS certificate_policy_tenant_org_id_unique;

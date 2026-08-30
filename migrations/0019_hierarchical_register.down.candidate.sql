-- INTEGIN hierarchical equipment register rollback.
-- CANDIDATE ONLY: do not apply without a verified pre-apply backup and isolated up/down review.
BEGIN;

-- Drop policies
DROP POLICY IF EXISTS equipment_type_field_tenant_isolation ON equipment_type_field;
DROP POLICY IF EXISTS equipment_type_tenant_isolation ON equipment_type;
DROP POLICY IF EXISTS location_zone_tenant_isolation ON location_zone;
DROP POLICY IF EXISTS location_area_tenant_isolation ON location_area;
DROP POLICY IF EXISTS location_branch_tenant_isolation ON location_branch;

-- Drop RLS
ALTER TABLE equipment_type_field DISABLE ROW LEVEL SECURITY;
ALTER TABLE equipment_type DISABLE ROW LEVEL SECURITY;
ALTER TABLE location_zone DISABLE ROW LEVEL SECURITY;
ALTER TABLE location_area DISABLE ROW LEVEL SECURITY;
ALTER TABLE location_branch DISABLE ROW LEVEL SECURITY;

-- Drop indexes on asset_registry
DROP INDEX IF EXISTS asset_registry_custom_gin;
DROP INDEX IF EXISTS asset_registry_type_idx;
DROP INDEX IF EXISTS asset_registry_zone_idx;
DROP INDEX IF EXISTS asset_registry_area_idx;
DROP INDEX IF EXISTS asset_registry_branch_idx;

-- Drop columns from asset_registry
ALTER TABLE asset_registry DROP COLUMN IF EXISTS custom_fields;
ALTER TABLE asset_registry DROP COLUMN IF EXISTS equipment_type_id;
ALTER TABLE asset_registry DROP COLUMN IF EXISTS zone_id;
ALTER TABLE asset_registry DROP COLUMN IF EXISTS area_id;
ALTER TABLE asset_registry DROP COLUMN IF EXISTS branch_id;

-- Drop indexes on equipment_type_field
DROP INDEX IF EXISTS equipment_type_field_type_idx;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS equipment_type_field;
DROP TABLE IF EXISTS equipment_type;

DROP INDEX IF EXISTS location_zone_area_idx;
DROP INDEX IF EXISTS location_area_branch_idx;

DROP TABLE IF EXISTS location_zone;
DROP TABLE IF EXISTS location_area;
DROP TABLE IF EXISTS location_branch;

COMMIT;
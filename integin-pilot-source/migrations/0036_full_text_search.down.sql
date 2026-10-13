BEGIN;

DROP TRIGGER IF EXISTS inspection_record_search_trigger ON inspection_record;
DROP FUNCTION IF EXISTS inspection_record_search_update();
DROP INDEX IF EXISTS inspection_record_search_idx;
ALTER TABLE inspection_record DROP COLUMN IF EXISTS search_vector;

DROP TRIGGER IF EXISTS work_order_search_trigger ON work_order;
DROP FUNCTION IF EXISTS work_order_search_update();
DROP INDEX IF EXISTS work_order_search_idx;
ALTER TABLE work_order DROP COLUMN IF EXISTS search_vector;

DROP TRIGGER IF EXISTS asset_registry_search_trigger ON asset_registry;
DROP FUNCTION IF EXISTS asset_registry_search_update();
DROP INDEX IF EXISTS asset_registry_search_idx;
ALTER TABLE asset_registry DROP COLUMN IF EXISTS search_vector;

COMMIT;

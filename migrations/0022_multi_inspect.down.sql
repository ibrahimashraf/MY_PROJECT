-- Migration 0022: Multi-Inspect Rack Workflow
-- Down migration (reverses up migration)

BEGIN;

-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_update_batch_completed_count ON multi_inspect_item;
DROP FUNCTION IF EXISTS update_multi_inspect_batch_completed_count();

-- Drop RLS policies
DROP POLICY IF EXISTS inspect_template_preset_org_isolation ON inspect_template_preset;
DROP POLICY IF EXISTS inspect_template_preset_tenant_isolation ON inspect_template_preset;

DROP POLICY IF EXISTS multi_inspect_item_org_isolation ON multi_inspect_item;
DROP POLICY IF EXISTS multi_inspect_item_tenant_isolation ON multi_inspect_item;

DROP POLICY IF EXISTS multi_inspect_batch_org_isolation ON multi_inspect_batch;
DROP POLICY IF EXISTS multi_inspect_batch_tenant_isolation ON multi_inspect_batch;

-- Disable RLS
ALTER TABLE inspect_template_preset DISABLE ROW LEVEL SECURITY;
ALTER TABLE multi_inspect_item DISABLE ROW LEVEL SECURITY;
ALTER TABLE multi_inspect_batch DISABLE ROW LEVEL SECURITY;

-- Drop indexes
DROP INDEX IF EXISTS idx_inspect_template_preset_tenant_org_equipment;

DROP INDEX IF EXISTS idx_multi_inspect_item_tenant_org_status;
DROP INDEX IF EXISTS idx_multi_inspect_item_inspection;
DROP INDEX IF EXISTS idx_multi_inspect_item_asset_lookup;
DROP INDEX IF EXISTS idx_multi_inspect_item_batch_sequence;

DROP INDEX IF EXISTS idx_multi_inspect_batch_assignment;
DROP INDEX IF EXISTS idx_multi_inspect_batch_work_order;
DROP INDEX IF EXISTS idx_multi_inspect_batch_tenant_org_status;

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS inspect_template_preset;
DROP TABLE IF EXISTS multi_inspect_item;
DROP TABLE IF EXISTS multi_inspect_batch;

COMMIT;
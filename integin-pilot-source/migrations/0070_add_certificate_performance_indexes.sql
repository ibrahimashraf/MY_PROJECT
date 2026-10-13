-- Migration 0070: Add composite B-tree indexes for certificate & work-order performance
-- Verified against live PostgreSQL 16 schema in integin_dev.

-- 1. Optimize certificate record lookups by status with tenant/org RLS
CREATE INDEX IF NOT EXISTS idx_certificate_record_tenant_status_org 
ON certificate_record(tenant_id, organization_id, status);

-- 2. Optimize certificate policy lookups with template version & status
CREATE INDEX IF NOT EXISTS idx_certificate_policy_tenant_org_ver_status 
ON certificate_policy(tenant_id, organization_id, template_code, template_version, status);

-- 3. Optimize inspection record lookups with tenant/org filtering
CREATE INDEX IF NOT EXISTS idx_inspection_record_tenant_org_id 
ON inspection_record(tenant_id, organization_id, id);

-- 4. Optimize template cell rendering queries
CREATE INDEX IF NOT EXISTS idx_template_cell_tenant_org_template 
ON certificate_template_cell(tenant_id, organization_id, template_id, page_number);

-- 5. Optimize active asset lookups in registry
CREATE INDEX IF NOT EXISTS idx_asset_registry_asset_active 
ON asset_registry(asset_id) 
WHERE lifecycle_state = 'ACTIVE';

-- 6. Optimize work order assignment scope joins
CREATE INDEX IF NOT EXISTS idx_work_order_assignment_scope_assign_item 
ON work_order_assignment_scope(assignment_id, scope_item_id);

-- 7. Optimize work order scope item filtering by tenant & org
CREATE INDEX IF NOT EXISTS idx_work_order_scope_item_tenant_org 
ON work_order_scope_item(tenant_id, organization_id, work_order_id);

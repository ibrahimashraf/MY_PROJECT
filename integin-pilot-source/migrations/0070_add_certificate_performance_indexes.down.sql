-- Rollback for 0070_add_certificate_performance_indexes.
DROP INDEX IF EXISTS idx_certificate_record_tenant_status_org;
DROP INDEX IF EXISTS idx_certificate_policy_tenant_org_ver_status;
DROP INDEX IF EXISTS idx_inspection_record_tenant_org_id;
DROP INDEX IF EXISTS idx_template_cell_tenant_org_template;
DROP INDEX IF EXISTS idx_asset_registry_asset_active;
DROP INDEX IF EXISTS idx_work_order_assignment_scope_assign_item;
DROP INDEX IF EXISTS idx_work_order_scope_item_tenant_org;

-- Add composite B-tree indexes for certificate PG performance
-- These indexes address N+1 query patterns and sequential scan issues

-- Index 1: certificate_record(tenant_id, organization_id, status)
-- Optimizes: point lookups, RLS predicate filtering, status-based queries
CREATE INDEX IF NOT EXISTS idx_certificate_record_tenant_status_org 
ON certificate_record(tenant_id, organization_id, status);

-- Index 2: certificate_policy(tenant_id, organization_id, template_code, template_version, status)
-- Optimizes: policy lookups with RLS, template version filtering
CREATE INDEX IF NOT EXISTS idx_certificate_policy_tenant_org_ver_status 
ON certificate_policy(tenant_id, organization_id, template_code, template_version, status);

-- Index 3: inspection_record(tenant_id, organization_id, id)
-- Optimizes: inspection record lookups with tenant/org filtering
CREATE INDEX IF NOT EXISTS idx_inspection_record_tenant_org_id 
ON inspection_record(tenant_id, organization_id, id);

-- Index 4: certificate_snapshot(certificate_id) WHERE status IN ('ISSUED','EXPIRED','REVOKED','SUPERSEDED')
-- Optimizes: public view lookups by digest
CREATE UNIQUE INDEX IF NOT EXISTS idx_cert_snapshot_digest 
ON certificate_snapshot(certificate_id) 
WHERE status IN ('ISSUED','EXPIRED','REVOKED','SUPERSEDED');

-- Index 5: certificate_template_cell(tenant_id, organization_id, template_id, page_number)
-- Optimizes: GetRenderData second-query for template cells
CREATE INDEX IF NOT EXISTS idx_template_cell_tenant_org_template 
ON certificate_template_cell(tenant_id, organization_id, template_id, page_number);

-- Index 6: asset_registry(asset_id) WHERE lifecycle_state='ACTIVE'
-- Optimizes: loadPublicAssetFacts lookups
CREATE INDEX IF NOT EXISTS idx_asset_registry_asset_active 
ON asset_registry(asset_id) 
WHERE lifecycle_state = 'ACTIVE';

-- Index 7: work_order_assignment_scope(assignment_id, scope_item_id)
-- Optimizes: scope item lookups
CREATE INDEX IF NOT EXISTS idx_work_order_assignment_scope_assign_item 
ON work_order_assignment_scope(assignment_id, scope_item_id);

-- Index 8: webhook_deliveries(status, next_retry_at)
-- Optimizes: webhook retry worker batch fetching
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status_retry 
ON webhook_deliveries(status, next_retry_at);

-- Index 9: shortlinks(expires_at) WHERE expires_at IS NOT NULL
-- Optimizes: shortlink expiration cleanup
CREATE INDEX IF NOT EXISTS idx_shortlinks_expires_at 
ON shortlinks(expires_at) 
WHERE expires_at IS NOT NULL;

-- Index 10: work_order_scope_item(tenant_id, organization_id, work_order_id)
-- Optimizes: scope item lookups with tenant/org filtering
CREATE INDEX IF NOT EXISTS idx_work_order_scope_item_tenant_org 
ON work_order_scope_item(tenant_id, organization_id, work_order_id);
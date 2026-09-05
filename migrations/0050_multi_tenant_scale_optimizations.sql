-- Migration 0050: Multi-Tenant Scale Optimizations (1,001 Orgs x 10,001 Users x 100,001 Assets)
-- Adds compound multi-tenant GIN indexing for search and compound sorting indexes for asset registry.
-- CANDIDATE ONLY. Do not apply without a fresh backup and disposable review.

BEGIN;

CREATE EXTENSION IF NOT EXISTS btree_gin;

-- Compound multi-tenant GIN index on asset_registry:
-- Prevents global postings list cross-tenant scans under high cardinality.
CREATE INDEX IF NOT EXISTS asset_registry_tenant_search_idx
    ON asset_registry USING GIN (tenant_id, organization_id, search_vector);

-- Compound sorting index for inventory queries:
-- Eliminates memory/disk Quicksort when ordering 100,001 assets by created_at.
CREATE INDEX IF NOT EXISTS asset_registry_tenant_created_idx
    ON asset_registry (tenant_id, organization_id, created_at DESC);

-- Compound sorting index for work_order inventory/search:
CREATE INDEX IF NOT EXISTS work_order_tenant_search_idx
    ON work_order USING GIN (tenant_id, organization_id, search_vector);

COMMIT;

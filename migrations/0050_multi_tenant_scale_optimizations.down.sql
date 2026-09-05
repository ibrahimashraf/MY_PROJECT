-- Migration 0050 Down: Multi-Tenant Scale Optimizations Rollback
-- Drops compound multi-tenant GIN and sorting indexes.

BEGIN;

DROP INDEX IF EXISTS asset_registry_tenant_search_idx;
DROP INDEX IF EXISTS asset_registry_tenant_created_idx;
DROP INDEX IF EXISTS work_order_tenant_search_idx;

COMMIT;

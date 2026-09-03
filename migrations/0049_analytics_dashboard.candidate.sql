-- Migration 0049: Analytics Dashboard Indexes
-- Adds multi-tenant composite indexes to optimize dashboard analytics queries and removes duplicate index definitions.

BEGIN;

-- Tenant + Code + Timestamp compound index for aggregate timeline and stats queries
CREATE INDEX IF NOT EXISTS idx_scan_events_tenant_code_timestamp 
    ON short_link_scan_events (tenant_id, organization_id, short_link_code, timestamp DESC);

-- Index for geo heatmap queries with multi-tenant prefix
CREATE INDEX IF NOT EXISTS idx_scan_events_tenant_geo 
    ON short_link_scan_events (tenant_id, organization_id, country, region, city);

-- Index for device analytics with multi-tenant prefix
CREATE INDEX IF NOT EXISTS idx_scan_events_tenant_device 
    ON short_link_scan_events (tenant_id, organization_id, device_type, os, browser);

-- Index for funnel queries (IP + timestamp) with multi-tenant prefix
CREATE INDEX IF NOT EXISTS idx_scan_events_tenant_ip_timestamp 
    ON short_link_scan_events (tenant_id, organization_id, ip, timestamp DESC);

GRANT SELECT ON short_link_scan_events TO integin_runtime;

COMMIT;

-- Migration 0049: Analytics Dashboard Indexes
-- Adds indexes to optimize dashboard analytics queries

BEGIN;

-- Composite index for tenant + timestamp queries (most common filter)
CREATE INDEX IF NOT EXISTS idx_scan_events_tenant_timestamp ON short_link_scan_events (short_link_code, timestamp DESC);

-- Index for geo heatmap queries
CREATE INDEX IF NOT EXISTS idx_scan_events_country_region_city ON short_link_scan_events (country, region, city);

-- Index for device analytics
CREATE INDEX IF NOT EXISTS idx_scan_events_device_os_browser ON short_link_scan_events (device_type, os, browser);

-- Index for funnel queries (IP + timestamp)
CREATE INDEX IF NOT EXISTS idx_scan_events_ip_timestamp ON short_link_scan_events (ip, timestamp DESC);

-- Index for top assets (scan count aggregation)
CREATE INDEX IF NOT EXISTS idx_scan_events_short_link_timestamp ON short_link_scan_events (short_link_code, timestamp DESC);

-- Index for recent events (most common query range)
CREATE INDEX IF NOT EXISTS idx_scan_events_recent ON short_link_scan_events (short_link_code, timestamp DESC);

GRANT SELECT ON short_link_scan_events TO integin_runtime;

COMMIT;

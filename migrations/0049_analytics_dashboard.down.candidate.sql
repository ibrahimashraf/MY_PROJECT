-- Migration 0049 down: Remove Analytics Dashboard Indexes

DROP INDEX IF EXISTS idx_scan_events_tenant_timestamp;
DROP INDEX IF EXISTS idx_scan_events_country_region_city;
DROP INDEX IF EXISTS idx_scan_events_device_os_browser;
DROP INDEX IF EXISTS idx_scan_events_ip_timestamp;
DROP INDEX IF EXISTS idx_scan_events_short_link_timestamp;
DROP INDEX IF EXISTS idx_scan_events_recent;
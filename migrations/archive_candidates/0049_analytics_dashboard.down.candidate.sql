-- Migration 0049 Down: Analytics Dashboard Indexes Rollback

BEGIN;

DROP INDEX IF EXISTS idx_scan_events_tenant_code_timestamp;
DROP INDEX IF EXISTS idx_scan_events_tenant_geo;
DROP INDEX IF EXISTS idx_scan_events_tenant_device;
DROP INDEX IF EXISTS idx_scan_events_tenant_ip_timestamp;

COMMIT;
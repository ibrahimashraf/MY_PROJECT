-- Migration 0040: Short Link Scan Events
-- TEXT IDs, combined RLS, FORCE RLS

BEGIN;

CREATE TABLE IF NOT EXISTS short_link_scan_events (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    short_link_code TEXT NOT NULL REFERENCES short_links(code) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    ip TEXT,
    country TEXT,
    region TEXT,
    city TEXT,
    device_type TEXT,
    os TEXT,
    browser TEXT,
    referrer TEXT,
    utm_source TEXT,
    utm_medium TEXT,
    utm_campaign TEXT,
    utm_term TEXT,
    utm_content TEXT
);

CREATE INDEX IF NOT EXISTS idx_scan_events_tenant_org ON short_link_scan_events (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_scan_events_short_link_code ON short_link_scan_events (short_link_code);
CREATE INDEX IF NOT EXISTS idx_scan_events_timestamp ON short_link_scan_events (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_scan_events_country ON short_link_scan_events (country);
CREATE INDEX IF NOT EXISTS idx_scan_events_device_type ON short_link_scan_events (device_type);
CREATE INDEX IF NOT EXISTS idx_scan_events_browser ON short_link_scan_events (browser);
CREATE INDEX IF NOT EXISTS idx_scan_events_referrer ON short_link_scan_events (referrer);
CREATE INDEX IF NOT EXISTS idx_scan_events_utm ON short_link_scan_events (utm_source, utm_medium, utm_campaign);

ALTER TABLE short_link_scan_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE short_link_scan_events FORCE ROW LEVEL SECURITY;

REVOKE ALL ON TABLE short_link_scan_events FROM PUBLIC, integin_test_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE short_link_scan_events TO integin_test_runtime;

DROP POLICY IF EXISTS short_link_scan_events_isolation ON short_link_scan_events;
CREATE POLICY short_link_scan_events_isolation ON short_link_scan_events
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;


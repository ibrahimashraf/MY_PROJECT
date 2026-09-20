-- Migration 0039: Short Links
-- TEXT IDs, combined RLS, FORCE RLS

BEGIN;

CREATE TABLE IF NOT EXISTS short_links (
    code TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    target_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    scan_count BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_short_links_tenant_org ON short_links (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_short_links_expires_at ON short_links (expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_short_links_revoked_at ON short_links (revoked_at) WHERE revoked_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_short_links_created_at ON short_links (created_at DESC);

ALTER TABLE short_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE short_links FORCE ROW LEVEL SECURITY;

REVOKE ALL ON TABLE short_links FROM PUBLIC, integin_test_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE short_links TO integin_test_runtime;

DROP POLICY IF EXISTS short_links_isolation ON short_links;
CREATE POLICY short_links_isolation ON short_links
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE OR REPLACE FUNCTION increment_scan_count(code TEXT) RETURNS VOID AS $$
BEGIN
    UPDATE short_links SET scan_count = scan_count + 1 WHERE code = $1;
END;
$$ LANGUAGE plpgsql VOLATILE;

COMMIT;


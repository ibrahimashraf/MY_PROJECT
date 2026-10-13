-- Migration 0047: Short Link HMAC
-- TEXT IDs, combined RLS, FORCE RLS

BEGIN;

ALTER TABLE short_links
    ADD COLUMN IF NOT EXISTS hmac_secret_ref TEXT,
    ADD COLUMN IF NOT EXISTS hmac_algorithm TEXT DEFAULT 'HS256',
    ADD COLUMN IF NOT EXISTS hmac_signature TEXT;

CREATE TABLE IF NOT EXISTS short_link_hmac_secrets (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    version INT NOT NULL DEFAULT 1,
    secret TEXT NOT NULL,
    algorithm TEXT NOT NULL DEFAULT 'HS256',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ,
    UNIQUE (tenant_id, organization_id, version)
);

CREATE INDEX IF NOT EXISTS idx_hmac_secrets_tenant_org ON short_link_hmac_secrets (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_hmac_secrets_tenant_org_version ON short_link_hmac_secrets (tenant_id, organization_id, version);
CREATE INDEX IF NOT EXISTS idx_hmac_secrets_tenant_active ON short_link_hmac_secrets (tenant_id, organization_id) WHERE revoked_at IS NULL;

ALTER TABLE short_link_hmac_secrets ENABLE ROW LEVEL SECURITY;
ALTER TABLE short_link_hmac_secrets FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS short_link_hmac_secrets_isolation ON short_link_hmac_secrets;
CREATE POLICY short_link_hmac_secrets_isolation ON short_link_hmac_secrets
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

GRANT SELECT, INSERT, UPDATE, DELETE ON short_link_hmac_secrets TO integin_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON short_links TO integin_runtime;

COMMIT;

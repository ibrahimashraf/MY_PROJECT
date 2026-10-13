-- Migration 0034: License Entitlement Engine
-- TEXT IDs, combined RLS, FORCE RLS

BEGIN;

CREATE TABLE IF NOT EXISTS tenant_license (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    license_key_hash BYTEA NOT NULL,
    tier TEXT NOT NULL CHECK (tier IN ('starter','pro','enterprise')),
    status TEXT NOT NULL DEFAULT 'trial' CHECK (status IN ('trial','active','expired','revoked')),
    issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    max_inspectors INT DEFAULT 5,
    max_inspections_per_month INT DEFAULT 100,
    features JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS license_audit (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    license_id TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('issued','renewed','expired','revoked','trial_extended')),
    actor_id TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_tenant_license_tenant_org ON tenant_license (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_tenant_license_key_hash ON tenant_license (license_key_hash);
CREATE INDEX IF NOT EXISTS idx_tenant_license_status ON tenant_license (tenant_id, organization_id, status);
CREATE INDEX IF NOT EXISTS idx_license_audit_tenant_org ON license_audit (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_license_audit_license ON license_audit (tenant_id, organization_id, license_id, occurred_at DESC);

ALTER TABLE tenant_license ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_license FORCE ROW LEVEL SECURITY;
ALTER TABLE license_audit ENABLE ROW LEVEL SECURITY;
ALTER TABLE license_audit FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_license_isolation ON tenant_license;
CREATE POLICY tenant_license_isolation ON tenant_license
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

DROP POLICY IF EXISTS license_audit_isolation ON license_audit;
CREATE POLICY license_audit_isolation ON license_audit
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;

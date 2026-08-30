-- Migration 0028: Xero/M365 Sync + Token-Auth Data API + HubSpot/Hilti Adapters
-- Compatible with existing schema (TEXT IDs, integin.* settings, FORCE RLS)
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.

BEGIN;

CREATE TYPE integration_provider AS ENUM ('XERO','M365','HUBSPOT','HILTI_ONTRACT','CUSTOM');
CREATE TYPE integration_status AS ENUM ('ACTIVE','INACTIVE','ERROR');
CREATE TYPE sync_job_type AS ENUM ('FULL','INCREMENTAL');
CREATE TYPE sync_job_status AS ENUM ('PENDING','RUNNING','COMPLETED','FAILED');

CREATE TABLE IF NOT EXISTS integration_config (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    provider integration_provider NOT NULL,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    status integration_status NOT NULL DEFAULT 'ACTIVE',
    last_sync_at TIMESTAMPTZ,
    error_log JSONB,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, provider)
);

CREATE TABLE IF NOT EXISTS data_api_token (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    token_hash BYTEA NOT NULL CHECK (octet_length(token_hash) = 32),
    scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sync_job (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    provider integration_provider NOT NULL,
    job_type TEXT NOT NULL CHECK (job_type IN ('FULL','INCREMENTAL')),
    status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','RUNNING','COMPLETED','FAILED')),
    stats JSONB,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_log JSONB
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_integration_config_tenant_org ON integration_config (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_integration_config_provider ON integration_config (tenant_id, organization_id, provider);
CREATE INDEX IF NOT EXISTS idx_data_api_token_tenant_org ON data_api_token (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_data_api_token_token_hash ON data_api_token (token_hash);
CREATE INDEX IF NOT EXISTS idx_data_api_token_expires ON data_api_token (tenant_id, organization_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_sync_job_tenant_org ON sync_job (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_sync_job_provider ON sync_job (tenant_id, organization_id, provider);
CREATE INDEX IF NOT EXISTS idx_sync_job_status ON sync_job (tenant_id, organization_id, status);
CREATE INDEX IF NOT EXISTS idx_sync_job_started ON sync_job (tenant_id, organization_id, started_at);

-- RLS
ALTER TABLE integration_config ENABLE ROW LEVEL SECURITY;
ALTER TABLE integration_config FORCE ROW LEVEL SECURITY;
ALTER TABLE data_api_token ENABLE ROW LEVEL SECURITY;
ALTER TABLE data_api_token FORCE ROW LEVEL SECURITY;
ALTER TABLE sync_job ENABLE ROW LEVEL SECURITY;
ALTER TABLE sync_job FORCE ROW LEVEL SECURITY;

CREATE POLICY integration_config_tenant_isolation ON integration_config
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE POLICY data_api_token_tenant_isolation ON data_api_token
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE POLICY sync_job_tenant_isolation ON sync_job
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;
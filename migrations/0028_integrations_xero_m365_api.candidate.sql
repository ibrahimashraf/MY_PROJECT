-- Migration 0028: Xero/M365 sync + token-auth Data API + HubSpot/Hilti adapters
-- Up migration

CREATE TYPE integration_provider AS ENUM ('XERO', 'M365', 'HUBSPOT', 'HILTI_ONTRACT', 'CUSTOM');
CREATE TYPE integration_status AS ENUM ('ACTIVE', 'INACTIVE', 'ERROR');
CREATE TYPE sync_job_type AS ENUM ('FULL', 'INCREMENTAL');
CREATE TYPE sync_job_status AS ENUM ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED');

CREATE TABLE integration_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    provider integration_provider NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    status integration_status NOT NULL DEFAULT 'ACTIVE',
    last_sync_at TIMESTAMPTZ,
    error_log JSONB,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, org_id, provider)
);

CREATE TABLE data_api_token (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    token_hash BYTEA NOT NULL,
    scopes JSONB NOT NULL DEFAULT '[]',
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sync_job (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    provider integration_provider NOT NULL,
    job_type sync_job_type NOT NULL,
    status sync_job_status NOT NULL DEFAULT 'PENDING',
    stats JSONB,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_log JSONB
);

ALTER TABLE integration_config ENABLE ROW LEVEL SECURITY;
ALTER TABLE data_api_token ENABLE ROW LEVEL SECURITY;
ALTER TABLE sync_job ENABLE ROW LEVEL SECURITY;

CREATE INDEX idx_integration_config_tenant_org ON integration_config (tenant_id, org_id);
CREATE INDEX idx_integration_config_provider ON integration_config (provider);
CREATE INDEX idx_data_api_token_tenant_org ON data_api_token (tenant_id, org_id);
CREATE INDEX idx_data_api_token_token_hash ON data_api_token (token_hash);
CREATE INDEX idx_data_api_token_expires_at ON data_api_token (expires_at);
CREATE INDEX idx_sync_job_tenant_org ON sync_job (tenant_id, org_id);
CREATE INDEX idx_sync_job_provider ON sync_job (provider);
CREATE INDEX idx_sync_job_status ON sync_job (status);
CREATE INDEX idx_sync_job_started_at ON sync_job (started_at);

CREATE POLICY integration_config_tenant_isolation ON integration_config
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY data_api_token_tenant_isolation ON data_api_token
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY sync_job_tenant_isolation ON sync_job
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
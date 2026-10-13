-- Migration 0028: Xero/M365 sync + token-auth Data API + HubSpot/Hilti adapters
-- Down migration

DROP POLICY IF EXISTS sync_job_tenant_isolation ON sync_job;
DROP POLICY IF EXISTS data_api_token_tenant_isolation ON data_api_token;
DROP POLICY IF EXISTS integration_config_tenant_isolation ON integration_config;

DROP INDEX IF EXISTS idx_sync_job_started_at;
DROP INDEX IF EXISTS idx_sync_job_status;
DROP INDEX IF EXISTS idx_sync_job_provider;
DROP INDEX IF EXISTS idx_sync_job_tenant_org;
DROP INDEX IF EXISTS idx_data_api_token_expires_at;
DROP INDEX IF EXISTS idx_data_api_token_token_hash;
DROP INDEX IF EXISTS idx_data_api_token_tenant_org;
DROP INDEX IF EXISTS idx_integration_config_provider;
DROP INDEX IF EXISTS idx_integration_config_tenant_org;

DROP TABLE IF EXISTS sync_job;
DROP TABLE IF EXISTS data_api_token;
DROP TABLE IF EXISTS integration_config;

DROP TYPE IF EXISTS sync_job_status;
DROP TYPE IF EXISTS sync_job_type;
DROP TYPE IF EXISTS integration_status;
DROP TYPE IF EXISTS integration_provider;
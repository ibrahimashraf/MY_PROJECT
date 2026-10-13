SET lock_timeout = '2s';

-- Note: CREATE INDEX CONCURRENTLY cannot run inside a transaction block.
-- The migration runner must execute this file outside a transaction or support concurrent index creation.

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_assurance_asset_eval
    ON assurance_records (tenant_id, asset_id, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_current_assurance_filter
    ON current_assurance_states (tenant_id, state)
    WHERE state != 'ASSURED';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_current_assurance_expiry
    ON current_assurance_states (tenant_id, expires_at)
    WHERE expires_at IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_assurance_audit_cid
    ON assurance_records (tenant_id, audit_ledger_cid)
    WHERE audit_ledger_cid IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_assurance_trigger_event
    ON assurance_records (tenant_id, trigger_event_id)
    WHERE trigger_event_id IS NOT NULL;

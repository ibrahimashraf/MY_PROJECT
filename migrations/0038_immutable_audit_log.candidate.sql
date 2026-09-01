-- Immutable audit log with hash chain for tamper evidence.
-- CANDIDATE ONLY. Do not apply without a fresh backup and disposable review.
BEGIN;

CREATE TABLE audit_log (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    actor_name TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    old_value JSONB,
    new_value JSONB,
    metadata JSONB,
    ip_address TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    previous_hash TEXT NOT NULL,
    entry_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, id)
);

CREATE INDEX audit_log_tenant_org_idx ON audit_log (tenant_id, organization_id, created_at DESC);
CREATE INDEX audit_log_entity_idx ON audit_log (tenant_id, organization_id, entity_type, entity_id);
CREATE INDEX audit_log_actor_idx ON audit_log (tenant_id, organization_id, actor_id, created_at DESC);
CREATE INDEX audit_log_event_idx ON audit_log (tenant_id, organization_id, event_type, created_at DESC);
CREATE INDEX audit_log_hash_chain_idx ON audit_log (tenant_id, organization_id, previous_hash, entry_hash);

COMMIT;

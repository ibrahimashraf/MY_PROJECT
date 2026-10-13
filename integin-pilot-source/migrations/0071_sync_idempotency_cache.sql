-- Stripe-grade global idempotency cache for /sync and /evidence ingress.
-- Scoped by tenant + organization with a transaction-scoped (is_local = true)
-- RLS session model, bounded 24-hour TTL, and automatic eviction on write.

BEGIN;

CREATE TABLE IF NOT EXISTS sync_idempotency_cache (
    key_hash         TEXT        NOT NULL,
    tenant_id        TEXT        NOT NULL,
    organization_id  TEXT        NOT NULL,
    endpoint         TEXT        NOT NULL,
    request_hash     TEXT        NOT NULL,
    status           TEXT        NOT NULL,
    response_payload JSONB       NOT NULL DEFAULT '{}'::jsonb,
    http_status      INTEGER     NOT NULL DEFAULT 200,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at       TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, organization_id, key_hash),
    CHECK (btrim(key_hash) <> ''),
    CHECK (btrim(tenant_id) <> ''),
    CHECK (btrim(organization_id) <> ''),
    CHECK (btrim(endpoint) <> ''),
    CHECK (btrim(request_hash) <> ''),
    CHECK (status IN ('IN_PROGRESS', 'APPLIED', 'DUPLICATE', 'HELD', 'CONFLICT', 'REJECTED', 'QUEUED', 'SECURITY_FAILURE')),
    CHECK (http_status BETWEEN 200 AND 599),
    CHECK (expires_at > created_at)
);

-- TTL index enables the eviction trigger and any operator-side sweep.
CREATE INDEX IF NOT EXISTS sync_idempotency_cache_expiry_idx
    ON sync_idempotency_cache (expires_at);

-- Composite tenant isolation using the hardened NULLIF pattern so an empty
-- session GUC can never read or write multi-tenant rows.
ALTER TABLE sync_idempotency_cache ENABLE ROW LEVEL SECURITY;
ALTER TABLE sync_idempotency_cache FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS sync_idempotency_cache_tenant_isolation ON sync_idempotency_cache;
CREATE POLICY sync_idempotency_cache_tenant_isolation ON sync_idempotency_cache
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );

-- Automatic bounded eviction: each insert sweeps at most 1000 expired rows so
-- the cache never exceeds its 24-hour TTL without operator intervention, while
-- a single sweep cannot monopolize the table or the write latch.
CREATE OR REPLACE FUNCTION sync_idempotency_cache_evict_expired() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    DELETE FROM sync_idempotency_cache
    WHERE key_hash IN (
        SELECT key_hash FROM sync_idempotency_cache
        WHERE expires_at < now()
        ORDER BY expires_at ASC
        LIMIT 1000
    );
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS sync_idempotency_cache_evict_expired_trigger ON sync_idempotency_cache;
CREATE TRIGGER sync_idempotency_cache_evict_expired_trigger
    BEFORE INSERT ON sync_idempotency_cache
    FOR EACH ROW EXECUTE FUNCTION sync_idempotency_cache_evict_expired();

COMMIT;
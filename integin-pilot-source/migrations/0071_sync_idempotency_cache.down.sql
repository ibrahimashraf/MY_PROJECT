BEGIN;

DROP TRIGGER IF EXISTS sync_idempotency_cache_evict_expired_trigger ON sync_idempotency_cache;
DROP FUNCTION IF EXISTS sync_idempotency_cache_evict_expired();

DROP POLICY IF EXISTS sync_idempotency_cache_tenant_isolation ON sync_idempotency_cache;
ALTER TABLE sync_idempotency_cache DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS sync_idempotency_cache;

COMMIT;
SET lock_timeout = '2s';

DROP INDEX CONCURRENTLY IF EXISTS idx_assurance_trigger_event;
DROP INDEX CONCURRENTLY IF EXISTS idx_assurance_audit_cid;
DROP INDEX CONCURRENTLY IF EXISTS idx_current_assurance_expiry;
DROP INDEX CONCURRENTLY IF EXISTS idx_current_assurance_filter;
DROP INDEX CONCURRENTLY IF EXISTS idx_assurance_asset_eval;

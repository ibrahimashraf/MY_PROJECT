-- Down migration for 0037_search_backfill (candidate, backfill is irreversible - no DDL to revert)
BEGIN;
-- No DDL to revert; backfill UPDATE is data-only and irreversible
COMMIT;

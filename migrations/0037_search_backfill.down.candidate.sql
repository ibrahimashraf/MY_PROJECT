-- Down migration for 0037_search_backfill (candidate, backfill table)
BEGIN;
DROP TABLE IF EXISTS search_backfill_progress;
COMMIT;
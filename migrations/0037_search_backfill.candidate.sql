-- Backfill search vectors for existing rows.
-- Safe to re-run (triggers will override on next UPDATE).
BEGIN;

UPDATE asset_registry SET search_vector =
    setweight(to_tsvector('english', COALESCE(asset_id, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(asset_type, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(serial_number, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(description, '')), 'C')
WHERE search_vector IS NULL;

UPDATE work_order SET search_vector =
    setweight(to_tsvector('english', COALESCE(job_number, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(client_id, '')), 'B')
WHERE search_vector IS NULL;

UPDATE inspection_record SET search_vector =
    setweight(to_tsvector('english', COALESCE(asset_id, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(inspector_id, '')), 'B')
WHERE search_vector IS NULL;

COMMIT;

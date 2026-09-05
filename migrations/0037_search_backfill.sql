-- Backfill search vectors for existing rows. Requires 0036_full_text_search.
-- Safe to re-run (triggers will override on next UPDATE).
BEGIN;

DO $$ BEGIN IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='asset_registry' AND column_name='search_vector') THEN
UPDATE asset_registry SET search_vector =
    setweight(to_tsvector('english', COALESCE(asset_id, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(asset_type, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(serial_number, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(description, '')), 'C')
WHERE search_vector IS NULL;
END IF; END $$;

DO $$ BEGIN IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='work_order' AND column_name='search_vector') THEN
UPDATE work_order SET search_vector =
    setweight(to_tsvector('english', COALESCE(job_number, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(client_id, '')), 'B')
WHERE search_vector IS NULL;
END IF; END $$;

DO $$ BEGIN IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='inspection_record' AND column_name='search_vector') THEN
UPDATE inspection_record SET search_vector =
    setweight(to_tsvector('english', COALESCE(asset_id, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(inspector_id, '')), 'B')
WHERE search_vector IS NULL;
END IF; END $$;

COMMIT;

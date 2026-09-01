-- Full-text search foundation: tsvector columns + GIN indexes on searchable tables.
-- CANDIDATE ONLY. Do not apply without a fresh backup and disposable review.
BEGIN;

-- Asset registry: search by description, serial, type, asset_id
ALTER TABLE asset_registry
    ADD COLUMN search_vector tsvector;

CREATE INDEX asset_registry_search_idx ON asset_registry USING GIN (search_vector);

CREATE OR REPLACE FUNCTION asset_registry_search_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.asset_id, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.asset_type, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.serial_number, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER asset_registry_search_trigger
    BEFORE INSERT OR UPDATE ON asset_registry
    FOR EACH ROW EXECUTE FUNCTION asset_registry_search_update();

-- Work order: search by job_number, client_id
ALTER TABLE work_order
    ADD COLUMN search_vector tsvector;

CREATE INDEX work_order_search_idx ON work_order USING GIN (search_vector);

CREATE OR REPLACE FUNCTION work_order_search_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.job_number, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.client_id, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER work_order_search_trigger
    BEFORE INSERT OR UPDATE ON work_order
    FOR EACH ROW EXECUTE FUNCTION work_order_search_update();

-- Inspection record: search by asset_id, inspector_id
ALTER TABLE inspection_record
    ADD COLUMN search_vector tsvector;

CREATE INDEX inspection_record_search_idx ON inspection_record USING GIN (search_vector);

CREATE OR REPLACE FUNCTION inspection_record_search_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.asset_id, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.inspector_id, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER inspection_record_search_trigger
    BEFORE INSERT OR UPDATE ON inspection_record
    FOR EACH ROW EXECUTE FUNCTION inspection_record_search_update();

COMMIT;

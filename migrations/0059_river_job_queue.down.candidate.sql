-- INTEGIN Background Queue: River durable job queue schema rollback.
BEGIN;

DROP TABLE IF EXISTS river_leader CASCADE;
DROP TABLE IF EXISTS river_job CASCADE;

COMMIT;

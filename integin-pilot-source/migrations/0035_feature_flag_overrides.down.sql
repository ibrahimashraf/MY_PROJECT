-- Down migration 0035: Feature Flag Override Persistence
BEGIN;
DROP TABLE IF EXISTS feature_flag_override;
COMMIT;

BEGIN;
DROP INDEX IF EXISTS inspection_record_passport_idx;
DROP INDEX IF EXISTS inspection_record_geo_idx;
ALTER TABLE inspection_record DROP COLUMN IF EXISTS rfid_tag;
ALTER TABLE inspection_record DROP COLUMN IF EXISTS product_passport;
ALTER TABLE inspection_record DROP COLUMN IF EXISTS geo;
COMMIT;

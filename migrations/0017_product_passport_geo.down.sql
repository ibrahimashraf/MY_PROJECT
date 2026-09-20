-- Rollback for 0017_product_passport_geo.
DROP INDEX IF EXISTS inspection_record_geo_idx;
DROP INDEX IF EXISTS inspection_record_passport_idx;
ALTER TABLE IF EXISTS inspection_record DROP COLUMN IF EXISTS rfid_tag;
ALTER TABLE IF EXISTS inspection_record DROP COLUMN IF EXISTS product_passport;
ALTER TABLE IF EXISTS inspection_record DROP COLUMN IF EXISTS geo;

-- INTEGIN product passport / geo additive (no DPP patent clone, no cross-tenant sharing).
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.
BEGIN;

-- Additive JSONB on inspection_record so existing rows stay valid (NULL = not set).
-- Geo is WGS84 lat/lng + optional RFID tag; passport is serial/batch/manufacturer.
ALTER TABLE inspection_record ADD COLUMN IF NOT EXISTS geo JSONB CHECK (geo IS NULL OR (geo ? 'lat' AND geo ? 'lng'));
ALTER TABLE inspection_record ADD COLUMN IF NOT EXISTS product_passport JSONB CHECK (product_passport IS NULL OR (product_passport ? 'serial'));
ALTER TABLE inspection_record ADD COLUMN IF NOT EXISTS rfid_tag TEXT CHECK (rfid_tag IS NULL OR rfid_tag ~ '^[A-Za-z0-9._-]{1,64}$');

CREATE INDEX IF NOT EXISTS inspection_record_geo_idx ON inspection_record USING GIN (geo) WHERE geo IS NOT NULL;
CREATE INDEX IF NOT EXISTS inspection_record_passport_idx ON inspection_record USING GIN (product_passport) WHERE product_passport IS NOT NULL;

-- Certificate side: per-certificate passport snapshot is derived from inspection_record at issuance
-- so no new RLS policy change; still tenant-isolated via certificate_record FORCE RLS.

COMMIT;

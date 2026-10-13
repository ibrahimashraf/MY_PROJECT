-- 0031_nfc_rfid_qr_tagging_photo_markup.down.candidate.sql
-- Reverse migration: drop NFC/RFID/QR Tagging + Photo Markup/Annotation tables

BEGIN;

DROP INDEX IF EXISTS idx_photo_markup_evidence_id;
DROP INDEX IF EXISTS idx_asset_tag_asset_idx;
DROP INDEX IF EXISTS idx_asset_tag_digest_idx;
DROP TABLE IF EXISTS photo_markup;
DROP TABLE IF EXISTS asset_tag;

COMMIT;
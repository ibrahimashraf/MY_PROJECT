-- Rollback integin Work-Order Field Package: authenticated QR/NFC entry.
-- CANDIDATE ONLY: do not apply without a verified backup and isolated SQL review.

BEGIN;

DROP POLICY IF EXISTS qr_nfc_entry_log_tenant_organization_isolation ON qr_nfc_entry_log;
DROP POLICY IF EXISTS qr_nfc_entry_tenant_organization_isolation ON qr_nfc_entry;

ALTER TABLE qr_nfc_entry_log NO FORCE ROW LEVEL SECURITY;
ALTER TABLE qr_nfc_entry_log DISABLE ROW LEVEL SECURITY;
ALTER TABLE qr_nfc_entry NO FORCE ROW LEVEL SECURITY;
ALTER TABLE qr_nfc_entry DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS qr_nfc_entry_log;
DROP TABLE IF EXISTS qr_nfc_entry;

COMMIT;

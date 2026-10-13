-- Rollback for 0016_certificate_lifecycle_renewal.
DROP INDEX IF EXISTS certificate_renewal_attempt_cert_idx;
DROP INDEX IF EXISTS certificate_renewal_attempt_authority_idx;
DROP INDEX IF EXISTS certificate_record_renewal_idx;
ALTER TABLE IF EXISTS certificate_record DROP COLUMN IF EXISTS renewal_authority_token;
ALTER TABLE IF EXISTS certificate_record DROP COLUMN IF EXISTS last_renewal_attempt;
ALTER TABLE IF EXISTS certificate_record DROP COLUMN IF EXISTS renewal_count;
DROP TABLE IF EXISTS certificate_renewal_attempt;

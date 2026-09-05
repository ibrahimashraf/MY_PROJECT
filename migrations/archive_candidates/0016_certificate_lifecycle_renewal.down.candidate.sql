-- INTEGIN certificate lifecycle renewal rollback.
-- CANDIDATE ONLY: do not apply without a verified pre-apply backup and isolated up/down review.
-- This rollback restores the pre-migration state by removing renewal tracking fields.
BEGIN;

-- Drop policies first (order matters due to dependencies)
DROP POLICY IF EXISTS certificate_renewal_attempt_tenant_isolation ON certificate_renewal_attempt;

-- Drop indexes
DROP INDEX IF EXISTS certificate_renewal_attempt_authority_idx;
DROP INDEX IF EXISTS certificate_renewal_attempt_cert_idx;
DROP INDEX IF EXISTS certificate_record_renewal_idx;

-- Drop renewal tracking table
DROP TABLE IF EXISTS certificate_renewal_attempt;

-- Restore certificate_record to pre-migration state (remove added columns)
ALTER TABLE certificate_record DROP COLUMN IF EXISTS renewal_count;
ALTER TABLE certificate_record DROP COLUMN IF EXISTS last_renewal_attempt;
ALTER TABLE certificate_record DROP COLUMN IF EXISTS renewal_authority_token;

COMMIT;
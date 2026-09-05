-- INTEGIN Stage A additive manifest-required metadata extension rollback candidate.
-- CANDIDATE ONLY: execute only after a verified backup and explicit evidence-retention decision.

BEGIN;

ALTER TABLE evidence_metadata
    DROP COLUMN IF EXISTS encryption_key_reference,
    DROP COLUMN IF EXISTS encryption_algorithm;

COMMIT;

-- INTEGIN Stage A additive manifest-required metadata extension.
-- CANDIDATE ONLY: encryption references are immutable evidence provenance and may not be inferred from signature metadata.

BEGIN;

ALTER TABLE evidence_metadata
    ADD COLUMN encryption_algorithm TEXT NOT NULL,
    ADD COLUMN encryption_key_reference TEXT NOT NULL;

COMMIT;

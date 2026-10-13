-- Rollback for 0011_evidence_metadata_encryption_export.
ALTER TABLE IF EXISTS evidence_metadata DROP COLUMN IF EXISTS encryption_key_reference;
ALTER TABLE IF EXISTS evidence_metadata DROP COLUMN IF EXISTS encryption_algorithm;

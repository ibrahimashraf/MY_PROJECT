-- Rollback for 0010_evidence_metadata.
DROP INDEX IF EXISTS evidence_metadata_inspection_lookup;
DROP TABLE IF EXISTS evidence_metadata;

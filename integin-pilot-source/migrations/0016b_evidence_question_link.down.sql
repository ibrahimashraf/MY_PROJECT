-- Rollback for 0016b_evidence_question_link.
DROP INDEX IF EXISTS evidence_metadata_inspection_question_idx;
ALTER TABLE IF EXISTS evidence_metadata DROP COLUMN IF EXISTS question_index;
ALTER TABLE IF EXISTS evidence_metadata DROP COLUMN IF EXISTS question_code;

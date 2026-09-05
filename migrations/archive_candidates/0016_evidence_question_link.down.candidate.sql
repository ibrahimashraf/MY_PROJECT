BEGIN;
DROP INDEX IF EXISTS evidence_metadata_inspection_question_idx;
ALTER TABLE evidence_metadata DROP COLUMN IF EXISTS question_index;
ALTER TABLE evidence_metadata DROP COLUMN IF EXISTS question_code;
COMMIT;

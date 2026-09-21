-- integin per-question evidence linkage (additive, no RLS downgrade).
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.
BEGIN;

-- Additive nullable link so existing evidence rows stay valid; old rows = NULL question.
ALTER TABLE evidence_metadata ADD COLUMN IF NOT EXISTS question_code TEXT CHECK (question_code IS NULL OR (question_code ~ '^[A-Za-z0-9._-]{1,64}$'));
ALTER TABLE evidence_metadata ADD COLUMN IF NOT EXISTS question_index INT CHECK (question_index IS NULL OR question_index >= 0);

CREATE INDEX IF NOT EXISTS evidence_metadata_inspection_question_idx ON evidence_metadata (tenant_id, organization_id, inspection_id, question_code, captured_at DESC) WHERE question_code IS NOT NULL;

COMMIT;

-- INTEGIN Stage A evidence-retention foundation rollback candidate.
-- CANDIDATE ONLY: execute only after a verified backup and an explicit data-retention decision.

BEGIN;

DROP TABLE IF EXISTS evidence_metadata;

COMMIT;

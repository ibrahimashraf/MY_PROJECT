-- integin certificate-template binding registry rollback candidate.
-- CANDIDATE ONLY: execute only after a verified backup and explicit certificate-retention decision.

BEGIN;

DROP TABLE IF EXISTS certificate_template_cell;
DROP TABLE IF EXISTS certificate_template;

COMMIT;

-- Rollback for 0084: restore the original certificate_audit_event action allowlist.
-- Fails if ATTESTED or SIGN_WAIVED rows exist (intentional: no silent loss).

BEGIN;

ALTER TABLE certificate_audit_event DROP CONSTRAINT IF EXISTS certificate_audit_event_action_check;
ALTER TABLE certificate_audit_event ADD CONSTRAINT certificate_audit_event_action_check
    CHECK (action IN ('DRAFT_CREATED', 'REVIEWED', 'SIGNED', 'ISSUED', 'EXPIRED', 'REVOKED', 'SUPERSEDED'));

COMMIT;

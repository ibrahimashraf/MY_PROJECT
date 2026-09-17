-- Certificate signature actions: attestation and waived sign-off audit events.
-- Extends the certificate_audit_event action allowlist; additive only.

BEGIN;

ALTER TABLE certificate_audit_event DROP CONSTRAINT IF EXISTS certificate_audit_event_action_check;
ALTER TABLE certificate_audit_event ADD CONSTRAINT certificate_audit_event_action_check
    CHECK (action IN ('DRAFT_CREATED', 'REVIEWED', 'SIGNED', 'ISSUED', 'EXPIRED', 'REVOKED', 'SUPERSEDED', 'ATTESTED', 'SIGN_WAIVED'));

COMMIT;

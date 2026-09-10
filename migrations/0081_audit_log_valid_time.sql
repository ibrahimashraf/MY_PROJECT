-- Bitemporal audit ledger: add valid time (the moment the audited event
-- actually occurred). NULL means the event occurred at created_at, and the
-- domain layer maps NULL <-> zero time. RLS and immutability trigger from 0038
-- are untouched.
BEGIN;

ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS valid_time TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS audit_log_valid_time_idx ON audit_log (tenant_id, organization_id, valid_time DESC);

COMMIT;
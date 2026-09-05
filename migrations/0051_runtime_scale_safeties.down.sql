-- Migration 0051 Down: Runtime Scale Safeties Rollback
-- Resets PostgreSQL runtime session defaults.

BEGIN;

ALTER ROLE integin_runtime RESET statement_timeout;
ALTER ROLE integin_runtime RESET idle_in_transaction_session_timeout;
ALTER ROLE integin_runtime RESET deadlock_timeout;

COMMIT;

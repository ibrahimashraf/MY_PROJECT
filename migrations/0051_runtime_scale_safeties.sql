-- Migration 0051: Runtime Scale Safeties (10,000 Concurrent Clients)
-- Hardens PostgreSQL session timeouts and deadlock detection for high-concurrency operations.
-- CANDIDATE ONLY. Do not apply without a fresh backup and disposable review.

BEGIN;

-- Terminate queries that exceed 5 seconds to prevent hung queries from starving connection pools.
ALTER ROLE integin_test_runtime SET statement_timeout = '5000ms';

-- Terminate idle-in-transaction sessions after 10 seconds to prevent forgotten transactions from locking rows.
ALTER ROLE integin_test_runtime SET idle_in_transaction_session_timeout = '10000ms';

-- Reduce deadlock detection wait from default 1,000ms to 50ms so conflicting locks are resolved in 50ms.
ALTER ROLE integin_test_runtime SET deadlock_timeout = '50ms';

COMMIT;


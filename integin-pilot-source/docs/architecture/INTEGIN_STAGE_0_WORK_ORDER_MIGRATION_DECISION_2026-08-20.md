# INTEGIN Stage 0 Work-Order Migration Decision

**Status:** Draft reviewed; not executed.  
**Date:** 2026-08-20  
**Owner:** Manus AI under the user’s continuing implementation authorization.

## Decision

The additive PostgreSQL draft for the Work-Order Foundation is recorded in `migrations/0005_work_order_foundation.draft.sql` with rollback draft `migrations/0005_work_order_foundation.down.draft.sql`. The drafts cover work orders, scope items, assignments, assignment scope, idempotency receipts, partial submission segments, provisional reconciliation, and append-only state events.

The drafts passed read-only completeness review and rollback-order validation. The review confirmed the intended organization-scoped uniqueness, composite organization/work-order relationships, optimistic revision fields, idempotency uniqueness, and dependency-safe rollback order.

## Hold conditions

The SQL is not executable yet. Before any migration is applied, the repository owner must approve the exact organization-context function, runtime role, RLS policies, identifier types, foreign-key compatibility with existing client/location/asset tables, index plan, and non-destructive deployment/rollback procedure. The current draft intentionally stops before adding RLS policies because the existing repository-specific runtime context must be confirmed rather than guessed.

No database connection, migration runner, SQL apply command, or destructive statement was invoked during this work. No protected manifest tables or acceptance fixtures were changed.

## Next bounded step

Continue with a repository-specific RLS and identifier compatibility review, then produce a revised migration draft containing the exact policies and foreign keys. Only after that review should a separate execution authorization be requested for a controlled test database or approved migration environment.

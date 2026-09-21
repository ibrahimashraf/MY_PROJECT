# INTEGIN Stage 0 Work-Order RLS and Identifier Compatibility Decision

**Status:** Candidate reviewed; not executed.  
**Date:** 2026-08-20

## Findings

The existing repository convention uses `tenant_id` and `organization_id` as `TEXT` fields and establishes transaction-local settings named `INTEGIN.tenant_id` and `INTEGIN.organization_id`. Existing protected tables enable and force row-level security and use both settings in their policies.

The first Work-Order draft used UUID identifiers and intentionally omitted policies. That draft is retained as historical design material but is not the executable candidate. The revised candidate uses repository-compatible `TEXT` identifiers, carries both tenant and organization scope on every tenant-owned row, uses composite organization/work-order foreign keys, and adds forced tenant-isolation policies using the established settings.

## Candidate

`migrations/0005_work_order_foundation.candidate.sql` is now the preferred migration candidate for further review. It is still not executable because existing client, location, and asset table definitions must be confirmed before adding those foreign keys, and the deployment role/transaction-context setup must be verified in the target environment.

Static review passed for required entities, composite scope relationships, idempotency uniqueness, exact RLS setting names, forced RLS, transactional framing, and absence of destructive statements. PostgreSQL client tooling is not available in the current workstation, so no parser-level validation was claimed.

## Explicit hold

No SQL was applied. No database connection was opened. The next gate is a human-approved or separately authorized controlled-database syntax and migration test after the referenced canonical tables and runtime role are confirmed. The candidate must not be renamed into an automatically applied migration until that gate passes.

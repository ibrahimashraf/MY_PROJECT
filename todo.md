
- [x] Implement the first pure-Go Stage 0 Work-Order Foundation contract: tenant/client-bound work-order identity, explicit scope items, assignments, idempotent operation metadata, provisional records, and partial-work execution transitions. Package tests and all internal domain tests pass; no migration or runtime behavior changed.

- [x] Implement the reviewed typed Work-Order service contract: actor context, create/assign/transition/partial-submit/reassign/provisional-reconcile/certificate-validation commands, tenant and scope validators, and focused negative tests. Full go test ./... passes; no database migration or runtime authority change performed.

- [x] Design the PostgreSQL persistence and transaction boundary for work orders and assignments: tenant isolation, revisions, idempotency, scope items, partial submissions, reassignment, provisional reconciliation, and certificate separation. Reviewed design is recorded in docs/architecture/INTEGIN_STAGE_0_WORK_ORDER_PERSISTENCE_AND_TRANSACTION_DESIGN_2026-08-20.md; migration execution remains deferred.

- [x] Add repository and transaction contracts for the Work-Order Foundation: authoritative reads/writes, idempotency receipt lookup, optimistic revision conflicts, tenant authorization, partial submission, reassignment, and provisional reconciliation. Focused and full Go tests pass; migration execution remains deferred.

- [x] Draft and independently review additive PostgreSQL migration SQL for work orders, scope items, assignments, idempotency receipts, partial submissions, provisional reconciliation, and audit events. Draft review and rollback-order checks pass; SQL was not executed.

- [x] Review repository-specific organization context, runtime role, identifier types, existing client/location/asset keys, and RLS policy conventions before revising the Work-Order migration candidate. Revised candidate uses TEXT tenant/organization identifiers and exact integin.* settings; no SQL execution performed.

- [x] Verify the target controlled database, non-owner runtime role, connection scope, backup/rollback conditions, and canonical referenced tables before applying the Work-Order migration candidate. Local integin-pilot-postgres was verified, backup created, candidate applied, forced RLS verified on all eight tables, runtime role verified as non-superuser/non-BYPASSRLS, and no production target was touched.

- [x] Run controlled migration evidence on local integin-pilot-postgres: candidate applied successfully, all eight Work-Order tables have enabled/forced RLS, runtime role has no superuser/BYPASSRLS privileges, matching tenant saw one fixture row, cross-scope query saw zero rows, and synthetic fixtures were removed.

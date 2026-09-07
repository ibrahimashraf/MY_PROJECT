# INTEGIN Work-Order Field Package and Form Contract — Design Only

> **Status:** Design-only, approved 2026-09-01. Authorizes no migration apply, route mount, runtime exercise, package-enforcement change, OIDC/OpenBao change, or private-material access. Based on Ticket 02 `a0a8360`/`eab16ab` (`application_service.go:23`), `INTEGIN_POST_MANIFEST_WORK_ORDER_FOUNDATION_SPEC_2026-08-22.md` and `TICKET_01_DECISION_RECORD_2026-08-22.md` D1-D8.

## 1. Tenant / Organization Isolation and RLS

Tenant and organization are server-derived, never client-authoritative (`FOUNDATION_SPEC:12-14,32`; `DECISION_RECORD D2:37`). Every mutation derives `tenant_id`/`organization_id` from server context.

- Tables `work_order*` in `migrations/0009_work_order_persistence.sql:4-254` all carry `(tenant_id, organization_id)` and composite FKs.
- RLS `ENABLE/FORCE` on all 10 tables `migrations/0010_work_order_rls.sql:17-144` with policy `tenant_id = current_setting('integin.tenant_id') AND organization_id = current_setting('integin.organization_id')`. `integin_runtime` has no `BYPASSRLS`; only `SELECT,INSERT,UPDATE,DELETE` (`0010:5-15`).
- Every Repository method: `BEGIN` then `SELECT set_config('integin.tenant_id',$1,true), set_config('integin.organization_id',$2,true)` from `ActorContext` in `internal/workorderpg/postgres.go:45-61`. `TransactionRunner.WithinTransaction` delegates (`transaction_runner.go:9-16`).
- All reads use explicit `WHERE id=$1 AND tenant_id=$2 AND organization_id=$3` (`postgres.go:551-583`).
- HTTP contract `internal/workorderhttp/http.go:28-35` exposes only `operation_id`, `idempotency_key`, `expected_revision`, `work_order_id`, `assignment_id`, `inspection_ids`; `DisallowUnknownFields:68-77` rejects `tenant_id` etc (`http_authority_rejection_test.go:13-30` → `400`).

## 2. Authority — Server-Derived ActorContext

Chain: `OIDC Principal → identity.PostgresResolver → Membership → workorderauth.ActorFromMembership → ActorContext → Authorizer → Service`.

- `identity.Resolver.Resolve` (`identity/contracts.go:31-33`) returns one `Membership` or `ErrUnknownSubject`.
- `PostgresResolver` (`identity/postgres.go:14-30`) calls `integin_resolve_identity_membership($1,$2)` (`migrations/0004_identity_subject_membership.sql:34-50`); runtime has only `GRANT EXECUTE` on function (`0004:52-55`).
- `workorderauth.ActorFromMembership` (`actor_projection.go:13-25`) validates `ActorContext`.
- `workorderauth.Authorizer` (`authorizer.go:22-84`) enforces role + tenant/org binding; `CanSubmitPartial:38-46` checks `assignment.InspectorID == actor.ActorID`.
- Service validates commands then authorizer before mutation, with idempotency guard `FindOperationReceipt` before `WithinTransaction` (`application_service.go:23-27`).

No senior self-issue exception in this foundation (`DECISION_RECORD D2:47`).

## 3. Entitlement — Assigned Asset Scope and Bounded Offline Cache

- `WorkOrder` + `ScopeItem` validated (`workorder.go:106-127`); `work_order_assignment` (`0009:48-72`) + `work_order_assignment_scope` (`0009:74-89` PK) is entitlement join; `loadAssignment:571-583`, `AssignScope` verifies `verifyScopeIDs:586-595`.
- `SubmitPartial` (`application_service.go:82-103` + `postgres.go:214-292`) checks order/assignment binding, authorizer, and `InspectionMembershipValidator`.
- Per `DECISION_RECORD D3:49-54`, device retains only currently assigned work-order scope, immutable `packageHash`, bounded drafts, signed pending submissions. One active work-order per device (`D3:53`). Server validates `tenant/org/actor/capability/package/ordering/payload-hash/current-state` before accept/hold (`FOUNDATION_SPEC:37`).

## 4. Versioning — Composite Vector

| Dimension | Source | File:Line |
|---|---|---|
| `packageHash` | `sha256` canonical package JSON | `workpackages/approved_work_package_cache.dart:58`, `postgres.go:632` |
| `assignmentRevision` | `work_order_assignment.revision` | `postgres.go:606` `bumpOrder` |
| `workOrderRevision` | `work_order.revision` monotonic | `postgres.go:606` |
| `execution_state` | `ExecutionReady/Assigned/InProgress/PartiallySubmitted` | `workorder.go:21` |

Rules:
1. `AssignScope` requires `ExecutionReady:22` and `ExpectedRevision == order.Revision:147` → `ExecutionAssigned:166`.
2. `TransitionExecution` validates `From == order.ExecutionState:138` + `CanTransition:141` (`workorder.go:210`).
3. `SubmitPartial` requires `CanTransition(...PartiallySubmitted):154` and `ExpectedRevision == order.Revision:241`.
4. Stale `ExpectedRevision` → `ErrStaleRevision:19` `ReceiptConflict`.

Field cache `bindingKey = inspectionId\0packageHash:30` (`approved_work_package_cache.dart:30`) — new package creates new binding, never overwrites historic hash.

## 5. Offline Reconciliation — Receipt-Driven, No Silent Deletion

### Field Draft & Held
- Drafts append-only `PersistentOutbox`; `ApprovedWorkPackageCache.isUsableAt:47` gates rendering.
- Offline creates `ProvisionalRecord{LocalID, TenantID, WorkOrderID, ReconcileState=pending:183}` (`workorder.go:171`).
- Held invariants `syncstate/postgres.go:235`: `SaveReceipt` allows only `HELD→APPLIED:235`, `SaveHeld:284` atomic `INSERT receipt + held:312`, `SaveReceipt(APPLIED):250` atomic `UPDATE last_accepted_sequence +1:250` + `DELETE held:261`.
- `ListHeld:319` ordered by `expected_sequence`.

### Provisional Reconciliation
- `ReconcileProvisionalCommand:62` (`service_contract.go:62`) validated `:199`; repo `postgres.go:361` fingerprints via `hashPayload:377`, `ValidateProvisionalFingerprint:389` → `ErrProvisionalConflict:77`. Device `canonical_id` never trusted (`:397`).

### Client Flow
1. Flush outbox in `expected_sequence` order → `accepted|held|conflict|rejected`.
2. `held`: retain draft, retry via `ListHeld`.
3. `conflict` (`ErrStaleRevision`): `GetWorkOrder`/`GetAssignment` for new revision, rebase draft, new `OperationMeta`, retry — never auto-merge.
4. No silent deletion: `loadAll(includeExpired:true)` + `ListHeld` remain readable.

All mutations through `applicationService.execute:18` → `FindOperationReceipt:23` (`postgres.go:81` `findReceipt:515` `FOR UPDATE:518`) → `startMutation:492` checks `storedHash != requestHash → ErrIdempotencyMismatch:20` else returns existing receipt `found=true:510`; `finishMutation:532` persists `work_order_operation` `request_hash:539`. Replay returns identical `MutationReceipt` without re-executing.

## 6. Migration Order (Design-Only, No Apply)

| Seq | File | Scope |
|---|---|---|
| 0008 | `migrations/0008_identity_actor_alignment.sql:4` | `identity_actor` projection; `integin_resolve_identity_membership:61` |
| 0009 | `migrations/0009_work_order_persistence.sql:4` | `work_order`, `work_order_scope_item`, `work_order_assignment`, `inspection_record`, `work_order_submission_segment/_item`, `work_order_operation`, `work_order_state_event`, `work_order_provisional_record` |
| 0010 | `migrations/0010_work_order_rls.sql:17` | `ENABLE/FORCE RLS` + `GRANT` to `integin_runtime:12` |
| D7-4 | `migrations/0011_work_order_signed_submission.sql` *(planned)* | `payload_hash`, `signature`, receipt `JSONB` |
| D7-5 | `migrations/0012_work_order_handover.sql` *(planned)* | `handover_state` |
| D7-6 | `migrations/0013_work_order_evidence.sql` *(planned)* | evidence reference `content_hash` |
| D7-7 | `migrations/0014_work_order_indexes_retention.sql` *(planned)* | indexes + `retention_expires_at` |

Down migrations reverse 0014→0008. Disposable isolated apply only on `integin_repo_test@15432` (`createdb → psql -f → dropdb`), never pilot.

## 7. Test Seam — Disposable Server-Side Mutation Contract

- **Server-derived ActorContext:** `oidcauth.Principal` → `identity.PostgresResolver` (`http_postgres_integration_test.go:38` rejects `tenant_id` body `400:93`).
- **Bounded fixtures:** nano-id per test `fmt.Sprintf("it-...-%d", UnixNano):40,54,122,176`, minimal graph via Tx `set_config:94,222`.
- **Proofs:** `postgres_integration_test.go:21` idempotency/RLS, `:108` cross-org isolation, `:161` full flow (fail-closed `ErrInspectionMembershipInvalid:236`).
- **Cleanup:** `cleanupWorkOrderFixture:294` deletes `state_event → operation → submission_item → submission_segment → inspection_record → assignment_scope → assignment → scope_item → work_order:306`; `cleanupRuntimeMembership:187`, `assertHTTPRuntimeFixtureCleanup:329` asserts `count==0`.

Usage:
```go
INTEGIN_TEST_DATABASE_URL=postgres://integin_repo_test@127.0.0.1:15432/... INTEGIN_TEST_FIXTURE_DATABASE_URL=...
go test -run TestCreateRequestIdempotencyAndRLSIntegration -count=1 ./internal/workorderpg
go test -run TestAuthenticatedPartialSubmissionHTTPPostgresIntegration -count=1 ./internal/workorderhttp
```

No test may write `tenant_id`/`actor_id` from client; all mutations via `workorder.Service` with server-resolved `ActorContext`.

## 8. Next Ticket Gate

This design authorizes Ticket 03+ implementation slices via `integin-tdd` disposable + `integin-code-review` before commit. No code in this commit.

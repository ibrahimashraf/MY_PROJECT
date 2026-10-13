# INTEGIN Stage 0 Work-Order Persistence and Transaction Design

**Status:** Proposed schema and transaction design; migration execution is explicitly deferred.  
**Date:** 2026-08-20  
**Repository boundary:** `integin-pilot-source`  
**Authority owner:** Server-side work-order service.

## Direct decision

The first persistence slice must preserve the domain contract without making the Field cache or transport handler authoritative. The database key used by the current repository for tenant isolation is `organization_id`; the domain-level `tenant_id` maps to that database boundary until a separate naming decision is approved. Every work-order, assignment, scope, operation, submission, and provisional-reconciliation row must carry the same organization scope either directly or through an RLS-safe parent relationship.

No migration should be generated or executed until this design is independently reviewed against the existing RLS, transaction, and migration conventions.

## 1. Proposed tables

| Table | Purpose | Critical controls |
|---|---|---|
| `work_order` | Canonical operational container and human job number. | `organization_id`, immutable ID, unique job number within organization, independent lifecycle states, revision, audit timestamps. |
| `work_order_scope_item` | Explicit client/location/asset target inside an order. | Composite organization/work-order relationships, stable asset reference, no cross-client scope. |
| `work_order_assignment` | Time-bounded assignment to an inspector or team. | Organization/work-order scope, assignment revision, effective period, lifecycle state. |
| `work_order_assignment_scope` | Subset of scope items granted to an assignment. | Composite foreign keys prevent assignment from escaping its work order. |
| `work_order_operation` | Idempotency and operation receipt ledger. | Unique organization/idempotency key, operation hash, status, expected revision, resulting revision, conflict code. |
| `work_order_submission_segment` | Partial submission across one or more inspection records. | Assignment link, immutable segment ID, submitted inspection references, server revision, separate from order closure. |
| `work_order_provisional_record` | Offline client/work-order/location/asset/inspection candidate. | Local ID scoped to organization/device, reconciliation status, canonical ID only after match/create decision. |
| `work_order_state_event` | Audit trail for lifecycle and assignment transitions. | Organization scope, actor, previous/new state, reason, correlation ID, append-only semantics. |

Inspection answers, evidence, certificate requests, and certificate issuance remain separate domains. They may reference work-order and scope IDs, but they must not be folded into the first work-order migration merely for convenience.

## 2. Identity and key strategy

Canonical IDs should be generated server-side and remain immutable. Human job numbers are presentation/business identifiers and must not be used as the sole foreign-key identity. A job number should be unique within `organization_id`, not globally, unless a later integration contract requires a global number.

Every child table should include `organization_id` and use composite foreign keys where practical. This makes accidental cross-organization joins fail at the database layer instead of relying only on application predicates.

The proposed common fields are:

| Field family | Requirement |
|---|---|
| Identity | UUID or repository-approved canonical ID, never client-generated as authoritative identity. |
| Organization scope | `organization_id` on every tenant-owned row. |
| Revision | Positive monotonic `revision` on mutable aggregates and `expected_revision` on commands. |
| Audit | `created_at`, `updated_at`, `created_by`, `updated_by` where appropriate. |
| Correlation | Request/correlation ID and operation ID for support and reconciliation. |
| Soft lifecycle | State fields and append-only state events instead of destructive deletion. |

## 3. Work-order table shape

The work-order row should contain the server identity, organization/client reference, job number, request state, execution state, commercial state, certificate state, current revision, and timestamps. The four lifecycle families remain separate because an order can be operationally partially submitted while waiting for a client and still be commercially not ready for invoicing.

The database must enforce positive revision and the service must update the row with an optimistic-concurrency predicate such as `WHERE organization_id = $1 AND id = $2 AND revision = $3`. A successful transition increments the revision exactly once and records a state event in the same transaction.

## 4. Scope-item integrity

A scope item must carry organization, work-order, client, location, asset, and asset-type references. The service must resolve the asset’s authoritative organization and current client relationship before attaching it. A client-provided asset ID must never be accepted solely because it exists.

The recommended constraints are:

- the scope item’s organization must equal the work order’s organization;
- the scope item’s client must equal the work order’s client;
- the assignment-scope row must reference a scope item from the same work order;
- retired or transferred assets must be rejected or routed to review according to server policy; and
- a provisional asset candidate must remain separate until reconciliation returns `matched` or `created`.

## 5. Assignment and reassignment transactions

Creating or changing an assignment must occur in one transaction that locks the work-order revision, validates the actor and organization, validates the scope-item subset, closes or transfers the previous assignment where required, creates the replacement assignment, writes the assignment-scope rows, increments the work-order revision, and appends a state event.

A reassignment must never mutate client ownership or asset ownership. It changes operational responsibility only. Completed operations from the previous inspector remain immutable and are reconciled separately.

## 6. Idempotency ledger

The operation ledger is required before accepting offline retries. The unique key should be scoped to organization and idempotency key. The ledger stores a request fingerprint/hash so that reusing the same idempotency key with different command content is rejected rather than treated as a new operation.

The transaction pattern is:

1. Begin a transaction under the authenticated organization context.
2. Insert the operation key with a unique constraint.
3. If the key already exists, compare the request hash and return the stored receipt only when the hash matches.
4. Lock and validate the target aggregate revision.
5. Apply the mutation and append its state event.
6. Store the authoritative receipt and resulting revision.
7. Commit atomically.

A retry after a committed transaction receives the original receipt. A retry after a rolled-back transaction may be attempted again with the same idempotency key, provided no committed ledger row exists.

## 7. Partial submission transaction

A partial submission should create one immutable submission-segment row and associate selected inspection IDs after validating that every inspection belongs to the organization, work order, assignment scope, and permitted state. The transaction must reject already-finalized or incompatible duplicate inclusion.

The work order may transition to `partially_submitted`, but this does not close the order, release an invoice, issue a certificate, or mark remaining items complete. Each inspector may receive a personal completion/timesheet summary while the parent work order remains open.

## 8. Provisional reconciliation transaction

Reconciliation must be explicit and outcome-based. For `matched`, the transaction links the provisional record to an existing canonical record after a server identity decision. For `created`, it creates a canonical record only if the actor/policy allows it. For `conflict` or `rejected`, it preserves the provisional row and local evidence references while blocking canonical issuance as required.

A provisional row cannot be changed to `matched` or `created` merely by sending a canonical ID from the device. The server must perform the match/create decision inside the authenticated organization transaction and record the actor, reason, candidate fingerprint, canonical ID if any, and resulting state.

## 9. RLS and transaction context

The repository’s established runtime pattern must set the authenticated organization context before tenant-owned queries. The work-order repository must use a non-owner runtime role, establish the transaction-local organization context, perform all reads and writes inside that transaction, and clear or end the transaction before reuse.

Application predicates remain necessary for scope and assignment checks, but RLS is the database backstop. Tests must demonstrate that a valid ID from another organization cannot be read, updated, assigned, or included in a partial submission.

## 10. Migration boundary

This document does not authorize SQL generation or execution. Before migration, the owner must review:

- naming consistency with existing migrations;
- UUID/ID type compatibility;
- organization context and RLS policy syntax;
- composite foreign-key feasibility;
- index and uniqueness behavior under concurrent retries;
- rollback strategy and non-destructive handling of existing data; and
- test fixtures that prove cross-organization denial and idempotent retry behavior.

The first migration should be additive and should not alter protected manifest tables or acceptance fixtures.

## 11. Next safe implementation gate

After independent review, the next implementation should add repository interfaces and transaction-shaped tests using a controlled database fixture or test double. Only after those tests demonstrate the intended locking, idempotency, and organization-scope behavior should the SQL migration be generated for review. Migration execution remains a separate, explicitly authorized step.

**No migration, database mutation, runtime authority change, or external integration is claimed by this design.**

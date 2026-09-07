# INTEGIN Post-Manifest Work-Order Foundation — Ticket Draft

> **Status:** Draft for owner approval. These tickets are not published to a tracker and authorize no implementation, migration, runtime action, route mount, package-enforcement change, OIDC/OpenBao change, private-material access, Git action, deployment, or customer-data processing.

**Source specification:** `docs\architecture\INTEGIN_POST_MANIFEST_WORK_ORDER_FOUNDATION_SPEC_2026-08-22.md`

**Option A reconciliation:** The work-order coordination view maps onto INTEGIN’s existing `RequestState`, `ExecutionState`, `AssignmentState`, `CommercialState`, and `CertificateState` axes. No second aggregate lifecycle may be introduced by these tickets.

**Drafting rule:** Each ticket is a bounded, verifiable slice. The blocker graph lists only true gates. Ticket 01 is a contract-resolution prerequisite because the approved specification intentionally leaves state names, authorization, package contract, handover, evidence interaction, offline limits, and migration decomposition unresolved.

## Proposed dependency graph

```text
01 Contract and policy resolution
├── 02 Server-authoritative work-order create/assign
│   └── 03 Asset/session progress and partial completion
│       └── 04 Signed offline submission, replay, and conflict handling
│           └── 05 Handover and reconciliation receipts
└── 06 Evidence references and audit trace

02 + 03 + 04 + 05 + 06
└── 07 Disposable integration matrix, migration evidence, and cleanup proof
```

## 01 — Resolve the work-order contract and policy boundaries

**What to build:** A durable, owner-confirmed contract for work-order states, transition rules, authorization, package-bound data, offline limits, handover semantics, evidence interaction, and migration decomposition. This ticket produces design artifacts and non-secret test vectors; it does not implement runtime behavior.

**Blocked by:** None — can start immediately.

**Acceptance criteria:**

- [ ] Final or explicitly deferred work-order state names and transition rules are recorded.
- [ ] The authorization matrix, including any senior self-issue exception, is explicit or explicitly deferred with an owner and follow-up gate.
- [ ] Maximum offline scope, local storage/evidence limits, package-bound data, handover policy, and evidence-retention interaction are explicit or explicitly deferred.
- [ ] The server-authoritative mutation boundary and highest safe public testing seam are restated as an implementation contract.
- [ ] Migration decomposition is sequenced without applying a migration.
- [ ] Negative, replay, isolation, conflict, and rollback test vectors are listed.
- [ ] The owner confirms that the unresolved items are either resolved or intentionally deferred before Ticket 02 begins.

**Required verification:** Read-only review against the approved specification, current state, operating model, RLS/identifier records, and migration guidance.

**Out of scope:** Code, schema changes, migrations, route registration, runtime changes, tracker publication, and private material.

## 02 — Add the server-authoritative work-order create and assign slice

**What to build:** A source-only vertical slice that verifies and, only where necessary, hardens the existing server-side create-request and assign-scope seams in `internal/domain/workorder`, `internal/workorderauth`, `internal/workorderpg`, and `internal/server/workorder_composition.go`. The slice must preserve the existing orthogonal state axes rather than introducing a second lifecycle, derive authority from server context, and return a durable auditable result through the approved isolated mutation seam.

**Blocked by:** 01 — Resolve the work-order contract and policy boundaries.

**Acceptance criteria:**

- [ ] Existing `RequestState`, `ExecutionState`, and `AssignmentState` behavior is mapped to the approved coordination view without adding a duplicate persisted state machine.
- [ ] Create-request and assign-scope behavior follows the approved state-axis and authorization contract.
- [ ] Tenant, organization, actor, capability, assignment, and applicable policy are derived server-side rather than accepted as client authority.
- [ ] The mutation rejects missing, mismatched, expired, or unauthorized context without creating partial authoritative state.
- [ ] The work order records only the approved service-container responsibilities and does not issue certificates or override inspection truth.
- [ ] The approved isolated disposable database seam has focused positive and negative tests.
- [ ] Cross-tenant enumeration and context leakage are rejected or unavailable at the tested seam.
- [ ] No route is mounted and no persistent runtime is changed.

**Required verification:** Focused behavior tests for the existing service and authorizer seams, tenant/org negative tests, repository/RLS contract review, confirmation that no duplicate lifecycle is introduced, and independent read-only review before commit consideration.

**Out of scope:** Offline submission, handover, evidence export, certificate issuance, package-enforcement activation, migration application, and protected runtimes.

## 03 — Add asset/session progress and explicit partial completion

**What to build:** A vertical slice that associates approved assets and independent inspection sessions with a work order, records per-asset progress, and supports explicit partial completion without collapsing job, session, and inspection lifecycles.

**Blocked by:** 02 — Add the server-authoritative work-order create and assign slice.

**Acceptance criteria:**

- [ ] An assigned work order can reference approved assets without allowing one asset outcome to stand in for another.
- [ ] Job, session, and asset-inspection states remain distinct and are independently queryable through the approved seam.
- [ ] Partial completion is explicit and does not imply final inspection approval, certificate eligibility, or commercial release.
- [ ] Unauthorized, out-of-scope, duplicate, and invalid-transition attempts are rejected deterministically.
- [ ] The isolated test seam proves both a valid multi-asset partial path and a negative cross-asset/context path.
- [ ] No protected runtime, route, migration, package enforcement, OIDC, or OpenBao state changes.

**Required verification:** Focused behavior and negative tests plus independent review of state separation and tenant scope.

**Out of scope:** Offline reconciliation, handover transfer, evidence export, certificate state, and migration execution.

## 04 — Add signed offline submission, replay, and conflict handling

**What to build:** A vertical slice for bounded offline drafts and signed idempotent submissions against the assigned work-order/package context, with deterministic duplicate, stale, unauthorized, scope-mismatch, payload-hash, held, rejected, and security-failure outcomes.

**Blocked by:** 03 — Add asset/session progress and explicit partial completion.

**Acceptance criteria:**

- [ ] Offline data is limited to the approved work-order/package scope and storage/evidence limits.
- [ ] The server derives and validates tenant, organization, actor, capability, authority, package, ordering, payload hash, and current state.
- [ ] Replaying the same valid transaction produces the approved deterministic duplicate outcome.
- [ ] Reusing an identifier with a different payload hash is rejected or held according to the approved contract.
- [ ] Stale, unauthorized, scope-mismatched, malformed, and invalid-transition submissions produce bounded auditable outcomes.
- [ ] Accepted, duplicate, held, rejected, conflict, and security-failure receipts do not silently delete drafts or revise prior evidence.
- [ ] Isolation and replay tests run only against the approved isolated disposable database seam.

**Required verification:** Focused acceptance and negative tests, replay/idempotency tests, RLS/tenant isolation tests, and independent review.

**Out of scope:** Applying migrations to pilot or acceptance, enabling package enforcement, production transport, candidate execution, and private material.

## 05 — Add controlled handover and reconciliation receipts

**What to build:** A vertical slice for explicit handover and server-side reconciliation, recording actor, reason, scope, receipt, and audit outcome while preserving affected offline drafts and independent inspection facts.

**Blocked by:** 04 — Add signed offline submission, replay, and conflict handling.

**Acceptance criteria:**

- [ ] Handover is an explicit governed action with approved actor, reason, scope, and transition checks.
- [ ] Handover cannot silently transfer responsibility, erase drafts, alter prior evidence, or approve a certificate.
- [ ] Reconciliation records accepted, duplicate, held, rejected, conflict, and security-failure outcomes through the approved receipt contract.
- [ ] Partial completion and handover remain distinct from review, approval, eligibility, certification, and commercial release.
- [ ] Negative tests cover unauthorized handover, stale handover, scope mismatch, duplicate handover, and affected-draft preservation.
- [ ] The test seam remains isolated, disposable, server-authoritative, and non-customer-facing.

**Required verification:** Focused lifecycle and negative tests, receipt/audit review, cleanup check, and independent review.

**Out of scope:** Automatic renewal, certificate issuance, external authority submission, invoicing, deployment, and runtime activation.

## 06 — Add controlled evidence references and immutable audit trace

**What to build:** A vertical slice that attaches only approved tenant-scoped evidence metadata and controlled object references to the appropriate work-order or inspection context, while emitting an immutable audit trace without making evidence an authority decision.

**Blocked by:** 02 — Add the server-authoritative work-order create and assign slice.

**Acceptance criteria:**

- [ ] Evidence metadata and object references are tenant-scoped and cannot be used to select authority, eligibility, review outcome, or certificate state.
- [ ] Unauthorized, cross-tenant, malformed, stale, and out-of-scope evidence references are rejected deterministically.
- [ ] Audit records identify the relevant work-order, inspection, evidence reference, actor/context, and state transition without private payload leakage.
- [ ] Prior evidence and audit records are not silently rewritten or deleted by work-order changes.
- [ ] Retrieval/export behavior remains separately governed and is not implied by this slice.
- [ ] Focused isolation, redaction, and negative tests pass at the approved seam.

**Required verification:** Focused evidence-reference and audit tests, redaction review, tenant-isolation review, and independent review.

**Out of scope:** Evidence upload/export implementation, certificate issuance, public verification, customer data processing, and private material.

## 07 — Prove the disposable integration matrix and migration/cleanup evidence

**What to build:** A separately authorized integration-verification slice that exercises the approved work-order mutation path against a disposable isolated database, proves positive and negative behavior, and records migration apply/cleanup evidence only after the migration design and owner gates are separately approved.

**Blocked by:** 02, 03, 04, 05, and 06 — all foundation behavior slices and their focused evidence.

**Acceptance criteria:**

- [ ] The isolated database, fixtures, roles, tenant/org context, and cleanup boundary are explicitly approved before execution.
- [ ] The matrix covers create, assign, partial completion, signed submission, replay, conflict/hold/reject, handover, reconciliation, evidence reference, and audit outcomes.
- [ ] Tenant/org isolation is proven under the approved non-owner/RLS test conditions without bypassing RLS.
- [ ] Migration apply, rollback or cleanup, fixture ownership, and zero-residue evidence are recorded only under a separate migration authorization.
- [ ] Acceptance at `127.0.0.1:8080` and the candidate boundary at `127.0.0.1:18080` are not used as the foundation test seam.
- [ ] The final evidence record distinguishes source behavior, isolated integration behavior, and any unproven runtime claim.
- [ ] Independent review finds no unresolved high-severity blocker before any commit or later governance request.

**Required verification:** Approved disposable integration matrix, negative and rollback evidence, cleanup proof, and independent review.

**Out of scope:** Persistent runtime activation, protected acceptance changes, candidate launch, package enforcement, OIDC/OpenBao changes, deployment, publication, tracker mutation, and customer-facing release.

## Owner approval questions

Before these tickets are published or implementation begins, confirm:

1. Is the seven-ticket granularity correct, or should any ticket be merged or split?
2. Is Ticket 01 the right prerequisite for resolving or explicitly deferring the approved unresolved decisions?
3. Are the blocker edges minimal and correct?
4. Should Ticket 06 be blocked by Ticket 03 as well as Ticket 02 because evidence may attach to asset/session progress?
5. Is the proposed isolated server-side mutation contract the correct testing seam?
6. Should Ticket 07 remain one integration-evidence ticket or be split into migration proof and behavioral matrix proof?
7. Are the scope exclusions and governance holds complete?

No ticket is approved for publication or implementation until these questions are answered.

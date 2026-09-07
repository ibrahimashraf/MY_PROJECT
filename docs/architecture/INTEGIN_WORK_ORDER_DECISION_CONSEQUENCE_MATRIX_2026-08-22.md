# INTEGIN Work-Order Decision Consequence Matrix — 2026-08-22

This matrix explains what changes if each Ticket 01 decision is resolved, deferred, or revised. It is an owner-decision aid; it does not authorize implementation, migration, runtime action, tracker publication, or a change to protected boundaries.

## Decision consequences

| Decision | If resolved now | If deferred | If revised later | Main consequence area |
|---|---|---|---|---|
| **D1 — Work-order states and transitions** | Later tickets can implement a stable lifecycle, reject invalid transitions consistently, and write meaningful tests and audit records. | Every later state-changing ticket risks inventing incompatible names or transitions. Ticket 02 may create data that Ticket 03–05 cannot interpret consistently. | Existing code, tickets, tests, and data contracts may need migration or compatibility handling. | Product behavior, data integrity, auditability, schedule. |
| **D2 — Authorization matrix** | Server-side mutations can enforce who may create, assign, hand over, reconcile, review, or close. Negative tests have a clear authority oracle. | Developers may accidentally encode permissive or contradictory authority rules. Security review and implementation must stop at each unclear action. | Authorization checks, audit events, tests, and possibly data ownership rules may need redesign. | Security, tenant isolation, governance, legal/assurance risk. |
| **D3 — Offline scope and local retention** | Field behavior, storage limits, expiry, synchronization, and cleanup can be designed without guessing. | The offline slice cannot safely be completed. A client might retain too much data, evidence, or stale authority, or behave inconsistently after expiry. | Local schemas, cleanup behavior, sync payloads, and user-facing offline states may need rework. | Privacy, data volume, Field usability, sync reliability. |
| **D4 — Package-bound data contract** | The client/server boundary becomes testable: version, hash, assignment context, authority context, and asset mappings have stable meaning. | Field work cannot safely bind to a package. Developers may create client-fabricated context or later break compatibility with the manifest work. | Manifest/cache formats, verification vectors, persistence, and client/server contracts may need compatibility migration. | Authority, integrity, offline correctness, cross-language compatibility. |
| **D5 — Handover policy** | Handover can be implemented as an explicit governed action that preserves drafts and records responsibility. | Handover-related tickets must remain blocked. Teams may not know who owns partial or conflicted work, and operational accountability remains incomplete. | Transition rules, permissions, receipts, user flows, and audit history may need changes. | Operations, accountability, conflict resolution, auditability. |
| **D6 — Evidence interaction** | Evidence references can be attached safely without making evidence acceptance an accidental authority or certificate decision. Audit and redaction tests can be precise. | Ticket 06 must be narrowed or blocked. Teams may create references that later conflict with retention, export, or evidence-acceptance policy. | Evidence metadata, object references, retention, export, and audit contracts may need migration or compatibility work. | Assurance, privacy, storage, certificate/evidence separation. |
| **D7 — Migration decomposition** | Future schema work has an ordered, reviewable dependency plan with RLS/identifier review and cleanup evidence. | No migration can be applied, which is safe, but implementation tickets cannot truthfully promise persistent storage or integration proof. | A changed decomposition can reorder tickets, split integration work, or require expand/migrate/contract compatibility steps. | Data safety, rollback, schedule, operational risk. |
| **D8 — Highest safe testing seam** | Source behavior can be tested at a server-side mutation contract against a disposable database, without touching acceptance or the pilot runtime. | Implementation may be blocked because there is no approved oracle or safe place to prove tenant/RLS behavior. | Tests, fixture setup, repository boundaries, and evidence claims may need redesign. | Evidence quality, tenant safety, reproducibility, confidence. |

## More detailed consequences by decision

### D1 — State model

Resolving D1 now establishes the vocabulary that every later ticket, test, audit event, and UI state will use. It prevents a “closed” work order from being confused with a completed inspection, certificate eligibility, evidence acceptance, or commercial release. The cost is that the owner must make a policy choice now, even if some state names are provisional.

Deferring D1 is safe only if Ticket 01 records exactly who will decide it, by when, and which later tickets remain blocked. Deferring it while allowing implementation creates the highest risk of incompatible state machines and silent transitions. Revising it later is possible, but the change may affect persisted values, API contracts, event history, test fixtures, and migration strategy.

**Recommended posture:** resolve the minimum state and transition contract now; defer only states outside the first foundation slice.

### D2 — Authorization matrix

Resolving D2 determines the security oracle for every mutation. It clarifies whether an actor may create, assign, reschedule, hand over, reconcile, review, or close, and ensures the server derives authority rather than trusting client-provided fields. It also makes negative tests meaningful.

Deferring D2 is not equivalent to allowing a permissive default. It means no authority-bearing implementation may proceed for the affected action. Revising the matrix later can be expensive because authorization is cross-cutting: it affects handlers, repositories, audit records, UI affordances, tests, and tenant-boundary reasoning.

**Recommended posture:** resolve all actions needed by Tickets 02–05 before implementation; defer unrelated roles explicitly.

### D3 — Offline scope and retention

Resolving D3 sets the boundary for what the device may retain and submit. A narrower scope reduces privacy and stale-authority risk but may require more server round trips or prevent broad offline scheduling. A broader scope improves field continuity but increases exposure, synchronization complexity, and cleanup obligations.

Deferring D3 can be acceptable for server-only Tickets 01–02, but it must block the offline portion of Ticket 04. It is unsafe to implement local storage, evidence retention, or expiry behavior using guessed limits. Revising D3 later may require client schema changes, cache invalidation, sync compatibility, and user communication.

**Recommended posture:** resolve the bounded classes of data and expiry behavior before any offline implementation; numeric limits can be a separate explicitly gated decision if necessary.

### D4 — Package-bound contract

Resolving D4 preserves continuity with the completed manifest-delivery work. The work-order foundation can then rely on immutable package/version/hash/context rules rather than allowing the client to fabricate authority or scope. This is the strongest dependency for safe Field integration.

Deferring D4 allows only package-neutral server design. It blocks any client binding, offline submission, or package-scoped acceptance behavior. Revising it later is costly because it can invalidate canonicalization, signatures, cache records, proof vectors, and cross-language compatibility.

**Recommended posture:** resolve the minimum package-bound input and divergence contract before Ticket 04.

### D5 — Handover policy

Resolving D5 makes responsibility transfer explicit and auditable. It determines whether drafts survive a handover, whether a conflicted work order can move, and who can accept or reverse the transfer. This protects against silent loss of operational accountability.

Deferring D5 leaves work-order assignment and partial progress possible only if no handover behavior is claimed. Ticket 05 must remain blocked. Revising it later may require transition, permission, receipt, and UI changes but is less dangerous than silently inventing a policy.

**Recommended posture:** resolve before Ticket 05; it need not block the initial create/assign slice if no handover is included.

### D6 — Evidence interaction

Resolving D6 keeps evidence as a controlled reference and prevents it from becoming an accidental authority, review, certificate, or commercial decision. It also establishes what can safely appear in audit records.

Deferring D6 allows the foundation to proceed without evidence attachment, but Ticket 06 must be blocked or reduced to an interface placeholder. Revising it later may affect object-reference contracts, retention, export, redaction, and audit data.

**Recommended posture:** resolve the attachment and audit boundary before Ticket 06; defer export and retention details to a separate evidence-policy gate if needed.

### D7 — Migration decomposition

Resolving D7 produces a safe sequence for future schema work, not an immediate migration. It makes dependencies, RLS review, identifier review, disposable apply, cleanup, and rollback visible.

Deferring D7 keeps the project safer by preventing premature migration application, but it blocks persistent integration proof and any ticket that requires a durable schema. Revising it later is normal if the approved contract changes; use expand/migrate/contract where compatibility is required.

**Recommended posture:** resolve the conceptual order in Ticket 01, but retain a separate authorization gate for every actual migration action.

### D8 — Testing seam

Approving D8 gives the project a safe evidence boundary: an isolated server-side mutation contract with server-derived context, bounded fixtures, and disposable storage. It avoids claiming proof from protected acceptance or the pilot runtime.

Deferring D8 means implementation has no approved place to prove the foundation’s most important properties. Revising it may be necessary if the seam cannot demonstrate tenant/RLS behavior, but the replacement must be at least as authoritative and isolated.

**Recommended posture:** approve D8 with explicit exclusions and require the seam to prove positive, negative, replay, isolation, cleanup, and rollback claims proportionately.

## Consequence of approving the ticket breakdown

Approving the seven-ticket breakdown permits the team to work through the planning sequence. It does **not** mean all seven tickets are authorized for implementation at once. Ticket 01 remains the gate for the unresolved contract decisions. Ticket 02 and later implementation slices remain blocked until Ticket 01 is complete.

The breakdown’s current dependency shape has one deliberate parallel opportunity: Ticket 06 can begin after Ticket 02 if its evidence-reference boundary does not depend on asset/session progress. If evidence attaches to partial session state, Ticket 06 should also be blocked by Ticket 03. The owner should decide this before publishing or implementing Ticket 06.

## Consequence of proceeding without Ticket 01 completion

Proceeding directly to code would create a high probability of rework and could encode unauthorized assumptions about states, authority, offline retention, package scope, handover, evidence, or migrations. It would also weaken later testing because the expected result would not be independently defined. It is therefore not recommended.

## Consequence of completing Ticket 01

Completing Ticket 01 does not create a work-order feature. It produces a stable decision record, explicit deferrals, test vectors, and a design-only migration sequence. After owner confirmation, Ticket 02 can be considered for a separate source-only implementation slice using TDD.

## Safe recommendation

The safest practical path is:

1. Resolve D1, D2, D4, and D8 before any implementation ticket.
2. Resolve D3 before implementing offline behavior; D5 before handover; D6 before evidence attachment; and D7 before any schema or integration action.
3. Allow explicit deferrals only when they name an owner, follow-up gate, and implementation-blocking condition.
4. Keep all migrations, route mounts, runtime exercises, package enforcement, OIDC, OpenBao, protected acceptance, private material, Git publication, and deployment behind separate approvals.

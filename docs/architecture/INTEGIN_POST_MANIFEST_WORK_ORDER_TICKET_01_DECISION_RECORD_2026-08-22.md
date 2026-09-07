# INTEGIN Ticket 01 — Work-Order Contract and Policy Decision Record

> **Status:** Owner-approved design baseline for Ticket 01, approved 2026-08-22. This record authorizes preparation of Ticket 02 only; it authorizes no code, schema, migration, route, runtime action, tracker publication, Git action, deployment, private-material access, package-enforcement change, OIDC/OpenBao change, or protected-runtime action.

**Parent specification:** `docs\architecture\INTEGIN_POST_MANIFEST_WORK_ORDER_FOUNDATION_SPEC_2026-08-22.md`

**Parent ticket draft:** `docs\architecture\INTEGIN_POST_MANIFEST_WORK_ORDER_TICKET_DRAFT_2026-08-22.md`

## Reconciliation basis

The decision set was checked against the accepted high-volume operating model, integrated operating model, current state, continuation guide, workspace map, and work-order consequence matrix. Architecture, safety, assurance, tenant/data, and verification lenses found no reason to defer the minimum foundation contract.

## Owner approval — 2026-08-22

The owner approved D1–D8: work-order states and transitions as an orthogonal-axis coordination view, the server-derived authorization matrix, conservative offline scope and retention, the package-bound data contract, explicit handover policy, evidence interaction boundaries, conceptual migration decomposition, and the isolated disposable server-side mutation contract as the highest safe testing seam.

This approval closes Ticket 01 as a design gate and unblocks preparation of Ticket 02. It does not authorize implementation, migration, route mounting, runtime action, tracker publication, Git action, deployment, package enforcement, OIDC/OpenBao, or private-material access.

## D1 — Work-order states and transitions: RESOLVE — Option A

Preserve the existing INTEGIN orthogonal state model. The work-order coordination view must not replace or duplicate the existing state axes:

| Existing axis | Existing responsibility |
|---|---|
| `RequestState` | Request/container lifecycle, including draft, requested, accepted, and cancelled. |
| `ExecutionState` | Operational execution lifecycle, including ready, assigned, in progress, partial submission, awaiting review, and completed. |
| `AssignmentState` | Assignment responsibility, including active, transferred, completed, and revoked. |
| `CommercialState` | Commercial/invoice lifecycle, independent of operational completion. |
| `CertificateState` | Certificate validation/issuance lifecycle, independent of operational completion. |

The previously described lifecycle is retained only as a **coordination view** derived from these axes. It must not be persisted as a second authoritative state machine. Mapping rules must be explicit: request authorization maps to the request axis; assignment and handover map to the assignment axis; operational progress and partial completion map to the execution axis; commercial and certificate outcomes remain independent. Invalid transitions must be rejected by the owning axis without partial authoritative mutation.

The existing domain names and behavior are the baseline for Ticket 02. Any proposed new state or transition must first demonstrate that it cannot be represented safely by the relevant existing axis and must receive a new architecture decision.

## D2 — Authorization matrix: RESOLVE

All authoritative mutations derive actor, tenant, organization, capability, assignment, and policy from server context.

| Actor or capability | Foundation actions | Not permitted by this foundation |
|---|---|---|
| Operations coordinator | Create, authorize, assign, reschedule, policy-governed handover, and submit closure for review. | Issue certificates, override inspection truth, bypass RLS, or choose tenant/org context. |
| Assigned field inspector | Open assigned scope, record bounded progress, submit signed work, request handover, and view permitted receipts. | Self-authorize, assign outside scope, approve own review outcome, or alter authority. |
| Authorized reviewer/senior inspector | Review submitted operational facts, resolve permitted operational holds, and accept/reject operational completion under policy. | Approve own work under an unapproved exception, issue certificates, or override tenant/RLS controls. |
| Tenant manager | Manage permitted assignment/asset scope, accept policy-governed handover, resolve controlled operational conflicts, and inspect audit history. | Grant authority through client data, bypass policy, or rewrite prior evidence. |
| Auditor | Read permitted audit and reconciliation records. | Mutate operational, inspection, evidence, authority, or certificate state. |

There is no senior self-issue exception in this foundation. Any future exception requires a separate policy decision, capability, audit event, negative-test set, and owner approval.

## D3 — Offline scope and local retention: RESOLVE

A device may retain only the currently assigned work-order scope, immutable package/version/hash/context, permitted assigned-asset and inspection/session references, bounded local drafts, signed pending submissions, and redacted reconciliation status. It may not retain unassigned work orders, another tenant’s data, unbound authority, certificate decisions, review approvals, or evidence binaries as part of this foundation.

New authoritative submission is blocked or held when assignment, package, authority, or expiry validation fails. Existing drafts are preserved for governed reconciliation and are not silently deleted or rewritten. The first foundation contract supports one active assigned work-order context per local session. Broader multi-order offline scope requires a separate capacity/privacy decision.

## D4 — Package-bound data contract: RESOLVE

A work-order session is bound to immutable work-order and assignment identity/version, package identity/version/canonical hash, server-derived tenant/org scope, server-derived actor/device/authority scope, expiry, assigned asset and inspection/session references, deterministic field-to-asset mappings, package schema version, submission purpose, and replay/idempotency binding.

The server supplies or signs this context. The client may cache and submit it but may not replace tenant, organization, authority, package hash, assignment, asset scope, expiry, or state transitions. Any divergence produces a bounded rejection or hold and preserves the prior draft.

## D5 — Handover policy: RESOLVE

Handover is an explicit governed action with a new audit record. The current assignee may request it; an operations coordinator or tenant manager may authorize it; and the receiving assignee must accept it before the new assignment becomes active. The record includes actor, receiving party, reason, scope, source assignment, destination assignment, and authoritative server time.

Handover may occur during `ASSIGNED`, `IN_PROGRESS`, or `PARTIALLY_COMPLETE`. A held or conflicted submission may be handed over only with the hold preserved and visible. Handover never deletes drafts, rewrites prior evidence, changes inspection truth, approves a certificate, or changes tenant/organization. Reversal is a new governed handover action.

## D6 — Evidence interaction: RESOLVE

The foundation may attach only tenant-scoped evidence metadata and controlled object references to the relevant work-order or inspection context. Evidence acceptance, retrieval, export, retention, deletion, and certificate use remain separate governed flows. Evidence references cannot choose authority, review outcome, eligibility, or commercial release.

Each attachment/reference mutation validates server context and scope, emits an audit event, and rejects cross-tenant, unauthorized, malformed, stale, or out-of-scope references. No binary upload, public export, certificate issuance, or external authority submission is part of this foundation.

## D7 — Migration decomposition: RESOLVE

Resolve the conceptual dependency order but do not apply any migration:

1. Tenant/org identity context, identifier conventions, and RLS policy primitives.
2. Server-authoritative work-order aggregate and audit/event boundary.
3. Assignment, package-bound context, asset links, and session/progress relations.
4. Signed submission, replay/idempotency, receipts, holds, rejection, conflict, and reconciliation relations.
5. Governed handover relations and audit links.
6. Tenant-scoped evidence metadata/reference relations and audit links.
7. Required indexes, retention controls, and integrity checks.
8. Identifier/RLS review under a non-owner role, isolated disposable apply, behavior validation, cleanup/rollback proof, and independent review.

Broad mechanical changes use expand → bounded migration batches → contract. No migration is authorized by this record.

## D8 — Highest safe testing seam: APPROVE

Approve the proposed seam: a server-side work-order mutation contract exercised against an isolated disposable database with server-derived actor context, bounded non-secret fixtures, explicit tenant/org scope, and cleanup proof.

It must eventually cover authorized create/assign, invalid authorization, tenant/org mismatch, RLS visibility, invalid transitions, package/context mismatch, replay/idempotency, partial completion, handover, reconciliation, evidence reference, redaction, and cleanup. It must not use protected acceptance, the pilot candidate, a persistent package runtime, OIDC, OpenBao, customer-facing endpoints, or private material.

## Ticket 01 completion gate

Ticket 01 is complete only after the owner confirms D1–D8, all implementation-blocking assumptions are explicit, any remaining uncertainty has a named follow-up owner and gate, and an independent read-only review confirms that no decision was silently invented.

Owner confirmation of this record authorizes preparation of Ticket 02 only. It does not authorize code, schema, migration, route, runtime, tracker, Git, deployment, package-enforcement, OIDC, OpenBao, or private-material action.

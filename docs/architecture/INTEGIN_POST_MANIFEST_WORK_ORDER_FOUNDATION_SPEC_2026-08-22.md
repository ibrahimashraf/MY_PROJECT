# INTEGIN Post-Manifest Work-Order Foundation Specification — Approved Design Baseline

> **Status:** Owner-approved design-only baseline, approved 2026-08-22. This document authorizes preparation of Ticket 02 only; it authorizes no implementation, migration, route mount, runtime exercise, package-enforcement change, OIDC/OpenBao change, private-material access, Git operation, deployment, publication, or tracker action.

## Problem Statement

The manifest-delivery gate is closed by the authorized isolated-pilot eight-case evidence. The next product design boundary is a work-order foundation that coordinates service delivery without collapsing the independent lifecycles of assets, inspections, packages, authority, evidence, handover, certificates, or commercial/external processes. The foundation must preserve server authority, tenant and organization isolation, offline-first field work, bounded synchronization, partial progress, accountable reconciliation, and auditability.

## Solution

Define a server-authoritative work-order aggregate as the operational and commercial service container. A work order coordinates assignment, package scope, asset selection, field progress, handover, and later eligibility signals; it does not replace the separate state machines of each asset inspection, certificate, authority package, evidence object, or external/commercial outcome.

Every authoritative mutation derives tenant, organization, actor, capability, and applicable policy from the server-side context. The client may submit bounded facts, signed offline transactions, or package-bound actions, but it cannot choose authority, scope, certificate eligibility, review outcome, tenant, organization, or server state transitions.

The foundation remains design-only after approval of this specification. Ticket 02 preparation is unblocked, but implementation, migration, route mounting, runtime action, and tracker publication require their own gates.

## User Stories

| Actor | Need | Governed outcome |
|---|---|---|
| Operations coordinator | Create, authorize, assign, reschedule, hand over, and close a service container. | Work order records operational state without issuing certificates or overriding inspection truth. |
| Field inspector | Open only an assigned, package-scoped work order while offline; record per-asset progress and submit later. | The device keeps bounded drafts and signed submissions; the server decides acceptance and reconciliation. |
| Reviewer or authorized senior inspector | Review completed inspection facts and act under a governed exception where policy permits. | Review, approval, and certificate eligibility remain separate from field completion. |
| Tenant manager | Move assets, resolve controlled conflicts, and review audit history. | Scope changes preserve prior facts, flag affected drafts, and never silently rewrite history. |
| Auditor | Trace a work order, each inspection, evidence reference, handover, receipt, and state transition. | The system exposes immutable audit events and reconciliation outcomes without exposing another tenant. |

## Implementation Decisions

| Boundary | Design decision |
|---|---|
| Work-order responsibility | The work order is a server-authoritative service container. It coordinates assignment, package scope, assets, session progress, partial submission, handover, and later review eligibility; it is not a replacement for inspection, certificate, evidence, authority, or commercial lifecycles. |
| Tenant and organization isolation | Tenant and organization are server-derived, not client-authoritative fields. All persistent access expects explicit tenant/org predicates and enforced RLS in the eventual migration environment. A local owner bypass must not be mistaken for RLS proof. |
| Asset and inspection responsibility | Assets remain independently identified and may have ownership/location changes under controlled administration. An inspection records asset-specific facts and retains its own lifecycle; one asset outcome cannot validate another asset. |
| Package and authority responsibility | A package bounds the field-visible work scope and authority context. Package enforcement remains disabled in persistent runtimes in this design stage; no package runtime activation is proposed. |
| Evidence responsibility | Evidence is tenant-scoped metadata plus controlled object reference. Evidence acceptance, export, and retrieval remain independent governed flows and do not become work-order authority. |
| Handover responsibility | Handover is an explicit controlled state/action with actor, reason, scope, and audit record; it cannot silently transfer responsibility or erase offline drafts. |
| Offline and synchronization boundary | Offline clients may retain only authorized, bounded package/work-order data and local drafts. They submit signed, idempotent transactions when connected. The server validates scope, actor, authority, package, payload hash, ordering, and current state before accepting or holding a submission. |
| Partial completion and conflict | Job, session, and asset-inspection state remain separate. Partial completion is explicit. Duplicate or replayed requests receive deterministic outcomes; stale, unauthorized, scope-mismatched, or state-conflicting submissions are rejected or held with an auditable reason. |
| Reconciliation and audit | Reconciliation is server-side and receipt-driven. It records accepted, duplicate, held, rejected, conflict, and security-failure outcomes without silently deleting drafts or revising prior evidence. |
| Migration order | Do not apply a migration from this specification. Future schema work must follow the accepted dependency order, design contract, explicit RLS/identifier review, isolated migration evidence, cleanup proof, and owner approval. |

## Testing Decisions

The highest safe public testing seam is a future server-side work-order mutation contract exercised only against an isolated disposable database with server-derived actor context and bounded fixtures. It is not protected acceptance at `127.0.0.1:8080`, the pilot candidate at `127.0.0.1:18080`, a persistent package runtime, OIDC, OpenBao, or a customer-facing endpoint. This seam is **owner-approved for future implementation testing**; its use remains limited to a separately approved source-only slice and isolated disposable evidence.

| Test class | Decision |
|---|---|
| Acceptance | Later prove authorized create/assign/partial-submit/handover/reconcile flows with separate job, session, and asset inspection states. |
| Negative | Later prove tenant/org mismatch, actor/capability mismatch, package/scope mismatch, invalid transition, stale state, malformed payload, and unauthorized evidence reference rejection. |
| Replay and idempotency | Later prove same transaction/receipt behavior, payload-hash mismatch rejection, deterministic duplicate outcome, and durable held/rejected result. |
| Isolation | Later prove transaction-local tenant/org context, explicit repository predicates, forced-RLS behavior under a non-owner role, and no cross-tenant enumeration. |
| Offline and conflict | Later prove bounded offline drafts, ordered reconciliation, partial completion, conflict hold/rejection, controlled handover, and no silent deletion of affected drafts. |
| Rollback and cleanup | Only after separately approved migration work: prove isolated apply, rollback/cleanup, fixture ownership, and zero residue. No migration test is authorized by this draft. |
| Independent verification | Before any commit or runtime claim, require the approved test seam, focused checks, relevant negative evidence, and independent review of the scoped change. |

## Out of Scope

This draft does not implement code, schema, routes, database migrations, RLS changes, package enforcement, OIDC, OpenBao, authentication configuration, runtime exercises, protected acceptance work, pilot-candidate operation, evidence export, certificate issuance, public verification changes, external authority submission, invoicing, ZATCA/SASO/SAAC integration, deployment, publishing, tracker work, Git staging/commit/merge, private-material access, or customer-data processing.

## Further Notes

**Confirmed facts.** The manifest hard gate is closed by isolated-pilot eight-case evidence; the accepted operating model is future/design-only; work order is the service container; tenant/org authority must be server-derived; RLS is an eventual enforced persistence expectation; and migrations are separately governed.

**Approved assumptions.** The owner approved the isolated server-side mutation contract as the future testing seam, bounded partial submission and handover as distinct operations, and disposable fixtures rather than protected runtimes for future tests. The D1–D8 decision record is `docs\architecture\INTEGIN_POST_MANIFEST_WORK_ORDER_TICKET_01_DECISION_RECORD_2026-08-22.md`.

**Decision status.** The D1–D8 decisions were resolved and owner-approved on 2026-08-22 in `docs\architecture\INTEGIN_POST_MANIFEST_WORK_ORDER_TICKET_01_DECISION_RECORD_2026-08-22.md`. Any new policy, numeric retention limit, exception, contract field, migration detail, or testing-seam expansion is a new decision and must be separately recorded before implementation.

**Evidence sources.** `CURRENT_STATE.md`; `ENGINEERING_CONTINUATION_GUIDE.md`; `WORKSPACE_MAP.md`; `task_plan.md`; `findings.md`; `progress.md`; `todo.md`; `docs\\continuity\\INTEGIN_ROOT_HANDOFF_RECONCILIATION_MATRIX_2026-08-22.md`; `docs\\architecture\\INTEGIN_HIGH_VOLUME_WORK_ORDER_OPERATING_MODEL_DRAFT_2026-08-19.md`; `docs\\architecture\\INTEGIN_INTEGRATED_OPERATING_MODEL_FINAL_RECONCILIATION_2026-08-19.md`; `docs\\architecture\\INTEGIN_DEVELOPMENT_HIERARCHY_AND_OSS_ADOPTION_2026-08-18.md`; and the Stage 0 service-contract, persistence, RLS/identifier, migration, roadmap, and evidence-ledger records. `docs\\INTEGIN_ROADMAP_AND_GATES.md` was not present and is not referenced.

Owner approval was recorded on 2026-08-22 for this specification, its assumptions, D1–D8 decisions, testing seam, and scope. This approval authorizes preparation of Ticket 02 only; implementation planning, migration, route mount, runtime action, and ticket publication remain separately gated.

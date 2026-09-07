# INTEGIN Ticket 01 — Work-Order Contract and Policy Decision Worksheet

> **Status:** Owner decision worksheet. This document authorizes no code, schema, migration, route, runtime action, package-enforcement change, OIDC/OpenBao change, private-material access, Git action, deployment, publication, or tracker mutation.

**Parent specification:** `docs\architecture\INTEGIN_POST_MANIFEST_WORK_ORDER_FOUNDATION_SPEC_2026-08-22.md`

**Parent ticket draft:** `docs\architecture\INTEGIN_POST_MANIFEST_WORK_ORDER_TICKET_DRAFT_2026-08-22.md`

## Purpose

Ticket 01 must resolve or explicitly defer the decisions that gate all later work-order implementation slices. An explicit deferral must name the follow-up gate and must not be treated as implementation approval.

## Decision register

### D1 — Work-order state model

**Decision needed:** Confirm the state names, allowed transitions, terminal states, and invalid-transition behavior for the work-order container. Keep job, session, asset-inspection, review, certificate, evidence, and commercial lifecycles separate.

**Owner decision:** [RESOLVE / DEFER]

**If resolved, record:** State names, transition table, actor/policy gate for each transition, and negative cases.

**If deferred, record:** The decision owner, the follow-up stage, and the condition that blocks implementation.

### D2 — Authorization matrix

**Decision needed:** Confirm which actor/capability may create, authorize, assign, reschedule, partially complete, hand over, reconcile, review, and close a work order. Confirm whether a senior self-issue exception exists and under what policy.

**Owner decision:** [RESOLVE / DEFER]

**If resolved, record:** Actor/capability × action matrix, server-derived context requirements, denial outcomes, and audit requirements.

**If deferred, record:** The missing policy authority and the follow-up gate. Do not implement an assumed authorization rule.

### D3 — Offline scope and local retention

**Decision needed:** Confirm the maximum offline work-order scope, retained package/work-order data, local draft limits, evidence metadata limits, expiry behavior, and what must be rejected before synchronization.

**Owner decision:** [RESOLVE / DEFER]

**If resolved, record:** Bounded data classes, retention limits, expiry rules, and storage/cleanup behavior.

**If deferred, record:** The product or assurance owner and the exact decision required before offline implementation.

### D4 — Package-bound data contract

**Decision needed:** Confirm which immutable package version/hash, assignment context, authority context, asset context, and field-to-asset mappings may be used by a work-order session. The client must not fabricate authority, tenant, organization, scope, eligibility, or server transitions.

**Owner decision:** [RESOLVE / DEFER]

**If resolved, record:** The package-bound input contract, server validation rules, divergence behavior, and version/hash mismatch outcome.

**If deferred, record:** The contract owner and the follow-up design gate.

### D5 — Handover policy

**Decision needed:** Confirm who may initiate, accept, reject, or reverse a handover; what reason and scope are mandatory; how offline drafts are preserved; and whether handover may occur during partial completion or conflict hold.

**Owner decision:** [RESOLVE / DEFER]

**If resolved, record:** Handover transitions, actor/policy gates, draft-preservation rules, conflict behavior, and audit receipt fields.

**If deferred, record:** The responsible policy owner and the blocked implementation slice.

### D6 — Evidence interaction

**Decision needed:** Confirm which tenant-scoped evidence metadata and controlled object references may attach to a work order or inspection, and how evidence acceptance, retrieval, export, and retention remain separate from authority, eligibility, and certificate decisions.

**Owner decision:** [RESOLVE / DEFER]

**If resolved, record:** Reference contract, authorization, redaction, retention, and audit rules.

**If deferred, record:** The evidence-policy owner and the separate evidence decision gate.

### D7 — Migration decomposition

**Decision needed:** Confirm the conceptual dependency order for future schema work without applying any migration. The sequence must include explicit identifier/RLS review, isolated apply, cleanup/rollback evidence, and owner approval.

**Owner decision:** [RESOLVE / DEFER]

**If resolved, record:** Dependency order and the proof gate for each migration group.

**If deferred, record:** The migration-design owner and the prerequisite review.

### D8 — Highest safe testing seam

**Decision needed:** Confirm the proposed highest safe seam: a server-side work-order mutation contract exercised against an isolated disposable database with server-derived actor context and bounded fixtures.

**Owner decision:** [APPROVE / REVISE / DEFER]

**If approved, record:** The seam’s input/output boundary, fixture policy, tenant/RLS setup, negative cases, cleanup proof, and what it explicitly excludes.

**If revised, record:** The replacement seam and why it is safer or more authoritative.

## Completion gate for Ticket 01

Ticket 01 is complete only when every decision above is either resolved or explicitly deferred with an owner, follow-up gate, and implementation-blocking condition; the testing seam is approved or replaced; the migration sequence is design-only; and an independent read-only review confirms that no decision was silently invented.

## Immediate owner response

Please respond to this worksheet by specifying for D1–D7 whether each is **RESOLVE** or **DEFER**, and for D8 whether it is **APPROVE**, **REVISE**, or **DEFER**. Free-form policy decisions are acceptable, but do not approve implementation through this worksheet.

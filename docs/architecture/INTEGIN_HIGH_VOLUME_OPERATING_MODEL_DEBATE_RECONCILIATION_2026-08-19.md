# INTEGIN High-Volume Operating Model — Debate Reconciliation

**Status:** Accepted as a future product-design baseline; not implemented.  
**Decision owner:** INTEGIN product owner, reconciled by the engineering owner.  
**Scope:** Design only. No source, database, runtime, certificate, customer communication, scheduler, notification, inspection, or approval state changed.

## Decision

INTEGIN should adopt the integrated operating model defined in `INTEGIN_HIGH_VOLUME_WORK_ORDER_OPERATING_MODEL_DRAFT_2026-08-19.md`. The product’s center of gravity is the controlled **work order**: it carries customer/service context, authorized scope, assignments, operational progress, timesheet-style completion reporting, and closure. The individual, versioned, asset-bound inspection record remains the technical and evidentiary unit of truth. Training competence is a separate person/competence lifecycle, not an equipment asset.

The model supports high-volume work through native bulk preparation, safe cloning, selected/all-ready submission, exception queues, controlled import staging, partial completion across days, and explicit handover. None of those operational mechanisms can perform a bulk technical attestation, certificate issuance, evidence substitution, or automatic renewal.

## Debate evidence

| Lens | Result | Reconciled contribution |
| --- | --- | --- |
| Systems architecture | Completed, high confidence | Required separate state machines and aggregates; server authority; tenant scoping; explicit handover/reconciliation; quarantined imports; staged delivery. |
| Field operations red team | Completed, high confidence | Strengthened clone/handover abuse protections, exception visibility, offline/remote conflict handling, and non-technical limits on dashboard bulk actions. |
| Assurance/certificate governance | Incomplete | The reviewer returned malformed structured content. It is not counted as affirmative evidence; the owner reconciliation retains existing human-review and certificate-authority boundaries. |

## Accepted design rules

| Area | Accepted rule |
| --- | --- |
| Work order | Operational parent for request, scope, client/site, assignment, progress, reports, proposals, and closure—not a substitute for individual inspection evidence or certificates. |
| Inspection record | One asset identity, approved package/version, current answers, findings/evidence, submission receipt, review path, and certificate eligibility state. |
| Bulk work | Safe for preparation, draft creation, selected/all-ready queueing, and administrative routing; never a group pass/fail, signature, review, or certificate action. |
| Clone | Requires a new unique asset/serial identity; retains only permitted stable data; visibly marks carried-forward values; clears all prior evidence, defects, signatures, receipts, approvals, and certificate references. |
| Evidence | Template policy, not a universal assumption, controls disabled, optional, finding-triggered, always-required, or reviewer-requested evidence. A rejected item may have no photo requirement when its approved template permits that; required rejection reason/measurement/remark rules still apply. |
| Submission | `Submit selected` and `Submit all ready` submit each record independently and preserve separate server outcomes/receipts. |
| Handover | Sync first where practical; create a handover snapshot; preserve original inspector attribution; transfer only remaining scope or explicitly adopted drafts; surface late-offline returns as reconciliation conflicts. |
| Reports/certificates | Timesheet/completion report is derived from job and individual records. Certificate/report candidates are derived only from eligible individually receipted and reviewed records. Summary documents may have evidence-led annexes. |
| Training | Person/competence entity with course, issuer, trainer/approver, photo/identity policy, issue/expiry, verification, renewal, and governed certificate lifecycle; never modeled as an equipment asset. |
| Renewal dashboard | Shows expired/near-due asset and competence items. Authorized users may select eligible rows to create a draft work order, add to a compatible existing order, create a proposal, or assign follow-up—never to renew automatically. |
| Dashboards | Role/tenant-scoped filters, sorting, saved views, and safe administrative bulk actions only. |

## Rejected or narrowed suggestions

| Suggestion | Decision and reason |
| --- | --- |
| Require photos for every rejection | **Rejected.** The user’s template-governed policy is retained: a rejection can be supported by required reason, measurement, comments, or other evidence without photos where the approved rule allows it. |
| Force GPS/high-precision metadata for all evidence | **Not adopted.** Location metadata is a future per-template/per-customer policy decision with privacy, device, and offline implications; it is not a universal requirement. |
| Automatic certificate or training renewal from date/job action | **Rejected.** Expiry can trigger a reminder/proposal/work-order draft only. Renewal requires fresh governed completion and approval. |
| Treat a bulk batch as an approval or certificate event | **Rejected.** Batch scope is operational only; record-level authority remains mandatory. |
| Automatically merge conflicting offline work after handover | **Rejected.** Conflicts require explicit reconciliation preserving both contributors and attribution. |
| Direct external import into authoritative state | **Rejected.** Import must be tenant-scoped, quarantined, mapped, validated, previewed, approved, and audited. |

## Residual risks and open decisions

The model still needs explicit policies for offline-duration/staleness, conflict-resolution roles, authoritative server versus device time, evidence storage and retention limits, asset-identity duplication rules, import trust levels, certificate candidate eligibility, package-version changes during a job, and client-visible notification consent/preferences. Those are sequencing gates, not reasons to weaken the current design.

## Staged implementation direction

1. Define tenant-scoped domain contracts and state machines for work orders, scope items, inspections, evidence, reports, certificates, people/competencies, reminders, and handovers.
2. Build server-authoritative work-order creation, authorization, assignment, record receipt, and append-only audit boundaries.
3. Build the Field assigned-order inbox, local drafts, per-record queue, partial submissions, and honest offline status.
4. Add high-volume native preparation, safe cloning, exception queue, and controlled evidence modes.
5. Add controlled handover/reconciliation and role dashboards with safe multi-select administrative actions.
6. Add report/certificate candidate orchestration, then the person/competence and training-certificate lifecycle.
7. Add renewal dashboard, consent-aware notifications, proposal/scheduling actions, and quarantined import tooling only after their foundational authority boundaries are proven.

This sequence follows the active manifest gate: it is a future roadmap and does not authorize implementation before the current controlled gate closes.

# INTEGIN Integrated Operating Model — Final Reconciliation

**Status:** Accepted as a **future, post-manifest design roadmap**.  
**Decision date:** 2026-08-19.  
**Scope:** Product architecture, workflow, and integration readiness only.  
**Non-interference:** No source, database, migration, runtime, tenant data, certificate, invoice, reminder, external submission, package-enforcement, OIDC, OpenBao, or acceptance behavior changed.

## Decision in one sentence

INTEGIN will evolve as a server-authoritative, tenant-isolated, offline-first operational platform where the **work order** coordinates service delivery, while each **asset inspection**, **certificate**, **person competence**, **external submission**, and **invoice** retains its own governed lifecycle and can never improperly control another.

## Final debate outcome

Three independent lenses returned **`adopt_with_stages`**. The architecture lens validated tenancy, immutable receipts, offline conflict handling, state separation, and adapter boundaries. The assurance/regulatory lens added segregation of duties, certificate verification/revocation, anti-false-claim controls, template/version freeze, and strict commercial separation. The field-operations red-team lens affirmed the high-volume model while emphasizing clone/handover abuse protection and inspector-fatigue controls.

| Debate conclusion | Owner reconciliation |
| --- | --- |
| Adopt the model in stages | Accepted. No single module should be implemented as if it can bypass the earlier foundation stages. |
| Make server receipts and immutable versioning foundational | Accepted. Finalized records require server authority; later changes are governed versions/corrections, never silent overwrites. |
| Keep high-volume productivity features | Accepted. Bulk preparation, safe clone, grid work, selected submission, and exception queues are essential. |
| Keep configurable photo policy | Accepted. Photos are not universally required for pass or rejection; approved template policy governs evidence mode. |
| Keep technical, regulatory, and commercial state separate | Accepted. Work orders link these domains but do not merge their authority. |
| Use adapters for SASO/SAAC/other platforms and ZATCA | Accepted as future readiness only. Official onboarding, sandbox/test validation, credentials, and current requirements are mandatory before a connection or compliance claim. |

## Product operating model

### A. Request to closure

1. A client, sales representative, customer-service user, or authorized manager creates a **service request**.
2. An authorized coordinator creates a **work order/job number** with tenant/client/site, controlled inspection locations, approved scope, work-package/template version, due window, and responsible team.
3. The work order appears in the assigned inspector’s authenticated app inbox. Email/push may notify the inspector, but the authenticated INTEGIN order is the operational source of truth.
4. The inspector performs work inside that order—online or offline—using expandable location sections, scans, approved assets, type-correct forms, structured drafts, safe cloning, and an exception-focused queue.
5. The inspector submits selected ready records or all independently ready records. Every inspection receives its own server result and receipt.
6. The system derives a work-order completion/timesheet report and routes eligible inspection records to governed technical review/certificate candidate queues.
7. The work order closes only after every scope item is submitted, deferred, cancelled, moved to follow-up work, or otherwise formally dispositioned.

### B. Three levels of status

| Level | Purpose | Illustrative states |
| --- | --- | --- |
| Work order | Operational/commercial service container | Draft, authorized, assigned, in progress, partially submitted, awaiting resolution, under review, closed. |
| Bulk session | Inspector productivity container | Open, local draft, synchronized, partially queued, reconciliation required, archived. |
| Inspection record | Asset-bound technical/evidence truth | Draft, ready, queued, submitted, receipted, correction requested, rejected, reviewed, certificate eligible, issued/revoked/expired. |

An order being partially submitted never means that unfinished assets are complete. A batch never means that every contained record has the same technical outcome.

## High-volume inspection workflow

The native INTEGIN bulk workspace is the recommended primary channel. It supports a work day containing 100 or more similar inspections without converting convenience into authority.

| Capability | Accepted behavior | Boundary |
| --- | --- | --- |
| Asset scan/select | Opens only authorized scope items or a controlled exception. | No cross-tenant/cross-job work from a scan alone. |
| Safe clone | Carries permitted editable draft data/template structure, including a visible copied finding/defect where applicable. | Requires new asset/serial identity and explicit current-condition confirmation/change; clears prior evidence, signatures, receipts, review, and certificate state. |
| Grid/bulk preparation | Speeds draft entry and row-level validation. | No `select all → pass`, bulk technical sign-off, or bulk certificate action. |
| Submit selected/all-ready | Submits independently eligible rows with separate receipts. | Failures/exceptions do not block unrelated valid items, but cannot progress themselves. |
| Controlled import | Future intake for asset lists/office preparation. | Quarantined, mapped, previewed, tenant-scoped, approved, and audited before it can create drafts; never direct authority. |
| Exception queue | Highlights missing data, blocks, duplicate identities, evidence requirements, conflicts, and reviewer requests. | It must not hide exceptions to make a job appear complete. |

## Evidence, photos, findings, and outputs

Evidence policy is owned by a versioned approved template. Available modes are **off**, **optional**, **finding-triggered**, **always required**, and **reviewer requested**. A rejected inspection can legitimately proceed without a photo if that template permits it and the other required reason/measurement/remark rules are satisfied.

The inspector app may show photo capture/upload controls. Final certificates must not show those app controls. A certificate or detailed annex shows only the governed result and relevant evidence according to its report template. Evidence from one asset must never be reused as current evidence for another asset.

| Output | Derived from | Authority meaning |
| --- | --- | --- |
| Completion/timesheet report | Work-order details plus individually receipted records | Job-level operational summary; it does not replace asset records. |
| Inspection certificate/report + annex | Reviewed asset inspection, versioned template, findings/evidence, approved signer | Asset-level technical outcome with controlled status/expiry/revocation. |
| Training certificate | Person/competence, course, trainer/approver, issue/expiry, verification state | Separate competence lifecycle; person is not equipment asset. |

## Handover and offline reconciliation

When an inspector cannot finish, a coordinator initiates a controlled handover. Synchronization occurs first whenever possible. INTEGIN stores a handover snapshot of submitted records, inherited drafts, blockers, unstarted scope, source inspector, reason, time, and replacement assignment. Submitted work keeps its original attribution.

The new inspector receives only remaining scope or explicitly adopted drafts. If the first device returns later with overlapping offline work, INTEGIN creates a reconciliation conflict. It preserves both histories and requires authorized human resolution; it never auto-merges or silently overwrites.

## Inspector closeout and office commercial release

Each inspector closes only their own assigned/dispositioned scope and receives a derived personal timesheet/work summary. A shared work order remains open until all assigned scope is closed, transferred, deferred, cancelled, or otherwise resolved. The office receives a combined completion report, confirms the commercial completion basis, and then may create an invoice draft under finance controls. Invoice state remains unable to mutate the technical inspection/certificate lifecycle.

## Dashboards, expiry, reminders, and scheduling

Each role receives a tenant-scoped dashboard: inspectors receive assigned orders and exceptions; coordinators receive capacity, blockers, partial work, and handovers; sales/client service receives requests, proposals, and renewals; reviewers receive technical queues; clients receive only their permitted status/documents; management receives aggregates.

The expiry dashboard covers asset inspection/certificate and training-competence expiry. Users can filter, sort, and save views by client, site, type, due window, current job, owner, urgency, and status. Authorized internal staff can multi-select eligible items to create a **draft** work order, add them to a compatible open order, create a proposal, or assign follow-up. They can send permitted reminders to internal team/client contacts. None of these actions renews a certificate, creates a pass, or implies renewal.

## Support, feedback, customer communication, and advisory AI

Future INTEGIN support must distinguish four separate, role-scoped spaces: formal support tickets for human assistance and application/sync issues; feedback for improvement suggestions; work-order-linked customer communication; and a dedicated advisory-only **INTEGIN Assistant** workspace with user-visible, searchable history. AI conversation history remains separate from inspection evidence and technical findings. The assistant may explain a template, summarize permitted order status, guide a user through an error, or draft neutral text for human review; it cannot approve, submit, sign, certify, alter primary workflow state, or bypass a validation rule.

Support tickets create an accountable support inbox with category, priority, owner, timestamps, linked permitted context, resolution history, and closure feedback. Authorized managers may review response time, permitted transcript, resolution/reopen/escalation patterns, and post-close user feedback for quality improvement. This is not uncontrolled surveillance and must use tenant/role access, privacy, and retention policies.

## Verification identity and completed-scope match

The future read-only verification portal must show the issued document's full asset ID, full serial number, full equipment description, and exact completed inspection/test scope alongside validity/version status. The completed-scope block must clearly distinguish results such as `load test performed`, `load test not performed`, `load test not applicable`, `load test required but not completed`, or `routine visual inspection only`. The same identity/scope block is bound into the immutable issued-document payload and integrity reference so altered visible PDF text cannot make a routine inspection appear to be a load test or make one physical asset's certificate appear to belong to another.

## Lawful external integration readiness

INTEGIN is to be **integration-ready**, not prematurely connected. A versioned, tenant-isolated adapter framework will support external regulator/platform workflows only through official onboarding, credentials, sandbox/test validation where provided, data minimization, audit trails, and truthful status mapping.

| External status | Meaning in INTEGIN |
| --- | --- |
| Internal/private only | An accurately labelled company/internal document; no external claim. |
| Ready for external submission | Internal eligibility is complete but no external call yet occurred. |
| Submitted externally / pending | Official submission is awaiting a response. |
| Accepted/registered | Shown only with a verified reference/status returned through the authorized route. |
| Rejected/correction required | Internal correction queue; no false acceptance state. |
| External outage/retry | Visible operational exception, never a hidden bypass. |

Official public SASO material describes open data/APIs for published datasets, and the Saudi Accreditation Center describes accreditation/monitoring of conformity-assessment bodies.[1] [2] This does **not** establish an issuance API or entitlement for this particular certificate workflow. Any future SASO/SAAC/other regulator adapter needs separate official validation and legal/assurance review.

## Separate ZATCA commercial readiness

ZATCA e-invoicing readiness belongs to an independent commercial module. Official ZATCA material describes generation/storage and integration phases, the latter rolling out in waves for notified taxpayer groups, and provides developer technical resources.[3] [4] This justifies readiness architecture but not a present compliance, onboarding, or applicability claim.

The commercial model keeps **proposal**, **customer/order**, **invoice draft**, **issued invoice**, **credit/debit note**, **payment**, and **ZATCA status** separate. Invoice/payment/ZATCA state must never modify inspection, evidence, review, certificate, training competence, or technical authority. The future adapter uses only official onboarding, current specifications, isolated tenant credentials, truthful statuses, sandbox/testing where available, and auditable retry/error handling.

## Mandatory safeguards

1. Tenant isolation at every API, data, storage, cache, audit, adapter, and dashboard boundary.
2. Separation of duties between inspector, coordinator, reviewer/certifier, commercial/finance, and client roles.
3. Immutable server receipts for finalized inspections and later corrections as governed new versions.
4. Version-pinned templates and evidence rules; no silent change to an in-progress job.
5. Server-authoritative time for final state, with device time retained as contextual metadata and monitored for skew/tampering.
6. Encrypted device storage and short-lived tenant-scoped sync authority; no regulator or tax credentials on a field device.
7. Bulk-action previews, eligibility checks, confirmation, per-record results, and audit log.
8. Honest certificate, integration, renewal, and invoice status wording; no false official registration, clearance, renewal, or acceptance claim.

## Explicitly rejected shortcuts

The following are not accepted: bulk pass/fail or bulk technical signing; automatic certificate/training renewal; cloned evidence/signature/receipt inheritance; automatic conflict merge after handover; direct import into authoritative inspection/certificate state; external-regulator bypass/forged reference/false claim; forcing photo evidence contrary to template policy; using commercial payment/invoice/ZATCA state to control technical outcomes; and editing finalized certificates in place without version/revocation history.

## Staged future sequence

1. **Core contracts:** domain model, tenant boundaries, roles, state machines, template/evidence policy, clone rules, audit contract.
2. **Authority foundation:** server receipts, versioning/finalization, RBAC/ABAC, offline storage/sync security, controlled local drafts.
3. **Work-order and Field core:** request-to-assignment lifecycle, assigned-order inbox, per-record submission, partial work, initial high-volume performance proof.
4. **High-volume controls:** safe clone, bulk preparation, exception queue, evidence modes, controlled handover and reconciliation.
5. **Review and outputs:** separation of duties, certificate/report candidates, annexes, timesheets, verification/revocation design.
6. **Person/competence:** training lifecycle, issuance, verification, expiry, and renewal planning.
7. **Dashboards and renewal planner:** role-based filters/sorts/saved views, expiry actions, reminders, proposal/scheduling controls.
8. **Adapter framework:** optional regulator/platform integrations with sandbox, audit, status taxonomy, and official pilot approval.
9. **Commercial module:** separate ZATCA-ready invoicing workflow with finance controls and no technical authority coupling.
10. **Hardening:** security, performance, recovery, legal/assurance review, operational runbooks, and controlled rollout evidence.

## Open decisions before implementation

The next design work must settle: versioning/event model; per-field offline conflict policy; maximum offline duration; evidence retention/encryption/key management; device policy; certificate numbering, signing, verification, and revocation; template change governance; geolocation/time metadata/privacy policy; training authority/verification; exact regulator adapter contracts; ZATCA onboarding scope; multilingual outputs; commercial entitlement non-interference tests; and performance/storage limits for high-volume jobs.

## References

[1]: https://www.saso.gov.sa/en/mediacenter/pages/open_data.aspx "SASO Open Data"
[2]: https://saac.gov.sa/en/home/ "Saudi Accreditation Center"
[3]: https://zatca.gov.sa/en/E-Invoicing/Introduction/Pages/Roll-out-phases.aspx "ZATCA E-Invoicing Roll-out Phases"
[4]: https://zatca.gov.sa/en/E-Invoicing/SystemsDevelopers/Pages/default.aspx "ZATCA E-Invoicing Systems Developers"

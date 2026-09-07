# INTEGIN Comprehensive Accountable Tracking Policy

**Decision date:** 2026-08-18  
**Status:** Proposed policy and event-catalog expansion pending bounded review.  
**Authority:** The owner authorized comprehensive tracking of INTEGIN usage and operations.  
**Scope:** Every accountable product, security, operational, administrative, integration, Field, and advisory interaction that INTEGIN can safely observe.

## 1. Meaning of “track everything” in INTEGIN

INTEGIN will track every **accountable action, system transition, security event, operational condition, and workflow outcome** that it creates or receives through an approved boundary. It will not silently collect raw private content merely because that content passed through the platform.

> Comprehensive tracking means that an authorized reviewer can reconstruct **who did what, when, in which organization/site/work package/device/session, by which product version, with what result, and when the server received or accepted it**. It does not mean retaining passwords, raw evidence bodies, private secrets, protected fixture material, or unrestricted background surveillance.

This distinction preserves both forensic usefulness and INTEGIN’s evidence/tenant/authority boundaries.

## 2. Comprehensive event catalog

| Event family | Required tracked facts | Examples | Content explicitly excluded |
| --- | --- | --- | --- |
| Identity and session | Tenant/org, actor/service reference, role, session/interaction correlation, login/result, authentication factor class, client/app version, server receipt time. | Login success/failure, logout, session renewal, role switch, revoked session. | Passwords, authentication cookies, bearer tokens, MFA secrets, raw identity-provider assertions. |
| Web/API activity | Actor, action, target reference, HTTP method/status, route/action code, request/correlation ID, server time, outcome/reason class. | Read, create, update, export, rejected authorization, API capability use. | Request/response bodies, authorization headers, tokens, connection strings. |
| Field device lifecycle | Device reference, key/registration lifecycle, trust state, app/database/package version, local event time, server receipt time, offline/online state, outcome. | Device bind, package verify/cache, sync queued/sent/replayed/rejected, app update required. | Private keys, raw device proofs, key material, device file contents. |
| Field workflow activity | Actor/device/work-package/site context, form/task action code, state before/after reference, offline status, timing, outcome. | Start inspection, save draft, submit item, upload requested, conflict resolved, package superseded. | Raw form answer values, raw photos/videos/documents, hidden notes unless deliberately accepted as evidence. |
| Evidence lifecycle | Evidence object reference/digest, size/media class, capture/receipt/acceptance times, custody transition, storage result, integrity result. | Evidence queued, uploaded, hash verified, accepted/rejected, retained/exported. | Raw evidence bytes, image pixels, document text, OCR source body, secret metadata. |
| Assurance workflow | Applicable standard/package version reference, review/action/certificate state transition, accountable decision, reason code, evidence reference, human decision time. | Review returned, finding raised, CAPA assigned/closed, eligibility ready, certificate issued/suspended/withdrawn. | Private deliberation text outside an accepted decision record; commercial plan/tier as a decision input. |
| Administrative configuration | Actor, change class, before/after digest or approved reference, reason, approval/maker-checker reference, result. | User/role change, tenant configuration, retention change, entitlement override, integration configuration change. | Raw secrets, private configuration blobs, connection credentials. |
| Security and abuse | Source class, event code, outcome, severity, rate/threshold references, session/device correlation, investigation disposition. | Auth failure, authorization denial, proof failure, replay, unsafe input, export anomaly, service configuration change. | Attack payloads, raw exploit strings, credentials, tokens, keys. |
| Integration and adapter | Adapter identity/version, inbound/outbound direction, schema version, payload digest, validation result, reconciliation ID, outcome. | CMMS import received, ERP export queued, document import rejected, callback refused. | Full external payload, third-party secrets, user credentials, raw document contents. |
| Advisory AI/OCR | Advisory artifact reference, model/template/version, permitted corpus/version references, source digest, confidence/limitations, human disposition, cost/usage class. | OCR suggested a field, reviewer corrected result, retrieval cited a controlled document, advisory service degraded. | Prompt payload body, raw tenant evidence/document text, model credentials, embeddings containing protected content. |
| Commercial and capacity | Plan/entitlement reference, meter name/unit, aggregate count, quota result, administrative change history, finance-export status. | Seat/device count, storage bytes, advisory job count, optional-feature quota notice. | Payment credentials, invoice card/bank data, commercial status in technical decision engines. |
| Operational platform | Service/version, deployment/config version reference, health/readiness, queue/storage capacity, backup/restore/drill outcome, failure class. | Start/stop, health degradation, recovery drill, migration attempt, RustFS health, database connectivity. | Environment-variable values, private configuration, secrets, payload dumps. |
| Audit access itself | Actor, scope/filter, reason, view/export/break-glass action, result, time. | Usage Journal viewed, session exported, audit record searched, break-glass granted. | Query result bodies beyond the already-authorized rendered/exported records. |

## 3. “Where” and temporal reconstruction

Tracking uses a hierarchy of location and time context.

| Context | Collection rule | Visibility rule |
| --- | --- | --- |
| Tenant, organization, site, work package | Collected whenever already established by the authoritative action boundary. | Tenant/org/site scoped through RBAC. |
| Device and client context | Collected as opaque device reference, app version, platform, and coarse network/client class. | Organization administrator or operational role according to policy. |
| Action-bound geographic location | Collected where an inspection action requires it and the Field client has permission. Include precision, source, and accuracy. | Exact location only to authorized roles; default displays may use site or reduced precision. |
| Continuous/background location | Not collected by default. It requires a separately documented operational need, user notice, retention period, and reviewed implementation. | Never made a silent product default. |
| Client action time and server receipt time | Both retained for offline-capable actions. | Admin views must make delay, replay, rejection, conflict, and authoritative acceptance visible. |

## 4. Non-negotiable data boundary

The following classes remain prohibited from all tracking stores, exports, diagnostics, metrics, and session views: passwords; secrets; keys; tokens; raw auth session IDs; connection strings; raw request/response bodies; raw form values; raw evidence; protected fixture material; private manifest bodies; private identity payloads; and cross-tenant query results.

The tracking implementation must enforce these exclusions through allow-listed event schemas, typed event builders, pre-persistence redaction, pre-export redaction, field-level tests, and failure-path tests. Developers must not rely on comments or personal memory to maintain this boundary.

## 5. Storage, authority, and access rules

Accountable events are append-only operational facts. A server-authoritative state transition requires a durable event in the same database transaction or outbox transaction. Offline Field events are locally buffered and explicitly labeled `local_pending` until the server accepts, rejects, replays, or conflicts them.

All events carry tenant context. Tenant administrators see only their organization-scoped records; site administrators see only authorized site scope; platform operations visibility is separated from tenant administration; any cross-tenant emergency access requires a documented break-glass process and is itself fully tracked.

Event records provide accountability and operational reconstruction. They do not become inspection evidence, evidence acceptance, review conclusion, corrective-action sufficiency, waiver, eligibility, or certification decision input unless an accountable human explicitly attaches a relevant record through the formal evidence workflow with a recorded rationale.

## 6. Retention, aggregation, and export

The implementation will define data-class-specific retention rather than a single permanent default. Detailed accountable records remain available for the governing retention requirement; low-value diagnostics may roll up into aggregates after their operational window; legal holds prevent normal deletion where an authorized policy requires it. The exact retention periods are deployment/policy decisions and must be configured per tenant/scheme rather than fabricated in source code.

Exports must be tenant-scoped, allow-list/redaction enforced, versioned, and themselves tracked. Aggregate usage and operational metrics can support capacity and subscription reporting, but raw activity tracking is never exported to a finance or external billing system by default.

## 7. Implementation sequence

| Order | Deliverable | Release condition |
| ---: | --- | --- |
| 1 | Event taxonomy, field classification register, outcome/reason vocabulary, and retention classes. | Architecture/security review accepts every event family and prohibited field rule. |
| 2 | Typed accountable-event library plus redaction library for Go, FastAPI, and Flutter. | Unit/failure-path tests prove excluded fields cannot persist or export. |
| 3 | Server durable audit/outbox integration for authoritative actions; Field pending-event buffer. | Transaction/replay tests prove action and accountability consistency. |
| 4 | Tenant-scoped Usage Journal and Session Explorer with dual timestamps, filters, and audit-of-audit. | Tenant-negative, RBAC, location-precision, export-redaction, and offline-state tests pass. |
| 5 | Operational dashboards, capacity/usage aggregates, integration events, and advisory artifact visibility. | Non-interference tests prove no telemetry/commercial/AI influence on authority. |
| 6 | Retention/hold/export administration, recovery drills, and break-glass access. | Drill records and access-control review are accepted. |

## 8. Enduring non-interference rules

Comprehensive tracking cannot make INTEGIN a surveillance engine or hidden authority path. Subscription, usage, telemetry, security anomalies, location, session activity, AI/OCR, integration state, and administrator views remain operational/commercial/advisory facts. They cannot alter inspection authority, evidence acceptance, review outcome, corrective-action closure, eligibility, waiver, risk acceptance, or certificate status.

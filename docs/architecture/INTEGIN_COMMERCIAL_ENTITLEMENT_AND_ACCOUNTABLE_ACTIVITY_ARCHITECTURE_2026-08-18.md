# INTEGIN Commercial Entitlement and Accountable Activity Architecture

**Decision date:** 2026-08-18  
**Status:** Approved architecture direction; **design only**.  
**Scope:** Subscription/tier/entitlement, usage metering, administrator activity visibility, and session correlation.  
**Non-authorization statement:** This document does not authorize payment processing, billing-provider selection, OIDC enablement, OpenBao wiring, package enforcement, candidate wiring, acceptance change, or any production activation.

## 1. Decision

INTEGIN will introduce **commercial entitlement** and **accountable activity** as two isolated bounded contexts inside the existing Go modular monolith. They are commercial and operational overlays, not sources of inspection, evidence, review, corrective-action, eligibility, waiver, or certification authority.

> A commercial state may determine whether an optional product capability is available. It must never determine whether an inspection is true, whether evidence is accepted, whether a review passes, whether a corrective action closes, or whether a certificate is issued.

The immediate engineering priority remains the source-owned manifest receipt hard gate. Nothing in this document displaces the requirement to prove opaque candidate-run correlation and trustworthy source-derived signals for the eight public receipt cases before any isolated-pilot receipt wiring.

## 2. Why a separate accountable-activity design is necessary

The requested view of who used the system, where they used it, and what they did is an **accountability function**, not generic surveillance analytics. Application-level event records are necessary because the application has the actor, action, outcome, and business context that infrastructure logs often lack. OWASP recommends recording the relevant *when, where, who, and what*, including an interaction identifier, while excluding secrets, session identifiers, private payloads, and unnecessary sensitive information.[1]

INTEGIN will use a stable internal `interaction_correlation_id` or `session_observation_id` for product accountability. It may later map to OpenTelemetry attributes, but it will not depend on OpenTelemetry’s session convention as a product contract because that convention is still marked **Development**.[2] [3]

## 3. Non-interference architecture

| Bounded context | Owns | May read | Must not read or influence |
| --- | --- | --- | --- |
| Inspection, evidence, review, corrective action, eligibility, certification | Authoritative assurance facts and human decisions. | Its approved domain dependencies. | Subscription state, payment state, plan tier, usage counters, telemetry, AI confidence, activity-log views, or location data as decision predicates. |
| Subscription and entitlement | Plans, allowed optional features, quotas, commercial account state, usage aggregates, overrides, and commercial change history. | Tenant identity and permitted aggregate meter events. | Inspection acceptance, evidence contents, review decisions, corrective actions, waiver, eligibility, certificate state, or work-package authority manifests. |
| Accountable activity | Immutable records of accountable actions, session summaries, redacted operational context, and audit-of-audit events. | Redacted action metadata and authorized tenant/org/site context. | Secrets, auth tokens, raw evidence, private payloads, protected fixture material, and direct workflow mutation. |
| Advisory AI/OCR | Read-only advisory artifacts, citations, confidence, limitations, and human disposition. | Approved controlled sources and authorized read models. | Commercial authority, audit mutation, acceptance/review/certification mutation, or mandatory workflow conditions. |

The source tree will later enforce this separation with module ownership rules, database ownership, forbidden import/query tests, and repository contracts requiring tenant context. There must be no foreign key, hidden predicate, or convenience lookup from an authoritative assurance module to a commercial table.

## 4. Subscription, plan, and entitlement model

The first release is an internal, provider-neutral entitlement ledger. It supports product packaging without prematurely adopting a finance platform.

| Concept | Purpose | Allowed effect | Prohibited effect |
| --- | --- | --- | --- |
| `plan` | Describes a commercial product package. | Enables optional product capabilities and quota policy. | Does not affect authority, evidence, review, or certificate facts. |
| `entitlement` | Effective tenant feature/limit decision. | Controls optional dashboard, advisory-processing, reporting, support, or administrative capability access. | Cannot block Field capture, cache binding, sync replay, evidence preservation, or historical accountability access. |
| `usage_counter` | Coarse server-derived count for capacity and commercial reporting. | Enables usage display and non-authoritative quota notices. | Cannot inspect evidence contents or influence inspection outcomes. |
| `entitlement_override` | Time-bounded, reasoned commercial exception or trial. | Allows an authorized administrator to change optional access. | Cannot act as a waiver for technical or certification rules. |
| `invoice_reference` | A non-authoritative reference to future finance records. | Supports later one-way finance export. | Does not import payment or dunning state into INTEGIN authority modules. |

The effective evaluation is intentionally narrow:

```text
entitlement_decision(tenant, feature_code, at_time)
  -> enabled | limited | unavailable, plus limit, source, and expiry
```

It is not an inspection-policy evaluator. It receives no inspection result, evidence payload, review state, certificate state, or human judgment input.

### 4.1 Product packages without pricing commitment

The following are **capability categories**, not final names, pricing, or sales commitments. They allow a future commercial catalogue to be designed without making commercial status an assurance control.

| Capability category | Intended optional capabilities | Core continuity that remains unaffected |
| --- | --- | --- |
| **Core Operations** | Organization administration, approved work packages, Field collection, sync, evidence custody, essential review records, and standard exports. | All existing accepted records, Field capture, sync, and accountable-history access. |
| **Assurance Management** | Advanced review queues, corrective-action dashboards, quality sampling, structured management review, and expanded reporting. | Technical validity and historical evidence remain unchanged. |
| **Controlled Intelligence** | Advisory OCR capacity, controlled-document guidance, anomaly queues, richer analytics, and advisory reporting. | No advisory feature can block work or make an authoritative decision. |
| **Enterprise Operations** | Higher administrative limits, additional tenant-safe integrations, deployment support tooling, extended capacity, and advanced operational reporting. | No plan changes the scope, validity, or outcome of an inspection/certificate. |

If an entitlement becomes uncertain, expired, unavailable, or corrupted, INTEGIN must preserve Field collection, server replay, evidence custody, accountable history, and authorized continuity/export access. The only permitted response is a commercial notice, optional-feature limitation, or later administrative reconciliation.

### 4.2 Billing-system posture

No billing platform is selected or enabled. Kill Bill is a modular, Apache-2.0 open-source subscription and payment platform that could be evaluated later for isolated back-office invoicing.[4] Lago supports self-hosted subscription and usage billing, but its published project material identifies AGPLv3 licensing and self-hosted action tracking by default, which requires separate legal, privacy, and egress review before any lab consideration.[5]

The approved direction is therefore **internal entitlement first, finance export later**. A future finance integration, if approved, must be one-way from INTEGIN aggregate usage to a separately operated back-office service. No payment provider, invoice event, dunning signal, or external callback may write into authoritative assurance workflows.

## 5. Accountable activity and session-observability model

INTEGIN will distinguish **authoritative accountable events** from **low-priority diagnostics**. An authoritative event must commit with its server state change in the same transaction or durable outbox transaction. Diagnostics may be sampled, bounded, or circular-buffered, but they may never crowd out evidence or accountable-event storage.

| Event category | Examples | Persistence rule |
| --- | --- | --- |
| Accountable authority event | Evidence accepted/rejected, review submitted, corrective-action decision, certificate lifecycle decision, entitlement override, role change, export, audit-data view. | Durable append-only event with the corresponding state transaction/outbox. |
| Accountable Field event | Local capture, local submission, sync received, replayed, rejected, conflicted, or accepted. | Buffered locally while offline; remains local/pending until server replay validates it. |
| Security event | Authentication result, authorization denial, device proof failure, manifest replay, configuration/security change. | Durable server event with explicit result and reason class. |
| Diagnostic event | Performance span, non-sensitive error class, retry count, queue depth, service health. | Best-effort and storage-bounded; never contains private payloads. |

### 5.1 Minimum allow-listed event fields

| Field | Purpose | Privacy rule |
| --- | --- | --- |
| `tenant_id`, `organization_id` | Mandatory isolation scope. | Required on tenant-bearing events. |
| `actor_ref`, `service_ref`, `role_context` | Identifies accountable human or service actor. | Use a stable internal reference; display names only to authorized views. |
| `action_type`, `target_type`, `target_ref` | States what occurred and to what. | Use opaque/redacted target references where detailed identity is not needed. |
| `outcome`, `reason_class` | States result without leaking private text. | Use a controlled vocabulary; do not store raw exception/payload dumps. |
| `client_event_time`, `server_receipt_time` | Distinguishes offline occurrence from later receipt. | Both are required when a Field event was offline. |
| `interaction_correlation_id`, `request_id`, `idempotency_key` | Reconstructs a session/action path. | Internal opaque values only; never copy auth-session cookies or bearer tokens. |
| `device_ref`, `work_package_ref`, `site_ref` | Provides accountable operational context. | Include only when justified by the action. |
| `location_context` | Describes where the action occurred. | Prefer site/work-package or coarse, action-bound location. Continuous tracking is prohibited. |
| `event_version`, `prev_event_hash`, `event_hash` | Supports event evolution and later tamper-evidence hardening. | Hashes do not replace storage access control or recovery evidence. |

The following are categorically excluded: passwords, access tokens, API keys, private keys, connection strings, raw authentication session IDs, raw request/response bodies, raw evidence/image/document contents, private fixture material, and sensitive personal data that is not necessary for accountability.

### 5.2 Administrator views

Tenant administrators will receive a scoped **Usage Journal**, not unrestricted surveillance analytics. It will allow authorized organization or site administrators to filter accountable events by actor, role, device, site/work-package context, action type, outcome, and time range. It will show both action and server-receipt times, whether an event is pending/synced/replayed/rejected/conflicted/accepted, and only coarse or action-bound location where a documented purpose exists.

Viewing, filtering, exporting, or invoking a rare break-glass audit path must itself create an accountable event with actor, query scope, reason, time, and outcome. Exports remain tenant-scoped and redacted. Platform operations observability remains separate from the organization administrator’s accountability view.

## 6. Preventable-risk register

| Preventable risk | Required control | Evidence before release |
| --- | --- | --- |
| Commercial soft-lock blocks Field continuity | Core capture/sync/evidence flows bypass entitlement evaluation. | Failure-mode and outage tests with absent, stale, denied, and unavailable entitlement state. |
| Audit logging leaks private material | Allow-list schema plus redaction library before persistence/export. | Unit, failure-path, and export tests proving excluded fields cannot appear. |
| State change lacks accountability | State and accountable event commit in one transaction/outbox. | Transaction/outbox tests and replay tests. |
| Cross-tenant activity/report leakage | Tenant context at repository/API/job boundaries, scoped object keys, and negative tests. | API, database, object-store, job, audit, report, and export isolation suite. |
| Advisory output becomes authority | Separate advisory artifacts, human disposition, no write credentials, `blocking=false`. | Permission, API, UI, and regression tests. |
| Observability becomes surveillance | Purpose limitation, coarse location, no replay/screen capture, bounded retention, audit-of-audit. | Data classification, RBAC, retention, and UI evidence. |
| Recovery cannot match operational complexity | Drill PostgreSQL, RustFS, manifests, receipts, device records, configuration, and keys. | Completed restore/hash-verification drills and reviewed runbooks. |
| Supply-chain/deployment drift | Pinned dependencies, lockfiles, SBOMs, checksums/signatures, vulnerability review, immutable images where feasible. | CI/release evidence and deployment review. |

## 7. Controlled implementation sequence

| Sequence | Work item | Entry gate |
| ---: | --- | --- |
| 0 | Complete source-owned manifest receipt correlation hard gate. | The existing eight-case source-derived evidence requirement. |
| 1 | Produce bounded-context map, data-authority matrix, forbidden-dependency rules, and disabled-control record. | Receipt boundary frozen; no runtime activation. |
| 2 | Define accountable-event taxonomy, redaction library, session-correlation schema, and durable audit/outbox contract. | Design and negative tests approved. |
| 3 | Add tenant-isolation and redaction harnesses across audit, receipts, storage, jobs, reports, and exports. | Tests fail closed on cross-tenant or redaction violations. |
| 4 | Implement minimal internal entitlement ledger and feature registry. | Non-interference tests prove authoritative workflows are unaffected. |
| 5 | Implement tenant-scoped Usage Journal with audit-of-audit and redacted exports. | RBAC, retention, action-vs-sync time, and location-minimization tests pass. |
| 6 | Add offline Field accountable-action buffering and storage prioritization. | Evidence and authoritative action records demonstrably outrank diagnostics. |
| 7 | Run recovery and supply-chain evidence drills. | Restore and build-integrity reports pass. |
| 8 | Resume governed standards, applicability, evidence acceptance, review/action, certificate, reporting, and advisory-AI programs. | Each program’s own evidence gate is approved. |

## 8. What this review cannot guarantee

This architecture reduces preventable coupling, privacy, tenant-isolation, and operational risks. It cannot guarantee zero defects, zero breaches, perfect Field-device integrity, perfect AI/OCR behavior, perfect human judgment, legal sufficiency in every jurisdiction, or immutable storage behavior under every deployment. Residual risk is managed through explicit evidence gates, tests, recovery drills, review, retention policy, and accountable human governance.

## References

[1]: https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html "OWASP Logging Cheat Sheet"
[2]: https://opentelemetry.io/docs/specs/semconv/general/session/ "OpenTelemetry Semantic Conventions for Session"
[3]: https://opentelemetry.io/docs/concepts/semantic-conventions/ "OpenTelemetry Semantic Conventions"
[4]: https://github.com/killbill/killbill "Kill Bill Open-Source Subscription Billing Platform"
[5]: https://github.com/getlago/lago "Lago Open-Source Billing Platform"

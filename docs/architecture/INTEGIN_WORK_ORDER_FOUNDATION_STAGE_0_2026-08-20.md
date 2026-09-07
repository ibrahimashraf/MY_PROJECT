# INTEGIN Work-Order Foundation — Stage 0

**Status:** Proposed Stage 0 architecture and implementation plan; no database migration, API implementation, or runtime behavior is authorized by this document.  
**Date:** 2026-08-20  
**Purpose:** Establish the server-authoritative work-order foundation before Field screens, high-volume inspection workflows, certificate issuance, or external integrations.

## Direct decision

> A work order is the authoritative operational boundary for inspection work. It determines which client, locations, assets, forms, inspectors, evidence rules, and submission actions are available for a particular job.

The Field app may cache assigned work orders and capture work offline, but it must not become the authority for client identity, asset identity, assignment, ownership, approval, certificate issuance, or external submission.

## 1. Stage 0 scope

Stage 0 establishes the domain language and invariants that every later component must use. It includes work-order identity, tenant scope, client/location/asset scope, assignment, lifecycle, partial completion, reassignment, provisional offline records, conflict handling, and the boundary between inspection completion and certificate issuance.

Stage 0 does not yet implement the Flutter UI, bulk-entry screens, certificate renderer, Binding Key Registry, external SASO/SAAC/ZATCA connectors, invoicing, or delegated offline certificate issuance.

## 2. Core entities

| Entity | Role in the work-order foundation |
|---|---|
| **Tenant/organization** | Security and ownership boundary. Every work-order record and child operation is tenant-scoped. |
| **Client** | Customer/owner context for the requested work. A client is not recreated merely because an inspector works offline. |
| **Location** | Site, area, vessel, yard, facility, or other operational place where work is performed. |
| **Work order** | Server-authoritative request and execution container. It has a stable identity and lifecycle. |
| **Work-order scope item** | The asset, person, equipment group, or inspection target included in the order. |
| **Assignment** | A time-bounded grant of work-order scope to one or more inspectors or teams. |
| **Inspection record** | A capture and inspection execution against one scope item and one approved form version. |
| **Submission segment** | A partial set of completed inspection records submitted from an order without closing the entire order. |
| **Provisional candidate** | An offline client, location, asset, or work-order reference awaiting authoritative server reconciliation. |
| **Certificate request** | A server-controlled request to validate and issue a certificate after inspection acceptance. |

The work order should reference stable server IDs for the tenant, client, locations, and assets. Provisional offline identifiers must be explicitly typed as provisional and must never be mistaken for canonical IDs.

## 3. Work-order identity and lifecycle

A work order receives a server-generated immutable identifier and a human-readable job number. The human-readable number is not the sole identity because numbers may be formatted, imported, or displayed differently across languages and systems.

The lifecycle should distinguish commercial/request status from operational execution status and certificate status. A single overloaded status field would make partial work and external submission difficult to reason about.

| Lifecycle family | Recommended states |
|---|---|
| **Request** | `draft`, `requested`, `accepted`, `cancelled` |
| **Execution** | `ready`, `assigned`, `in_progress`, `partially_submitted`, `awaiting_client`, `awaiting_review`, `completed` |
| **Commercial** | `not_ready_for_invoice`, `ready_for_office_confirmation`, `released_for_invoice`, `invoiced`, `closed` |
| **Certificate** | `not_started`, `pending_validation`, `partially_issued`, `issued`, `needs_correction`, `revoked_or_superseded` |

These state families must not be collapsed into one status. For example, an order may be `partially_submitted` operationally, `awaiting_client` because some equipment is unavailable, and `not_ready_for_invoice` commercially.

Every state transition must record the actor, timestamp, reason code, source channel, previous state, new state, and relevant revision. The server rejects transitions that violate the lifecycle policy.

## 4. Work-order scope and asset access

A work order explicitly contains the clients, locations, and scope items that may be inspected. The Field app filters local records by the active assignment and work-order scope, not by whatever happens to exist in its local cache.

An inspector may inspect an existing assigned asset and create a new inspection/renewal record against that asset. The inspector may not create a duplicate canonical asset, select an unrelated client asset, move ownership, or use a cloned inspection as evidence for a different asset without changing and validating every relevant field.

If an onsite item is absent from the order, the inspector may create a provisional scope candidate only when the tenant’s emergency-offline policy allows it. The server later matches or creates the canonical record and decides whether the inspection can proceed to issuance.

## 5. Assignment model

Assignments must be explicit records rather than a field on the work order. A work order can be shared by multiple inspectors, transferred, or divided by location and scope item.

An assignment should contain the work-order ID, inspector/team ID, authorized scope, start and end timestamps, assignment role, allowed actions, assignment revision, and status. The allowed actions may distinguish viewing, capturing, submitting, correcting, reviewing, and closing a personal work segment.

| Assignment action | Inspector | Manager/office |
|---|---:|---:|
| View assigned scope | Yes | Yes |
| Capture inspection | Yes | Optional by policy |
| Submit own completed segment | Yes | Yes |
| Transfer assignment | No, request only | Yes |
| Change client or ownership | No | Yes |
| Close entire work order | No, unless policy explicitly delegates | Yes |
| Release invoice | No | Yes |
| Issue final certificate | Only through server-authorized workflow | Yes through server-authorized workflow |

When two inspectors share one work order, each inspector may close their own assignment segment and receive an individual completion/timesheet summary. The work order remains open until all required segments are complete or a manager explicitly resolves the remaining scope.

## 6. Partial completion and submission

The work order must support multiple inspection days and partial submission. An inspector can select completed records and submit only those records while leaving unavailable, waiting, defective, or scheduled-for-another-day records open.

A submission segment should contain the work-order ID, assignment ID, selected inspection IDs, completion date/time, inspector identity, location context, evidence state, notes, and a server-generated submission revision. The segment is not the same as closing the work order.

The server must prevent a record from being included in two incompatible final submissions. If a duplicate or concurrent submission is detected, the server returns a conflict while preserving both operation histories for review.

## 7. Reassignment and transfer

A manager or authorized office user may transfer all or part of a work order to another inspector. The original inspector must synchronize completed local work before transfer where possible. The server records the previous assignment, new assignment, actor, reason, effective time, and transferred scope.

The receiving inspector downloads the new assignment snapshot. The original inspector’s local drafts are not silently erased. They are marked as transferred, completed, conflicted, or requiring office review according to the server result.

A work-order transfer must not transfer client ownership or asset ownership. Operational assignment and commercial ownership are separate concepts.

## 8. Offline and emergency work-order records

For a preassigned order, the Field app receives a signed assignment snapshot. It can work offline against that snapshot and queue operations for later synchronization.

For a new remote client, the emergency workflow may create provisional client, work-order, location, asset, and inspection records locally. These records are chained by local IDs and idempotency keys. The app may generate a provisional field report, but final canonical linkage and certificate issuance remain server-controlled.

The server must be able to accept, match, reject, or request correction for each provisional record independently. A failed client match must not cause the local inspection evidence to disappear.

## 9. Certificate and invoice boundaries

Completing a work-order segment is evidence of field work, not automatic certificate issuance or invoice release. The server must separately validate inspection completeness, required evidence, package version, asset identity, ownership context, approvals, and certificate rules.

The office may use segment completion and timesheet summaries to prepare an invoice. The invoice should be linked to the work order and may include dates, inspectors, locations, scope items, quantities, and completion reports. This commercial output must not mutate the immutable inspection or certificate record.

## 10. Tenant-safe server invariants

The following invariants are mandatory:

1. Every work order, assignment, scope item, inspection, submission segment, provisional candidate, and certificate request carries a tenant scope enforced by the server.
2. Every asset selection is checked against the active work-order scope and assignment scope.
3. Every mutation includes an idempotency key and expected revision where concurrency matters.
4. Every state transition is authorized by role and recorded in an audit trail.
5. Local cache contents never expand the inspector’s authority.
6. Client and asset ownership changes are administrative operations.
7. Acknowledgment of a queued operation never implies certificate issuance or external authority approval.
8. Immutable issued records are never rewritten; corrections produce a controlled supersession, reissue, or revocation path.
9. Provisional offline records cannot silently become canonical without an authoritative server response.
10. Deletion of local drafts cannot delete server records and cannot silently discard unsynchronized work.

## 11. Stage 0 implementation sequence

The implementation should proceed in dependency order:

| Stage | Deliverable | Exit condition |
|---|---|---|
| 0.1 | Domain types and state-transition catalog | All entity identities, lifecycle families, and transition rules are reviewed. |
| 0.2 | Tenant-scoped server tables and authorization contracts | Tenant isolation and assignment scope are enforced in tests. |
| 0.3 | Work-order and assignment service/API | Create, assign, partially submit, transfer, and close flows have typed contracts. |
| 0.4 | Scope and conflict service | Asset scope, revision checks, idempotency, and provisional candidate resolution are defined. |
| 0.5 | Read-only office/Field contract fixtures | Representative scenarios cover one inspector, two inspectors, partial days, transfer, and remote onboarding. |
| 0.6 | Independent review and tests | Negative authorization and concurrency tests pass before UI or Flutter integration. |

## Handoff to later stages

After Stage 0 is accepted, the next dependency is the **Binding Key Registry**, because form and certificate fields need stable server-owned semantic bindings. The fixed-layout designer can then bind cells to those keys. The Field local storage and assignment-scoped synchronization implementation follows after the work-order contracts are stable.

High-volume inspection tables, clone actions, partial work, timesheets, invoicing, and certificate generation should all consume the same work-order and assignment vocabulary rather than creating separate parallel models.

**No implementation, migration, or runtime exercise is claimed by this document.**

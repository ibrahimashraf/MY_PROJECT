# INTEGIN Field Assignment-Scoped Asset Synchronization

**Status:** Proposed architecture decision; no database migration or Field implementation is authorized by this document.  
**Date:** 2026-08-20  
**Scope:** Existing client assets, offline inspection capture, work-order access, renewals, administration-controlled ownership transfers, and reconnect synchronization.

## Direct decision

> The Field app must reuse server-authoritative assets. It must not create a second asset merely because an inspector is offline or because the asset already exists in INTEGIN.

The correct operating model is **assignment-scoped synchronization**, not unrestricted synchronization of the whole organization’s asset database. When a work order is assigned to an inspector, the Field app receives the minimum client, location, work-order, asset, certificate-history, and form metadata required for that assignment. The inspector may work offline with that controlled snapshot. On reconnect, the app synchronizes queued inspection work and receives authoritative changes.

This provides continuity without allowing an inspector to browse or select assets belonging to unrelated clients or assignments.

## 1. Existing assets are reused, not recreated

The server owns the canonical identity of every client, location, asset, person, equipment record, certificate, and inspection history. A Field device may create a **local draft reference** while offline, but it must not create a new canonical asset when a matching assigned asset is already present in its local snapshot.

Asset matching should use stable server identity first. Serial number and description are useful search fields, but they are not sufficient by themselves to prove identity because two clients may use the same serial format, equipment may be re-identified, and data entry may contain errors.

| Situation | Field behavior |
| --- | --- |
| Asset exists in the assigned work-order snapshot | Select and inspect the existing asset. Do not create another asset. |
| Asset is physically present but absent from the assigned snapshot | Create a **provisional asset candidate** linked to the current work order, not an immediately canonical asset. Mark it for server/office resolution. |
| Same serial number appears on another client’s asset | Do not expose or merge the other client’s record. Escalate a possible identity conflict to the server or office. |
| Inspector wants to renew a certificate | Start a new inspection/renewal record against the existing asset identity. Do not copy the asset into a new row. |
| Manager confirms an ownership transfer | The server changes the authoritative ownership relationship with an effective time and audit record. The asset identity remains stable. |

The distinction is important: **ownership may change; asset identity should normally remain stable**. Certificates and inspections should preserve the asset identity and record the owner relationship that was effective at the time of the event.

## 2. What synchronizes to the Field app

The Field app should not download every client and every asset in the organization by default. That would increase privacy exposure, storage use, synchronization cost, and the consequences of device loss.

Instead, the server issues an assignment-scoped synchronization package for each inspector/device session.

| Scope | Included data |
| --- | --- |
| **Inspector identity** | Current user, tenant, role, device/session policy, and assigned permissions. |
| **Work-order scope** | Work-order ID, lifecycle state, assignment, dates, priority, client, and allowed actions. |
| **Location scope** | Only locations/sites/areas referenced by the assigned work orders. |
| **Asset scope** | Only assets included in or explicitly added to those work orders, including the minimum identity and inspection-history fields required for safe capture. |
| **Form scope** | Signed form packages, binding catalogs, rules, option sets, and layouts required by the work-order asset types. |
| **Evidence scope** | Only evidence references needed for the assigned work, subject to retention and access policy. Full prior media should be downloaded only when the workflow requires it. |
| **Client scope** | Only client information required for the assigned work order and certificate/report output. |

The device may hold multiple assigned clients at once, but each record must carry an explicit `tenant_id`, `client_id`, and assignment scope. The user interface must filter and authorize every lookup using the currently selected work order, not merely the device cache.

## 3. Access rules inside the Field app

Selecting a work order establishes the active operational environment. The inspector sees the client, permitted locations, permitted asset list, applicable forms, and the records relevant to that work order.

The inspector must not be able to:

- search the entire tenant asset database;
- select an asset belonging to an unrelated client or unassigned work order;
- create a canonical asset for another client while offline;
- move an asset between clients;
- alter the authoritative owner relationship;
- bypass server-issued assignment or tenant scope through a modified local database; or
- use a cloned inspection as evidence that the wrong asset was inspected.

The local database is therefore a **working cache**, not an authorization layer. The server revalidates tenant, client, work-order, asset, assignment, package, and revision constraints when queued operations are synchronized.

## 4. Adding an asset during an offline job

The inspector may encounter equipment that is not present in the assignment snapshot. The app should support this without encouraging duplicate canonical records.

The inspector creates a provisional row containing the minimum fields needed to continue the work: local candidate ID, proposed type, visible serial number, description, location, work-order link, and any required evidence. The provisional row is visibly marked **Pending office/server confirmation**.

The resulting inspection may be saved locally, but certificate issuance or final canonical linkage must follow the server’s resolution policy. When connected, the server can match the candidate to an existing asset, create a new asset if authorized, or return a conflict for office review. The app must never silently merge based only on a fuzzy serial-number match.

## 5. Administration-controlled ownership transfer

Ownership transfer is an administrative operation, not an inspector operation. A manager or authorized office user may move the asset’s current ownership relationship to another client after verifying the request.

The transfer must preserve:

| Record | Required preservation |
| --- | --- |
| Asset identity | Stable asset ID and technical history remain unchanged. |
| Previous ownership | Previous client, effective end time, actor, reason, and supporting reference remain auditable. |
| New ownership | New client, effective start time, actor, reason, and approval state are recorded. |
| Existing inspections | Historical inspections remain attached to the asset and retain their original client/owner context. |
| Certificates | Existing issued certificates remain immutable; future certificates use the effective ownership context. |
| Offline devices | Devices receive a revocation/update event so stale assignments cannot continue using the asset under an invalid client context. |

If an ownership transfer is pending while an inspector is offline, the device may finish already-authorized work only if the server’s policy permits it. On reconnect, the server decides whether the inspection remains valid, requires review, or must be blocked from issuance.

## 6. Reconnect synchronization

Whenever the app reconnects, synchronization should occur in a controlled order:

1. Upload diagnostic heartbeat and device/package status without exposing inspection content.
2. Upload queued operations using their idempotency keys and dependency order.
3. Receive authoritative acknowledgments, revisions, conflicts, revocations, and assignment changes.
4. Apply ownership, client, location, asset, and work-order updates transactionally to the local cache.
5. Download newly assigned work-order packages and required form metadata.
6. Re-evaluate local drafts whose asset, client, assignment, or package revision changed.
7. Display any blocked, conflicted, revoked, or re-review-required records to the inspector.

The device should not erase a local capture simply because the server changed the assignment. It should mark the capture for resolution and retain the evidence until an authorized server workflow determines its disposition.

## 7. Synchronization conflict examples

| Conflict | Safe result |
| --- | --- |
| Office moved the asset to another client while inspector was offline | Keep the local inspection draft; block final issuance until office/server resolution. |
| Another inspector completed a newer inspection | Keep both operation histories; server applies the defined concurrency rule and may require review. |
| Work order was reassigned | Stop new capture under the old assignment; preserve completed local work and synchronize it for manager review. |
| Asset was removed or retired | Do not delete local history. Mark new inspections blocked and show the authoritative retirement state. |
| Form package was revoked | Preserve answers and evidence, stop rendering/submission under the revoked package, and request the replacement package or office decision. |
| Local provisional asset matches an existing server asset | Link the inspection to the authoritative asset only through an explicit server response; retain the match decision in the audit trail. |

## 8. Recommendation on “sync all clients”

The idea of synchronizing all clients and all assets may appear convenient, but it should not be the default. The better design is a **two-level cache**:

| Cache level | Recommendation |
| --- | --- |
| **Assignment cache** | Required. Contains all data needed for the inspector’s active and upcoming assigned work orders. |
| **Organization-wide cache** | Not available to ordinary inspectors. Reserved for authorized administration devices or server-side dashboards, with stronger controls and audit logging. |

If the business later needs an inspector to work across many preplanned clients, the office can assign those work orders in advance and the sync service can prefetch their scoped data. This preserves offline capability without granting unrestricted asset discovery.

## Final decision

Your idea is correct in its essential principle: the inspector should work from the client’s existing asset records and should receive updates whenever the app reconnects. The refinement is that the app should synchronize **all assigned clients and their required data**, not every client and every asset in the organization. Existing assets are reused; new offline discoveries become provisional candidates; ownership transfers remain administration-controlled; and every reconnect applies authoritative assignment, asset, package, and ownership changes without deleting local evidence.

**No implementation, migration, or runtime exercise is claimed by this document.**

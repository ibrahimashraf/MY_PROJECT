# INTEGIN Refined Capability Sequence

## Decision framing

The earlier recommendations were directionally sound but too broad. They combined foundational data contracts, field interaction, workflow projection, and export features into large parallel ideas. The refined sequence below turns them into smaller **server-authoritative design units** with explicit dependencies.

> **Principle:** the asset is a stable domain anchor, but it is not a second source of truth. Work orders, inspections, evidence, certificates, custody, and corrective work remain distinct server-authoritative records connected through controlled relationships.

This is a **design-only sequence**. It authorizes no code, migration, route, runtime, scheduler, credential, OIDC, OpenBao, deployment, or protected-environment action.

## Refined design sequence

| Order | Refined capability | Why it comes here | Required design decisions | Explicit exclusion |
|---:|---|---|---|---|
| **0** | **Preserve the approved work-order foundation boundary** | Work orders are already the next logical server-authoritative domain unit. New capabilities must attach to that boundary rather than form an independent field platform. | Confirm work-order state model, assignment, work-package manifest, inspection execution, partial completion, authority, audit, isolation, and highest public test seam. | No side queue, client-owned workflow, or direct inspector-to-certificate authority path. |
| **1** | **Form-definition and evidence-binding contract** | Dynamic forms are the earliest enabler; an asset scan or work-order screen cannot safely render mutable content without a canonical definition/version/snapshot model. | Form schema, section/field identity, conditional rules, validation, evidence requirements, fixed-document bindings, published-version lifecycle, compatibility envelope, server validation, offline cache/invalidations, and diagnostic receipt. | No arbitrary scripts, unbounded expressions, client-only validation, or unversioned field additions. |
| **2** | **Assigned asset entitlement and field-work package** | Before scan, NFC, or offline interaction, the inspector must receive only the assets, forms, evidence references, and authority necessary for an assigned work order. This implements least access rather than globally cached tenant data. | Assignment-to-asset query, organization/RLS isolation, visibility interval, package manifest, offline retention/deletion, delta/supersession, revoked assignment, moved asset, conflict/review state, and local diagnostic export. | No whole-client asset replication, cross-client selection, or silent deletion of affected drafts. |
| **3** | **Asset identity and authenticated field entry** | QR/NFC becomes safe only after the entitlement boundary is defined. The physical tag is a pointer into authorised server state, not identity, authority, or evidence. | One opaque tag identity; QR/NFC payload grammar; signed or random token policy; tag state/reissue/revoke; scan-channel audit; device behavior; offline lookup; duplicate/clone handling; and tenant resolution. | No internal database IDs, customer data, raw certificate token, or authorization decision stored on a tag. No public asset detail route. |
| **4** | **Derived assurance projection and corrective work derivation** | A timeline and compliance queue must be derived from existing canonical events, not introduced as a new mutable compliance database. Corrective work should normally be a controlled work-order type or linked action, not a parallel lifecycle. | Event sources; immutable versus derived entries; due/failure semantics; correction/supersession; user-visible versus internal history; corrective work link; authority to close; RLS queries; and replay/isolation tests. | No background reminders, scheduler, notification provider, or automatic certificate action in this unit. |
| **5** | **Evidence and release-pack composition** | An auditor or customer pack is only defensible after form/evidence versioning and the assurance projection are defined. It should be a reproducible selection, not an ad-hoc folder export. | Export authority; selection manifest; record/evidence version references; redaction; PDF/certificate inclusion rules; checksum/seal policy; expiry; access audit; cancellation; retry; and rollback/replay test seam. | No blanket download of tenant history, uncontrolled ZIP generation, or mutable “final” pack. |
| **6** | **Custody, site, and handover domain extension** | Asset movement is a later operational domain. It must distinguish legal/organizational ownership from temporary custody and physical/site location. | Owner versus custodian versus responsible organization; canonical site/location; observed location; handover acceptance; condition/evidence; asset move during offline work; RLS; and dispute/reconciliation path. | No continuous location tracking, map provider, or implicit authority transfer. |
| **7** | **Sector-specific evidence graphs and bulk operations** | Materials/MTC traceability, equipment-hire cycles, UHF RFID inventory, map views, and broad certificate regimes require their own evidence and operational models. They should be added only when a real operating case and test fixture justify them. | Sector requirements, data graph, retention, authority, import/export, reader hardware, density/range pilot, and targeted acceptance evidence. | No generic “RFID support,” MTC upload bucket, or map feature without a bounded use case. |

## What changed from the earlier list

| Earlier broad addition | Refined replacement | Why the refinement matters |
|---|---|---|
| Declarative field forms | **Form-definition and evidence-binding contract** | Makes the form registry, evidence requirement, fixed-document binding, versioning, and offline compatibility first-class rather than treating layout as configuration alone. |
| Authenticated asset scan | **Assigned asset entitlement**, then **asset identity and authenticated field entry** | Prevents a scan from becoming an unscoped asset-discovery channel. Scan is deliberately downstream of the work-order entitlement. |
| Unified timeline and corrective queue | **Derived assurance projection and corrective work derivation** | Avoids a duplicate mutable queue. Compliance state is projected from canonical events and links back to controlled work-order action. |
| Audit export packs | **Evidence and release-pack composition** | Requires reproducible selection, authority, redaction, and immutable version references rather than a report download. |
| QR, NFC, RFID | **QR/NFC within asset identity; RFID as a sector-scale extension** | QR/NFC are field-entry media. RFID is justified only by measurable bulk inventory or handover throughput, hardware constraints, and a yard/fleet pilot. |
| Maps/location | **Custody/site/handover first; map later** | Ensures a map only visualizes controlled location/custody state instead of becoming a competing source of truth. |

## QR, NFC, and RFID refinement

Start with one **server-issued opaque tag identity** and support QR plus NFC as two physical encodings of that identity. The server resolves the reference under current work-order entitlement, tenant/organization isolation, tag state, and device session context. The field client records the scan medium but does not derive authority from it.

UHF RFID is a separate later decision. It should be considered only if a named yard, depot, hire-fleet, or bulk-handover process demonstrates that individual QR/NFC scans create material delay, missed capture, or revenue/compliance loss. Its future acceptance gate must cover reader hardware, read range/density, false-positive/false-negative behavior, custody semantics, tag replacement, and restricted-zone testing.

## Cross-cutting non-negotiables

| Concern | Required rule |
|---|---|
| Tenant and organization isolation | Every read, scan resolution, work package, projection, export, and custody change remains scoped by server-side tenant/organization/RLS policy. |
| Offline behavior | Offline clients act only on an assigned bounded package. They queue verifiable intent and evidence metadata, then reconcile with reviewable conflicts rather than silently overwriting canonical state. |
| Authority | Inspection completion, evidence capture, review, certificate issuance, corrective closure, and export release remain distinct authority decisions. |
| Audit | Record actor, channel, declared device context, canonical record version, form/evidence version, request/receipt, and outcome—without storing secrets or raw private content in generic logs. |
| Versioning | Forms, tags, evidence selections, derived projections, and exports must reference durable source versions and have a supersession story. |
| Testing | Every unit needs acceptance, negative authorization, tenant-isolation, replay/idempotency, offline-conflict, and rollback/recovery decisions before implementation. |

## Recommended next design artifact

The next single design artifact should be a **Work-Order Field Package and Form Contract**. It should combine Unit 1 and Unit 2 only at their controlled seam:

1. A versioned form/evidence definition selected by work-order/asset scope.
2. A server-issued, tenant-scoped field package manifest enumerating allowed assets, forms, evidence requirements, and expiry/supersession behavior.
3. A client compatibility and offline-cache contract.
4. A highest safe public test seam that proves non-leakage, revoked assignment behavior, conditional-field validation, and deterministic reconciliation.

The tag identity contract should be designed immediately afterward, once the package’s asset entitlement and client workflow are settled.

## Evidence references

1. `C:\MY_PROJECT\docs\architecture\INTEGIN_POST_MANIFEST_WORK_ORDER_FOUNDATION_SPEC_2026-08-22.md`
2. `C:\MY_PROJECT\docs\architecture\INTEGIN_METADATA_DRIVEN_FIELD_FORM_DELIVERY_2026-08-20.md`
3. `C:\MY_PROJECT\docs\architecture\INTEGIN_FIELD_ASSIGNMENT_SCOPED_ASSET_SYNC_2026-08-20.md`
4. `C:\MY_PROJECT\docs\architecture\INTEGIN_HIGH_VOLUME_WORK_ORDER_OPERATING_MODEL_DRAFT_2026-08-19.md`
5. `C:\MY_PROJECT\docs\architecture\INTEGIN_INTEGRATED_OPERATING_MODEL_FINAL_RECONCILIATION_2026-08-19.md`
6. `MOBTAG_PUBLIC_CAPABILITY_ASSESSMENT_2026-08-25.md` — public product-positioning reference only, not implementation authority.

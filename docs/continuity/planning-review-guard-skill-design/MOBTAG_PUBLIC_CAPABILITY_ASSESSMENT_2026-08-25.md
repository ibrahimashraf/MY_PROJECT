# Mobtag Public Capability Assessment

## Scope and evidence boundary

This is a **read-only public-product assessment** of `mobtag.io`, not a source-code audit, security review, procurement review, integration assessment, or implementation authorization. No Mobtag code, configuration, API, SDK, account, browser session, install artifact, or workflow was accessed. Its public pages describe product positioning; they do not prove behavior, security, availability, data residency, offline capability, API support, licensing, or compatibility.

The useful question is therefore not whether to adopt Mobtag, but which **product capabilities** are worth specifying and implementing clean-room inside INTEGIN’s server-authoritative architecture.

## Observed public capability themes

| Publicly stated theme | Evidence | Assessment confidence |
|---|---|---|
| Asset-centred inspections with checklists, sign-off, evidence, history, overdue/failed visibility, and audit exports | Compliance-inspection page | Moderate for product positioning; unverified operational behavior. |
| QR/scan as the primary route to a live asset record | Compliance and insights pages | Moderate for product positioning; no implementation or security proof. |
| Equipment identity paired with inspection records, certificates, handovers, service reports, and location/stock-site concepts | Sector and insights pages | Moderate for product positioning; scope varies by sector. |
| Configurable fields, forms, and layouts by sector | Sectors page | Moderate for product positioning; no data-model proof. |
| Support for calibration, lifting/LOLER, Ex, materials/MTC, servicing, modular handover, PAT, tool hire, regulated compliance, and hose management | Sectors page | Moderate for market coverage; unverified regime detail. |

## Capability fit for INTEGIN

| Candidate clean-room capability | Why it is relevant to INTEGIN | Fit | Required design gate before implementation |
|---|---|---|---|
| **Asset-first field entry** | INTEGIN already has tenant-scoped canonical asset facts, inspection scope, certificate snapshots, and QR-safe verification work. A private authenticated scan route can make the asset the field-work entry point for work orders, inspections, and handovers. | High | Define tag grammar, signed/opaque identifier policy, tenant resolution, device/offline behavior, revocation/reissue, and abuse controls. Do not expose internal IDs or evidence through a scan. |
| **Unified asset compliance timeline** | A server-derived timeline can join inspection results, evidence, certificate lifecycle, handovers, service events, and corrective actions without inventing a second asset truth. | High | Specify event sources, ordering, immutable versus derived entries, tenant/RLS queries, retention, and the difference between evidence history and customer-visible history. |
| **Overdue/failure/action work queue** | It supports the accepted work-order foundation by turning due/failed inspection or certificate state into assigned, auditable corrective work rather than a dashboard-only warning. | High | Define server-authoritative due-state derivation, action lifecycle, authority, partial completion, notification policy, and offline conflict handling. No scheduler or notification service is authorized by this assessment. |
| **Auditor export-pack composition** | INTEGIN has evidence retention, export foundations, certificate snapshots, and QR/PDF work. A controlled pack can produce a verifiable selection of approved evidence and lifecycle history. | High | Define export authority, immutable selection manifest, record-level redaction, checksum/seal policy, expiry, access audit, and recovery/rollback tests. |
| **Declarative field-form layout versioning** | It matches the earlier INTEGIN requirement to add fields/forms without releasing an inspector app update. The configuration must remain data, versioned, validated, and server-authoritative. | High | Complete the form-schema/template binding design: allowed field types, expressions, conditional visibility, server validation, app compatibility envelope, signature/version rollout, and offline cache invalidation. |
| **Asset location, custody, stock-site, and handover model** | This strengthens the work-order operating model for depot, site, contractor, and customer changes without conflating asset ownership with temporary field custody. | Medium-high | Define canonical location versus last-observed location; tenant and organization RLS; asset movement authority; offline reconciliation; and handover evidence. |
| **Offline-first scan and inspection ergonomics** | The public advice highlights poor-signal field use. INTEGIN already has offline-sync durability evidence, which is a stronger base than assuming a third-party product’s offline behavior. | Medium-high | Extend the accepted offline scope with a per-work-order asset entitlement bundle, local-storage boundary, conflict/review states, and reliable diagnostics/export. |
| **Materials/MTC-to-asset traceability** | Potentially valuable for industrial projects, but it is a distinct evidence graph, not merely an attachment upload. | Medium | Establish a requirements-alignment decision on material lots, certificates, joins to installed assets, revision semantics, and customer release needs. |
| **Maps and operational location views** | Useful after authoritative asset/location/custody rules exist; a map should be a view of controlled state, not a second data authority. | Medium | Define location precision, privacy, permission, offline map policy, and whether a map integration is justified. |

## What should not be added from this assessment

No external product, code, form, marketing copy, tag format, API, account, deployment, or operational workflow should be imported from Mobtag. The public pages do not evidence its security, tenancy model, RLS equivalent, audit integrity, offline reconciliation, certificate authority lifecycle, or integration contract.

The following remain outside the current approved scope: public scan access beyond the existing constrained verifier design; scheduler-driven reminders; location tracking; bulk import; mobile-app changes; map-provider selection; MTC traceability; and customer-facing release. They each require an INTEGIN-owned specification, authorization, and test seam.

## Recommended sequence

The most valuable next **design-only** additions are: first, the declarative field-form and asset-scan entry contract; second, the unified asset compliance timeline and corrective-action queue; third, the controlled audit export-pack specification. This sequence builds on existing asset, certificate, evidence, offline-sync, and work-order records without changing protected runtime state.

The public Mobtag positioning reinforces a direction INTEGIN has already identified: make the **asset, not a spreadsheet or isolated PDF, the durable index** for field work, evidence, compliance, and handover. INTEGIN should implement that principle through its existing server-authoritative, tenant-isolated, evidence-led model rather than by integrating a third-party product.

## References

1. [Mobtag homepage](https://mobtag.io/)
2. [Mobtag — Compliance inspections in Rochester](https://mobtag.io/locations/rochester/compliance-inspections)
3. [Mobtag — Sectors & use cases](https://mobtag.io/sectors)
4. [Mobtag — Insights](https://mobtag.io/insights)

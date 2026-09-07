# INTEGIN Form and Fixed-Layout Document Designer Delivery Plan

**Status:** Recommended staged implementation plan. No database migration, source-code change, or runtime activation is authorised by this plan alone.  
**Date:** 2026-08-20  
**Goal:** Deliver one governed data model that supports responsive inspection data capture and professional fixed-layout certificates without duplicate data, uncontrolled layouts, or loss of auditability.

## Confirmed design decisions

> INTEGIN will use two different presentation systems built on one controlled data and binding foundation.

| System | Purpose | Layout behaviour |
| --- | --- | --- |
| Responsive inspection-form designer | Build the inspector’s data-entry form for phone/tablet/web use | Sections may stack and respond to screen size. |
| Fixed-layout document designer | Build certificates, reports, labels, annexes, and official output | Page geometry, tables, and zones are fixed after approval. |

The fixed document designer is not a normal responsive web form. A template zone contains `cell_id`, label, binding key, geometry, style, fit/overflow rule, and visibility rule. The binding key resolves a canonical value from the tenant-scoped server-side data model.

## Corrected data-model rules

The independent verification confirmed the central design, with the following non-negotiable refinements.

| Topic | Final rule |
| --- | --- |
| Visual cell | Stored as a template-zone object, not as a new database column. |
| Real value | Stored once in its authoritative normalized record, then resolved by the binding key. |
| Binding key | Must be registered, typed, permission-aware, versioned, and namespaced; it is never SQL. |
| Checkbox question | Stores one controlled answer code, for example `yes`, `no`, or `not_applicable`, not separate Yes/No columns. |
| Conditional question | Uses versioned visibility and required rules. A hidden child has explicit applicability/capture state rather than an ambiguous missing answer. |
| Repeating sections | Use stable question-instance IDs and parent/group/order references; a static field key alone is insufficient. |
| Core decisions | High-value workflow values such as `safe_to_operate` remain dedicated inspection columns and are mapped into the binding registry. |
| Dynamic values | Store typed values with template/ruleset version, instance identity, capture state, actor, source, and timestamps. |
| Files | Store in RustFS/object storage with database references, hash, type, size, scan, retention, and tenant metadata. |
| Certificate issuance | Revalidate server-side and create a self-contained immutable snapshot. Corrections create a new certificate with a supersession link. |

## Staged implementation plan

### Stage 0 — Data and authority foundations

Create the tenant-safe base model before designing any screen. This includes client, site, work order, asset, asset component, measurement, inspection, finding, test, evidence, user, role, and file-object concepts. Define tenant and organization RLS, role boundaries, audit conventions, UUID/ULID policy, and offline row-version behavior.

**Success criteria:** A tenant-scoped API cannot load another tenant’s records; each inspection links to a work order and asset; values have canonical types/units; files are referenced with immutable hashes rather than raw URLs.

### Stage 1 — Binding Key Registry and typed value system

Build the server-owned binding registry before allowing any designer to pick a field. Each key must declare its namespace, data type, cardinality, unit policy, nullability, sensitivity, allowed template components, and deprecation state. Add value types and localized option sets with stable codes.

**Success criteria:** A designer chooses a human-readable option such as **Asset Serial Number**, which resolves to a registered key such as `core.asset.serial_number`. An unregistered, cross-tenant, wrongly typed, deprecated, or list-valued key cannot be placed in a scalar text cell.

### Stage 2 — Responsive inspection-form template engine

Implement the data-entry template lifecycle: draft, review, approved, published. Support sections, question groups, controlled field types, measurements, attachments, signatures, option sets, requiredness, and validation. Use repeaters for multiple components, assets, test rows, or findings.

**Success criteria:** An administrator can create a draft inspection form with text, date, number/unit, Yes/No/N/A, controlled selection, photo, signature, table/repeater, and conditional question types. Only an approved template version can be assigned to an inspection.

### Stage 3 — Deterministic conditional rule engine

Implement a limited declarative rule DSL for show/hide, require-if, calculate, applicability, and allowed evidence conditions. Detect circular dependencies. Version and hash every ruleset. Run the same rules offline in Flutter and revalidate them server-side in Go.

**Success criteria:** If the parent answer is `yes`, the child question appears and is required; if the parent answer is `no`, the child becomes `inapplicable` and is not an error. The server rejects a synced record that violates the approved ruleset, regardless of what the client displayed.

### Stage 4 — Offline Field capture and synchronization

Implement deterministic client-generated IDs, local storage, operation log, resumable file placeholders, revisions/ETags, tombstones, conflict records, and server reconciliation. The server remains authoritative and returns machine-readable validation/rule/merge errors.

**Success criteria:** An inspector can complete an assigned approved form offline, add repeatable rows and evidence, reconnect, and receive an exact list of any conflict or validation corrections. Another inspector or office user never receives a silent overwrite.

### Stage 5 — Fixed-layout certificate/report template designer

Build a draft-only page canvas with reusable components: identity header, client/work-order block, asset block, inspection result block, fixed/bounded table, signatory block, evidence frame, QR/verification block, footer, and annex. Each zone selects only registered binding keys and declares geometry, font, line limit, fit behavior, direction, visibility, and page-break rules.

**Success criteria:** A designer can recreate one of the uploaded certificate grids in draft mode. The system detects overlap, out-of-page zones, unknown bindings, unsafe scalar/list binding use, unapproved fonts, and invalid overflow policies before approval.

### Stage 6 — Authoritative renderer and issuance pipeline

Choose and pin a self-hosted renderer, fonts, locale/Arabic-English direction rules, QR/barcode library, and page-unit policy. The server resolves bindings, evaluates rules, selects evidence, validates fit, creates a self-contained snapshot, renders the final PDF, computes integrity metadata, and archives it.

**Success criteria:** Browser preview and server approval proof use the same renderer profile. Long text never moves other fields, overlaps, silently disappears, or changes an approved page. A failing cell identifies the exact zone and correction needed. An issued PDF always reproduces from its template/snapshot/render profile.

### Stage 7 — Certificate lifecycle, verification, and correction

Implement certificate numbering, issuance, void/revocation/supersession, signatory assignments, verification QR/reference, public disclosure policy, verification event audit, and replacement chains. Keep templates and issued snapshots immutable.

**Success criteria:** A verifier can distinguish valid, voided, and superseded certificates; a corrected certificate points to the original without rewriting it; a QR reference is never treated as proof without server verification state.

### Stage 8 — Technical extensions, scale, and operational workflows

Add load-test, lifting accessory, container, and NDT extensions; calibration/competence; renewal reminders; handover/transmittals; analytics projections; external platform readiness; retention; and exception/deviation workflows only after their owner rules are approved.

**Success criteria:** Technical template extensions use registered bindings and preserve the same issuance/verification controls. Later external submission status is separate from internal certificate issuance and is never claimed without recorded acceptance evidence.

## Fixed-layout document success rules

The following are hard requirements for official documents:

| Requirement | Required system behaviour |
| --- | --- |
| Fixed geometry | A zone cannot move, enlarge, or push another zone after template approval. |
| Text overflow | Use only the zone’s approved policy: allowed wrapping, controlled minimum font, approved display value, approved annex, or block issuance. No silent clipping. |
| Tables | Fixed, bounded-repeat, or controlled annex mode; header/row association remains intact across pages. |
| Conditional content | A hidden zone has a defined render rule: hide, show `N/A`, or render an approved placeholder. |
| Photos/signatures | Fixed frame, crop/fit/quality rule, evidence provenance, and content hash. |
| Arabic/English | Renderer pins font pack, direction policy, locale, unit and date formats in the issuance snapshot. |
| Auditability | Template version, binding registry/ruleset version, snapshot, render profile, and output file hash are retained. |

## What is not part of the first release

The first release should not attempt unlimited customer scripting, arbitrary SQL bindings, unrestricted templates, external regulator submission, autonomous AI decisions, or every possible certificate family. It should prove a small number of high-value templates end-to-end: one lifting/thorough-examination certificate and one NDT/evidence-heavy report, using approved core components and bindings.

## Recommended immediate next action

The next engineering work after the manifest gate should begin with the **work-order foundation**, then add the data and binding foundations from Stages 0–1. The visual designers should begin only after the server can safely represent assets, inspections, typed field values, repeatable components, evidence, tenant scope, binding keys, and versioned rules.

This sequencing avoids building an attractive designer that later has no trustworthy data model, no offline reconciliation, or no reproducible issuance path.

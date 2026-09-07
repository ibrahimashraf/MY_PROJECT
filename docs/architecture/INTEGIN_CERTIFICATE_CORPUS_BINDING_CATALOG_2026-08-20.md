# INTEGIN Certificate Corpus Binding Catalog

**Date:** 2026-08-20  
**Status:** Proposed architecture and binding baseline; no schema migration or product behavior change is made by this document  
**Purpose:** Convert the uploaded inspection-company certificate examples into a practical INTEGIN template-cell and database-binding catalog.

## Executive conclusion

The uploaded certificates share one strong pattern: each official document is a **fixed grid of cells** filled from a small set of repeatable business records. The system should therefore not create one database column for every visual cell and should not let templates issue free-form database queries.

INTEGIN should store three distinct things:

1. **Business records** containing the authoritative values: client, site, work order, asset, inspection, result, tests, findings, evidence, signatories, and certificate.
2. **An approved binding catalog** containing safe names such as `asset.serial_number` and `inspection.outcome`.
3. **A versioned certificate template** containing the fixed cells/zones and the selected binding key for each zone.

At issuance, the server resolves only approved binding keys against the tenant-scoped certificate aggregate, validates text/image/table fit, renders the fixed document, and preserves the final output plus a controlled issuance snapshot.

> A certificate cell stores an instruction such as “show the asset serial number here.” It does not store SQL, credentials, a cross-tenant record reference, or arbitrary executable logic.

## Evidence corpus

The analysis examined seven uploaded examples. Customer names, asset identifiers, signatures, and other instance data are intentionally not reproduced in this catalog; the analysis uses only abstracted field labels and structural patterns.

| Uploaded example | Document family | Binding implications observed |
| --- | --- | --- |
| `shecklesCert.pdf` | Thorough examination of lifting equipment | Certificate identity, order/customer/site, equipment description and ID, load/SWL history, statutory questions, defects, tests, safety decision, signatures, stamp, verification footer. |
| `WFDTBGWireSlings.pdf` | Wire-rope sling thorough examination | Asset/item table, dimensional details, end/termination, location, standard, test history, checklist, findings, decision, colour code, signatory fields. |
| `PossCont.POSS-MC-01Cert._17082026103734.pdf` | Lifting-equipment/container examination | Purchase/order/report identifiers, asset grid, weights, reference standard, load-test fields, optional NDT method/equipment mini-table, QR, stamp, controlled template footer. |
| `Hallib.ECDCont.AIS-WS-01Cert.(1).pdf` | Container, sling, and shackle visual examination | Shared visual-examination grid with equipment-specific dimensions, manufacturer, size, capacity/angle, load-test detail, preparation/authentication, QR. |
| `CONNECTOR-LINKLoadTestCert..pdf` | Load-test / thorough-examination certificate | Multiple asset IDs, bounded description, quantity, SWL/proof load, standard, checklist, corrective action, two approvals, statement. |
| `SINGLELEGWIREROPESLING.pdf` | Single-leg wire-rope sling examination | Multiple assets per certificate, sling construction/dimensions, capacity, inspection/test history, statutory checklist, corrective action, sign-off. |
| `ndt.pdf` | Liquid penetrant and magnetic particle inspection | NDT method and category, specification/acceptance, inspection area, consumables and batch/expiry, photo/contact-sheet frames, results, conclusion, client/inspector/reviewer sign-off. |

## The correct database shape

### Do not make every certificate cell a live database column

The text box at a particular visual position is **not** a business entity. For example, a cell labelled “Serial Number” should bind to one canonical asset field. If ten templates show serial number in ten different positions, they should all reuse `asset.serial_number`; the system must not create ten serial-number columns.

| Layer | Stores | Example |
| --- | --- | --- |
| Authoritative business record | The real, normalised value and its lifecycle | `asset.serial_number` stored once on an asset. |
| Binding catalog | Which safe values are available to templates | `asset.serial_number`, type `string`, allowed in a certificate identity block. |
| Template zone/cell | Where and how a selected value is shown | `binding_key=asset.serial_number`, one line, centre aligned, hard-block overflow. |
| Issued-certificate snapshot | Resolved value used at issuance and rendering manifest | The serial value used for this issued certificate, with template version. |

### Proposed logical record families

These are proposed logical names for the future INTEGIN database model. They are not claims that every table already exists in the current implementation.

| Record family | Responsibility | Typical stable identity |
| --- | --- | --- |
| `tenant` and `organization` | Isolation, ownership, role scope | Tenant/organization ID. |
| `client` | Certificate recipient/owner/employer details | Client ID. |
| `site` and `location` | Work/examination location hierarchy | Site/location ID. |
| `work_order` | Commercial/operational request, scope, assignment, partial closure | Work-order ID and human order number. |
| `asset` | Equipment/person/training subject master record | Asset ID; serial number is an attribute, not always the primary identity. |
| `inspection` | One inspection event against an asset within a work order | Inspection ID. |
| `inspection_result` / `finding` | Outcome, defect, corrective action, recommendation | Finding ID and inspection ID. |
| `test` | Load, dimensional, NDT, or other test occurrence | Test ID and test type. |
| `evidence` | Controlled photo, signature, attachment, stamp reference | Evidence ID. |
| `user` / `inspector_assignment` | Examiner, reviewer, authorizer, qualification/role | User ID plus assignment. |
| `certificate` | Issuance decision, certificate number, status, next due date, verification reference | Certificate ID and controlled certificate number. |
| `certificate_item` | A certificate/document line linked to one inspection/asset; permits one certificate to cover one or several independently identifiable items | Certificate-item ID and certificate ID. |
| `asset_component` | A fitted, attached, or nested component such as a shackle, master link, termination, or other assembly part | Component ID plus parent asset ID. |
| `template` / `template_version` | Draft/approved fixed layout, binding rules, rendering profile | Template version ID. |
| `document_render` | Authoritative render instance, page count, renderer profile, print/reprint metadata | Render ID plus certificate issue ID. |
| `certificate_snapshot` | Immutable resolved rendering input and final document linkage | Certificate ID plus immutable snapshot version. |

## Core approved binding catalog

The following catalog is the recommended shared vocabulary. A template should choose from human-readable labels, while storing the stable binding key shown in the second column.

| Designer label | Binding key | Data type | Primary record source | Recommended certificate use |
| --- | --- | --- | --- | --- |
| Certificate number | `certificate.number` | Controlled text | Certificate | One-line identity; hard block. |
| Certificate issue date | `certificate.issued_at` | Date | Certificate | Fixed formatted date. |
| Certificate status | `certificate.status` | Controlled enum | Certificate | Usually not printed; may support internal copies. |
| Next examination due | `certificate.next_due_at` | Date | Certificate | One-line date. |
| Certificate item sequence | `certificate_item.sequence` | Integer | Certificate item | Use when a document contains multiple asset lines. |
| Certificate item quantity | `certificate_item.quantity` | Integer | Certificate item | Table-cell integer; distinct from an asset’s master quantity. |
| Verification QR | `verification.qr_payload` | Generated value | Verification | Fixed QR zone, capacity checked. |
| Verification reference | `verification.reference` | Controlled text | Verification | One-line reference/URL display. |
| Template/version reference | `template.version_label` | Controlled text | Template version | Footer/audit zone. |
| Form number/revision | `template.form_number` / `template.revision` | Controlled text | Template version | Footer/audit zone. |
| Static compliance statement | `template.compliance_statement` | Approved static text | Template version | Fixed locked text; not live business data. |
| Organization legal name | `organization.legal_name` | Controlled text | Organization | Certificate provider identity. |
| Organization address/contact | `organization.certificate_contact_block` | Bounded text | Organization | Fixed provider footer/header block. |
| Accreditation/certification marks | `organization.accreditations[]` | Approved image/text collection | Organization | Fixed approved logo/accreditor zone. |
| Render timestamp | `document.rendered_at` | Timestamp | Document render | Controlled footer metadata; not an inspection date. |
| Page number / total pages | `document.page_number` / `document.total_pages` | Integer | Document render | Renderer-supplied repeated header/footer values. |
| Controlled print/reference label | `document.print_reference` | Controlled text | Document render | Optional internal/public copy marker; never a raw storage URL. |
| Client legal name | `client.legal_name` | Text | Client | One/two-line bounded field. |
| Client certificate address | `client.certificate_address` | Address | Client | Bounded multi-line zone. |
| Work-order number | `work_order.number` | Controlled text | Work order | One-line identity. |
| Purchase/order reference | `work_order.client_reference` | Text | Work order | One-line identity. |
| Site name | `site.name` | Text | Site | One/two-line bounded field. |
| Examination location | `inspection.location_display` | Text | Inspection/location | One/two-line bounded field. |
| Asset ID | `asset.asset_id` | Controlled text | Asset | One-line; hard block. |
| Asset serial number | `asset.serial_number` | Text | Asset | One-line; hard block. |
| Asset type | `asset.type_display_name` | Controlled text | Asset/type catalog | One-line or small label. |
| Certificate display description | `asset.certificate_display_description` | Controlled text | Asset | Bounded multi-line. It is not a silently shortened source value. |
| Manufacturer | `asset.manufacturer` | Text | Asset | One/two lines. |
| Model | `asset.model` | Text | Asset | One line. |
| Manufacture date | `asset.manufactured_on` | Date/partial date | Asset | Fixed date/partial-date format. |
| Quantity | `asset.quantity` | Integer | Asset/inspection item | Table-cell integer. |
| Parent asset | `asset.parent_asset_id` | Controlled identifier | Asset | Used where a certificate item is part of an assembly. |
| Fitted components | `asset.components[]` | Repeating collection | Asset component | Bounded nested table/list; never an unstructured hidden serial list. |
| Component identifier | `asset_component.identifier` | Controlled text | Asset component | One-line item/serial/asset identifier. |
| Component type | `asset_component.type_display_name` | Controlled text | Asset component | One-line component classification. |
| Component capacity | `asset_component.rated_capacity` | Measurement | Asset component | Numeric value plus canonical/display unit. |
| Inspection type | `inspection.type_display_name` | Controlled text | Inspection template/type | One line. |
| Inspection date | `inspection.completed_at` | Date | Inspection | Fixed formatted date. |
| Inspection outcome | `inspection.outcome` | Controlled enum | Inspection | Accepted/rejected/result zone. |
| Safe-to-operate decision | `inspection.safe_to_operate` | Boolean/enum | Inspection decision | Fixed decision cell; no free text. |
| Applicable standard(s) | `inspection.applicable_standards[]` | Controlled collection | Inspection/template | Bounded list or controlled table. |
| Inspection summary | `inspection.summary` | Bounded text | Inspection | Approved narrative zone. |
| Next action/recommendation | `inspection.recommendation` | Bounded text | Inspection/finding | Bounded text or annex. |
| Defects/findings | `inspection.findings[]` | Repeating collection | Findings | Fixed/bounded table or approved annex. |
| Corrective action | `finding.corrective_action` | Bounded text | Finding | Fixed narrative zone or annex. |
| Inspector name | `signatory.inspector.name` | Text | User/assignment | One-line signatory cell. |
| Inspector qualification | `signatory.inspector.qualification` | Text | User/qualification | One/two lines. |
| Inspector signature | `signatory.inspector.signature` | Signature asset | Evidence/signatory | Fixed signature frame. |
| Reviewer/authorizer name | `signatory.authorizer.name` | Text | User/assignment | One-line signatory cell. |
| Reviewer/authorizer signature | `signatory.authorizer.signature` | Signature asset | Evidence/signatory | Fixed signature frame. |
| Role-indexed signatory | `signatory[role].name` / `signatory[role].qualification` / `signatory[role].signature` / `signatory[role].signed_at` | Role-indexed structured record | Signatory assignment | Supports inspector, issuer, authorizer, reviewer, customer witness, or other explicitly approved roles without template-specific database columns. |
| Stamp/seal | `organization.approval_stamp` | Image asset | Organization | Fixed stamp frame; no external URL. |

### Binding display rules for measurements, collections, and repeated pages

The corpus contains values such as quantity, capacity, safe working load, proof load, dimensions, length, weight, and angle. These must not be represented as arbitrary combined strings in the authoritative record. Each measurement binding resolves from a structured value, unit, precision, and display policy. For example, `lifting.safe_working_load` may resolve as `value=18.37`, `canonical_unit=tonne`, `display_unit=tonne`, and a template-defined display precision.

| Binding pattern | Meaning | Document control |
| --- | --- | --- |
| `measurement.value` plus `measurement.unit` | Scalar technical value and unit | Renderer formats as one approved display; numeric alignment/precision is template controlled. |
| `certificate.items[]` | Repeating certificate-item collection | Table rows remain associated with their linked asset/inspection; table header repeats after a controlled page break. |
| `asset.components[]` | Nested component collection | Renders only inside an approved nested list/table or annex. |
| `signatories[role]` | Role-indexed signatory collection | Only template-approved roles render; each name remains bound to its own signature and qualification. |
| `document.page_number` / `document.total_pages` | Render-time page variables | Repeated only in approved header/footer zones; not persisted as business data on a template. |

The template model must distinguish **static approved template content**, **organization/provider identity**, **resolved business-record data**, and **render-time document metadata**. A compliance statement is normally static template content. A company address is organization data. An inspection date is business-record data. A page number is renderer metadata. Keeping these sources separate prevents accidental historical drift.

## Required inspection and assessment bindings

The lifting-equipment examples repeatedly use a statutory/examination-basis questionnaire. INTEGIN should model answers as controlled named assessments, not a unique column for every visual yes/no cell.

| Designer label | Binding key | Type | Notes |
| --- | --- | --- | --- |
| First examination after installation | `assessment.first_examination_after_installation` | Boolean/enum | `yes`, `no`, `not_applicable`, with explicit display policy. |
| Installed correctly | `assessment.installed_correctly` | Boolean/enum | May be conditionally required after the first question. |
| Within six months | `assessment.within_six_months` | Boolean/enum | Template-defined question. |
| Within twelve months | `assessment.within_twelve_months` | Boolean/enum | Template-defined question. |
| In accordance with examination scheme | `assessment.in_accordance_with_scheme` | Boolean/enum | Template-defined question. |
| Exceptional circumstances | `assessment.after_exceptional_circumstances` | Boolean/enum | Template-defined question. |
| Immediate danger | `assessment.immediate_danger` | Boolean/enum | Safety-critical, must bind to authoritative decision logic. |
| Defect narrative | `assessment.defect_summary` | Bounded text | Usually derived from findings, never hand-copied into a second source of truth. |
| Repair/renewal/alteration required | `assessment.remedial_action_summary` | Bounded text | Derived from finding recommendations. |
| Tests carried out | `assessment.tests_summary` | Bounded text | Derived from test records. |

## Inspection-type extensions

The core catalog should not be overloaded with every technical attribute. Each inspection template/type can expose an extension namespace.

### Lifting accessories and wire-rope slings

| Binding key | Typical source | Expected cell/layout |
| --- | --- | --- |
| `lifting.safe_working_load` | Asset/specification or inspection result | Numeric plus unit; may be angle-qualified. |
| `lifting.safe_working_load_at_angle` | Capacity rule/result | Structured numeric, unit, and angle. |
| `lifting.proof_load` | Test record | Numeric plus unit. |
| `lifting.leg_count` | Asset specification | Integer. |
| `lifting.rope_diameter` | Asset specification/measurement | Measurement plus unit. |
| `lifting.rope_length` | Asset specification/measurement | Measurement plus unit. |
| `lifting.rope_construction` | Asset specification | Bounded text or controlled catalog value. |
| `lifting.termination_type` | Asset specification | Controlled value, such as hard eye/ferrule/thimble. |
| `lifting.end_configuration` | Asset specification | Bounded text or controlled value. |
| `lifting.colour_code` | Inspection/certification program | Controlled colour/status value. |
| `lifting.last_load_test` | Test history | Structured test date/certificate/reference. |

### Shackles and connector links

| Binding key | Typical source | Expected cell/layout |
| --- | --- | --- |
| `connector.link_type` | Asset type/specification | Controlled text. |
| `shackle.type` | Asset type/specification | Controlled text. |
| `shackle.size` | Asset specification | Measurement/text. |
| `shackle.grade` | Asset specification | Controlled text. |
| `connector.dimensions[]` | Asset measurements | Compact dimension row or bounded table. |
| `connector.fabrication_description` | Asset specification | Bounded text. |
| `connector.proof_load` | Test record | Numeric plus unit. |

### Containers and offshore containers

| Binding key | Typical source | Expected cell/layout |
| --- | --- | --- |
| `container.dimensions` | Asset specification | Length/width/height structured values. |
| `container.tare_weight` | Asset specification | Numeric plus unit. |
| `container.gross_mass` | Asset specification | Numeric plus unit. |
| `container.attachment_location` | Asset/inspection | Controlled text. |
| `container.container_conformity` | Asset/certification record | Standard/reference text. |
| `container.load_test_certificate` | Test history | Certificate number/date/reference. |

### Load testing

| Binding key | Typical source | Expected cell/layout |
| --- | --- | --- |
| `load_test.required_load` | Test plan | Numeric plus unit. |
| `load_test.applied_load` | Test result | Numeric plus unit. |
| `load_test.test_date` | Test record | Date. |
| `load_test.test_certificate_number` | Test record | One-line reference. |
| `load_test.tested_by` | Test record/user | Controlled person/organization display. |
| `load_test.result` | Test record | Controlled outcome. |
| `load_test.observations` | Test record | Bounded text or annex. |

### Non-destructive testing

| Binding key | Typical source | Expected cell/layout |
| --- | --- | --- |
| `ndt.method` | NDT inspection record | Controlled method: PT, MT, etc. |
| `ndt.procedure_reference` | NDT procedure | Controlled reference. |
| `ndt.specification` | NDT inspection/template | Controlled standard/specification. |
| `ndt.acceptance_criteria` | NDT inspection/template | Controlled text/reference. |
| `ndt.inspection_area` | NDT inspection | Controlled choice/checklist. |
| `ndt.condition` | NDT inspection | Controlled enum. |
| `ndt.consumables[]` | NDT consumables rows | Product, batch, expiry, role; controlled repeating table. |
| `ndt.equipment[]` | NDT equipment rows | Equipment, serial, calibration/qualification due date. |
| `ndt.photos[]` | Evidence selection | Fixed photo frames or controlled annex contact sheet. |
| `ndt.conclusion` | NDT result | Bounded conclusion. |
| `ndt.result` | NDT result | Controlled accepted/rejected/other outcome. |
| `ndt.reviewer` | Assignment/signatory | Technical review signatory. |

## How a template cell is stored

The template should store an approved binding instruction and presentation rule. This is illustrative JSON, not an executable query or a final database schema.

```json
{
  "zone_id": "asset-serial",
  "component": "text_cell",
  "binding_key": "asset.serial_number",
  "page": 1,
  "bounds_mm": { "x": 117, "y": 54, "width": 52, "height": 6 },
  "format": { "align": "center", "direction": "auto", "font": "template_primary" },
  "fit_policy": { "max_lines": 1, "min_font_pt": 8, "on_overflow": "block_issuance" },
  "visibility": { "when": "always" }
}
```

For a repeating fixed table, a zone binds to a controlled collection rather than embedding many individual record fields:

```json
{
  "zone_id": "finding-table",
  "component": "bounded_table",
  "binding_key": "inspection.findings[]",
  "max_rows": 4,
  "columns": ["finding.code", "finding.description", "finding.severity", "finding.corrective_action"],
  "overflow": { "on_excess_rows": "approved_annex", "annex_template": "findings-annex-v1" }
}
```

For a multi-page certificate or batch print, template logic must use controlled scope rather than visually repeating arbitrary content. A document declares whether it is `single_asset`, `multi_item_certificate`, or `batch_print`. The renderer then repeats only the approved header/footer zones, prevents a certificate-item row from splitting across pages unless its zone explicitly allows continuation, and maintains page-specific `document.page_number` and `document.total_pages` bindings.

## Server-authoritative issuance contract

1. The server receives an authorised request to issue a certificate for a specific certificate/inspection context.
2. It resolves tenant, organization, work order, client, site, asset, inspection, tests, findings, evidence, signatories, and template version using server-side tenant scope.
3. It builds a **read-only certificate snapshot** containing only catalog-approved binding values.
4. The renderer reads the approved template’s binding keys from that snapshot. It never receives unrestricted database access.
5. Preflight validates every zone: data type, visibility, font availability, language direction, fit, table count, QR capacity, photo quality, asset scope, and annex limit.
6. On any failure, issuance stops with an exact zone/error description. No value is silently hidden, deleted, or moved.
7. On success, the server persists the certificate number, approved template version, rendering-manifest identifiers, final output, verification reference, audit event, and immutable issuance snapshot/reference.

## Recommended physical persistence model

The future schema may use normalised tables, a controlled JSON layout document, or both. The recommended split is:

| Proposed persistence object | Stores | Important constraint |
| --- | --- | --- |
| `certificate_template` | Tenant-scoped template identity and business name | No cross-tenant reuse without an explicit owner-controlled copy/share process. |
| `certificate_template_version` | Draft/review/approved state, fixed page definition, renderer profile, form/revision metadata | Approved versions immutable. |
| `template_zone` or versioned `layout_json` | Bounds, component type, binding key, fit/visibility/overflow rules | Binding key must exist in the server-approved catalog. |
| `binding_catalog` | Key, data type, allowed component types, data-sensitivity class, supported inspection types | Server-owned, versioned, no SQL expressions. |
| Normalised business tables | Client, work order, asset, inspection, finding, test, NDT, evidence, signatory | Canonical source of values. |
| `certificate_issue` | Number, status, template version, linked inspection/work order, verification reference | Server-authoritative issuance state. |
| `certificate_issue_item` | Ordered link between a certificate issue and its source asset/inspection/item scope | Prevents a multi-line or multi-item layout from losing its source relationship. |
| `document_render` | Renderer version/profile, page count, render/print time, optional controlled reprint marker | Supports deterministic archived output without treating print metadata as inspection data. |
| `certificate_snapshot` | Immutable resolved values, selected evidence IDs, locale/renderer profile | Protected from later live-record edits. |
| `issued_document` | Final PDF/file checksum, archive metadata, page count | Does not permit post-issuance visual mutation. |

## What must remain a controlled selection in the designer

The designer should offer approved labels and component-specific choices. It must not ask designers to type database field names or query syntax.

| Designer operation | Safe UI choice |
| --- | --- |
| Add a text cell | Choose “Asset Serial Number,” “Client Legal Name,” or another approved label from the binding catalog. |
| Add a result cell | Choose an approved controlled outcome binding such as “Safe to Operate.” |
| Add an asset table | Choose a collection such as “Inspection Assets” and select approved columns. |
| Add NDT photos | Choose “Selected NDT Evidence Photos” and a fixed frame/contact-sheet policy. |
| Add a signatory | Choose “Assigned Inspector” or “Authorizing Reviewer,” with controlled signature/qualification fields. |
| Change long-content behavior | Select allowed lines, hard block, approved display field, or a designated annex—not arbitrary clipping. |

## Next bounded product design actions

1. Confirm which proposed record families will be introduced in the work-order foundation and which remain later phases.
2. Define the first server-owned binding catalog as a Go type/schema, beginning with the shared core bindings.
3. Model a small set of first template components: identity header, client/work-order block, asset block, result block, sign-off block, fixed finding table, and QR/footer block.
4. Define tenant-scoped certificate snapshot construction and issued-document archival semantics before a renderer is connected.
5. Define certificate-item and asset-component relationships before implementing multi-asset certificates or nested lifting assemblies.
6. Define a canonical measurement/unit/precision service before connecting capacity, dimensions, weights, or load-test fields to templates.
7. Prototype one lifting certificate and one NDT certificate using mock schema values only after the underlying data contracts are approved.
8. Add template test cases for one-line hard blocks, bounded descriptions, table limits, nested components, repeated page metadata, image frames, QR capacity, Arabic/English direction, and controlled annexes.

## Third-pass systems-level certificate assurance model

The visual certificate cells are only the presentation layer. The certificate itself also needs non-visible source records that establish whether it may be issued, whether it remains valid, what it replaced, how it was rendered, and what evidence supports it. These are **not** all template cells. Most remain server-controlled operational metadata and appear only through limited conditional labels, annexes, or public verification results.

| Assurance family | Authoritative source record purpose | Optional/conditional document presentation | Suggested priority |
| --- | --- | --- | --- |
| Certificate lifecycle | Draft/review/approved/issued/void/superseded state, transition reason, time, actor role, and lock scope | Conditional status, withdrawal, or supersession note | Foundation now |
| Reissue/supersession chain | Original/replacement relation, continuity chain, effective date, and reason | “Superseded by” or replacement reference only when applicable | Foundation now |
| Issuance authority | Issuing body, signatory authority/delegation scope, approval route, seal selection, and sign-on-behalf context | Authority text or approved seal if the template needs it | Foundation |
| Verification events | Verification status, method, public disclosure policy, offline token scope, and verification-event audit | QR/reference and a minimal verification instruction only | Foundation |
| Evidence provenance | Evidence origin, collector/collection time, original checksum, transformation/redaction history, inclusion rationale, and review decision | Optional evidence index or annex only | Foundation |
| Instrument calibration and competence | Instrument identity, calibration certificate/reference/validity, uncertainty class, operator competence, and supervision context | Short calibration/competence statement or annex only when required by the certificate type | Foundation |
| Document integrity | Canonicalisation/render profile, render hash, signature/timestamp reference where separately implemented, watermark policy, and attested page count | Integrity mark or signed-document statement only when the defined policy supports it | Foundation |
| Data lineage | Template version, snapshot identity, source-record revision references, diff/reissue context, and read-only reason | Optional revision note | Foundation |
| Policy context | Tenant policy-set, language, locale, unit system, privacy treatment, and permitted jurisdiction/scheme context | Locale/unit/jurisdiction badge only where required | Foundation |
| Offline capture integrity | Field session/device authority reference, local-clock treatment, sync conflict outcome, and reconciliation state | Not printed on the certificate | Foundation |
| Retention and archival | Retention policy, legal hold, archival location/class, purge protection, and preservation state | Not printed on the certificate | Foundation |
| External submission readiness | Scheme, mapping version, missing required fields, export package, external filing state, and receipt reference | Optional scheme/compliance mark only after the external workflow records a valid state | Later |
| Expiry notifications | Recipient group, trigger, schedule, throttling, delivery outcome, and acknowledgement state | Not printed on the certificate | Later |
| Operational handover | Distribution/transmittal, recipient acknowledgement, handover state, and package manifest relationship | Separate transmittal/cover sheet, not normally the certificate body | Later |
| Deviation/waiver | Deviation class, waiver/risk-acceptance reference, concession expiry, and linked non-conformance | Conditional deviation note or annex | Later |
| Confidential annex control | Annex visibility scope, redaction rules, separate distribution, and content index | Conditional annex placeholder only | Optional |

### Explicit separation of source records and template cells

The template must not treat every assurance item as a visible field. The server owns the full lifecycle, authority, verification, evidence, calibration, retention, and lineage state. Templates show only the approved presentation required by a particular certificate type.

| What the server stores | What a template may show |
| --- | --- |
| Full lifecycle transition history, including who/why/when | Current valid/superseded/withdrawn message where applicable. |
| Full evidence provenance and review record | Selected evidence frames, a reference, or an annex index. |
| Calibration/competence record set | A concise required statement or selected reference. |
| Verification-event history and disclosure policy | QR/reference and public verification outcome. |
| Renderer/profile/hash/attestation data | A policy-approved integrity statement, not internal hashes by default. |
| Notification delivery and handover logs | Normally nothing on the certificate; use separate operational screens/documents. |

### Important non-equivalences

The following distinctions should remain explicit in the data model:

1. A visible signature image is not, by itself, proof of issuance authority.
2. Printed status text is not the lifecycle state machine.
3. A QR/reference alone does not establish validity; the server verification result does.
4. Evidence file names or captions are not evidence provenance.
5. An instrument identifier is not proof that calibration was valid at the inspection time.
6. A due date is not a retention policy.
7. A sent email is not a tracked notification or recipient acknowledgement.
8. A render hash is not automatically a legal digital signature or trusted timestamp.
9. A tenant is not automatically a jurisdiction or external certification scheme.
10. An issued document is not automatically an externally accepted submission.
11. Work-order closure is not automatically operational handover completion.

These additions stay inside the product’s existing constraints: the server remains authoritative; AI, when later introduced, is advisory only; templates receive a limited tenant-scoped snapshot; and no persistent package enforcement or external-regulator workflow is activated by this catalog.

## Conclusion

The uploaded examples do not require a different database per certificate style. They demonstrate a stable common record set plus carefully defined technical extensions. INTEGIN should build the binding catalog around canonical records—especially work order, asset, inspection, test, finding, evidence, signatory, certificate, and template version—then let each fixed grid cell select a safe binding from that catalog. This creates flexible templates without compromising tenant isolation, certificate reproducibility, or the rule that official pages never reflow unexpectedly.

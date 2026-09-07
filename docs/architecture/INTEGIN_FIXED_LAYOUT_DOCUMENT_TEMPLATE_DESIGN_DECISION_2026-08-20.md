# INTEGIN Fixed-Layout Certificate and Report Template Designer

**Decision date:** 2026-08-20  
**Decision status:** Adopted as a post-manifest design baseline; implementation remains separately planned  
**Decision owner:** INTEGIN product owner, with owner-side engineering reconciliation  
**Scope:** Official certificates, reports, annexes, labels, and similar fixed-layout output documents

## Decision

INTEGIN will use a **fixed-page, component-and-zone document model** for official certificates and reports. The designer may be a visual drafting tool, but the approved template artifact will be a constrained, versioned, declarative model—not arbitrary HTML, user JavaScript, or an unconstrained collection of floating text boxes.

> A draft may be edited freely. An approved template version becomes immutable. Each issued document uses one exact approved template version and may fill data only within its approved zones.

The system will keep **responsive inspection data-entry forms** separate from **fixed-layout document templates**. React/TypeScript will provide the operator/designer experience, Flutter/Dart will render approved data-entry forms for Field use, and Go/PostgreSQL will remain the authority for template validation, approval, publication, version selection, issuance, and tenant scoping.

## Debate result

Four independent review lenses considered three options:

| Option | Outcome | Reason |
| --- | --- | --- |
| A. Generic absolute-position canvas of independent boxes | Rejected as the authoritative model | It can be useful as a visual drafting surface, but alone cannot reliably express tables, bounded cells, fit policies, data bindings, or unpublishable invalid states. |
| B. Fixed-page component-and-zone model with controlled annexes | **Adopted conditionally** | It directly supports fixed geometry, gridded certificates, deterministic validation, versioning, and safe handling of allowed variable-length content. |
| C. Responsive HTML-first certificate layout | Rejected for official output | Natural reflow and expansion conflict with the required fixed geometry and non-overlap rules. |

Architecture, document-assurance, and field/operator-UX lenses recommended Option B. The red-team lens accepted it only with pinned rendering dependencies, authoritative server preflight, strict asset/font governance, and explicit hard publication blocks.

## Model

Every template uses an approved page size such as A4 portrait, A4 landscape, A5, or a defined label format. The main body has fixed page geometry. The visual designer provides reusable components instead of forcing the owner to build every certificate from independent rectangles.

| Component family | Examples | Layout behavior |
| --- | --- | --- |
| Header and identity | Certificate number, logo, QR, issue date, revision | Fixed zone; bounded text and QR capacity. |
| Client and work order | Client, site, contact, job/order reference | Fixed fields or controlled multi-line cells. |
| Asset identity | Asset ID, serial, manufacturer, model, capacity, description | Bound to canonical asset/inspection data with explicit fit rules. |
| Inspection summary | Inspection type, dates, result, inspector, next due date | Fixed structured grid. |
| Inspection/test tables | Checkpoints, measurements, pass/fail entries, test scope | Fixed, bounded-repeat, or annex-capable table mode. |
| Findings and evidence | Defect reason, corrective action, photos, signatures | Conditional approved zones only; fixed photo frames and crop rules. |
| Approval and verification | Inspector/reviewer sign-off, stamp, verification QR | Fixed zone and controlled assets. |
| Footer/legal content | Fixed declarations, company details, page/version data | Fixed immutable placement. |

Each component contains bounded zones or table cells. A zone declares its position, size, data binding, direction, alignment, font policy, maximum lines, allowed overflow policy, visibility rule, and validation behavior. A zone can never move another zone, expand a fixed page, or silently draw outside its boundary.

## Drafting and publication lifecycle

1. A tenant-authorised designer creates or edits a **draft** fixed-page template.
2. The designer may move, resize, add, remove, and configure components during draft work.
3. The system runs structural validation, data-binding validation, collision checks, font/asset checks, and stress rendering with representative long and bilingual values.
4. An authorised reviewer approves a new immutable template version.
5. The server publishes that version as the only selectable artifact for future issuance.
6. Issued certificates retain their template version, rendering manifest, final output, and required audit linkage.
7. Any later visual or rule change creates a new draft version; it never rewrites an issued document or its approved template version.

## Text, table, and overflow contract

For every content zone, the template author explicitly chooses its approved behavior. The default for identity, result, and other material certificate information is **hard failure before issuance** when the value cannot fit.

| Content type | Recommended bounded behavior |
| --- | --- |
| Certificate number, asset ID, serial number | One line; fixed or minimally bounded font; hard fail if it does not fit. |
| Client name and short descriptions | Explicit maximum line count; wrapping only inside the zone; hard fail after the allowed capacity. |
| Equipment description | Bounded multi-line zone; optional controlled display-shortening only where the complete canonical value remains retained and the template policy permits it. |
| Result, dates, status, and standard references | Controlled values and formatting; no free-text expansion. |
| Photos/signatures | Fixed frame, minimum effective resolution, explicit crop/fit/rotation policy. |
| Fixed inspection table | Fixed rows and cell limits; never changes main-page geometry. |
| Bounded repeating table | Uses only the approved number of rows; fails or moves to an approved annex after the limit. |
| Annex-capable content | May continue only in a designated annex with page caps, ordering, headers, and separate fixed layout. It can never change main-certificate page geometry. |

No issued document may silently clip content, substitute a font, auto-shrink below its approved minimum, permit an unapproved line wrap, or reflow the main certificate page. The system must identify the exact failing zone before issuance.

## Deterministic rendering and bilingual support

The final PDF/archive output is server-rendered through one self-hosted, pinned rendering stack. Browser preview is useful for editing, but **approval and issuance trust the server-rendered proof**, not browser layout alone.

The approved template version must pin or reference versioned rendering dependencies that influence layout: page units, renderer release, font files, locale rules, Arabic/English directionality, numeral policy, embedded assets, QR policy, and permitted color/transparency behavior. Fonts must be whitelisted and embedded; missing glyphs or fallback font use must fail preflight. The renderer must normalise measurement units and provide configured tolerance checks for out-of-bounds content, overlap, thin strokes, and print-safe QR modules.

## Mandatory publication blocks

Publication or issuance must stop if any of the following occurs:

| Category | Hard-stop condition |
| --- | --- |
| Geometry | Overlapping zones, out-of-page bounds, disallowed layer overlap, or zone-boundary violation. |
| Content fit | A value exceeds permitted lines/size, uses forbidden wrapping, or would shrink below the approved minimum. |
| Data binding | Missing, incompatible, cross-tenant, or unversioned binding. |
| Fonts and language | Missing glyph, fallback font, unapproved BiDi control, unsupported locale behavior, or post-normalisation overflow. |
| Image/QR/signature | Insufficient resolution, unsupported orientation/format, incorrect crop, unfit QR capacity, or unsafe external reference. |
| Annex | Annex use is not approved for the zone, exceeds page cap, or would affect main-page geometry. |
| Renderer integrity | Non-pinned renderer/font/asset dependency, preview-to-server mismatch, or rendering drift. |
| Authority | Template is draft, deprecated, cross-tenant, unapproved, or not selected by the server-authoritative issuance workflow. |

There is no silent operator override for these blocks. Any future exceptional override process must be separately designed, role-controlled, auditable, and must never modify the stored issued output.

## Rejected shortcuts

The design explicitly rejects arbitrary template JavaScript, externally loaded assets or fonts, browser-only final rendering, independent tenant-unscoped asset caches, generic responsive reflow as certificate authority, automatic page growth in the fixed main body, and editing an approved version in place.

## Residual risks and next evidence

The decision is a design baseline, not proof that a renderer has been built. The highest residual risks are Arabic/English mixed-direction shaping, font metrics and fallback behavior, table pagination, QR/image capacity, printer scaling, renderer upgrades, and exact historical reproducibility.

The next bounded design/engineering actions are:

1. Define the canonical versioned template schema and data-binding contract.
2. Select and evaluate a self-hosted deterministic rendering stack with a representative certificate corpus.
3. Define authoritative server-rendered preview and preflight behavior.
4. Create a bilingual stress corpus for long text, mixed directions, tables, QR values, photos, and signatures.
5. Specify golden-render, no-overlap, no-clip, font, QR, annex, tenant-isolation, and immutability tests.
6. Define issued-artifact archival and renderer/font upgrade migration rules.

No product code, certificate data, runtime, enforcement state, OIDC setting, OpenBao state, or external integration is changed by this decision.

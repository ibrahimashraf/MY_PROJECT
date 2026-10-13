# INTEGIN Certificate-Template Binding Registry Decision

## Decision

Certificate templates are versioned server-side definitions. A template does **not** contain executable SQL, scripts, URLs, arbitrary object paths, or client-supplied data selectors. Each visible cell binds only to an approved, stable registry key. The key identifies a canonical INTEGIN fact; it does not create a database column or require a Field application release.

> A certificate cell is a bounded presentation slot. A binding key is the approved server-side name of the fact that may occupy that slot.

The first implementation prepares and validates templates; it does **not** issue certificates, generate PDFs, expose a visual designer, introduce unrestricted custom expressions, or claim that presently unavailable asset fields can already be resolved.

## Canonical binding catalog

The registry owns an explicit allow-list. Initial resolvable keys map only to the current `inspection_record` fields:

| Key | Current canonical source | Use |
|---|---|---|
| `inspection.id` | `inspection_record.id` | Internal/certificate inspection reference. |
| `inspection.work_order_id` | `inspection_record.work_order_id` | Job/work-order reference. |
| `inspection.asset_id` | `inspection_record.asset_id` | Stable current asset identifier. |
| `inspection.inspector_id` | `inspection_record.inspector_id` | Inspector reference; display-name projection is a later identity boundary. |
| `inspection.lifecycle_state` | `inspection_record.lifecycle_state` | Status label, subject to future issuance rules. |
| `inspection.revision` | `inspection_record.revision` | Revision traceability. |
| `inspection.finalization_state` | `inspection_record.finalization_state` | Finalization traceability. |
| `inspection.created_at`, `inspection.updated_at` | `inspection_record` timestamps | Controlled dates/times. |

The owner’s required future cells—asset serial number, full description, model/manufacturer, owner/client, location, inspection type, test details, certificate number, certificate validity, signatory identity, and QR verification reference—remain required registry keys but are deliberately **unavailable for template approval** until a canonical authoritative source and resolver are introduced. The present inspection persistence schema does not contain those values; creating a template that appears to bind them would be a false claim.

## Template and cell model

A template identity is `(tenant_id, organization_id, template_code, version)`. Versions are immutable once approved. Correcting a published layout creates a new version; a future issued certificate retains the exact approved template snapshot and binding catalog revision used at issuance.

Each cell declares a stable `cell_id`, the approved `binding_key`, a presentation kind, geometry in fixed PDF points, optional label/format, applicability rule, requiredness, and a bounded fitting policy. The initial presentation kinds are `TEXT`, `CHECKBOX`, `DATE`, `STATIC_TEXT`, and `REPEATING_REGION`. Every cell rectangle must be positive, within its page boundary, and must not overlap another visible cell unless the template explicitly defines the overlap as a future supported layer. The first registry rejects unapproved overlaps.

## Bounded values and overflow policy

Cells never grow, push other fields, or silently delete data. Text cells are bounded by their rectangle and selected policy:

| Policy | Behavior | Allowed initial use |
|---|---|---|
| `WRAP_REQUIRED` | Wrap only up to an explicit maximum line count; fail validation/render-readiness if text still does not fit. | Labels and approved descriptive blocks. |
| `SINGLE_LINE_REQUIRED` | One line only; fail readiness when the formatted value exceeds the approved cell capacity. | Serial numbers, certificate references, dates, short IDs. |
| `CHECKBOX_MAP` | Map an approved boolean/enum value to fixed marks or labels. | Pass/fail and yes/no fields. |
| `REPEAT_REQUIRED` | Place a bounded list in a defined repeating region; fail readiness when entries exceed the approved region maximum. | Finding/test rows. |

`CLIP`, silent truncation, automatic deletion, and unconstrained font shrinking are not allowed for certificate-bound values in the first implementation. A later explicitly approved policy may support ellipsis or controlled shrink only for non-authoritative decorative text, never for identity, result, test, validity, or verification facts.

## Applicability and repeat behavior

Applicability is declarative and server-validated. It is expressed as an approved key, operator, and literal value; free-form scripts, SQL, and arbitrary template expression strings are rejected at the certificate binding boundary. A cell that is not applicable is absent according to its rule; a required applicable cell with an unavailable or empty canonical value blocks certificate readiness. Repeating regions use an explicit item source from a future approved catalog and an explicit maximum; excess rows block readiness rather than overflow to a hidden page or silently disappear.

## Security, authority, and rollout

Only a tenant-scoped template administrator may create drafts; a later certificate-template approval authority may approve a version. Inspector clients consume published form/template definitions but cannot mutate binding catalogs, layout, approval, or renderer policy. The Field app will synchronize metadata-driven template definitions; routine new cells/key mappings therefore do not require an app release. A new mobile release is required only when a template uses a presentation kind the installed app does not declare as supported.

The registry resolver receives server-derived tenant and organization scope and an inspection ID. It selects only the allow-listed canonical fields in an RLS-scoped transaction. It never accepts a value, tenant, organization, asset, inspector, or SQL selector from a certificate request body.

## Acceptance requirements

The next implementation must prove version identity and immutability; tenant/organization isolation; duplicate-cell and overlap rejection; unknown/unavailable binding-key rejection; unsupported fitting and checkbox-map rejection; applicability validation; canonical inspection-only resolution; and zero residue from namespaced integration fixtures. No pilot migration may be applied without a verified backup and isolated up/down SQL review.

## Implementation and controlled-proof record — 2026-08-21

The registry implementation is complete. `internal/domain/certificatetemplate` owns catalog validation, immutable definition rules, layout and fitting validation, and canonical-value resolution. `internal/certtemplatepg` persists drafts and approvals in forced-RLS tables and resolves only approved `inspection_record` values under server-derived actor scope. Migration `0012_certificate_template_binding_registry.candidate.sql` was applied only after the previously recorded verified backup and disposable-copy review.

Focused package checks passed after formatting: `go test ./internal/domain/certificatetemplate ./internal/certtemplatepg`. The controlled pilot proof `TestPostgresResolveCanonicalInspectionTemplateIntegration` passed canonical inspection resolution and cross-organization denial. A post-run count across `certificate_template` and `inspection_record` confirmed zero `it-certificate-template-resolver-*` fixtures. Direct repository-wide `go test -count=1 ./...` and `go vet ./...` passed afterward.

This proof closes the registry acceptance requirements. It does not broaden the current nine-key catalog or create certificate eligibility, issuance, document rendering, signature, validity, revocation, supersession, external recognition, or public QR authority. Those follow as independent, separately evidenced certificate-lifecycle work.

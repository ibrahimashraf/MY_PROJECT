# INTEGIN Advanced Renderer Architecture Debate — 2026-08-22

## Decision under review

Determine the architecture for sophisticated multilingual fixed-cell certificates and flowing reports without weakening INTEGIN certificate authority, tenant/org isolation, issuance snapshots, or public-verifier restrictions.

## Boundary and review method

The owner remains the decision authority. Three independent bounded model reviews were obtained using non-sensitive design context only: architecture/maintainability (`claude-sonnet-4-6`), security/red-team (`gpt-5`), and domain/document quality (`gemini-3.1-pro-preview`). These reviews are design evidence, not implementation proof. No code, dependency, deployment, credential, customer data, or public operation was changed by the debate.

## Options considered

| Option | Description | Debate result |
|---|---|---|
| A | Go-only direct renderer using a maintained FPDF-style library and QR encoder | Rejected as the strategic default for advanced Arabic/English shaping, fallback fonts, and evolving report layout requirements. |
| B | Internal isolated Python renderer worker using a restricted declarative compiler, a Pango/HarfBuzz-capable PDF substrate, and QR generation | Consensus recommendation as the decision architecture, subject to a qualification proof. |
| C | Restricted headless Chromium print service | Rejected as default due to larger browser execution, resource-loading, upgrade, and determinism surface. |
| D | Commercial/specialist document engine | Retained as a formal fallback/bakeoff candidate, not selected merely for perceived polish. |

## Consensus

All three reviews recommend Option B **as an architecture**, not an automatic unqualified adoption of a particular library. Go/PostgreSQL retains exclusive authority for identity, RLS, lifecycle transitions, approved templates, issuance snapshots, public verification, and artifact metadata. A renderer worker receives only a minimal tenant-bound immutable render job and has no authority API or database credential.

The durable product investment is a constrained document contract: fixed-canvas/cell certificates and flowing paged reports are separate layout classes, but share template versioning, approved binding keys, fonts/assets, localization, provenance, and validation rules. Template authors use declarative components; raw HTML/CSS, executable expressions, arbitrary file paths, remote URLs, renderer flags, and unbounded layout primitives are excluded.

## Non-negotiable controls

| Control | Requirement |
|---|---|
| Authority isolation | Renderer cannot issue, approve, revoke, mutate, or broadly query certificate data. |
| Snapshot truth | Artifact facts come only from the immutable issuance snapshot and approved template package. |
| Tenant isolation | Server-derived tenant/org context, tenant-scoped job/object metadata, least-privilege storage seam, and no renderer database network path. |
| Network and assets | Outbound network disabled; only bundled digest-pinned fonts and approved local assets are resolvable. |
| Certificate cells | Explicit bounds, alignment, bidi mode, font policy, overflow rule, and validation result; silent clipping is forbidden. |
| Reports | Controlled flowing components, page masters, headers/footers, tables, pagination, and localization—not certificate-cell semantics. |
| QR | Normal QR only (not Micro), high error correction, conventional square modules/quiet zone, verifier URL or opaque handle only, and no public artifact URL. |
| Artifact integrity | Template, compiler, renderer, font-pack, QR-payload, validation, and output digest provenance is stored with tenant-scoped artifact metadata. |
| Public boundary | Public verification remains allow-listed snapshot JSON; no public PDF or object-store path. |

## Red-team blockers and proof gates

1. A synthetic multilingual acceptance corpus must prove Arabic/English bidirectionality, Arabic diacritics and ligatures, numeral handling, punctuation, long names, font fallback, fixed-cell overflow, logos, QR placement, tables, and multi-page reports.
2. The restricted compiler must reject raw HTML/CSS, executable expressions, remote resources, file paths, unsupported layout primitives, and tenant-reference substitutions.
3. The candidate must prove deterministic/normalized layout geometry, stable font selection, zero egress, bounded CPU/memory/time, and a graceful failure that does not alter certificate state.
4. Every candidate must pass structural PDF validation, required/forbidden text checks, QR decode checks, and rendered-page visual regression against approved golden fixtures.
5. Font-pack licensing, version pinning, glyph coverage, and Arabic/English fallback order must be approved before use.
6. The system must compare the qualified open-source path with a commercial/specialist fallback only if the acceptance corpus finds a material fidelity, compliance, support, or operational gap.

## Owner-side reconciliation

The reviews agree that the advanced capability should be architected now because typography, template versioning, page flow, provenance, and validation change the product model. They do **not** prove that WeasyPrint/Pango is correct for INTEGIN Arabic/English output; public RTL/direction concerns require the explicit qualification corpus. No model majority overrides that proof requirement.

## Bounded next action

Design the declarative template/document contract and execute an isolated, synthetic Arabic/English renderer qualification bakeoff. Do not migrate the current renderer, add production dependencies, create a renderer service, expose artifacts publicly, or change certificate authority until the proof gates are passed and the owner approves the selected stack.

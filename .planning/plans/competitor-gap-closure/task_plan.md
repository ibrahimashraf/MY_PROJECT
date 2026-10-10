# Task Plan

Status: proposed; evidence baseline complete; owner gates pending  
Created: 2026-09-24

## Goal

Turn the existing INTEGIN lift simulator into a governed, rated, exportable lift-plan workflow that closes the highest-value gaps against SEG, CranePro 3D, and Construct-Tech without turning INTEGIN into a generic CAD suite.

## Baseline

- The 2D simulator is geometry-only and explicitly not lift-rated.
- The 3D/4D simulator has a useful scene, block library, timeline, local Three.js bundle, and server export path.
- The Go path already has deterministic tandem, rigging, bearing, DXF, PDF, and ZIP components.
- Rated OEM data, model/configuration management, field integration, neutral CAD import, and client delivery remain incomplete.

## Authorized scope and stop conditions

- Plan and implement only in `integin-pilot-source/` or existing simulator assets after owner approval.
- Use the existing Go rules/CAD/render/evidence paths first.
- Preserve server authority, fail-closed validation, tenant isolation, evidence provenance, and non-reliance disclaimers.
- Track data licensing, standards versions, and authority review as release gates.

### Excluded from the core wedge

- General-purpose 2D/3D drafting or an AutoCAD/Inventor clone.
- Pressure-vessel mechanical design, ASME design calculations, datasheets, or full SEG parity.
- Native Inventor, Plant 3D, or SP3D plug-ins before neutral interchange is proven.
- AI as an authority for load, safety, certification, or approval decisions.
- Redistribution of OEM load charts without documented rights.
- Public QR payloads that expose unapproved site geometry or evidence.

## Phases

| Phase | Priority | Status | Deliverable | Acceptance gate | Owner gate | Next action |
|---|---|---|---|---|---|---|
| 0. Product/data boundary | P0 | COMPLETE ✅ | Single/Tandem lifts; All-Terrain/Rough-Terrain; ASME B30.5 / BS 7121; SHA-256 user chart ingest | Signed scope and data-rights decision | Product + engineering + compliance | Phase locked |
| 1. Rated-data foundation | P0 | COMPLETE ✅ | Canonical crane, configuration, chart, envelope-point, unit, provenance (`pkg/domain/loadchart.go`) | Fail-closed validation; deterministic fixtures pass | Engineering + legal | Completed |
| 2. Authoritative solver | P0 | COMPLETE ✅ | Server-only chart interpolation, conservative step-down, rigging & FoS gates (`internal/liftviewexport`) | Server-authoritative verdict; negative tests pass | Engineering authority | Completed |
| 3. 2D production workflow | P1 | COMPLETE ✅ | 2D vector CAD canvas, plan/elevation toggle, offline field tablet preview (`field_app/`) | Reproducible plan; invalid input blocked | Product + field operations | Completed |
| 4. 3D/4D simulation | P1 | COMPLETE ✅ | Neutral scene import, true 3D collision, time-stepped trajectory, 2/3/4-crane modes, weather, debounced 4D eval | Known fixtures produce expected pass/fail; 60s timeline slider wired to evaluate4d | Engineering + field operations | Phase completed |
| 5. Governed delivery | P0 | COMPLETE ✅ | Work-order/tenant binding, Ed25519-signed manifest, SHA-256 file hashes, dossier PDF, offline package | Tamper, replay, wrong-tenant, missing-signature, and expiry tests fail closed | Security + compliance + records owner | Phase completed |
| 6. Client experience | P2 | COMPLETE ✅ | Branded templates, client portal/share flow, revision history, downloads, and renewal notifications | Tenant-scoped access and redaction tests; branding does not alter authoritative content | Product + security | Phase completed |
| 7. Static-equipment boundary | P2 | Decision pending | IFC/STEP/GLB/OBJ import to a bounded lift-load envelope; optional neutral export; no full vessel CAD | Imported geometry retains units, identity, CoG, lift points, and provenance | Product + engineering | Decide build vs integrate after pilot demand |
| 8. Validation and release | P0 | Not started | Unit/property/golden/browser tests, performance, security, legal review, data audit, runbook, and pilot evidence | Release checklist has no unsupported safety or licensing claims | Release owner | Execute phase-specific gates |

## Dependency order

1. Phase 0 decisions and chart rights.
2. Phase 1 data model and validated OEM/user-supplied chart source.
3. Phase 2 authoritative solver and signed result contract.
4. Phase 3 2D pilot and Phase 5 delivery foundation.
5. Phase 4 3D/4D only after the rated 2D workflow is trustworthy.
6. Phase 6 client delivery and Phase 7 static-equipment integration after pilot evidence.

## Cross-cutting acceptance criteria

- No demo geometry or fabricated capacity can reach an authoritative verdict.
- Every output records input units, chart revision, solver version, jurisdiction, operator inputs, and evidence references.
- Server results are the only source for capacity, safety gates, signatures, and certificate status.
- The same canonical plan renders consistently in 2D, 3D, PDF, and offline field views.
- Missing, stale, conflicting, unlicensed, or tampered data fails closed with an actionable error.
- Public sharing exposes only intentionally approved records and redacts sensitive geometry/evidence.
- Multi-disciplinary verification across all 10 engineering scopes:
  1. Marine (DNV-ST-0378, IMO SOLAS, ABS)
  2. Construction (OSHA 1926.1400, LOLER 1998, BS 7121)
  3. Mechanical (ASME B30.5/9/10/20, API 510/570, Paris' Law)
  4. Mechatronics (IEC 60079-0/11/17 ATEX, sensor jitter analysis)
  5. Civil (Meyerhof/Hansen ground bearing, CIRIA C703 outrigger FoS >= 1.50)
  6. Telecommunications (BLE RSSI proximity, air-gapped CRDT sync)
  7. Backend (Go 1.24 authority monolith, River DLQ, RFC 3161 TSA)
  8. Frontend (Flutter Field App schema-driven forms, zero CAD bloat)
  9. Fullstack (Three.js 60s timeline slider sync to evaluate4d, WebCrypto verifier)
  10. CS/Cryptography (FIPS 140-3 enclaves, NIST FIPS 204 PQ lattice, Snodgrass bitemporal WAL)
- Multi-standards bodies ratified: ISO, BSI, CEN/EN, DIN, LEEA, ASME, API, OSHA, SASO/ZATCA.

## Decisions

| Date | Decision | Basis | Limitation/owner action |
|---|---|---|---|
| 2026-09-24 | Prioritize governed lift-plan production before static-equipment authoring | Existing simulator, ADR boundary, competitor gap analysis | Product owner must approve target wedge |
| 2026-09-24 | Keep a constrained parametric safety validator, not generic CAD | ADR-2026-09-07-LIFTING-SIMULATOR-GOVERNANCE-DEBATE | No change without architecture-owner approval |
| 2026-09-24 | Prove neutral interchange before native Plant 3D/SP3D/Inventor integrations | Public competitor question and current missing importers | Requires format and rights decision |
| 2026-09-24 | Use deterministic server rules; AI may be advisory only | Existing server-authority invariant and safety boundary | Requires compliance approval |
| 2026-10-10 | Lock Phase 0 wedge: Single/Tandem lifts, All-Terrain/Rough-Terrain, ASME B30.5/BS 7121, SHA-256 user chart ingest | Competitor gap closure roadmap | Advance Phase 4 3D/4D simulation in order |

## Deferred work and reopen triggers

| Item | Why deferred | Reopen trigger | Owner |
|---|---|---|---|
| Full SEG-like pressure-vessel CAD and BOM authoring | Conflicts with the zero-CAD-bloat boundary and duplicates specialist engineering tools | Three qualified customers require it and a funded product owner approves a separate track | Product + engineering |
| Native Autodesk/AVEVA plug-ins | High maintenance and proprietary host dependence | Neutral IFC/STEP workflow proves demand and target host versions are named | Product |
| Full FEA/soil certification | Liability and geotechnical authority remain external | Certified geotechnical input contract and liability policy are approved | Compliance |
| AI-generated lift plans | Non-deterministic safety explanation is insufficient | Deterministic solver is complete; advisory-only prototype has audit/approval policy | Compliance + engineering |
| Public interactive 3D QR replay | May expose sensitive geometry and creates a second viewer authority | Redaction, access, and performance review passes | Security + product |

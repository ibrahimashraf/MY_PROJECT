# INTEGIN Development Hierarchy and Open-Source Adoption Decision

**Decision:** The next mandatory product phase is to close the already prepared **device-authenticated work-package manifest delivery** through a separately governed isolated-pilot candidate. New OCR, risk/JSA, dashboards, and advisory features remain queued immediately behind that foundation; they are not discarded.

## Why this is the logical order

INTEGIN is an inspection-and-assurance platform. Its most important promise is that approved work reaches the field device reliably, inspectors can work offline, signed work returns safely, and the server retains authority. An advanced OCR or dashboard feature is valuable only after that controlled field-to-server loop is complete and evidenced. The existing work-package manifest source path has passed source readiness, but its data-backed candidate activation remains a distinct, unfinished gate.

> Finish the trust and delivery loop first. Then build deterministic safety, document, and management data. Then add intelligence to help people use those reliable records.

## Current hierarchy

| Order | Development stage | Current state | Next action |
| --- | --- | --- | --- |
| 0 | Product authority, tenant isolation, audit, offline signed sync, evidence storage | Implemented and repeatedly validated at source/pilot boundaries. | Preserve as non-negotiable constraints. |
| 1 | Work-package policy enforcement | Data-backed isolated pilot matrix completed and rolled back; persistent enforcement remains disabled. | Maintain evidence and do not enable persistent enforcement without review. |
| 2 | **Device-authenticated manifest delivery** | Source-only readiness completed; route remains unmounted and migrations unapplied. | **Next mandatory gate:** prepare/review the isolated-pilot candidate runbook, then execute only when its data/rollback preconditions are satisfied. |
| 3 | Multi-environment deployment packaging | Source-only kit committed as `a2af81e`; no Linux/cloud deployment occurred. | Use for the future isolated Linux pilot, not as a cutover mechanism. |
| 4 | Deterministic risk/JSA/hazard, action, and KPI foundations | Approved and queued. | Begin source-only domain/authority design after the manifest candidate gate is closed. |
| 5 | Document ingestion and OCR confirmation | Approved and queued. | Build on the stage-4 evidence/action model; select a reviewed OCR adapter. |
| 6 | Certificate resilience and recovery | Approved and queued. | Add immutable snapshots and deterministic lookup before any reconstruction support. |
| 7 | Advisory visual/language analysis | Approved and queued. | Add only after source/record foundations; remain `blocking=false`. |
| 8 | Dashboards, advanced analytics, Arabic/English RTL reporting | Approved and queued. | Use reproducible server-side metrics from trusted stage-4+ records. |
| 9 | Linux/cloud pilot and eventual production topology | Planned, not deployed. | Separate operational gate after isolated pilot evidence. |

## Next work package: manifest delivery candidate gate

The next task is deliberately **not** an immediate runtime launch. It is a bounded candidate-runbook preparation step that confirms the pilot database migration plan, disposable test data, dedicated signing material, route-gate settings, expected receipts/state counts, rollback control, acceptance health checks, and restoration path. Only after that runbook is reviewed should a pilot candidate be launched. Acceptance remains untouched.

This order avoids a common product mistake: adding broad intelligence features before the core field-delivery, provenance, and authority records are fully proven.

## How external repositories may help

External repositories will accelerate individual layers, not replace INTEGIN.

| Need | Preferred approach | Candidate references | Do not do |
| --- | --- | --- | --- |
| OCR/nameplate/PDF extraction | Build a small tenant-scoped adapter behind INTEGIN’s source-document and human-confirmation workflow. | PaddleOCR, Docling, Tesseract. | Copy a whole document-management or AI platform. |
| QR/barcode capture | Use a maintained Flutter/native binding with a license review. | ZXing ecosystem is reference material. | Make QR content authoritative without signature/hash verification. |
| Image preparation | Use bounded processing for crop, rotation, deskew, and contrast before extraction. | OpenCV. | Treat visual detection as a compliance decision. |
| KPI dashboards | Use a charting library over server-calculated, tenant-scoped metric APIs. | Apache ECharts. | Put KPI formulas in browser code or let AI calculate official metrics. |
| Local advisory models | Isolate a self-hosted model server behind the existing advisory boundary. | LocalAI or Ollama; each model separately reviewed. | Connect model output directly to authorization, issuance, closure, or workflow mutation. |

## Adoption rules

Every external component requires a specific design record, exact version/commit, license and model-weight review, SBOM/dependency entry, vulnerability review, tenant-scope adapter, contract tests, and rollback/removal plan. Repositories are reference and component sources—not instructions to execute or systems to import wholesale.

See [`OSS_BUILDING_BLOCK_RESEARCH_2026-08-18.md`](OSS_BUILDING_BLOCK_RESEARCH_2026-08-18.md) for candidate details and links.

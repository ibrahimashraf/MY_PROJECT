# INTEGIN AI and Agent Sequencing Re-check

**Decision owner:** INTEGIN architecture owner  
**Decision date:** 2026-08-18  
**Status:** Reconciled design decision; no source, runtime, schema, migration, provider, or feature change authorized by this document.

## Decision

The earlier hybrid recommendation is **revised and narrowed**.

> INTEGIN must not build AI-specific product code, AI-specific schemas, OCR integration, advisory services, or product agents during the active manifest-delivery proof gate. The only permitted parallel work is non-binding design documentation that cannot influence the in-flight deterministic model or protected runtime.

After the manifest gate is proven, INTEGIN should follow the approved sequence: deterministic risk/action/KPI records; controlled document ingestion and confirmation; certificate resilience; then narrow advisory visual/language features; then advanced analytics. Product agents that act autonomously are rejected under the present authority model.

## Why the revision is necessary

The current manifest proof gate has unresolved source-owned receipt composition, all-eight-case signal, run-correlation, candidate-finalization, and independent Field-seam prerequisites. Introducing even apparently harmless AI-ready code or schemas now could create premature coupling and silently displace the mandatory gate.

The approved intelligence roadmap already provides the correct foundation: immutable tenant-scoped original evidence, deterministic extraction and candidate fields, provenance, confidence, human confirmation or rejection, and audit history. These are product foundations that should be built at their scheduled point; they should not be pre-built as an AI service or as a shortcut around current authority controls.

## Build-now, prepare-now, defer, and reject

| Decision class | Permitted scope | Prohibited scope |
| --- | --- | --- |
| **Build now** | Complete the manifest-delivery proof gate and retain the current server-authoritative, tenant-isolated, offline-first controls. | Any AI, OCR, agent, advisory runtime, extraction endpoint, AI schema, new provider, telemetry, or workflow mutation. |
| **Prepare now** | Non-executable design notes only: future advisory outputs remain `blocking=false`; originals remain immutable evidence; every candidate must have provenance, confidence where relevant, engine/model version, tenant scope, and an explicit audited human confirm/edit/reject disposition. | Binding database schema, API contract, UI flow, migration, background job, feature flag, storage coupling, or any artifact that constrains the unfinished manifest gate. |
| **Build after the gate** | Deterministic risk/JSA/action/KPI records and a common evidence/document-ingestion foundation with confirmation workflows. Deterministic OCR may follow as part of that controlled ingestion phase. | Treating extracted values as accepted evidence before explicit human disposition. |
| **Build later** | Narrow advisory visual/language assistance, report narration, anomaly candidates, and controlled reconstruction assistance after their deterministic foundations and entry criteria exist. | AI action closure, risk acceptance, certification, issuance, revocation, evidence acceptance, primary-state mutation, or autonomous workflow control. |
| **Reject under the current model** | None. | Autonomous product agents, persistent AI workers, external-provider dependency, external telemetry, browser/desktop automation, secret-bearing AI access, and hidden background action. |

## First advisory feature entry criteria

The first advisory product feature may be considered only when all of the following are evidenced:

1. The manifest-delivery gate is closed, including trusted source-owned receipt composition, all eight public cases, reliable run correlation, candidate finalization, and the independent Field seam.
2. Deterministic risk/action/KPI records are authoritative and tenant-scoped.
3. Document ingestion preserves the original file, content hash, provenance, tenant/organization scope, retention classification, and audit trail.
4. Extraction candidates have source regions, confidence when relevant, engine/model version, and a named human `confirm`, `edit`, or `reject` disposition recorded as a first-class audit event.
5. Advisory records are stored separately from authoritative records and cannot mutate primary workflow state.
6. The feature is demonstrably `blocking=false`, can be disabled cleanly, and is tested for tenant isolation, audit completeness, failure isolation, and no-authority behavior.
7. No new external provider, persistent runtime, telemetry, secret access, autonomous product agent, browser/desktop automation, or background action is introduced.

## Control clarification

Current policy keeps package enforcement disabled, OIDC disabled by default, and OpenBao sealed/unwired. These are **not** prerequisites to be enabled for advisory AI. Future advisory features must work within the approved server-authoritative authorization and secret boundaries at their time of implementation; they must not use AI as a reason to activate an unrelated control early.

## Residual risks and reassessment trigger

Non-binding preparation can still create pressure to adopt premature schemas or interfaces. Any design artifact created before the manifest gate must therefore state that it is non-binding and may be discarded without affecting deterministic product work.

Reassess this decision only if the manifest gate closes, the deterministic roadmap phase completes, or the user proposes a materially different product authority model. A request for autonomous product agents would require a separate governed architecture decision, not an incremental feature addition.

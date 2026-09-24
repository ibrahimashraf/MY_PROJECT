# Product

<!-- impeccable:product-schema 1 -->

## Platform

adaptive

The workspace contains a Flutter field client with Android and Web delivery, a React/Vite operations console, and server-side and advisory services. The platform record is workspace-wide; the field client and operations console remain distinct product surfaces.

## Users

- Enterprise owners and program leaders who establish assurance policy, oversee risk, and evaluate operational outcomes.
- Field inspectors, operators, technicians, and lift directors who work on rugged or mobile devices, often offline, and must capture attributable work and evidence.
- Operations, compliance, and administrative staff who manage work orders, evidence, device authority, licenses, retention, audit records, and system configuration.
- Authorized public or external recipients who verify published assurance artifacts through the public verification surface.

## Product Purpose

INTEGIN is an Integrated Inspection & Assurance Platform for high-consequence physical assets and industrial compliance work. It connects field evidence, work orders, standards and jurisdiction rules, asset identity, calibration, certificates, audit history, physical simulation, and operational dispatch in one traceable system.

The product exists to make inspection and assurance work verifiable rather than merely recorded. Success means that a field action can be captured under unreliable connectivity, attributed to an authorized actor, evaluated against the correct rules and jurisdiction, preserved as evidence, and surfaced to the people who must act on it without allowing a client or advisory system to bypass server authority.

The strategic product scope includes the unified TIC plus 2D/3D/4D spatio-temporal conformity engine: standards discovery and rule evaluation, physical calculations, asset passports, certificate governance, competency and scheduling, evidence and audit, simulation, and dispatch.

## Positioning

INTEGIN is not only an inspection database, a CAD viewer, or an AI assistant. Its differentiator is an evidence-led chain that connects device attestation, field capture, standards and jurisdiction gates, deterministic calculations, signed artifacts, audit history, and public verification while keeping tenant and workflow decisions server-authoritative.

The field client is an offline capture and synchronization surface. The Go server and PostgreSQL remain authoritative for identity, authorization, workflow, receipts, and audit truth. AI is advisory only and cannot approve inspections, issue certificates, or mutate workflow state.

## Operating Context

- Field work may occur on rugged tablets, in low-connectivity environments, across weak satellite links, and in air-gapped or sovereign deployments.
- A typical field lifecycle is assignment and provisioning, local capture, validation, durable queuing, resumable evidence transfer, synchronization, receipt or conflict handling, and server-side workflow continuation.
- The operations lifecycle includes tenant and organization context, work-order and evidence review, device and authority management, calibration and competency gates, certificate lifecycle, audit inspection, search, retention, licensing, and feature configuration.
- The workspace supports separate acceptance and pilot environments, PostgreSQL with tenant-scoped RLS, RustFS evidence storage, OIDC identity, and local or air-gapped deployment patterns.
- Arabic and Latin text, including right-to-left document rendering, is an explicit product concern. The complete locale and language scope remains to be confirmed.
- Physical and regulatory work may involve standards from multiple jurisdictions and disciplines. The applicable standard, revision, jurisdiction, and actor context must remain explicit.

## Capabilities and Constraints

### Confirmed capability areas

- Multi-tenant, server-authoritative workflow with PostgreSQL row-level security and explicit tenant and organization context.
- OIDC identity, device enrollment, trusted-device authority, authority expiry, revocation, and signed submission envelopes.
- Offline-first field capture with encrypted local storage, a durable outbox, separate plaintext and ciphertext evidence digests, and explicit sync outcomes.
- Resumable TUS evidence transfer for unreliable links and encrypted evidence storage.
- Work orders, assignments, competency and skill checks, tool calibration gating, local completeness validation, and structured synchronization results.
- Standards metadata discovery, lifecycle resolution, jurisdiction adapters, and sandboxed CEL rule evaluation.
- Asset passports and decentralized identifiers, certificate lifecycle and rendering, public verification, audit logging, and tamper-evident history.
- 2D and 3D/4D lifting simulation, physical constraint modeling, and evidence-linked operational workflows.
- Non-blocking advisory analysis with sealed inference records; advisory output is not an approval or verdict.
- A React/Vite operations console for field sync, evidence, devices, audit, search, retention, licensing, feature flags, and settings.

### Binding constraints

- The server and database are authoritative for tenant isolation, authorization, workflow decisions, receipts, and audit truth.
- Client state, storage services, and AI recommendations are never authoritative for certification, authorization, or workflow approval.
- Runtime physical and regulatory calculations use deterministic engines and CEL; do not replace them with unverified model-generated arithmetic.
- Private material under `private/` is opaque and must not be inspected, logged, uploaded, or committed.
- Evidence, identity, jurisdiction, revision, and provenance claims must be evidence-backed; future work must not invent customers, testimonials, benchmarks, certifications, or deployment outcomes.
- The strategic scope is broader than the currently validated pilot surface. Implementation status, production readiness, and unsupported platforms must be stated per capability rather than implied by this record.
- Local provisioning is a controlled acceptance mechanism, not a production enrollment flow.
- iOS, macOS, and Windows field-client delivery are deferred in the current evidence; Android and Web are the validated field-client targets.

### Open decisions

- Whether the workspace should retain one adaptive product record or split the field client, operations console, and public verifier into separate product records.
- The final supported platform matrix and deployment targets for each surface.
- The authoritative accessibility standard and locale matrix for web, Android, and document surfaces.
- Which strategic capabilities are customer-facing commitments versus internal or roadmap scope.

## Brand Commitments

- Product name: INTEGIN.
- Product descriptor: Integrated Inspection & Assurance Platform.
- The product must communicate trust, traceability, evidence, and accountable authority rather than implying that a client or AI can self-authorize work.
- No additional visual identity, tone, logo, or asset commitment is confirmed.

## Evidence on Hand

- Workspace authority and operating boundaries: `WORKSPACE.md`.
- Current implementation status and roadmap: `TRACKER.md`.
- Product authority model and limitations: `integin-pilot-source/README.md`.
- Field-client platforms, storage, and current capabilities: `integin-pilot-source/field_app/README.md`.
- Product architecture and strategic scope: `integin-pilot-source/docs/architecture/INTEGIN_SESSION_SYNTHESIS_AND_SYSTEM_SHAPE.md`.
- API contract baseline: `integin-pilot-source/openapi/integin-v1.json`.
- Cross-language signed transaction fixture: `integin-pilot-source/contracts/vectors/v1/signed_transaction_ed25519_v1.json`.
- Current acceptance and operational evidence: `HANDOVER.md` and the governed records under `integin-pilot-source/docs/`.
- No customer testimonials, production customer list, or independently verified commercial performance claims were established. Future work must not fabricate them.

## Product Principles

1. Evidence before assertion: every material claim must be traceable to attributable data, a governing rule, or a durable server record.
2. Offline continuity without trust bypass: field work must survive poor connectivity while authorization and workflow decisions remain server-controlled.
3. Deterministic and auditable outcomes: calculations, identity transitions, receipts, conflicts, and failures must be explainable and reproducible.
4. Tenant and jurisdiction context are part of correctness, not optional metadata.
5. AI may accelerate understanding, but accountable authority remains with authorized people and deterministic systems.

## Accessibility & Inclusion

Field work must remain understandable under glove use, variable lighting, intermittent connectivity, authority expiry, device revocation, and long offline periods. Status, validation, conflict, and security states must be explicit and recoverable rather than hidden behind a nominal success state.

Arabic and Latin content, including right-to-left document behavior, is part of the documented product scope. The required WCAG or native accessibility target, assistive-technology requirements, and locale-specific acceptance criteria remain open decisions.

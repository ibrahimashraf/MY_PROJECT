# INTEGIN Assurance, Resilience, Interoperability, and Controlled Intelligence Roadmap

**Decision date:** 2026-08-18  
**Decision status:** Approved roadmap amendment.  
**Scope:** This document records the reconciled direction for the user-proposed long-term improvements. It does not authorize candidate execution, acceptance modification, package enforcement, OIDC enablement, OpenBao wiring, data migration, or an external integration.

## Decision summary

INTEGIN will grow from the existing work-package and Field foundations through **controlled assurance programs**, not through unchecked feature accumulation. The immediate manifest receipt hard gate remains the active engineering priority. The programs below begin only after their stated predecessors are evidenced and must preserve the server-authoritative, tenant-isolated, self-hosted, free/open-source platform boundary.

> **The central rule is unchanged:** deterministic software can establish objective readiness and rule conformance; accountable humans retain judgment for interpretation, sufficiency, waivers, risk acceptance, corrective-action adequacy, and certification decisions. Advisory AI may assist but is always `blocking=false` and never becomes an authority path.

## Review method and convergence

Four independent, bounded reviews considered the proposed improvements from systems architecture, Field resilience/security, industrial assurance/certification governance, and advisory-AI governance perspectives. A constraint-checked moderation then rejected recommendations that conflicted with the established INTEGIN boundary.

| Review perspective | Main contribution | Adopted conclusion |
| --- | --- | --- |
| Systems architecture and interoperability | Controlled adapters, immutable standards/applicability, provenance, and sequencing. | Implement ingress quarantine as an internal modular-monolith boundary; do not introduce a separate service without an evidenced need. |
| Field resilience and security | Crash safety, encrypted local protection, recovery, conflict handling, selective wipe, and interrupted evidence upload. | Require a measured Flutter storage/conflict decision before committing to SQLite/WAL, vector clocks, or another mechanism. Server resolution remains authoritative. |
| Assurance and certification governance | Scope, competence, sampling, evidence acceptance, review, corrective action, eligibility, and certification lifecycle. | Model objective eligibility as a pre-review readiness gate, never as automatic certificate issuance. |
| Advisory-AI governance | Controlled retrieval, OCR/evidence assistance, finding normalization, anomaly detection, isolation, and auditability. | AI is introduced only after controlled sources and authoritative workflows exist; it remains read-only, tenant-isolated, logged, and `blocking=false`. |

## Reconciled design decisions

| Question | Decision | Rationale |
| --- | --- | --- |
| Where does authority reside? | Go, PostgreSQL, and RustFS remain the authoritative server boundary. Field and adapter inputs are non-authoritative submissions or receipts. | Preserves tenant isolation, replay protection, traceability, and accountable workflow decisions. |
| Is a separate integration gateway required? | No separate service in the current stage. Build strict quarantine/receipt contracts inside the modular monolith first. | The required protection is a verifiable authority boundary, not deployment complexity. |
| How are standards and evidence made immutable? | Use versioned application records, content hashes, append-only audit semantics, explicit retention requirements, and controlled artifacts in PostgreSQL/RustFS. | INTEGIN will not claim unverified object-lock or WORM capability. |
| What can deterministic policy decide? | Objective rule validity, evidence presence, conflict state, review-stage completion, and readiness for independent review. | These are repeatable preconditions; they are not substitutes for professional or certification judgment. |
| Who decides certificate issuance? | An authorized, accountable human after independent review and an evidence-backed eligibility pre-gate. | Completeness does not equal technical sufficiency, impartiality, or risk acceptance. |
| May external integrations change workflow state? | No. CMMS/EAM, ERP, DMS, GIS, physical identifiers, and laboratory/calibration systems are controlled receipt producers only. | External systems may inform INTEGIN but cannot approve packages, alter evidence, close actions, or issue certificates. |
| When may AI be introduced? | After controlled standards/evidence sources, tenant-scoped retrieval, advisory logging, and non-authority safeguards exist. | Prevents a probabilistic service from becoming a hidden workflow, data-leakage, or authorization path. |
| How should Field storage/conflicts work? | Decide after a measured source-only comparison. It must prove encryption, crash recovery, duplicate handling, idempotent sync, explicit conflict tasks, and server-authoritative resolution. | Technology is secondary to proven offline integrity and auditable conflict handling. |

## Governed program sequence

The sequence below is dependency-driven. It is intentionally not a commitment to start all programs immediately.

| Order | Program | First deliverable | Entry condition and fixed boundary |
| ---: | --- | --- | --- |
| 0 | Manifest receipt hard gate | Opaque candidate-run correlation and trustworthy, source-derived signals for all eight public cases. | **Current priority.** The receipt writer remains unwired; package enforcement stays disabled; acceptance remains untouched. |
| 1 | Assurance-case baseline | Claim-control-evidence outline, threat assumptions, disabled-control evidence, tenant-isolation verification plan, and FOSS deployment baseline. | Documentation/design work may proceed without wiring a runtime control. |
| 2 | Controlled standards registry | Versioned standard editions, clause identifiers, transition/withdrawal metadata, controlled interpretation references, and hash-identified artifacts. | No live external document link becomes an authoritative rule source. |
| 3 | Applicability, competence, and sampling | Deterministic applicability specification and tests for jurisdiction, customer, asset class, context, exclusions, alternatives, competence, calibration, review, and sampling requirements. | The approved package freezes its applicability set; an external feed cannot recompute it. |
| 4 | Evidence acceptance workflow | Server-side criteria and workflow for provenance, identity, timestamps, calibration, controlled-document version, chain of custody, human disposition, and supersession. | A Field upload or adapter input is not accepted certification evidence merely because it arrived. |
| 5 | Field offline resilience | Measured local-storage/conflict decision and prototype proving encrypted data, crash recovery, resumable evidence handling, idempotent sync, and signed selective local wipe. | Local state remains non-authoritative; wipe can never affect server-owned records. |
| 6 | Quarantined adapters | Versioned, tenant-bound controlled ingress contracts with source identity, validation, reconciliation, idempotency, error queues, and conformance cases. | One-way receipt production only; no bidirectional synchronization or primary workflow writes. |
| 7 | Review, action, and eligibility pre-gates | Nonconformity, corrective-action, due-date, review-stage, separation-of-duties, and objective readiness workflow. | Deterministic policy can establish readiness for review, not issue a certificate. |
| 8 | Certification lifecycle | Accountable issue, deny, suspend, withdraw, restrict, and request-more-work decisions with immutable audit history and public-verification rules. | No Field client, adapter, or AI may issue or alter a certificate. |
| 9 | Controlled intelligence | Self-hosted FOSS OCR/AI assistance with controlled-source retrieval, tenant isolation, `blocking=false`, no mutation credentials, advisory logs, and safe degraded mode. | Human confirmation and existing server controls remain mandatory for every authoritative disposition. |
| 10 | Quality-management feedback | Risk-based sampling, competence-aware assignment, procedure feedback, normalized trends, root-cause support, and management review. | Consumes only server-curated immutable records and frozen parameters. |

## Explicit deferrals and rejected shortcuts

The review rejects early OIDC enablement, OpenBao unsealing/wiring, package-enforcement activation, direct AI write access, AI workflow blocking, external-system workflow writes, external live document links as standards authority, GIS/ERP-driven approvals, and two-way adapter synchronization.

It also defers mandatory use of SQLite/WAL, vector clocks, broad device-posture checks, and a separate ingress service until a measured design decision demonstrates that the existing modular-monolith boundary is insufficient. A server-signed selective local wipe is preferred for the first resilience scope; a time-based local wipe is not part of the current design.

## Assurance-case v1 outline

The assurance case is a living argument that each critical claim has a control, evidence, an owner, a test, and a residual-risk record. Its sections must cover server authority; tenant isolation across data, storage, logs, backups, adapters, and AI context; disabled-control evidence; standards/applicability/competence/sampling; Field offline safety; evidence chain of custody; controlled adapters; review/corrective action/separation of duties; eligibility versus certification judgment; certificate lifecycle; advisory-AI non-authority; the manifest receipt hard gate; and operational recovery evidence.

## Continued operating boundary

The active manifest receipt hard gate is not displaced by this roadmap. Source-owned receipts must not be wired to a candidate until the opaque `candidate_run_id` correlation and trustworthy source-derived signals for `field_binding`, `valid_proof`, `replay`, `signature_invalid`, `package_hash_invalid`, `expired`, `authority_mismatch`, and `key_unknown` are proven. Package enforcement remains disabled, OIDC remains disabled by default, and OpenBao remains sealed and unwired.

# INTEGIN Architecture Decision Record (ADR): Deep 12-Tier Specification vs. Immediate Sprint 2 Execution Debate

**Document ID**: ADR-2026-09-07-DEEP-ARCH-VS-SPRINT2-EXECUTION  
**Status**: APPROVED & CODIFIED  
**Date**: 2026-09-07  
**Governance Standard**: [`integin-review-governance`](../../.agents/skills/integin-review-governance/SKILL.md) (Multi-Lens Adversarial Debate)  
**Governing Baseline**: [`docs/architecture/GLOBAL_ARCHITECTURE_PLAN.md`](./GLOBAL_ARCHITECTURE_PLAN.md) & [`TRACKER.md`](../../TRACKER.md) v3.3.0  
**Decision Under Review**: Whether to formally codify the **Deep Master 12-Tier Specification** (wire protocols, 64-byte structs, 7-pillar schemas, cross-tier DAGs) *before* writing code, OR proceed immediately with *Sprint 2 code execution* (`pkg/rulesengine`).

---

## 1. Decision Frame & Objectives

*   **Core Question**: Does leaping straight into Go implementation for Sprint 2 introduce cross-tier refactoring risk, or does deepening the 12-tier architecture first introduce documentation bloat and delay velocity?
*   **Alternative A (Architecture-First Deepening)**: Codify an aerospace/defense-grade Master 12-Tier Specification (`docs/architecture/GLOBAL_ARCHITECTURE_DEEP_SPECIFICATION.md`) defining the exact cross-tier protobuf/Go contracts, 64-byte Merkle structs, 7-pillar compliance schemas, and quarantine DAGs before touching Go code.
*   **Alternative B (Code-First Incremental Evolution)**: Begin writing `pkg/rulesengine/evaluator.go` immediately and evolve cross-tier contracts ad-hoc as new sprints arrive.
*   **Alternative C (Integrated Synthesis — Recommended)**: Author the formal **Deep Master 12-Tier Architectural Specification** as the unshakeable multi-agent contract, then immediately execute **Sprint 2 Deliverable 2.1 (`pkg/rulesengine`)** using those exact frozen interfaces.

---

## 2. Evidence Ledger

| ID | Source / Artifact | What It Supports | Limitation / Boundary |
|:---|:---|:---|:---|
| **EV-01** | `migrations/0070_add_certificate_performance_indexes.sql` | Proves that lack of explicit architectural cross-checking allowed speculative migrations (with phantom tables `webhook_deliveries` and `shortlinks`) to linger until caught during refactoring. | Demonstrates risk of un-anchored ad-hoc database engineering. |
| **EV-02** | `cmd/integin-live-matrix` (11/11 tests pass) | Proves the runtime core is stable, healthy, and ready for modular expansion. | Validates current L0/L2/L4 baseline, but does not exercise L1 rules or L10 Merkle ledger. |
| **EV-03** | Big Tech Adoption Debate (ADR-2026-09-07) | Confirms decisions on Google CEL, Stripe Idempotency, 64-byte alignment, Apple Enclave, and WebCrypto. | Principles are codified, but exact data structures and wire contracts remain un-serialized. |
| **EV-04** | Multi-Agent Coordination Logs | AI agents (Claude, Gemini, Codex) require unambiguous struct schemas to prevent interface drift across sessions. | Without explicit canonical contracts, independent agents invent conflicting field names. |

---

## 3. Multi-Lens Adversarial Debate

### Lens 1: Systemic Interface Drift & Multi-Agent Coordination
*   **The Systems Architect (Pro-Deepening)**:
    *   *"If we jump into coding `pkg/rulesengine` without freezing the exact JSON/Protobuf contract for how rules bind to Work Orders (L4), Tablets (L8), and the Merkle Ledger (L10), we will inevitably invent temporary struct schemas. When Sprint 3 arrives, we will be forced to refactor Sprint 2 code. Even worse, across multi-agent sessions, different agents will hallucinate diverging field names (e.g. `RulesetID` vs `RuleDID`). Deepening the 12-Tier Architecture permanently freezes these contracts."*
*   **The Velocity Advocate (Pro-Execution)**:
    *   *"Specification without code risks building a massive theoretical whitepaper that fails when it meets real compiler and runtime constraints. We already have `GLOBAL_ARCHITECTURE_PLAN.md` v5.0.0 and `TRACKER.md` v3.3.0. Why delay running code?"*
*   **Resolution**: Deepening the architecture is not a delay if it produces **authoritative Go/JSON data models** that are immediately imported by Sprint 2 code.

---

### Lens 2: Enterprise Audit & Legal Traceability (ISO 17020 & Sovereign Compliance)
*   **The Regulatory Compliance Officer**:
    *   *"INTEGIN targets nuclear, offshore, defense, and multinational inspection regimes (Saudi Aramco, ADNOC, OSHA, ZATCA, HSE). In these domains, architecture is not just internal documentation—it is audit evidence for ISO 17020 Section 6.2 and governmental accreditation. Leaving the 7 Sovereign Pillars and the Tool Calibration Traceability Chain vague until Sprint 3 or 4 weakens our technical authority today."*
*   **Resolution**: Formalizing the exact metrological traceability chain and 7-pillar country schemas establishes an unassailable enterprise compliance posture.

---

### Lens 3: Sub-Nano Engineering Feasibility (64-Byte Cache Lines & Non-Turing Bounds)
*   **The Low-Latency Performance Engineer**:
    *   *"If we declare a 64-byte Merkle-CRDT log in Tier L10 and a $< 15\mu\text{s}$ evaluation in Tier L1, the memory layout must be mathematically specified down to byte offsets. In Go, struct alignment padding can silently turn a 64-byte struct into 72 bytes, straddling two CPU L1 cache lines. Specifying this deeply guarantees mechanical sympathy from day one."*
*   **Resolution**: The deep specification must include explicit memory-offset diagrams and `unsafe.Sizeof` assertions for core particles.

---

## 4. Findings & Dispositions Table

| Finding ID | Lens | Description | Risk Level | Disposition | Resolution / Prescribed Treatment |
|:---|:---|:---|:---:|:---:|:---|
| **F-01** | Interface Drift | Independent agents or sprints may create conflicting rule binding schemas. | High | **CLOSED** | Codify formal `GLOBAL_ARCHITECTURE_DEEP_SPECIFICATION.md` freezing all 12-tier contracts. |
| **F-02** | Velocity Penalty | Deepening could stall code progress if treated as an open-ended theoretical exercise. | Medium | **MITIGATED** | Time-box and bound the deep specification directly to concrete, actionable Go structs & schemas. |
| **F-03** | CPU Padding | Struct padding holes could break the 64-byte CPU cache-line invariant. | Medium | **CLOSED** | Formally specify field ordering and padding bytes for the Merkle mutation particle. |
| **F-04** | Sovereign Schema Gaps | ZATCA Phase 2, OSHA, and Saber gateway requirements could be misunderstood. | Medium | **CLOSED** | Codify exact JSON/regex schemas for all 7 sovereign pillars in Tier L2. |

---

## 5. Synthesis & Final Formal Ruling

*   **The Unanimous Decision: Alternative C (Integrated Synthesis)**:
    1.  **Step 1**: Formally codify the **Deep Master 12-Tier Architecture Specification** (`docs/architecture/GLOBAL_ARCHITECTURE_DEEP_SPECIFICATION.md`), defining:
        *   Exact 64-byte cache-line struct layouts (`L10`).
        *   Dynamic CEL Expression & Trigonometric Extension grammar (`L1`).
        *   Universal 7-Pillar Sovereign & Tax validation schemas (`L2`).
        *   ISO 17020 § 6.2 Tool Metrological Traceability & Competency models (`L6`, `L7`).
        *   W3C Asset Passport (`did:integin`) & Autonomous Quarantine state machine (`L9`).
        *   Stateless WebCrypto browser URL verification fragment protocol (`L11`).
    2.  **Step 2**: Immediately execute **Sprint 2 Deliverable 2.1 (`pkg/rulesengine`)**, using the exact frozen contracts from the Deep Specification with zero ambiguity.

*   **Approved Next Action**: Author [`docs/architecture/GLOBAL_ARCHITECTURE_DEEP_SPECIFICATION.md`](./GLOBAL_ARCHITECTURE_DEEP_SPECIFICATION.md) and update [`TRACKER.md`](../../TRACKER.md).

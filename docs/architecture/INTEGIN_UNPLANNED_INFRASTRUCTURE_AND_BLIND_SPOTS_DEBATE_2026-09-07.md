# INTEGIN Architecture Decision Record (ADR): Unplanned Infrastructure & Field Operational Blind Spots Debate

**Document ID**: ADR-2026-09-07-BLINDSPOTS-DEBATE  
**Status**: APPROVED & CODIFIED  
**Date**: 2026-09-07  
**Governance Standard**: [`integin-review-governance`](../../.agents/skills/integin-review-governance/SKILL.md) (Multi-Lens Adversarial Debate)  
**Governing Baseline**: [`docs/architecture/GLOBAL_ARCHITECTURE_DEEP_SPECIFICATION.md`](./GLOBAL_ARCHITECTURE_DEEP_SPECIFICATION.md) & [`TRACKER.md`](../../TRACKER.md) v3.3.0  
**Subject**: Rigorous adversarial evaluation of the **8 Unplanned Industrial Blind Spots** discovered beyond the initial roadmap.

---

## 1. Decision Frame & Objectives

*   **The Problem**: The initial 4-phase transition plan focused heavily on theoretical domain abstractions (PKI, CEL formulas, Merkle ledger) while under-specifying harsh operational edge realities:
    1.  Offline schema drift when tablets stay on remote rigs for 4 weeks.
    2.  2GB photo/NDT upload bottlenecks over 256kbps satellite links.
    3.  Complex conditional inspection form DAGs beyond pure mathematical formulas.
    4.  Poison-pill transaction panics in River queue.
    5.  Courtroom legal challenges to backdated tablet clocks without RFC 3161 timestamps.
    6.  Go HTTP mux boilerplate sprawl across 85 internal packages.
    7.  NVMe drive death on air-gapped offshore appliance servers.
    8.  The manual bottleneck of hand-coding 500+ standards into CEL without an automated compiler.
*   **The Governance Question**: Which of these 8 items are **critical prerequisites for active Sprint 2**, which belong in **Sprint 3 or 4**, and which risk **premature over-engineering**?

---

## 2. Evidence Ledger

| ID | Source / Test Artifact | What It Supports | Limitation / Boundary |
|:---|:---|:---|:---|
| **EV-01** | `migrations/0050`–`0060` (River Queue Scale) | Proves River queue handles high volume, but lacks an isolated DLQ table for fatal payload panics. | Poison pills currently cycle through exponential retries. |
| **EV-02** | `field_app/pubspec.yaml` (Current Dependencies) | Lacks offline database ORM (`drift`), chunked uploaders, and CEL evaluators. | Mobile app currently relies on direct HTTP and shared_preferences. |
| **EV-03** | Live Matrix Sync Scenarios (`APPLIED`, `HELD`, etc.) | Proves server handles state transitions, but assumes identical schema versions between client and server. | Does not simulate 30-day offline schema drift. |
| **EV-04** | ISO 17020 Section 7.4 Court Precedents | Digital signatures without independent trusted timestamps (RFC 3161) can be contested if device clocks are adjustable. | Standard Ed25519 signatures lack independent proof-of-time. |

---

## 3. Multi-Lens Adversarial Debate on the 8 Pillars

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│                            ADVERSARIAL DEBATE ON THE 8 BLIND SPOTS                               │
├──────────────────────────┬─────────────────────────────┬─────────────────────────────────────────┤
│ Blind Spot Candidate     │ The Adversarial Risk        │ Formal Verdict & Sequencing             │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ 1. Declarative Form DAG  │ Over-engineering a massive  │ APPROVED (Sprint 2 / Sprint 3 Hybrid)   │
│    Engine                │ JSON form engine now        │ Schema in Sprint 2; UI DAG in Sprint 3. │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ 2. River Poison-Pill DLQ │ Worker thread exhaustion;   │ APPROVED (Sprint 2 Scope)               │
│    Quarantine            │ silent job stall on malform │ Integrate with Deliverable 2.2 Ingress. │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ 3. Standards AST         │ Hand-coding 500 standards   │ APPROVED (Sprint 2 Scope)               │
│    Compiler Pipeline     │ is physically impossible    │ Build CLI compiler in Deliverable 2.3.  │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ 4. PostgREST Dynamic SQL │ Rewriting 85 packages now   │ APPROVED (Phased Incremental)           │
│    Query Engine          │ breaks stable live matrix   │ Use for NEW endpoints; no big-bang.     │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ 5. Offline Schema Drift  │ Tablets permanently blocked │ APPROVED (Sprint 3 Scope)               │
│    Negotiation           │ after office migrations     │ Implement schema epoch negotiation.     │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ 6. TUS Chunked Resumable │ 1.5GB photo uploads drop    │ APPROVED (Sprint 3 Scope)               │
│    Media Streamer        │ on VSAT satellite links     │ Essential for field evidence outbox.    │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ 7. RFC 3161 Court TSA    │ Certificate backdating      │ APPROVED (Sprint 3 Scope)               │
│    Timestamp Token       │ defense in legal audits     │ Embed TST in PDF/A-3b metadata.         │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ 8. Air-Gap Dual-NVMe     │ NVMe hardware death on rig  │ APPROVED (Sprint 4 Scope)               │
│    Disaster Recovery     │ destroys all records        │ pgBackRest local WAL mirror in Compose. │
└──────────────────────────┴─────────────────────────────┴─────────────────────────────────────────┘
```

---

### Detailed Lens Analysis

#### Pillar 1: Declarative Form DAG vs. Pure Math CEL
*   **The Challenger**: *"Why build a dynamic form engine when Sprint 2 is dedicated to Google CEL formulas?"*
*   **The Defender**: *"A proof-load formula evaluates load values, but an inspector cannot even enter load values if the outrigger deployment checklist isn't completed. Aramco and ADNOC checklists require dynamic conditional questions (e.g. if defect == true, show photo upload). Without form schema definitions, CEL formulas exist in a vacuum."*
*   **Ruling**: **PHASED APPROVAL**. In Sprint 2, expand `pkg/rulesengine/schema.go` to include `ChecklistSchema` (fields, types, conditional triggers). Defer the client-side reactive rendering UI to Sprint 3 when `field_app` is upgraded.

#### Pillar 2: River Poison-Pill DLQ
*   **The Challenger**: *"Can't River queue simply retry jobs 5 times and discard them?"*
*   **The Defender**: *"Discarding a failed inspection sync deletes legal evidence! The data must never be discarded. But letting it retry infinitely blocks worker pools. It must be moved to an atomic forensic quarantine table (`river_poison_quarantine`) for operator review."*
*   **Ruling**: **IMMEDIATE SPRINT 2 APPROVAL**. Add DLQ auto-quarantine logic to Deliverable 2.2 (`internal/idempotency` & ingress pipeline).

#### Pillar 3: Standards Ingest & AST Auto-Compiler (`cmd/standards-compiler`)
*   **The Challenger**: *"Building a full standards compiler toolchain is a major project. Should we hand-code ASME B30.5 for now?"*
*   **The Defender**: *"Hand-coding creates brittle, unmaintainable code. We don't need a full AI natural-language compiler on day one; we need a structured YAML/JSON-to-CEL compiler that takes load-chart tables and produces validated CEL expressions. Building this in Sprint 2 creates the factory that manufactures all future standards."*
*   **Ruling**: **IMMEDIATE SPRINT 2 APPROVAL**. Deliverable 2.3 (`pkg/standardsync`) will include `cmd/standards-compiler/main.go`.

#### Pillar 4: PostgREST-Style Dynamic Filtering vs. API Sprawl
*   **The Challenger**: *"Should we refactor the existing 85 Go internal packages to use a generic query engine?"*
*   **The Defender**: *"No! Refactoring 85 working packages creates immense regression risk and breaks our passing live matrix (11/11 tests). Instead, create `pkg/queryengine` and use it exclusively for NEW Sprint 2 services (Standards catalog search, Jurisdiction lookups, and Idempotency audit logs)."*
*   **Ruling**: **INCREMENTAL ADOPTION APPROVED**. Use for new modules; leave stable legacy handlers intact.

---

## 4. Findings & Dispositions Ledger

| Finding ID | Description | Severity | Disposition | Resolution / Action Item |
|:---|:---|:---:|:---:|:---|
| **F-01: Schema Drift** | Offline tablet rejected by server after office migration. | Critical | **ASSIGNED: Sprint 3** | Implement `schema_epoch` negotiation in `internal/domain/sync`. |
| **F-02: Media Choke** | 1.5GB uploads fail over 256kbps rig satellite connection. | High | **ASSIGNED: Sprint 3** | Implement TUS chunked resumable upload in `internal/storage`. |
| **F-03: Poison Pills** | Malformed sync payloads exhaust River queue workers. | High | **ASSIGNED: Sprint 2** | Add `river_poison_quarantine` table in Deliverable 2.2. |
| **F-04: Court Backdating**| Backdated tablet clocks contestable in court without RFC 3161. | High | **ASSIGNED: Sprint 3** | Add RFC 3161 Timestamp Token (TST) embedder in Tier L5. |
| **F-05: Standards Bottleneck**| Hand-coding 500 standards in Go is unscalable. | High | **ASSIGNED: Sprint 2** | Implement `cmd/standards-compiler` in Deliverable 2.3. |
| **F-06: NVMe Loss on Rig**| Server disk crash on offshore rig destroys air-gapped DB. | Critical | **ASSIGNED: Sprint 4** | Add `pgBackRest` dual-NVMe local shuttling in edge appliance. |

---

## 5. Final Formal Resolution

1.  **Sprint 2 Scope Enhanced**:
    *   **Deliverable 2.1**: Incorporate `ChecklistSchema` inside `pkg/rulesengine/schema.go`.
    *   **Deliverable 2.2**: Incorporate `river_poison_quarantine` and PostgREST-grade `is_local=true` session GUCs.
    *   **Deliverable 2.3**: Incorporate `cmd/standards-compiler` toolchain for automated YAML-to-CEL generation.
2.  **Sprint 3 Scope Enhanced**:
    *   Add TUS Resumable Chunked Media Engine.
    *   Add `schema_epoch` Offline Version Negotiation.
    *   Add RFC 3161 Trusted Timestamping integration.
3.  **Sprint 4 Scope Enhanced**:
    *   Add Dual-NVMe local WAL mirror (`pgBackRest`) to the Coolify-style sovereign appliance.

*The roadmap is now hardened against real-world industrial and offshore operational failure modes.*

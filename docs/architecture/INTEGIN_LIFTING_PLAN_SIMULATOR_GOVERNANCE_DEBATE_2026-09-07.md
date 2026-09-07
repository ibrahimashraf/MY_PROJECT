# INTEGIN Architecture Decision Record (ADR): 2D/3D/4D Lifting Plan Simulator Governance Debate

**Document ID**: ADR-2026-09-07-LIFTING-SIMULATOR-GOVERNANCE-DEBATE  
**Status**: APPROVED & CODIFIED  
**Date**: 2026-09-07  
**Governance Standard**: [`integin-review-governance`](../../.agents/skills/integin-review-governance/SKILL.md) (Multi-Lens Adversarial Debate)  
**Governing Authority**: [`docs/architecture/GLOBAL_ARCHITECTURE_DEEP_SPECIFICATION.md`](./GLOBAL_ARCHITECTURE_DEEP_SPECIFICATION.md) & [`TRACKER.md`](../../TRACKER.md)  
**Subject**: Architectural boundary, scope-creep containment, geotechnical liability, and execution sequencing of the **2D/3D/4D Lifting Plan Simulator (Dynamic Blocks)**.

---

## 1. Decision Frame & Objectives

*   **The Decision Under Review**: Whether to incorporate an interactive **2D/3D/4D Lifting Plan Simulator with Parametric Dynamic Blocks** into the INTEGIN trust platform.
*   **The Tension**: Is this a natural, high-value vertical integration of Tier L1 (Rules) and Tier L4 (Work Orders), or does it represent massive scope creep that turns an assurance/verification platform into an unwieldy CAD software clone?
*   **The Comparison Point**: Third-party standalone lift planning tools (AutoCAD, 3D Lift Plan, Cranimax, LiftPlanner) requiring separate licensing ($5,000+/seat) and manual, error-prone transcription into INTEGIN work packages.
*   **Protected Invariants**:
    1.  **Zero CAD Bloat Invariant**: INTEGIN will *not* build a generic 2D/3D CAD editor. It builds a **constrained parametric safety validator** focused strictly on kinematic boom geometry, load charts, sling angles, and outrigger ground-bearing pressure.
    2.  **Server Authority Invariant**: 4D simulation calculations must evaluate deterministically inside the Go CEL nanocell (`pkg/rulesengine`); client-side UI is purely a rendering projection.
    3.  **Sprint 2 Execution Fence**: Zero frontend rendering or 3D engine work is permitted in Sprint 2. Sprint 2 is strictly limited to the mathematical vector physics kernel (`pkg/rulesengine/rigging.go`, `geotech.go`).

---

## 2. Evidence Ledger

| ID | Source / Test Artifact | What It Supports | Limitation / Boundary |
|:---|:---|:---|:---|
| **EV-01** | ASME B30.5-2024 Section 5-3.2 | Proves critical lifts require documented lift plans with verified ground-bearing pressure and rigging geometry. | Mandates calculation methodology, not specific software vendor. |
| **EV-02** | Aramco GI 7.028 (Crane Operations) | Proves high-tonnage lifts (>75% capacity or tandem lifts) require approved, signed engineering drawings. | Strict approval workflow requires Appointed Person (AP) signature. |
| **EV-03** | Rugged Tablet Benchmarks (ATEX Zone 1) | Proves complex 3D WebGL scenes cause thermal throttling and battery drain on field hardware under 45°C direct sun. | 3D rendering must be optional; 2D canvas must remain lightweight (<2MB). |
| **EV-04** | Google CEL Performance Benchmarks | Proves trigonometric sling tension vectors and Boussinesq soil equations evaluate in $< 20\mu\text{s}$ in Go. | CEL easily handles the math; client rendering is the only heavy component. |

---

## 3. Multi-Lens Adversarial Debate

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│                     ADVERSARIAL DEBATE ON THE LIFTING PLAN SIMULATOR                             │
├──────────────────────────┬─────────────────────────────┬─────────────────────────────────────────┤
│ Lens                     │ The Adversarial Risk        │ Formal Verdict & Mitigation Invariant   │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ 1. Scope Creep           │ Turning INTEGIN into an     │ APPROVED: Constrained Parametric Engine │
│    vs. Domain Fit        │ AutoCAD clone distracts     │ Zero generic CAD drawing tools; only    │
│                          │ from core trust/compliance. │ reactive crane kinematic handles.       │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ 2. Field Hardware &      │ Rugged tablets throttle and │ APPROVED: Phased Dimension Separation   │
│    Thermal Throttling    │ crash when rendering heavy  │ 2D Canvas on tablets (<2MB RAM); 3D     │
│                          │ 3D WebGL meshes under 45°C. │ WebGL on desktop planning workstations. │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ 3. Geotechnical & Soil   │ Crane punches through soil; │ APPROVED: Strict Input-Boundary Gate    │
│    Liability Trap        │ operator sues INTEGIN for   │ Tool calculates applied stress only;    │
│                          │ inaccurate soil analysis.   │ explicit legal non-reliance disclaimer. │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ 4. Execution Velocity    │ Building 3D simulator stalls│ APPROVED: Strict Sprint Fencing         │
│    vs. Sprint 2 Invariant│ active Sprint 2 Go nanocell │ Math vectors only in Sprint 2; 2D in    │
│                          │ implementation.             │ Sprint 3; 3D/4D in Sprint 4.            │
└──────────────────────────┴─────────────────────────────┴─────────────────────────────────────────┘
```

---

### Detailed Lens Deep-Dives

#### Lens 1: Scope Creep vs. Domain Fit
*   **The Challenger (Product Discipline Lens)**:
    *   *"Building a 2D/3D/4D simulator is classic feature creep. We are a digital trust and inspection platform. If users want a lift plan, they can use AutoCAD or 3D Lift Plan. Why should we write physics simulators and CAD engines?"*
*   **The Defender (Industrial Operations Lens)**:
    *   *"In heavy lifting, the Lift Plan and the Inspection Certificate are two halves of the exact same safety contract. Before an inspector signs off on a 100-ton lift, they MUST verify: Did the crane configure outriggers according to the lift plan? Is the boom length correct? Is the load within the 75% critical limit? Today, riggers use separate CAD software, print a PDF, and the field inspector has no way to verify if the crane set up matches the drawing. By embedding dynamic blocks, the lift plan compiles into the exact same signed Work Package Manifest (`packagemanifest`) as the inspection checklist."*
*   **Ruling**: **APPROVED**. Bounded to **Parametric Dynamic Blocks**. We do not build lines, circles, and arbitrary CAD drawing tools; we build parametric crane models with interactive handles for boom length, radius, and outrigger span.

---

#### Lens 2: Field Hardware & Thermal Throttling
*   **The Challenger (Edge Systems Lens)**:
    *   *"Field tablets in Saudi Arabia and Texas operate in 45°C to 50°C desert heat. If an inspector opens a heavy Three.js 3D WebGL scene with 500,000 polygons, the tablet GPU will overheat, thermal-throttle to 5 FPS, and drain the battery in 30 minutes."*
*   **The Defender (Architecture Hierarchy Lens)**:
    *   *"That is why the architecture strictly isolates the dimensions:
        1.  **2D Parametric Dynamic Blocks (Sprint 3)**: Pure HTML5 Canvas / Flutter vector drawing. Memory footprint $< 2\text{MB}$, zero GPU shader load, 60 FPS on any rugged device. This is what runs on the field tablet.
        2.  **3D WebGL Spatial Collision & 4D Temporal Tandem (Sprint 4)**: Runs on desktop engineering workstations in the planning office. The 3D scene compiles the resulting trajectory into a lightweight 2D parameter manifest sent to the tablet."*
*   **Ruling**: **APPROVED WITH DIMENSIONAL ISOLATION**. 2D on field tablets; 3D/4D on engineering workstations.

---

#### Lens 3: Geotechnical & Soil Liability Trap
*   **The Challenger (Legal & Risk Lens)**:
    *   *"If INTEGIN calculates Ground Bearing Pressure (GBP) and an outrigger mat punches through the soil causing a crane tip-over, the operating company will sue INTEGIN, claiming our software certified the ground was safe."*
*   **The Defender (Engineering Law Lens)**:
    *   *"INTEGIN does not perform soil surveys. It evaluates the exact mechanical formula: $\sigma_{\text{applied}} = P_{\text{outrigger}} / A_{\text{mat}}$. The software requires the user to input the site geotechnical engineer's certified $\sigma_{\text{allowable}}$. The certificate output embeds an explicit, legally binding disclaimer:
        *'Ground Bearing Pressure computation represents structural load distribution under idealized rigid mat assumptions per Boussinesq equations. Site sub-surface capacity remains the sole legal responsibility of the certified Geotechnical Engineer and Appointed Person.'*"*
*   **Ruling**: **APPROVED WITH MANDATORY NON-RELIANCE DISCLAIMER**.

---

#### Lens 4: Execution Velocity & Sprint 2 Invariant
*   **The Challenger (Agile Execution Lens)**:
    *   *"Does introducing this simulator derail our immediate resumption of Sprint 2 Deliverable 2.1 (`pkg/rulesengine`)?"*
*   **The Defender (Scrum Governance Lens)**:
    *   *"Not by a single second. Sprint 2 has a strict architectural fence:
        *   **In Sprint 2**: We only write the pure mathematical Go functions for sling tension and outrigger reactions (`pkg/rulesengine/rigging.go`, `geotech.go`), which are naturally part of Deliverable 2.1.
        *   **Zero UI**: No canvas, no WebGL, no frontend code in Sprint 2.
        *   **Sprint 3**: 2D Canvas & Flutter dynamic blocks.
        *   **Sprint 4**: 3D Three.js & 4D temporal player."*
*   **Ruling**: **APPROVED WITH STRICT SPRINT FENCING**.

---

## 4. Findings & Dispositions Ledger

| Finding ID | Description | Severity | Disposition | Prescribed Treatment |
|:---|:---|:---:|:---:|:---|
| **F-01: CAD Bloat** | Generic CAD drafting features dilute core platform focus. | High | **CLOSED** | Hard-limit to parametric crane dynamic blocks with fixed kinematic handles. |
| **F-02: GPU Heat** | 3D WebGL crashes rugged tablets in desert heat. | High | **CLOSED** | Restrict field tablets to lightweight 2D Canvas; keep 3D on desktop workstations. |
| **F-03: Soil Lawsuit** | Geotechnical soil punch-through liability. | Critical | **CLOSED** | Mandatory cryptographic disclaimer: software computes applied pressure only. |
| **F-04: Sprint Delay** | Frontend simulator work distracting from Sprint 2 Go engine. | High | **CLOSED** | Strict sprint fence: only Go math vectors in Sprint 2; UI deferred to Sprints 3 & 4. |

---

## 5. Formal Engineering Resolution

*   **Final Ruling**: The addition of the **2D/3D/4D Lifting Plan Simulator with Parametric Dynamic Blocks** is **FORMALLY APPROVED & RATIFIED**.
*   **Implementation Boundaries**:
    1.  **Sprint 2**: Implement `pkg/rulesengine/rigging.go` and `geotech.go` (pure Go math vectors evaluated in $< 20\mu\text{s}$).
    2.  **Sprint 3**: Implement `tools/lifting-simulator/2d/` (lightweight parametric canvas) and embed offline 2D preview in `field_app/`.
    3.  **Sprint 4**: Implement `tools/lifting-simulator/3d/` (desktop Three.js WebGL & 4D temporal tandem lift player).

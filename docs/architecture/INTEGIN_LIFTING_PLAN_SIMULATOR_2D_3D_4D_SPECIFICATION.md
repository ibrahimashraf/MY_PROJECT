# 🏗️ INTEGIN 2D/3D/4D Lifting Plan Simulator & Dynamic Blocks Engine Specification

**Document ID:** SPEC-2026-09-07-LIFTING-SIMULATOR-4D  
**Status:** APPROVED & CODIFIED  
**Date:** 2026-09-07  
**Governance Standard:** [`integin-review-governance`](../../.agents/skills/integin-review-governance/SKILL.md)  
**Companion Architecture:** [`GLOBAL_ARCHITECTURE_PLAN.md`](./GLOBAL_ARCHITECTURE_PLAN.md), [`GLOBAL_ARCHITECTURE_DEEP_SPECIFICATION.md`](./GLOBAL_ARCHITECTURE_DEEP_SPECIFICATION.md)  
**Scope:** Physics Kernel, 2D Parametric Dynamic Blocks, 3D Spatial Collision & Ground Bearing Pressure (GBP), 4D Time-Stepped Tandem Lift Trajectory, and 12-Tier Architectural Seams.

---

## 📑 Table of Contents

1. [Executive Vision: From Static Paper to 4D Living Physics](#1-executive-vision-from-static-paper-to-4d-living-physics)
2. [The Multi-Dimensional Simulation Matrix (2D vs. 3D vs. 4D)](#2-the-multi-dimensional-simulation-matrix-2d-vs-3d-vs-4d)
3. [Mathematical Physics & Geotechnical Engine](#3-mathematical-physics--geotechnical-engine)
4. [Parametric 2D/3D Dynamic Blocks Architecture](#4-parametric-2d3d-dynamic-blocks-architecture)
5. [4D Time-Series Trajectory & Tandem Lift Dynamics](#5-4d-time-series-trajectory--tandem-lift-dynamics)
6. [Integration into the Master 12-Tier Architecture](#6-integration-into-the-master-12-tier-architecture)
7. [Implementation Roadmap across Sprints](#7-implementation-roadmap-across-sprints)

---

## 1. Executive Vision: From Static Paper to 4D Living Physics

In heavy industrial operations (oil & gas, petrochemical, offshore wind, nuclear, infrastructure), **Critical Lift Plans** are legally mandatory before moving any high-tonnage load. 

Today, lift planning is severely fragmented:
*   Draftsmen create static 2D AutoCAD drawings with disconnected lines.
*   Engineers use manual spreadsheets to calculate ground-bearing pressure under outrigger mats.
*   Riggers on site guess boom clearances and swing trajectories.
*   When crane radius shifts by 1 meter in the field, the entire paper lift plan becomes obsolete and dangerous.

### The INTEGIN Vision:
A fully integrated **Browser & Tablet 2D/3D/4D Dynamic Lifting Simulator** natively compiled into the trust platform:
*   **2D Dynamic Blocks**: Interactive parametric canvas where dragging the load dynamically articulates the crane boom, stretches the cable, updates outrigger reactions, and flags load chart % in real-time.
*   **3D Spatial Clearance**: Full WebGL volumetric scene calculating obstacle clearance (pipe racks, flare stacks, power lines) and soil stress distribution heatmaps under outrigger timber mats.
*   **4D Time Trajectory**: Time-stepped simulation ($t_0 \rightarrow t_{\text{final}}$) modeling load rotation, wind gust dynamics, and **tandem dual-crane load share shifts** as a column is lifted from horizontal to vertical.

---

## 2. The Multi-Dimensional Simulation Matrix

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│                             INTEGIN MULTI-DIMENSIONAL LIFT MATRIX                                │
├──────────────────────────┬─────────────────────────────┬─────────────────────────────────────────┤
│ Dimension                │ Core Capabilities           │ Key Output / Engineering Safety Gate    │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ **2D Dynamic Blocks**    │ • Parametric SVG/Canvas CAD │ • Immediate ASME B30.5 Load Chart %     │
│ (Plan & Elevation)       │ • Interactive boom radius   │ • Rigging sling angle tension derating  │
│                          │ • Outrigger span geometry   │ • Minimum hook clearance & boom angle   │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ **3D Spatial Geometry**  │ • WebGL / Three.js meshes   │ • 3D Volumetric obstacle collision      │
│ (Volumetric & Soil)      │ • Rig/Refinery 3D IFC/OBJ   │ • Boussinesq Ground Bearing Pressure    │
│                          │ • Outrigger mat sizing      │ • Outrigger pad punch-through warning   │
├──────────────────────────┼─────────────────────────────┼─────────────────────────────────────────┤
│ **4D Temporal Physics**  │ • Time-stepped kinematics   │ • Tandem crane load-share percentage    │
│ (Time + Trajectory)      │ • Center of Gravity shift   │ • Dynamic wind gust load oscillation    │
│                          │ • Tailing crane rotation    │ • 1-Click Approved 4D Lift Execution    │
└──────────────────────────┴─────────────────────────────┴─────────────────────────────────────────┘
```

---

## 3. Mathematical Physics & Geotechnical Engine

The simulator runs on deterministic mathematical formulas evaluated inside the **Google CEL nanocell (`pkg/rulesengine`)**:

### 3.1 Rigging Geometry & Sling Tension Vector
For a load $W$ lifted with $n$ sling legs at angle $\theta$ to the horizontal:
$$T_{\text{leg}} = \frac{W \cdot d_{\text{opposite}}}{(d_1 + d_2) \cdot \sin(\theta)} \times \text{DynamicFactor}$$
*   **Angle Derating Gate**: If $\theta < 30^\circ$, hard-block lift plan generation (`ERR_CRITICAL_SLING_ANGLE_COLLAPSE`).

### 3.2 Outrigger Reaction & Ground Bearing Pressure (GBP)
The reaction force on each outrigger pad $i \in \{1, 2, 3, 4\}$:
$$P_i = \frac{W_{\text{crane}} + W_{\text{counterweight}} + W_{\text{load}}}{4} \pm \frac{M_x \cdot y_i}{\sum y^2} \pm \frac{M_y \cdot x_i}{\sum x^2}$$
The maximum ground bearing pressure $\sigma_{\text{actual}}$ under timber mat area $A_{\text{mat}}$:
$$\sigma_{\text{actual}} = \frac{P_i}{A_{\text{mat}}} \le \sigma_{\text{allowable\_soil}}$$
*   **Geotechnical Safety Gate**: If $\sigma_{\text{actual}} > \sigma_{\text{allowable\_soil}}$, the simulator automatically enlarges mat dimensions ($A_{\text{mat}} = 2.0\text{m} \times 3.0\text{m}$) or halts plan approval.

---

## 4. Parametric 2D/3D Dynamic Blocks Architecture

### 4.1 What is a "Dynamic Block" in INTEGIN?
A Dynamic Block is not a static drawing; it is a **reactive mathematical object**:
*   A crane block possesses kinematic handles:
    *   `Handle(Radius)`: Dragging changes boom angle $\alpha$ and recalculates capacity.
    *   `Handle(OutriggerSpread)`: Toggles between 100%, 75%, 50%, and 0% (on-rubber) outrigger extension.
    *   `Handle(BoomLength)`: Snaps to physical crane boom insert sections (e.g. 12.8m, 17.5m, 22.2m, 31.0m).

### 4.2 Dynamic Block JSON Data Schema (`pkg/simulator/schema.go`)
```go
type DynamicCraneBlock struct {
    BlockID          string             `json:"block_id"`           // "LIEBHERR_LTM_1100_4_2"
    Manufacturer     string             `json:"manufacturer"`       // "Liebherr"
    Model            string             `json:"model"`              // "LTM 1100-4.2"
    ChassisLength    float64            `json:"chassis_length_m"`   // 13.98
    ChassisWidth     float64            `json:"chassis_width_m"`    // 2.75
    OutriggerSpanX   float64            `json:"outrigger_span_x_m"` // 8.58
    OutriggerSpanY   float64            `json:"outrigger_span_y_m"` // 7.00
    CounterweightTon float64            `json:"counterweight_ton"`  // 28.2
    TelescopicBooms  []BoomSectionSpec  `json:"boom_sections"`
    LoadChartMatrix  map[string]float64 `json:"load_chart_matrix"`  // "len:rad" -> MaxTon
    KinematicHandles []KinematicHandle  `json:"kinematic_handles"`
}

type KinematicHandle struct {
    HandleType   string    `json:"handle_type"`   // "RADIUS", "BOOM_ANGLE", "SLEW_ANGLE"
    CurrentValue float64   `json:"current_value"`
    MinValue     float64   `json:"min_value"`
    MaxValue     float64   `json:"max_value"`
    Unit         string    `json:"unit"`          // "METERS", "DEGREES"
}
```

---

## 5. 4D Time-Series Trajectory & Tandem Lift Dynamics

### 5.1 The Tandem Lift Challenge
When two cranes lift a heavy distillation column (one main crane at the head, one tailing crane at the skirt):
*   At $t = 0$: Column is horizontal. Main crane carries 50%, Tailing crane carries 50%.
*   At $t = \text{mid}$: Column tilts to $45^\circ$. Center of gravity shifts toward the main crane. Main crane carries 72%, Tailing crane carries 28%.
*   At $t = \text{vertical}$: Tailing crane load drops to 0%; Main crane carries 100%.

### 5.2 4D Simulation Engine Kinematics
The 4D engine evaluates $N$ discrete time slices $\Delta t = 0.5\text{s}$:
$$\text{State}(t) = \left\{ \text{BoomAngle}_1(t), \text{Radius}_1(t), \text{LoadShare}_1(t), \text{BoomAngle}_2(t), \text{Radius}_2(t), \text{LoadShare}_2(t) \right\}$$
*   **The 4D Safety Gate**: If at any millisecond $t$ during the rotation, either crane exceeds **75% of rated capacity** (the international tandem lift limit per ASME B30.5), the simulation pauses, flashes red, and outputs the exact collision or overload timestamp.

---

## 6. Integration into the Master 12-Tier Architecture

```
┌───────┬──────────────────────────────┬─────────────────────────────────────────────────────────┐
│ Tier  │ Layer Name                   │ Simulator Role & Operational Seam                       │
├───────┼──────────────────────────────┼─────────────────────────────────────────────────────────┤
│ **L1**│ Dynamic AST Engine           │ Evaluates load charts, sling angles, and GBP in <15µs.  │
│ **L3**│ Inspection Package Scoping   │ Binds Lift Plan complexity (Standard vs. Critical vs. 4D)│
│ **L4**│ Enterprise Work Orders       │ Serializes 4D trajectory JSON into Work Package Manifest│
│ **L5**│ Certificate Governance       │ 4-Eyes Appointed Person (AP) review & cryptographic seal│
│ **L7**│ Tool Calibration Registry    │ Enforces verified calibration on load cells & mats      │
│ **L8**│ Rugged Field Tablet          │ Displays interactive 2D/3D lift plan offline to riggers │
│ **L10**│ Merkle-CRDT Audit Ledger    │ Cryptographically hashes full 4D trajectory into audit  │
│ **L11**│ Stateless QR Trust Gateway  │ Public QR scan displays 3D animated lift plan playback   │
└───────┴──────────────────────────────┴─────────────────────────────────────────────────────────┘
```

---

## 7. Implementation Roadmap across Sprints

### 📌 Phase 1: Sprint 2 (Foundational Data Models)
*   Implement `pkg/rulesengine/rigging.go`: Mathematical vector models for sling tensions and ASME B30.5 load charts.
*   Implement `pkg/rulesengine/geotech.go`: Ground Bearing Pressure (GBP) equations.

### 📌 Phase 2: Sprint 3 (2D Parametric Canvas & Mobile Tablet)
*   Build `tools/lifting-simulator/2d/`: Interactive HTML5 Canvas / Flutter vector engine for crane plan and elevation views.
*   Implement Dynamic Block library for major crane models (Liebherr, Tadano, Kato, Manitowoc).
*   Add offline 2D lift plan preview into `field_app/`.

### 📌 Phase 3: Sprint 4 (3D WebGL & 4D Temporal Tandem Engine)
*   Build `tools/lifting-simulator/3d/`: Three.js WebGL spatial clearance engine with refinery 3D OBJ/IFC model import.
*   Implement 4D time-stepped trajectory player simulating tandem dual-crane lifts.
*   Integrate with `verify.integin.com` so QR codes embed interactive 3D simulation replays.

---

*Formally codified into the INTEGIN Master Architecture Repository.*

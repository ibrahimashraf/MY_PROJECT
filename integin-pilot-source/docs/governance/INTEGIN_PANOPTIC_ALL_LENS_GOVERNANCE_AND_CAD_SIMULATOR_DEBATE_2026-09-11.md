# INTEGIN Panoptic All-Lens Architecture & CAD/3D Simulator Governance Debate

## Decision frame
- **Decision**: 
  1. Panoptic audit of all 12 INTEGIN architectural tiers (L0–L11), assessing isolation boundaries, cryptographic guarantees, offline replication, and runtime economics.
  2. Architectural feasibility, boundary definition, and implementation strategy for evolving the 2D/3D simulator into a parametric CAD/BIM/DCC (AutoCAD / Blender / 3ds Max / Cinema4D) engine with dynamic blocks, multi-format import/export (DXF, DWG, IFC, STEP, glTF, OBJ), and PDF vector ingestion in Go.
- **Comparison Point**: Current 3D WebGL tandem lift simulator (`tools/lifting-simulator/3d/`, Three.js + deterministic kinematics) vs Full Parametric Solid Modeling & Vector Translation Engine.
- **Protected Boundaries**:
  - Hazard 01 & Invariant 4: Hard multi-tenant RLS isolation in PostgreSQL 16.
  - Hazard 12: 100% air-gapped sovereign execution without internet egress.
  - Hazard 28: Zero full-text copyright-infringing ingestion of proprietary standards.
  - Invariant 1: Computational nanocells evaluate in $< 50\mu\text{s}$ with zero memory leaks.
  - Public Verifier boundary: $0 backend compute for third-party verification via browser WebCrypto.

---

## Technical Feasibility: Go as a Universal 2D/3D CAD & Modeling Engine

### Can Go do this? Engineering Reality Matrix
| Functional Requirement | Feasibility in Pure Go | Optimal Production Architecture | Technical Trade-off & Limitations |
|---|:---:|---|---|
| **Parametric Geometry & Dynamic Blocks** | **100% (Native)** | **Go Core Engine** (Algorithmic DAG, constraint solver, spatial indexing via R-tree/BVH). | Go is optimal for high-speed concurrent graph evaluation and parametric math. |
| **DXF (Import / Export)** | **100% (Native)** | **Pure Go Parser** (`pkg/cad/dxf`). DXF is open ASCII/binary tagged data. | Complete fidelity for 2D/3D wireframe, layers, polylines, and blocks. |
| **DWG (Import / Export)** | **Partial (FFI/Cgo)** | **LibreDWG / Open Design Alliance via Cgo or WASM sidecar**. | DWG is a proprietary binary format with variable R12–R2024 byte packing; pure Go re-implementation is high friction. |
| **glTF / OBJ / STL (3D DCC)** | **100% (Native)** | **Pure Go AST** (`qmuntal/gltf` or native JSON/binary parser). | Native Go handles scene hierarchies, PBR materials, node transforms, and mesh buffers. |
| **STEP (AP203/AP214) & IFC (BIM)** | **Moderate** | **Go Parser + OpenCASCADE / CGAL Cgo Bridge**. | Express schema parsing is viable in Go, but solid B-Rep topology (NURBS boundary representation, booleans) requires a dedicated B-Rep kernel. |
| **PDF Vector Extraction (like AutoCAD)** | **100% (Native)** | **Go Content Stream Parser** (`pdfcpu` or native PDF operator scanner). | Extracts path operators (`m`, `l`, `c`, `re`, `f`, `S`), transforms affine matrices, and maps bezier curves directly into CAD polylines. |
| **Viewport Rendering (60+ FPS)** | **Hybrid (Go + WebGPU)** | **Frontend WebGPU / Three.js** connected to Go backend/WASM over binary ArrayBuffers. | Browsers cannot execute native Go graphics directly without compiling to WebAssembly (`GOOS=js GOARCH=wasm`) or piping scene graphs to WebGL/WebGPU. |

---

## Evidence Ledger: 12-Tier Architecture Baseline
| ID | Tier / Source | What it Supports | Panoptic Limitation |
|---|---|---|---|
| E-01 | L0 Base: `migrations/0001_initial_schema.sql`–`0074_partitioning_and_cqrs.sql` | PostgreSQL 16 schema, tenant isolation, range partitioning, outbox pattern | Large CAD binary blobs (DWG/IFC/glTF > 50MB) cannot live in relational tables; must use S3/RustFS. |
| E-02 | L1 Query: `pkg/queryengine/` | Dual-mode hybrid search, vector embeddings, BM25 ranker | Text-centric; lacks spatial geometric indexing (R-Tree / 3D spatial query). |
| E-03 | L2 Rules: `pkg/rulesengine/` | Sandboxed CEL evaluation, $<50\mu\text{s}$ nanocell, sensor jitter analysis | Evaluates scalar/boolean telemetry; does not solve 3D geometric collisions or finite element stress. |
| E-04 | L4 Manifest: `internal/packagemanifest/` | 1-Click execution binding, sha256 package cryptographic manifests | Hashes tabular work orders; not yet bound to versioned parametric CAD block states. |
| E-05 | L8 Enclave: `tools/lifting-simulator/3d/` | Local Three.js tandem lift simulator, clearance checks, ground bearing pressure | Script-driven visual approximation; lacks full constraint-based dynamic block engine. |
| E-06 | L10 Ledger: `pkg/verification/transparency.go` | RFC 6962 Certificate Transparency append-only Merkle log | Logs inspection certificates; does not log CAD revision lineage or engineering delta trees. |
| E-07 | L11 Verifier: `internal/certificatepublichttp/` | Ed25519 WebCrypto public zero-knowledge verification | Verifies static JSON signatures; does not verify 3D scene clearance proofs. |

---

## Findings and Dispositions (All-Lens Panoptic Debate)

| ID | Lens | Finding & Adversarial Challenge | Evidence | Confidence | Disposition | Required Next Action / Architecture Decision |
|---|---|---|---|---|---|---|
| F-01 | **Scope & Core Mission** | Transforming INTEGIN from an industrial inspection & compliance engine into a monolithic CAD suite (AutoCAD + Blender + Cinema4D) creates severe scope drift and multi-year kernel development. | L0–L11 architecture | High | Confirmed | **Bound scope**: Build a *Domain-Specific Rigging & Heavy-Lift CAD Kernel* in Go (focused on rigging geometry, clearance envelopes, and lift rigging blocks), with import/export bridges (DXF, glTF, PDF) rather than a generic general-purpose 3D modeling suite. |
| F-02 | **Geometric Kernel Architecture** | Pure Go can parse formats and evaluate parametric trees, but implementing full non-manifold B-Rep booleans (NURBS solid union/difference) from scratch in pure Go requires $>100\text{k}$ lines of complex numerical geometry. | CAD industry history | High | Confirmed | Adopt a 3-tier geometry stack: **Go** for parametric state machine, DAG dependency resolution, constraint solving, and format serialization; **WebGPU/Three.js** for interactive manipulation; **OpenCASCADE WASM/Cgo** for heavy B-Rep booleans. |
| F-03 | **PDF Vector Ingestion** | Extracting vector paths from PDFs is trivial in Go, but converting raw unstructured PDF line segments into semantic CAD dynamic blocks requires topological polyline joining and text/layer heuristic clustering. | PDF spec ISO 32000-1 | High | Confirmed | Implement `pkg/cad/pdfvector` in Go: parse PDF path streams, execute line-snapping tolerance ($\epsilon = 0.05\text{mm}$), join contiguous bezier segments into closed loops, and extract CAD geometry. |
| F-04 | **Tenant Isolation & Binary Storage** | CAD models, point clouds, and mesh files generate 10MB–500MB payloads. Storing these in PostgreSQL violates RLS throughput and causes table bloat. | Hazard 01 & Hazard 21 | High | Confirmed | Enforce storage segregation: CAD metadata and cryptographic hashes live in PostgreSQL under strict RLS (`tenant_id`); binary geometric buffers (glTF, DXF, PDF) stream to tenant-prefixed RustFS S3 buckets (`s3://tenant-uuid/cad-blobs/`). |
| F-05 | **Air-Gapped Sovereign Boundary** | Advanced 3D/CAD tools often rely on cloud GPU rendering pipelines (WebRTC streaming), which violates air-gapped sovereign operational requirements (Hazard 12). | Hazard 12 | High | Confirmed | Viewport rendering must remain 100% client-side WebGL/WebGPU in the browser/appliance, requiring 0 external cloud compute or proprietary licensing servers. |
| F-06 | **Audit & Cryptographic Lineage** | CAD changes on critical rigging plans currently lack cryptographic audit trails; unauthorized modification of crane radii or outrigger coordinates could lead to site collapse. | L10 Merkle Log | High | Confirmed | Extend L10 Merkle Transparency Log (`pkg/verification/transparency.go`) to record SHA-256 state hashes of every CAD block mutation and lift plan revision. |
| F-07 | **Statutory PE Liability & Certification** | CAD simulators that calculate soil bearing pressure and dynamic boom deflection create significant civil liability if contractors treat simulated clearance as licensed engineering approval. | ASME B30.5 / OSHA 1926 | High | Confirmed | All exported CAD drawings, DXF sheets, and simulation reports must automatically watermark: "ENGINEERING STUDY ONLY — NOT VALID FOR FABRICATION OR CRITICAL LIFT WITHOUT LICENSED PE STAMP". |

---

## Recommended Architecture: INTEGIN Rigging-CAD Engine

```
                               ┌────────────────────────────────────────────────────────┐
                               │           INTEGIN WebGPU Client Viewport              │
                               │  - Three.js / WebGPU Scene Graph                       │
                               │  - Interactive Gizmos & Dynamic Block Grips            │
                               │  - 2D Canvas Drafting Overlay                          │
                               └───────────────────────────▲────────────────────────────┘
                                                           │ Binary ArrayBuffer / JSON
                                                           ▼
┌──────────────────────────────────────────────────────────────────────────────────────┐
│                            INTEGIN Go Backend Core Engine                            │
│                                                                                      │
│  ┌───────────────────────┐  ┌────────────────────────┐  ┌─────────────────────────┐ │
│  │   pkg/cad/engine      │  │    pkg/cad/importers    │  │    pkg/cad/exporters    │ │
│  │ - Parametric DAG      │  │ - DXF Parser (Pure Go)  │  │ - DXF Writer (Pure Go)  │ │
│  │ - Dynamic Block State │  │ - glTF / OBJ Parser     │  │ - glTF Binary (GLB)     │ │
│  │ - Constraint Solver   │  │ - PDF Vector Extractor  │  │ - Vector PDF Exporter   │ │
│  │ - Tandem Lift Physics │  │ - DWG / IFC Bridge      │  │ - SVG / STEP Bridge     │ │
│  └───────────▲───────────┘  └───────────▲────────────┘  └────────────▲────────────┘ │
│              │                          │                            │               │
└──────────────┼──────────────────────────┼────────────────────────────┼───────────────┘
               │                          │                            │
               ▼                          ▼                            ▼
┌───────────────────────────┐ ┌──────────────────────────┐ ┌───────────────────────────┐
│   PostgreSQL 16 (RLS)     │ │  RustFS S3 Object Store  │ │   L10 Merkle Transparency   │
│ - Parametric metadata     │ │ - Large CAD binaries     │ │ - Immutable revision log    │
│ - Tenant isolation GUCs   │ │ - glb/dxf blob streams   │ │ - Signed cryptographic cert │
└───────────────────────────┘ └──────────────────────────┘ └───────────────────────────┘
```

## Next Safe Implementation Steps
1. **Build `pkg/cad/dxf` in Go**: Pure Go reader and writer for AutoCAD 2D/3D entities (LINE, POLYLINE, CIRCLE, ARC, INSERT, BLOCK).
2. **Build `pkg/cad/pdfvector` in Go**: Ingestion engine that extracts vector paths and text from technical PDF drawing sheets.
3. **Build Dynamic Block Parametric Solver in Go**: Graph data structure where modifying a property (e.g. crane boom angle $\theta$) automatically propagates to dependent rigging lines, load center of gravity, and outrigger reactions.

# INTEGIN Go Coder Agent Adversarial CAD & Engine Debate

**Author / Lens**: `go_coder_agent` (Zero-Token-Waste Go Systems Engineer)  
**Target Subject**: `docs/governance/INTEGIN_PANOPTIC_ALL_LENS_GOVERNANCE_AND_CAD_SIMULATOR_DEBATE_2026-09-11.md`  
**Evaluation Standard**: Lean-ctx context compression, Ponytail reuse ladder, 64-byte cache alignment, multi-tenant RLS, Hazard 40 (Zero Transitive Cgo), and clean-cache quality gates.

---

## 1. Adversarial Technical Objections & Forensic Critique

### Challenge 1: The CGO Trap (Hazard 40 Violation)
- **Document Proposal (§ Technical Feasibility & F-02)**: Recommends an "OpenCASCADE / CGAL Cgo Bridge" for solid B-Rep booleans.
- **`go_coder_agent` Pushback**: **HARD REJECT**. Linking OpenCASCADE or CGAL via Cgo destroys the single-binary deployment model, inflates binary size by >200MB, breaks cross-compilation (`CGO_ENABLED=0`), and creates glibc runtime incompatibilities across sovereign appliance operating systems.
- **Mandated Architecture**: 
  - The Go server MUST remain 100% pure Go (`CGO_ENABLED=0`).
  - Solid boolean operations must either be:
    1. Evaluated client-side in the browser via an isolated WebAssembly (WASM) Web Worker (`opencascade.wasm`), OR
    2. Computed via pure Go CSG (Constructive Solid Geometry) mesh clipping for planar polyhedra.

### Challenge 2: Interface-Heavy Polymorphism vs CPU Cache Lines (L1 Data Cache Misses)
- **Document Baseline (`pkg/cad/dxf`)**: Uses `[]Entity` interfaces containing pointers to individual `Line`, `Circle`, `Polyline` structs.
- **`go_coder_agent` Pushback**: Interface slicing causes heap escape (`runtime.newobject`), pointer chasing, and CPU cache thrashing during high-frequency geometric traversals (e.g., 60Hz tandem lift clearance sweeps across 10,000 entities).
- **Mandated Optimization**:
  - Hot simulation loops must operate on contiguous primitive arrays: `struct { Positions []float32; Radii []float32 }` (Struct of Arrays / SoA) or packed 64-byte cache-aligned nodes.
  - Retain `dxf.Entity` interface strictly for ASCII file serialization/deserialization at I/O boundaries.

### Challenge 3: Cross-Tenant Asset Leakage via Object Storage (Hazard 01)
- **Document Architecture (§ F-04)**: Segregates CAD binaries to RustFS S3 buckets (`s3://tenant-uuid/cad-blobs/`).
- **`go_coder_agent` Pushback**: Storing tenant UUID in S3 keys is insufficient if presigned URLs or S3 access tokens are generated without enforcing PostgreSQL session GUCs (`INTEGIN.tenant_id`). A malicious tenant could request a presigned download for an adjacent tenant's CAD drawing if handler validation relies on client-supplied path params.
- **Mandated Enforcement**:
  - Every CAD blob retrieval must execute an initial database check:
    ```sql
    SELECT set_config('INTEGIN.tenant_id', $1, true);
    SELECT s3_key, sha256_digest FROM cad_blobs WHERE id = $2;
    ```
  - If RLS yields 0 rows (SQLSTATE 42501 or empty set), access is denied before S3 presigned generation occurs.

### Challenge 4: Floating-Point Division by Zero (Hazard 31)
- **Document Kinematics (`rigging_solver.go`)**: Computes outrigger moment arms with `deltaZ := mx / (2.0 * c.OutriggerSpreadZM)`.
- **`go_coder_agent` Pushback**: If outrigger spread is accidentally initialized to `0.0` or missing in payload, Go emits `+Inf` or `NaN`, which silently propagates through JSON serialization as `null` or crashes downstream WebGL matrices.
- **Mandated Guardrail**:
  - All kinematic divisors MUST enforce explicit non-zero thresholds:
    ```go
    if c.OutriggerSpreadZM <= 0.1 || c.OutriggerSpreadXM <= 0.1 {
        return OutriggerPressures{}, errors.New("outrigger spread too narrow for static equilibrium")
    }
    ```

### Challenge 5: Revision Tamper-Proofing in Exported Artifacts
- **Document Recommendation (§ F-07)**: UI watermarking for PE liability.
- **`go_coder_agent` Pushback**: UI watermarks can be removed with browser DevTools inspection. If a user downloads a DXF or GLB file, the UI watermark is lost.
- **Mandated Cryptographic Seam**:
  - In `pkg/cad/dxf`: Inject an immutable `COMMENT` entity (Group Code 999) with cryptographic sha256 digest of the work order manifest + statutory non-certified disclaimer.
  - In `pkg/cad/gltf`: Inject `asset.extras.integin_audit` containing the L10 Merkle leaf inclusion proof and verification signature.

---

## 2. Adversarial Governance Matrix

| Ref | Target Finding | `go_coder_agent` Disposition | Concrete Hardening Required |
|---|---|---|---|
| **ADV-01** | F-02 (Cgo Bridge) | **REVISE** | Strike Cgo; mandate client-side WASM or pure Go CSG. |
| **ADV-02** | F-04 (CAD S3 Blobs) | **AMEND** | Bind S3 presigned URLs strictly behind session RLS GUC validation. |
| **ADV-03** | F-07 (PE Liability) | **EXPAND** | Inject immutable cryptographic metadata stamps into exported DXF (Code 999) and GLB (`asset.extras`). |
| **ADV-04** | Engine Hot Path | **ADD** | Mandate flat contiguous float32 buffers for 60Hz physics, reserving interface ASTs for file I/O. |
| **ADV-05** | Hazard 31 | **CONFIRM** | Enforce non-zero denominator guards on all kinematic division operations. |

---

## 3. Conclusion & Next Safe Action
The panoptic debate correctly identified domain boundaries, but risked introducing Cgo dependency bloat and cache-unfriendly interface architectures. With ADV-01 through ADV-05 accepted, the INTEGIN CAD engine preserves sovereign air-gapped purity, zero transitive Cgo, and hard RLS tenant containment.

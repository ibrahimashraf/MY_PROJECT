# Findings

| ID | Finding or evidence | Source | Confidence | Limitation or unresolved question |
|---|---|---|---|---|
| F-01 | The 2D simulator explicitly performs geometry only and rejects lift-rating use; crane values are illustrative demo data. | `tools/lifting-simulator/2d/app.js:5-10`; `tools/lifting-simulator/2d/cranes.js:3-9` | High | Does not prove absence of other rated-data code; source review found no chart catalog in this simulator. |
| F-02 | The 3D simulator has a local Three.js scene, load/block library, timeline, environment presets, and an export button. | `integin-pilot-source/tools/lifting-simulator/3d/index.html:1-218`; `app.js:1-42` | High | Scene inputs and many defaults remain demo/user data rather than a governed product workflow. |
| F-03 | The 3D client submits scene state to `/api/v1/liftviews/export`; the handler solves, gates, draws DXF, builds a deterministic PDF, and returns a ZIP. | `3d/app.js:1117-1150,1400-1424`; `internal/liftviewexport/handler.go:38-208` | High | Current handler is stateless and accepts supplied capacity values; it does not establish OEM chart provenance or full work-order identity binding. |
| F-04 | Go already contains constrained CAD primitives, dynamic blocks, DXF read/write, GLB packing, rigging/outrigger math, and lift-view rendering. | `pkg/cad/engine/*`; `pkg/cad/dxf/*`; `pkg/cad/gltf/glb.go`; `pkg/rulesengine/*` | High | These are foundations, not a complete rated CAD/lift-plan product. |
| F-05 | The codebase has no visible IFC/STEP/OBJ importer or Plant 3D/SP3D adapter; GLB packing is present but not a full neutral interchange workflow. | Repository source search; `pkg/cad/gltf/glb.go:26-49` | Medium | Search can miss generated or external adapters; confirm with architecture owner before claiming absolute absence. |
| F-06 | The current export bundle contains `plan.dxf`, `elevation.dxf`, `liftplan.pdf`, and `export-meta.json`, with tests for ZIP contents and fail-closed overload. | `internal/liftviewexport/handler.go:179-207`; `handler_test.go:63-121` | High | It is a technical export, not yet a client portal, branded dossier, or signed execution record. |
| F-07 | The approved simulator ADR forbids generic CAD and requires server-authoritative physics plus explicit geotechnical non-reliance language. | `docs/architecture/INTEGIN_LIFTING_PLAN_SIMULATOR_GOVERNANCE_DEBATE_2026-09-07.md:17-20,67-104,119-125` | High | Static-equipment modeling is therefore a separate product decision, not an implicit extension of the simulator. |
| F-08 | SEG's public post claims Inventor-based parametric 3D modeling, 2D fabrication drawings, BOMs, and customization; comments ask about Plant 3D/SP3D import. | User-provided LinkedIn URL `https://lnkd.in/p/eg6A8sPK` | Medium | Public marketing claim; no independent technical validation or licensing evidence. |
| F-09 | CranePro 3D publicly demonstrates 3D lifting simulation and crane-selection optimization; its post is product promotion, not a specification. | User-provided LinkedIn URL `https://lnkd.in/p/e_bU7t4q` | Medium | Exact solver, data, and certification boundaries are undisclosed. |
| F-10 | Construct-Tech publicly advertises AI-assisted lift plans, cloud hosting, high-quality 3D visuals, LOLER/BS alignment, and custom branding, but gives little implementation detail. | User-provided URL `https://www.construct-tech.co.uk/3dliftplan` | Medium | Marketing page; verify claims, standards basis, data handling, and deliverables before comparison. |
| F-11 | Existing architecture already records IFC/OBJ import, load-chart interpolation, 3D/4D timeline, and statutory dossier export as planned work. | `docs/architecture/INTEGIN_LIFTING_PLAN_SIMULATOR_2D_3D_4D_SPECIFICATION.md:154-169`; `TRACKER.md:602-628` | High | Roadmap items are not implementation evidence and must not be counted as complete. |
| F-12 | Existing platform foundations include evidence exports, signed manifests, AASX, regulatory dossiers, and identity/authority gates, but their connection to lift-plan delivery is not complete. | `TRACKER.md:176-184`; `pkg/domain/enterprise_export.go:216-350` | High | Requires seam-level verification before reuse. |

## Gap classification

| Gap | Classification | Reason |
|---|---|---|
| Rated OEM load charts and configuration interpolation | Build | Core trust dependency; current simulator has no governed chart source. |
| Server-authoritative 2D pilot and signed work-order binding | Build | Existing stateless export is a useful seam but not a complete operational product. |
| IFC/STEP/OBJ neutral import and Plant 3D/SP3D interchange | Build or integrate | First prove neutral format; native plug-ins are optional follow-on. |
| 3D/4D collision and trajectory | Build after rated 2D | Existing demo visuals are not enough to authorize safety decisions. |
| Branded cloud lift-plan portal | Build after core workflow | Competitive delivery layer; security and tenant scope are prerequisites. |
| Full pressure-vessel CAD/BOM authoring | Defer or separate product | Conflicts with approved zero-CAD-bloat boundary and is not required for INTEGIN's inspection wedge. |
| AI-generated authoritative lift plans | Reject for safety path | Deterministic, reviewable rules are required. |

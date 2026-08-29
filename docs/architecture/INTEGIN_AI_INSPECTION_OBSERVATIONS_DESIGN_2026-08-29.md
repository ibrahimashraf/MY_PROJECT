# INTEGIN AI Inspection Observations — Design — 2026-08-29

**Status:** design-only, no model training, no route mount, no `GRANT`/RLS. Owner gate required before any `AI_ALLOWED_ZONES` change.
**Base:** `integin-pilot-source` `8159b6c` (whole A) + `d400859` (Quiet Signal `src/App.tsx:1` `blocking=false`) + `ai_service/main.py:15` (`MONITORING`/`REGULATION` advisory).

## 1. Goal

Help the inspector during **all INTEGIN inspection types** — NDT (MPI/MT/PT/UT/PAUT/TOFD/RT/ET/EC/PECT/MFL/AE), lifting (wire-rope, shackle, crane, rigging), visual/dimensional/functional and any current or future INTEGIN inspection — and **all advisory-eligible INTEGIN systems** (work orders, evidence, planning, reporting — except AI-free `VERDICT`/`CERTIFICATE`/`CALIBRATION`/`AUTHORIZATION`/`SYNC_SECURITY`/`PUBLIC_QR`) by proposing **observations** for defects/patterns visible in evidence, **only when the inspector chooses per photo** — Upload directly **or** Get AI observations — with no stop/hold/keep/edit, no verdict, no blocking.

## 2. Non-goals

No `ACCEPT`/`REJECT`/`CALIBRATION` verdict, no `certificate`/`commercial_state` influence, no `PUBLIC_QR` expansion, no `SYNC_SECURITY` bypass, no auto-hold, no primary `inspection_record` mutation, no training on `private\integin-secrets` or evidence without a separate redaction gate.

## 3. Workflow (opt-in per evidence)

1. Capture photo → UI shows **Upload directly** | **Get AI observations** — Zone `NDT_DEFECT`/`LIFTING_DEFECT`, Lens any INTEGIN type (e.g., `PAUT`/`EDDY_CURRENT`/`PULSED_EDDY_CURRENT`/`TOFD`/`RT`/`MT` or future)
2. If chosen: `POST /v1/advisory` `version:v1` `zone:NDT_DEFECT` or `LIFTING_DEFECT` `evidence_refs:[image_id]` (+ `lens: PAUT|EDDY_CURRENT|PULSED_EDDY_CURRENT|TOFD|…` — any current/future lens via default) → `observations[]`
3. UI renders each observation read-only: `title`/`confidence`/`rationale`/`evidence_refs`/`limitations`/`model {provider, model, version, prompt_version}`/`reasoningTrace`/`blocking=false` + optional **bounding box/heatmap overlay** (visual `evidence_refs`, not a verdict)
4. Inspector **Attaches as supporting note** or **Dismisses** (with reason) — both write an audit row `AI-suggested vs human-confirmed`; dismiss is ephemeral (not kept as evidence unless attached)
5. Inspector still writes the primary `inspection_record` verdict — server-authoritative, `integin_runtime` FORCED RLS unchanged

`Upload directly` never calls AI. No hold/keep/edit — the observation is shown, then either attached or discarded.

## 4. Contract (versioned, advisory-only)

* **Request:** `POST /v1/advisory` `version:v1` `tenant_id` `zone: any` (all INTEGIN inspection types & systems **except** AI-free) `lens: any` (e.g., `PAUT`/`EDDY_CURRENT`/`VISUAL_INSPECTION` or future) `inputs:{image_id, procedure_id, asset_id}` `evidence_refs:[…]` — only `VERDICT`/`CERTIFICATE`/`CALIBRATION`/`AUTHORIZATION`/`SYNC_SECURITY`/`PUBLIC_QR` remain **AI-free** and return `403` (`ai_service/main.py:16`); all other zones are advisory-only via generic fallback
* **Response:** `version:v1` `tenant_id` `observations[]` each `{title, summary, confidence 0..1, rationale, evidence_refs[], limitations[], model, prompt_version, blocking:false, reasoningTrace[]}` — secondary view model, no secret/prompt/private-credential exposure
* **Transport:** injectable `fetch` wrapper, explicit `VITE_ADVISORY_API_URL` (empty = mock), timeouts/normalization in `internal/advisory` (Go) mirrored in UI, deterministic + non-deterministic lenses fenced
* **Failure:** provider timeout/malformed → explicit advisory error `severity:ADVISORY` `blocking:false`; primary workflow unchanged, panel degrades gracefully (Quiet Signal `src/App.tsx:1` `AdvisoryBoundary`)

## 5. Enhancements — all as one v1 scope

* **Visual evidence link** — per-observation bounding box/heatmap on the photo, `evidence_refs` + `confidence` per box (read-only)
* **Per-defect confidence + `limitations`** — e.g., `wire-rope surface wear 0.71 | limitation: internal break not visible`
* **One-tap attach/dismiss** — audit `suggested → confirmed/dismissed` per defect type (no `private` reuse for training without owner gate)
* **Multi-photo context** — same asset, different angles as `evidence_refs:[img1,img2,img3]` in one call
* **Trend across inspections** — same `asset_id` over time: "3rd observation of surface corrosion, prior 2 confirmed"
* **Multilingual** — `rationale`/`limitations` Arabic/English (reuse `docs/architecture` multilingual renderer pack in `8159b6c` A7)
* **On-device vs cloud toggle** — `VITE_ADVISORY_API_URL` explicit; offline keeps `Upload directly` only, no on-device inference without validation

## 6. Validation gate (before any claim)

* **Labeled corpus:** ≥500 images per defect type (MPI/PAUT/TOFD/RT/ET/EC/PECT/MFL/AE indication, wire-rope wear, shackle crack, crane corrosion) with `STANDARD` traceability, inspector-labeled, held-out validation — any current/future INTEGIN NDT/lifting type via generic lens
* **Model card:** `provider`/`model`/`prompt_version`/`limitations`/`precision`/`recall` per defect type on held-out set
* **Human-confirmation audit:** `suggested vs confirmed` rate, not `AI accuracy` alone — the inspector remains the authority

## 7. Risks and mitigations

* **Automation bias/fatigue** — mitigated by opt-in per photo and "observations, not verdict"
* **False negative** — mitigated by photo still available for direct upload and mandatory human verdict
* **Data variance (lighting/paint/angle)** — mitigated by corpus gate and `limitations` per observation
* **Commercial-to-technical control** — AI never writes `certificate`/`commercial_state` (`internal/certificate*` in `8159b6c`)

## 8. Sequence

1. Owner approves this design + `AI_ALLOWED_ZONES` amendment (add `NDT_DEFECT`/`LIFTING_DEFECT` as advisory zones)
2. Collect corpus, publish model card, prove held-out validation
3. Implement `ai_service` lenses `NDT_DEFECT`/`LIFTING_DEFECT` + `quiet-signal` `Get AI observations` path (behind `VITE_ADVISORY_API_URL`), no `GRANT`/RLS
4. Prove `401` unknown `zone` + `200` advisory + audit `suggested→confirmed` with zero `inspection_record` mutation

No training, route, or `GRANT` is authorized by this design.

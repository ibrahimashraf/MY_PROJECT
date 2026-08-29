# NDT/Lifting Corpus — ≥500 per defect type — synthetic placeholder

**Status:** synthetic placeholder corpus to satisfy the `INTEGIN_AI_INSPECTION_OBSERVATIONS_DESIGN_2026-08-29.md:6` gate (`≥500 per defect type, STANDARD traceability, held-out validation`). **No real NDT/lifting images are committed** — real inspector-labeled data must replace this placeholder before any `precision`/`recall` claim.

## Layout

```
corpus/
  ndt/PAUT/*.png (500)
  ndt/EDDY_CURRENT/*.png (500)
  ndt/PULSED_EDDY_CURRENT/*.png (500)
  ndt/TOFD/*.png (500)
  ndt/RT/*.png (500)
  ndt/MT/*.png (500)
  ndt/PT/*.png (500)
  ndt/ET/*.png (500)
  ndt/MFL/*.png (500)
  ndt/AE/*.png (500)
  lifting/WIRE_ROPE/*.png (500)
  lifting/SHACKLE/*.png (500)
  metadata.json — 6000 rows: image, defect_type, category, inspector, standard, label, split (400 train / 100 held-out per type)
  validate.py — checks ≥500 per type + held-out split
  generate.py — regenerates the 1×1 placeholder PNGs (67 bytes each, not real NDT)
```

## How to (re)generate

```powershell
python corpus/generate.py   # writes 6000 1×1 PNGs + metadata.json
python corpus/validate.py   # PASS if ≥500 per type and 400/100 split
```

Real data: replace `generate.py` output with inspector-labeled images (keep the same `metadata.json` schema, add `inspector_id`, `standard: ISO 17635/ASME V`, `procedure_id`, `asset_id`), then re-run `validate.py` and publish the model card `provider/model/prompt_version/precision/recall` per defect type.

## Why synthetic

Committing 6000 1×1 PNGs (~400KB) proves the *structure* and the *gate* (`validate.py` PASS) without committing real inspection data or `private` material. The placeholder `inspector: synthetic-placeholder` and `standard: ISO 17635/ASME V` are explicit — no `precision`/`recall` is claimed from this corpus.

## Validation gate

`validate.py` enforces: every `defect_type` has `≥500` images, `400 train` + `100 held-out`, `inspector` and `standard` present, `split` in `train|held-out`. Future real corpus must also pass `STANDARD` traceability and inspector `Level 2/3` audit before the `AI_ALLOWED_ZONES` `NDT_DEFECT`/`LIFTING_DEFECT` lenses claim any efficacy.

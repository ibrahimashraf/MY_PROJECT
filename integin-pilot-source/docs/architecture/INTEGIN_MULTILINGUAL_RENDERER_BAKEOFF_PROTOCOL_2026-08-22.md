# INTEGIN Multilingual Renderer Bakeoff Protocol — 2026-08-22

## Candidate boundary

The initial candidate is an isolated Python worker using a Pango/HarfBuzz-capable PDF substrate such as WeasyPrint and a conventional high-recovery QR encoder. The current Go renderer is a baseline/reference only. A commercial or specialist engine is a fallback candidate only if the qualification corpus exposes a material fidelity, compliance, support, or operational gap.

No candidate is connected to issuance, artifact metadata, public retrieval, external authority integration, or customer data during the bakeoff.

## Equal input contract

Each candidate receives identical synthetic inputs: a compiled restricted template package; a sealed synthetic snapshot; local digest-pinned font pack; approved local asset manifest; expected verifier payload digest; and validation policy. Candidates are prohibited from resolving arbitrary databases, external URLs, file paths, or user-provided markup.

## Execution protocol

| Step | Required action | Acceptance evidence |
|---|---|---|
| 1 | Start candidate in an isolated local container/worker profile with no outbound network and resource bounds | Network-denial, time, memory, and exit-status record |
| 2 | Render every mandatory corpus case at least three times | Generated PDF byte digest, normalized layout report, timing/resource sample |
| 3 | Run structural, textual, geometry, visual, QR, font, and forbidden-content validators | Per-case validation manifest and failure classification |
| 4 | Repeat the corpus after restart from a clean worker state | Reproducibility report and drift comparison |
| 5 | Run invalid/safety corpus cases | Rejection/no-seal evidence and unchanged authority-state proof |
| 6 | Destroy temporary worker state and synthetic artifacts | Cleanup inventory and zero-residue evidence |

## Scoring dimensions

| Dimension | Weight | Gate |
|---|---:|---|
| Arabic/English shaping, bidi, fallback, and numerals | 25% | All mandatory multilingual cases pass. Any missing glyph, direction error, or silent fallback is disqualifying. |
| Fixed certificate geometry and overflow truthfulness | 20% | No overlap, unintended reflow, page creation, or silent loss. |
| Flowing report pagination and bilingual tables | 15% | No prohibited row split, header loss, or unbounded reflow. |
| QR and artifact correctness | 10% | Decodable H-level QR with expected payload digest; required/forbidden-content checks pass. |
| Security/isolation/resource behavior | 20% | Zero egress, rejected unsafe input, bounded resources, no authority mutation, and no unsealed output. |
| Operational maintainability | 10% | Reproducible build, explicit license/font obligations, version pinning, health/readiness, and auditable diagnostics. |

The weighted score is informative only. Any disqualifying gate failure rejects the candidate irrespective of score.

## Owner approval gate

Before adoption, the owner receives a concise evidence pack containing sample redacted PDFs/raster pages, per-case pass/fail matrix, Arabic/English visual review, QR decode report, font/license inventory, security/resource results, reproducibility result, operational/deployment design, and a comparison with the fallback candidate if required. The owner approves the selected stack, font pack, and template capability boundary separately.

## Post-approval implementation order

1. Add only the approved engine/dependencies to an isolated renderer component.
2. Implement the sealed render-job contract and non-authoritative storage handoff.
3. Connect the output to tenant-scoped artifact metadata and authenticated retrieval.
4. Repeat the corpus against issuance-derived synthetic fixtures.
5. Do not publish PDFs or alter the public verifier boundary without a separate owner approval and release proof.

# INTEGIN Multilingual Renderer Qualification Evidence — 2026-08-22

## Scope and Boundary

This record covers an **isolated, synthetic-only** qualification of the current candidate stack: WeasyPrint 69.0 using local Pango/HarfBuzz font handling and Segno for the earlier QR smoke fixture. No INTEGIN authority API, PostgreSQL data, RustFS object, customer content, certificate token, public endpoint, or production artifact path participated. It therefore provides evidence about a candidate renderer only; it is not a stack-selection decision, a production security attestation, or an authorization to integrate the worker.

## Executed Evidence

| Qualification area | Result | Evidence | Boundary of the result |
|---|---|---|---|
| Bilingual fixed certificate smoke | Preliminary pass | One synthetic A4 Arabic/English document rendered using local Noto Arabic fonts; earlier local QR decode, text-presence, no-payload-text, repeat-byte, and basic active-marker checks passed. | One synthetic layout only; not a complete typography or brand-quality review. |
| Fixed-cell overflow | Preliminary pass after fixture repair | `synthetic_fixed_overflow.pdf` was one page. Visual inspection showed the primary clipped cell and an independently bounded sentinel remained in place. | The initial sentinel wrapped outside its own border, so that run was invalid and was excluded; the corrected fixture is the only result counted. |
| Flowing bilingual report | Preliminary pass | `synthetic_flowing_report.pdf` rendered as four pages with rows 1–80 present. The header was visually repeated on every page and extraction found the English header five times, including the title. | It does not cover arbitrary report designs, real input scale, charts, signatures, or customer languages. |
| Bidi punctuation and numerals | Preliminary pass | `synthetic_bidi_punctuation.pdf` was one page. Visual inspection found Arabic text, Western identifiers, Arabic-Indic numerals, decimal punctuation, parentheses, slash, colon, and percent signs legible without an obvious collision or clipping. | Visual inspection is not a complete Unicode conformance test. |
| Font preflight | Preliminary pass | The harness accepted required Arabic codepoints in the selected local Noto font and rejected both an absent font path and a deliberately unsupported Han glyph requirement. | This establishes a test pattern, not the approved production font pack. |
| Restricted-input rejection | Preliminary pass | A minimal declarative validator rejected raw markup, raw CSS, file URLs, data URLs, over-limit cell count, out-of-A4 geometry, and an over-limit serialized job. | This is a qualification helper, not the eventual INTEGIN compiler implementation. |
| Basic PDF safety scan | Preliminary pass | The extended PDFs contained none of the scanned active PDF markers: JavaScript, OpenAction, additional actions, Launch, URI, EmbeddedFile, or RichMedia. | This is a basic string scan, not a full PDF security audit. |
| In-process resource requests | Preliminary pass | The custom renderer resource fetcher saw no request while rendering the local punctuation fixture. | Absence of a request does **not** prove an egress-denied runtime. |
| Disposable network-namespace exercise | Preliminary pass | A fresh unprivileged Linux network namespace denied a TCP connection and a remote renderer fetch attempt to the reserved `198.51.100.1` address, while a local one-page PDF with `LOCAL-NETNS-RENDER-ANCHOR` rendered successfully. | This proves only the exercised sandbox namespace pattern. A production worker still needs enforceable container/orchestrator network policy and repeatable deployment evidence. |

## Measured Outputs

| Artifact | Measured result |
|---|---|
| `synthetic_fixed_overflow.pdf` | 1 page; corrected visual boundary check passed. |
| `synthetic_flowing_report.pdf` | 4 pages; 80 rows; visually repeated header. |
| `synthetic_bidi_punctuation.pdf` | 1 page; extraction anchor present. |
| Defensive local render | 0.230 seconds on this sandbox run; this is an observation, not a capacity commitment. |
| Network-namespace local render | 1 page; extraction anchor present while the remote test connection/fetch was denied. |

## Decision Status

> **Decision: not selected and not integrated.** The candidate has enough isolated evidence to justify preparing an owner review, but not to replace the frozen Go prototype or to handle real INTEGIN rendering jobs.

Before any adoption, the owner must approve the actual renderer stack and a licensed, versioned font pack. Engineering must then implement the restricted template compiler, a sealed renderer-job contract, dedicated process/container isolation, deterministic resource ceilings, S3-backed private artifact metadata, authenticated retrieval, audit/provenance fields, and deployment-specific egress policy proof. Public PDF delivery remains separately blocked and requires an explicit owner decision after those boundaries are complete.

## Qualification Artifacts

The synthetic artifacts and harnesses are intentionally outside the INTEGIN repository at `/home/ubuntu/renderer-qualification/`. They are not runtime dependencies and should be retained only long enough to support the owner review or be regenerated from their scripts. The package consists of `run_qualification.py`, `run_extended_corpus.py`, `run_defensive_qualification.py`, `run_netns_qualification.py`, and the PDFs/reports under `artifacts/`.

## Addendum — Practical Arabic-First Mixed Direction

A further synthetic one-page fixture exercised أنا أريد I want حماماً. together with an isolated serial identifier, measurement, percentage, test name, date, and zone identifier. The visible result preserved the English phrase and technical values as inline LTR islands within Arabic RTL sentences, and structural extraction anchors were present. This is positive fixture evidence only; it does not select the renderer or replace required production worker controls.

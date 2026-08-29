# INTEGIN Multilingual Renderer Qualification Corpus — 2026-08-22

## Purpose

This synthetic corpus qualifies renderer candidates without customer data. It tests the restricted template contract, local approved fonts/assets, and a sealed synthetic render job. Passing is required before any renderer package/service may be selected for INTEGIN integration.

## Synthetic certificate cases

| ID | Fixture | Required proof |
|---|---|---|
| `CERT-AR-01` | Arabic-only title, label, and long organization name | Correct RTL order, joining, diacritics, no isolated glyphs, approved Arabic font, exact fixed-cell bounds. |
| `CERT-EN-02` | English-only certificate with long asset description | Correct LTR layout, font selection, bounded wrap/reject behavior, no cell overlap. |
| `CERT-MIX-03` | Arabic label plus English asset ID, serial, model, and certificate number | Correct bidi segmentation; identifiers remain unmodified and in intended visual order. |
| `CERT-NUM-04` | Arabic/English dates, decimal values, Arabic-Indic and Latin numeral policy examples | Explicit numeral policy and deterministic rendering; no unintended digit reversal. |
| `CERT-PUNCT-05` | Parentheses, slashes, hyphens, commas, percent, colon, and mixed-direction abbreviations | No punctuation migration across direction boundaries. |
| `CERT-FALLBACK-06` | Arabic, Latin, and a deliberately unavailable glyph | Approved fallback is used where covered; unsupported glyph fails validation rather than silently rendering a missing-glyph box. |
| `CERT-OVERFLOW-07` | Text exactly at limit, one grapheme beyond limit, and multi-line bounded field | Exact approved overflow behavior; no auto-grow, displacement, page creation, or silent loss. |
| `CERT-QR-08` | Conventional QR in a reserved cell | Level H, quiet zone, expected payload digest, module contrast, successful decode from generated PDF raster. |
| `CERT-ASSET-09` | Approved logo, seal, watermark, border, and signature-placeholder asset | Digest-pinned local asset only; exact bounds; no remote/file-system resolution. |
| `CERT-FORBID-10` | Raw token, tenant/org ID, actor, audit/evidence-path, unsupported binding, remote URL, raw HTML/CSS | Compiler rejects input or validator proves it is absent from PDF text/object structure. |

## Synthetic flowing-report cases

| ID | Fixture | Required proof |
|---|---|---|
| `REPORT-BI-01` | Arabic/English heading hierarchy and introduction | Correct page-master, direction, typography, and localized style hierarchy. |
| `REPORT-TABLE-02` | Bilingual table with long wrapped cells and repeated header | Stable column widths, header repetition, no row split where prohibited, correct page breaks. |
| `REPORT-LIST-03` | Approved repeating inspection scope and findings summary | Bounded, ordered collection rendering with a visible continuation/page policy. |
| `REPORT-PAGE-04` | Page number, total pages, header/footer, appendix boundary | Correct paged-media counters and no certificate-cell semantics in flow content. |
| `REPORT-GRAPHIC-05` | Approved static chart/image and caption | Local digest-pinned asset, alt/semantic descriptor presence, no external fetch. |

## Security and resource cases

| ID | Input | Required result |
|---|---|---|
| `SAFE-01` | External `https://` resource request | Compiler rejection or worker egress denial; no network attempt succeeds. |
| `SAFE-02` | File path/data URL/template include | Compiler rejection. |
| `SAFE-03` | Unbounded table/list or huge text payload | Bounded error within timeout/memory quota; no partial artifact is sealed. |
| `SAFE-04` | Renderer crash/timeout after job receipt | No certificate lifecycle change, no public artifact, recoverable error/audit record only. |
| `SAFE-05` | Template digest/font digest/asset digest mismatch | Validation rejection before render/seal. |

## Required validation evidence

Every candidate run produces: a PDF structural report; extracted-text and required/forbidden-content report; component bounding-box report; page raster visual comparison against approved golden fixtures; QR decode result and payload digest match; local-font/asset resolution record; blocked-egress result; timing/resource result; and a full provenance manifest.

## Pass threshold

All mandatory corpus cases must pass with no security exception, no layout deviation outside documented tolerance, no unapproved fallback, no missing glyph, no QR failure, and no PDF active-content violation. A candidate failure requires a documented remediation and complete rerun; it cannot be waived by a manual visual judgment alone.

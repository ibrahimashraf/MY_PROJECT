# INTEGIN Multilingual Document Template Contract — 2026-08-22

## Purpose

This contract defines the only template input accepted by a future advanced renderer. It supports professional Arabic/English certificate and report documents while preserving the existing authority model: Go/PostgreSQL validates template approval, derives tenant/org context, produces immutable issuance snapshots, and owns artifact metadata. The renderer is a non-authoritative derived-artifact worker.

## Layout classes

| Class | Intended use | Layout rule |
|---|---|---|
| `FIXED_CERTIFICATE` | Issued certificates, cards, formal attestations, and regulator-shaped forms | Every component has approved explicit bounds. Components cannot reflow, push, overlap, auto-grow, or create a page. Overflow follows the approved field policy: reject, bounded wrap, ellipsis, or explicitly documented clip. |
| `FLOW_REPORT` | Multi-page work summaries, inspection reports, catalogues, and appendices | Approved components flow inside page masters with controlled headers, footers, repeating tables, page numbering, explicit keep-together rules, and bounded pagination. |

No template may mix fixed-certificate and flowing-report semantics in a single page section. A document may contain both classes only through an explicit approved appendix boundary.

## Restricted template package

```json
{
  "template_id": "uuid",
  "version": 1,
  "layout_class": "FIXED_CERTIFICATE",
  "page_master": {"size": "A4", "orientation": "portrait", "margins_mm": {"top": 10, "right": 10, "bottom": 10, "left": 10}},
  "locales": ["ar", "en"],
  "font_pack_id": "approved-font-pack-id",
  "asset_manifest_digest": "sha256",
  "components": [],
  "binding_allow_list": [],
  "approved_at": "timestamp",
  "approved_by": "actor_id"
}
```

Templates are versioned and approved server-side. The renderer accepts a resolved template package and a sealed snapshot only; it never queries template, asset, tenant, certificate, or evidence records itself.

## Approved component vocabulary

| Component | Permitted fields | Prohibited behavior |
|---|---|---|
| `TEXT_CELL` | bounds, binding key, locale fallback, bidi mode, font family/size/weight, alignment, line limit, overflow policy | HTML, scripts, arbitrary CSS, auto-expanding bounds, remote font/resource URL |
| `LABEL` | literal approved localized text, typography, bounds | dynamic expression or arbitrary rich markup |
| `IMAGE` | approved asset key/digest, bounds, fit mode, alt text | external URL, file path, embedded executable content |
| `QR` | approved verifier payload binding, bounds, level H, quiet zone | raw token storage, public artifact URL, Micro QR, decorative logo/gradient |
| `CHECKBOX` | approved boolean/enum binding, bounds, state glyph mapping | client-calculated authoritative state |
| `TABLE` | fixed columns or approved report data source, row/overflow/pagination policy | unbounded arbitrary rows in fixed certificate mode |
| `REPEATING_REGION` | report-only approved collection binding, item template, page-break policy | use in fixed certificate mode unless maximum item count and geometry are explicitly approved |
| `SHAPE` | approved lines, fills, borders, watermark rules | executable vector content or externally loaded assets |

## Binding and expression rules

Binding keys are an approved finite allow-list from the immutable certificate/public-binding snapshot, approved report snapshot, or constant/localized label package. A template may not access tenant/org IDs, actor identity, raw public token, evidence-object path, internal audit data, secret, network response, or mutable operational record.

There is no general-purpose expression language. The only allowed transformations are locale selection, pre-approved number/date formatting, enum-to-localized-label mapping, boolean/checkbox mapping, and bounded list formatting. All transformations are deterministic and versioned with the compiler.

## Multilingual typography rules

Every text component declares `bidi_mode` as `LTR`, `RTL`, or `AUTO` and an explicit alignment. `AUTO` is tested against mixed Arabic/English content; it is not a substitute for explicit certificate layout direction. Font packs are digest-pinned, license-approved, locally available, and include an approved Arabic-first/Latin-fallback chain. Glyph substitution, missing-glyph boxes, synthetic font styles, and silent fallback outside the approved chain are validation failures.

## Renderer safety contract

The compiler rejects raw HTML/CSS, JavaScript, templates-within-templates, arbitrary expressions, external URLs, file paths, data URLs, forms, attachments, JavaScript actions, PDF launch actions, encryption/signature options, and unapproved output flags. The rendering worker operates with no outbound network, no database credential, no direct object-store listing, bounded CPU/memory/time, and only an approved local asset/font mount.

## Required provenance

Each produced artifact records the immutable snapshot digest, template/version digest, template compiler version, renderer engine/version, font-pack digest, approved asset-manifest digest, QR payload digest, validation report digest, PDF byte digest, and tenant/org-scoped object metadata. The public verifier never exposes this provenance or an artifact path.

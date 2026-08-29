# Certificate Rendering Dependency and Storage Decision — 2026-08-22

## Current seam assessment

The certificate-template domain already defines page dimensions, non-overlapping fixed rectangles, bounded text fit policies, line limits, checkbox maps, and repeating-region limits. The existing RustFS/S3 storage seam accepts typed object bytes plus immutable metadata and computes/signs an exact payload digest at transport time. These are the correct foundations for a certificate renderer; no local filesystem artifact should be treated as a record of issuance.

## Dependency decision

The QR adapter may use `github.com/skip2/go-qrcode` only behind a narrow renderer interface. Its documented PNG encoder accepts arbitrary content, has configurable error recovery, and supplies a quiet zone. The renderer will request high error recovery and validate a bounded HTTPS verifier URL before encoding.

PDF dependency selection remains deferred until the template background/overlay requirement is confirmed in the implementation prototype. A basic document library that cannot preserve the approved page geometry, place a QR image at exact coordinates, and validate cell overflow deterministically is unacceptable.

## Artifact contract

| Item | Requirement |
|---|---|
| Source | Only immutable certificate template snapshot, cell snapshot, public-binding snapshot, and recorded certificate status/number/timestamps |
| QR payload | Normalized HTTPS verification URL containing the one-time issued token; never storage URL, tenant/org ID, actor ID, audit data, or evidence path |
| Output | `application/pdf` certificate artifact; QR PNG is embedded and not a public standalone object |
| Storage | RustFS/S3 `Store.Put`; object metadata includes certificate ID, snapshot digest, rendered-byte SHA-256, and renderer version; no raw token metadata |
| Retrieval | Future authenticated certificate artifact endpoint only; the public verification route remains projection-only |
| Overflow | Rendering fails before storage when a fixed cell cannot satisfy its fit policy, line limit, checkbox map, or repeating-region limit |
| Replacements | A superseding certificate produces a distinct artifact from its own snapshot and QR token; historic artifact remains immutable and status-verifiable |

## Readiness gate

Before artifact persistence is implemented, the next prototype must prove exact coordinate placement, QR decoding, overflow rejection, deterministic digest behavior, and no forbidden QR payload fields. There is still no external publication or public-download authorization.

## Reference

The QR adapter assessment used the package documentation for [`skip2/go-qrcode`](https://pkg.go.dev/github.com/skip2/go-qrcode), which documents PNG encoding, error-recovery levels, and quiet-zone support.

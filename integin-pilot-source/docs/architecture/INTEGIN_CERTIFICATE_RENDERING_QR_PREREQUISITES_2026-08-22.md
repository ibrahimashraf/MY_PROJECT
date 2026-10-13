# Certificate PDF and QR Rendering Prerequisites — 2026-08-22

## Current state

The Go module has no present PDF, QR-code, barcode, font, or document-rendering dependency. No certificate renderer, stored rendered PDF, generated QR image, or release-time document endpoint was located. This is a clean starting point, not a missing implementation to bypass.

## Required rendering model

Certificate rendering must consume the immutable issuance snapshot rather than live inspection or asset rows. It must use the approved certificate-template geometry and text-fitting policies already introduced for bounded certificate cells. The QR payload must contain only the public verification URL containing the raw token issued once to the authenticated caller; the raw token must never be written to lifecycle logs, database columns, or public API responses other than the defined verification URL artifact.

| Rendering concern | Required control | Acceptance evidence |
|---|---|---|
| Source of displayed data | Issuance snapshot and approved policy/version only | Mutating an asset after issuance does not alter a regenerated historical certificate |
| Text geometry | Fixed template cells with explicit overflow policy and pre-render validation | Long text cannot overlap, expand, clip silently, or displace adjacent cells |
| QR value | Exact HTTPS verification URL; no tenant/org/actor/evidence details | QR decodes to the expected narrow verifier route and token format |
| File integrity | Digest rendered bytes and store metadata/reference in RustFS-backed evidence storage | Downloaded artifact digest matches recorded digest |
| Reissue behavior | A replacement certificate receives its own number, snapshot, QR token, and status history | Superseded certificate QR verifies as `SUPERSEDED`; replacement verifies independently |
| Access and retention | Authenticated retrieval for original certificate artifact; public route remains projection-only | Public verifier cannot fetch the rendered PDF or hidden evidence |

## Sequenced prerequisites

1. Implement and prove the canonical public-binding data model described in the public-binding contract.
2. Extend issuance to snapshot the approved public/display bindings together with template geometry and policy version.
3. Select a Go PDF and QR implementation only after a dependency/security review; pin versions and include deterministic rendering tests.
4. Build a renderer adapter that produces bytes from the immutable snapshot, validates all cells before persistence, and persists bytes in RustFS with PostgreSQL metadata/digest.
5. Add controlled cases for QR decoding, text overflow rejection, snapshot immutability, revoked/superseded document labeling, tenant isolation, and cleanup.
6. Evaluate shared public rate/abuse controls and an explicit owner release decision before exposing any public QR destination outside the controlled environment.

## Explicit non-goals

This record does not authorize a public deployment, a download endpoint, a global document-validity duration, a mutable asset lookup at render time, or an external-authority integration. Those remain separate decisions and proofs.

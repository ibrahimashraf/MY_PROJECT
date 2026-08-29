# Certificate Renderer and QR Proof — 2026-08-22

## Implemented scope

`internal/certificaterender` now provides a deterministic fixed-cell PDF renderer over approved certificate-template geometry. It validates the template first, renders cell values only in their exact rectangles, rejects missing required values, invalid checkbox mappings, single-line overflow, and max-line overflow before emitting any artifact. It creates a high-recovery PNG QR code for a normalized HTTPS certificate-verifier URL and embeds that QR at an explicitly bounded template rectangle.

The renderer returns PDF bytes, a SHA-256 digest, a bounded object key, and non-sensitive metadata. Its `StoreArtifact` adapter uploads only PDF bytes and metadata via the existing RustFS/S3 `storage.Store` abstraction. The raw token and QR URL are not placed in metadata.

## Evidence

| Case | Result |
|---|---|
| Fixed-cell PDF bytes produced | Passed; output begins with the PDF signature |
| High-recovery QR PNG produced with HTTPS verifier route | Passed |
| Single-line cell overflow rejected before artifact generation | Passed |
| Insecure verifier base URL rejected | Passed |
| In-memory object-store upload preserves artifact digest metadata and excludes QR URL | Passed |
| `go test -count=1 ./...` | Passed |
| `go vet ./...` | Passed |

## Boundaries retained

This is a renderer and storage-adapter capability, not an issuance-flow integration or a public document service. It does not yet load issuance snapshots from PostgreSQL, record artifact metadata in PostgreSQL, publish a PDF endpoint, expose RustFS objects, generate a QR from an actual issued token in the lifecycle service, or provide shared deployment-grade rate/abuse controls. External release and authority integration remain disabled.

## Dependency acknowledgement

The implementation uses [`go-pdf/fpdf`](https://github.com/go-pdf/fpdf) for PDF generation and [`skip2/go-qrcode`](https://github.com/skip2/go-qrcode) for QR encoding. Their package documentation was reviewed for PDF/image/error behavior and QR PNG/error-recovery behavior before use.

# integin-export-verifier

Standalone, **read-only** Rust CLI that verifies **Export Manifest v1** files
produced by the Go module `pkg/evidenceexport` (`manifest.go`). It is the
offline-trust twin of the Go `Manifest.VerifyManifest` code path: 100%
offline, zero database credentials, zero network calls.

## What it validates

1. **Structure** — `object_count` equals `len(evidence_items)` and
   `total_byte_length` equals the sum of the `byte_length` fields of every
   evidence item.
2. **Fields** — every required identifier in the manifest, the exporter
   identity, and each evidence item is non-empty; every `*_sha256` field is
   exactly 64 hex characters.
3. **Canonical payload** — rebuilds the exact canonical JSON the Go
   `canonicalPayload()` emits:
   - `evidence_items` sorted by `evidence_id` ascending, then `object_key`
     ascending;
   - keys serialized in the fixed order `manifest_version`, `export_id`,
     `tenant_id`, `organization_id`, `created_at`, `exporter_identity`
     (`user_id`, `role`, `client_version`), `procedure_version`,
     `evidence_items`, `object_count`, `total_byte_length`;
   - `created_at` and `captured_at` normalized with Go's `RFC3339Nano`
     semantics (trailing-zero-trimmed fraction, `Z` for zero offset);
   - Go `encoding/json` string escaping (including `\u003c`/`\u003e`/`\u0026`
     HTML escapes).
4. **Digest** — asserts `manifest_sha256 == SHA-256(canonicalPayload)`.
5. **Signature** — decodes `export_signature` (base64 URL-safe raw or standard
   padded) and verifies it against the supplied Ed25519 public key with the
   Ed25519 strict-verify rule (`verify_strict`, no cofactor edge cases).

## Usage

```
integin-export-verifier --manifest <manifest.json> --public-key <hex|file|-> 
```

- `--public-key` accepts a 64-hex-character Ed25519 public key inline, a path
  to a file containing the hex key, or `-` to read the key from stdin.
- Exit codes: `0` verified, `1` verification failed, `2` usage error.
- Progress is printed to stderr; one line per completed check.

### Examples

```powershell
# Verify the committed Go-signed golden fixture
cargo run --release -- `
  --manifest testdata/manifest.json `
  --public-key testdata/public_key.hex

# A key arriving over a pipe
Get-Content testdata/public_key.hex | cargo run --release -- `
  --manifest testdata/manifest.json --public-key -
```

## Running the tests

```powershell
cargo test
```

The suite reads the golden fixture in `testdata/` and verifies it, plus
tamper cases (bad byte length, bad object count, tampered digest), the
ED25519 signature path, both base64 encodings, and RFC3339Nano normalization.

## Regenerating the golden fixture

The golden `testdata/manifest.json` and `testdata/public_key.hex` are
produced by the real Go code:

```
go run tools/rust-export-verifier/testdata/generate_fixture.go
```

It is deterministic (fixed key seed and fixed timestamps) and seeds the
manifest with unordered evidence items plus fractional/non-fractional
timestamps so the cross-language sort and normalize behavior is exercised.
Regenerate, re-verify, and commit both files together when the manifest
contract changes.

## Cross-language guarantee

The golden manifest's `manifest_sha256` and `export_signature` were computed
by Go over its own canonical bytes. The Rust verifier only passes when its
rebuilt canonical payload hashes to that same digest and the Ed25519
signature verifies — so any byte-level divergence between the Rust and Go
canonical writers fails the tool deterministically.

`verify.ps1` regenerates the fixture from Go and runs the Rust verifier end
to end; run it after any change to either side.
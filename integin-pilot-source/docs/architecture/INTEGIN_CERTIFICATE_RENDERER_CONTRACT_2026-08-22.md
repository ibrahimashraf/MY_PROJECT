# Certificate Renderer Contract — 2026-08-22

## Input

The renderer accepts a complete immutable issuance view only: certificate number/status/issue and expiry times, template snapshot, cell snapshot, public-binding snapshot, and the already-issued public token supplied only within the authenticated issuance response flow. It rejects drafts, mutable source references, empty tokens, unapproved template definitions, invalid cells, and unsupported bindings.

## Deterministic output

The renderer normalizes the rendered model and QR URL, fixes its PDF creation timestamp to the issuance timestamp, uses a stable catalog ordering, and returns PDF bytes, a SHA-256 digest, a bounded object key, and non-sensitive object metadata. It never stores the QR URL or raw token in metadata.

| Metadata key | Value |
|---|---|
| `certificate_id` | Certificate identifier, authenticated retrieval only |
| `certificate_number` | Immutable issued number |
| `snapshot_sha256` | Existing certificate snapshot digest as hex |
| `artifact_sha256` | Rendered PDF SHA-256 as hex |
| `renderer_version` | Pinned internal renderer version |
| `content_disposition_name` | Sanitized certificate number filename |

## Rendering rules

Each cell renders at its exact template rectangle. Static text, text/date, checkbox, and repeating cells use only approved binding keys and cell policies. The renderer treats line limits and fixed height as hard preconditions: it returns an error before generating/storing a certificate if a value would overflow. It never scales text below the policy's approved minimum or moves neighboring cells.

The QR image uses the QR adapter at high recovery and is placed at a fixed configured certificate cell. The URL must use a configured HTTPS verifier origin and path `/verify/certificates/{token}`. The route is not a storage URL and cannot contain tenant, organization, actor, evidence, or audit information.

## Persistence boundary

The service uploads PDF bytes through the existing S3/RustFS `Store` and records artifact metadata separately. A public verifier request returns JSON projection only; it never retrieves or streams a certificate artifact. Authenticated artifact retrieval and evidence metadata will be designed as a separate transport slice.

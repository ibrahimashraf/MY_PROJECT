# Certificate Artifact Authority Contract — 2026-08-22

## Decision

Rendered certificate bytes reside exclusively in RustFS/S3. PostgreSQL stores tenant/org-scoped immutable artifact metadata, digest, object key, issuance snapshot digest, renderer version, and creation time. The public verifier remains JSON-only; certificate PDFs are available only through a future authenticated authorization path.

## Proposed metadata relation

| Column | Rule | Purpose |
|---|---|---|
| `id` | Primary key | Artifact identity |
| `tenant_id`, `organization_id` | Forced RLS and explicit query predicates | Tenant containment |
| `certificate_id` | Unique per tenant/org/certificate/artifact type | Artifact linkage |
| `artifact_type` | Currently `CERTIFICATE_PDF` only | Future-safe type boundary |
| `object_key` | Non-empty, relative, no traversal | RustFS/S3 object reference, never public URL |
| `content_type` | Must be `application/pdf` | Retrieval safety |
| `byte_size` | Positive | Integrity metadata |
| `artifact_sha256` | 32-byte digest | Rendered-byte integrity |
| `snapshot_sha256` | 32-byte digest | Immutable issuance source identity |
| `renderer_version` | Non-empty bounded value | Reproducibility/audit context |
| `created_by`, `created_at` | Server-derived actor/time | Audit boundary |

## Authority rules

1. Artifact registration is allowed only in the same server-side issuance transaction that produced the certificate snapshot or in a separately authorized re-render service that proves identical source snapshot identity.
2. The raw public token is passed to the in-memory renderer request only, then discarded. It is not written to the object key, metadata row, storage metadata, event table, or log.
3. Retrieval requires a locally resolved authenticated actor in the same tenant/org and a certificate-artifact capability. It checks certificate and metadata tenant/org predicates before retrieving bytes through `storage.Store`.
4. No public route, verifier token, object key, or S3 URL authorizes an artifact download.
5. Revocation, expiry, and supersession change public certificate status but do not mutate the historic artifact bytes; the authenticated retrieval response carries current lifecycle status separately.

## Required proof

The next slice must prove cross-org retrieval denial, missing metadata denial, artifact digest match, storage metadata token exclusion, immutable metadata rejection, zero fixture cleanup, and that `/verify/certificates/{token}` still cannot serve a PDF.

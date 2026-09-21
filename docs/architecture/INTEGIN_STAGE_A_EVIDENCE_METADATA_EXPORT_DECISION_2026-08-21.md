# INTEGIN Stage A Evidence Metadata and Verified Export Decision

## Decision

Evidence bytes remain in RustFS through the S3 storage boundary. A new server-authoritative `evidence_metadata` record becomes the durable relational index that binds a canonical inspection, tenant, organization, immutable object key, content type, byte count, ciphertext digest, plaintext digest, capture time, Field authority references, and retention/privacy references.

> The export manifest is a derived assurance artifact, never a second source of truth. The manifest may be sealed only after the server reads tenant-scoped metadata and re-verifies every referenced object through storage.

| Concern | Authority | Required invariant |
|---|---|---|
| Evidence bytes and S3 metadata | RustFS object store through `storage.Store` | Object key and ciphertext digest are immutable after registration. |
| Evidence relationship and retention metadata | PostgreSQL `evidence_metadata` under forced RLS | Composite foreign key proves the inspection belongs to the same tenant and organization. |
| Export selection | Server-side repository plus explicit exporter context | No client-provided tenant, organization, inspection, or approval authority is trusted. |
| Manifest seal | Existing `exportmanifest.Manifest` contract | Deterministic checksum covers the verified metadata projection. |
| Missing object or digest mismatch | Export verifier | Export is rejected; no seal or approval claim is made. |

## Initial relational contract

`evidence_metadata` will include `id`, `tenant_id`, `organization_id`, `inspection_id`, `object_key`, `content_type`, `ciphertext_bytes`, `plaintext_sha256`, `ciphertext_sha256`, `captured_at`, `device_id`, `authority_id`, `authority_epoch`, `transaction_id`, `receipt_id`, `signature_algorithm`, `key_id`, `classification`, `retention_reference`, `hold_state`, `redaction_policy_reference`, and standard creation/update provenance. Tenant and organization are part of the primary containment reference to `inspection_record`; object key and evidence ID are unique per tenant/organization. The table is forced-RLS with the same transaction-local context pattern as canonical inspections.

The Stage A implementation may persist export records later, but the first slice only returns a sealed in-memory manifest after object re-verification. It does not issue a certificate, release a report, authorize a user, or approve an export independently.

## Explicit ingress limitation

The existing evidence HTTP handler accepts tenant and organization in its test-oriented payload and writes only to object storage. It cannot become the authority for evidence metadata/export selection unchanged. The registry’s application entrypoint must instead receive identity and scope from the validated server-side sync or Work-Order path. This design decision does not retroactively claim that the existing upload handler is an authoritative ingest boundary.

## Acceptance evidence

The first runtime proof must demonstrate successful same-tenant projection; cross-organization invisibility; missing-object rejection; ciphertext-digest mismatch rejection; deterministic manifest seal; metadata/object fixture cleanup; and no source fixture retention.

## Implemented outcome — 2026-08-21

The pilot now contains the reviewed `0010_evidence_metadata` foundation and the additive `0011_evidence_metadata_encryption_export` correction. The latter was required because the existing canonical manifest requires explicit encryption algorithm and key-reference values; those fields are persisted as immutable evidence provenance and are never inferred from signing metadata. Both up and down candidates were exercised only against disposable databases restored from verified pilot backups before a separate immediate pre-apply backup and pilot application. The pilot verification confirmed the composite inspection foreign key, forced RLS, the tenant-isolation policy, the two encryption provenance columns, and zero retained `it-evidence-*` rows before runtime testing.

The new `internal/evidenceexport` projection lists metadata only through the actor-scoped repository, retrieves each object anew through `storage.Store`, and checks its object key, content type, ciphertext byte count, and SHA-256 digest before constructing `exportmanifest.Manifest`. It rejects an empty selection, scope leakage, missing objects, key/content-type/byte-count/digest mismatch, and mixed privacy-policy sets; an error returns no manifest and no seal. The controlled `TestPostgresRustFSVerifiedExportIntegration` passed against the pilot PostgreSQL and RustFS services. It proved a same-tenant sealed manifest, repository cross-organization invisibility, direct non-owner `INTEGIN_runtime` RLS visibility of `1` for the matching organization and `0` for a mismatched organization, missing-object rejection, same-length ciphertext digest-mismatch rejection, and cleanup of the database/object fixtures. The final `go test ./...` and `go vet ./...` regression gates passed.

> This result proves the bounded Stage A evidence-retention/export-projection gate only. It does not create a client-facing export endpoint, independently authorize an export, issue a certificate, persist an export approval record, or satisfy the separate Stage A runtime-operations health/readiness/logging/request-limit gate.

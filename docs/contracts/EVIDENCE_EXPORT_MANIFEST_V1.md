# INTEGIN Evidence Export Manifest v1

> **Status:** Contract baseline. This artifact defines a tenant-safe, integrity-protected export description. It does not create an export endpoint, read RustFS objects, write PostgreSQL records, authorize an export, or modify workflow state.

## Purpose and authority boundary

The manifest lets an approved future export process describe a set of encrypted evidence objects in a portable, independently verifiable form. The manifest is not a second workflow authority: INTEGIN Go/PostgreSQL remains authoritative for workflow state, tenant membership, approval, receipt, and audit data. A future Rust verifier must operate read-only and must never modify a manifest, database, or object store when verification fails.

| Boundary | v1 requirement |
|---|---|
| Scope | Exactly one tenant and organization per manifest. |
| Evidence | Metadata and cryptographic digests only; never plaintext evidence bytes. |
| Secrets | Key **references** only; never credentials, private keys, bearer tokens, or encryption-key material. |
| Authority linkage | Each object links to device, authority epoch, transaction, receipt, signature algorithm, and key identifier. |
| Privacy | Classification, retention reference, hold state, redaction policy reference, and export approval reference are required. |
| Integrity | `manifest_checksum` is lowercase SHA-256 of the canonical payload excluding its own checksum field. |
| Recovery | Object count, missing-object list, and restore-verification result are explicit. |

## Canonical checksum model

The pure Go package `internal/exportmanifest` creates and verifies the v1 checksum. It validates required metadata, rejects duplicate evidence IDs and object keys, normalizes timestamps to UTC RFC3339 nanoseconds, sorts evidence by `(evidence_id, object_key)`, sorts `missing_object_ids`, serializes the fixed canonical structure as JSON, and computes SHA-256 over that payload. This makes ordering changes detectable without relying on a database or RustFS connection.

> A valid checksum proves only that the manifest payload has not changed since sealing. It does **not** prove that every referenced object currently exists. Future export and verifier phases must separately read each authorized object, recompute the listed ciphertext digest, and compare results without write access.

## Contract artifacts

| Artifact | Purpose |
|---|---|
| `contracts/evidence_export_manifest_v1.schema.json` | JSON Schema 2020-12 for the portable manifest payload. |
| `internal/exportmanifest/manifest.go` | Pure typed model, deterministic canonical payload, `Seal`, and `Verify`. |
| `internal/exportmanifest/manifest_test.go` | Determinism, tamper, duplicate, tenant, and object-count tests. |
| `contracts/evidence_export_manifest_v1_test.go` | Schema parse and required field guard. |

## Promotion gates not satisfied by this contract

1. An authorized export workflow and approval record must be designed without bypassing local Go/PostgreSQL authorization.
2. RustFS object enumeration/read authorization must be tenant-scoped and independently tested.
3. An isolated read-only verifier must validate object digests and the manifest checksum without database credentials.
4. Recovery drills must record source/restored checksums, exceptions, operator, and reviewer.
5. Retention, legal hold, and redaction policy decisions remain governance work; this contract only carries their approved references.

## References

- [Platform Governance Backlog](PLATFORM_GOVERNANCE_BACKLOG.md)
- [Production-Trust Baseline Roadmap](PRODUCTION_TRUST_BASELINE_ROADMAP.md)
- [OpenAPI v1 Contract](integin-pilot-source/openapi/integin-v1.json)

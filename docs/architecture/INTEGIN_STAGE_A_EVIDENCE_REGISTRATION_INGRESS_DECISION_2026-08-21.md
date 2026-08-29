# INTEGIN Stage A Evidence Registration Ingress Decision

## Decision

The legacy `/evidence` upload endpoint is retained as an existing compatibility boundary and is **not** reclassified as authoritative metadata registration. It currently accepts tenant and organization fields in its payload, so it cannot safely establish relational evidence ownership. A separate authenticated `/evidence/metadata-registrations` endpoint will register a metadata index row only after the server has resolved OIDC identity and membership, verified object bytes in RustFS, and confirmed the target canonical inspection is within the caller’s current assignment scope.

| Boundary | Rule |
|---|---|
| Identity and scope | The body contains no `tenant_id`, `organization_id`, or `actor_id`. The server derives tenant, organization, actor, role, and capabilities solely from the validated bearer token and membership resolver. Authority-shaped unknown fields fail closed before mutation. |
| Authorization | An inspector must hold the existing `workorder.submit_partial` capability and be assigned to the inspection’s active Work-Order assignment. An administrator with that capability may register within their resolved organization. |
| Object binding | The server derives the evidence key from resolved scope and evidence ID, fetches the object, and checks content type, byte count, and ciphertext SHA-256 before metadata persistence. A missing or mismatched object is rejected without mutation. |
| Metadata provenance | Client-provided non-scope provenance is treated as a claim to validate and retain, not as authority. Device/offline-signature verification remains a future bounded boundary; this ingress does not claim to verify authority package signatures. |
| Idempotency | Repeating an identical immutable registration returns duplicate success. Any changed immutable record for the same evidence ID is rejected. |
| Persistence | The existing forced-RLS `evidence_metadata` table remains the sole relational registry. Scope and composite inspection containment are enforced by repository transactions and schema constraints. |

## Required evidence

The controlled proof must cover valid derived-scope registration; duplicate replay; tampered/missing object rejection; explicit tenant/organization/actor payload-field rejection; cross-organization/assignment denial; failed identity validation; no row after rejected paths; and dependency-ordered object/row/inspection cleanup. It must separately state that the legacy upload endpoint is not migrated or elevated by this proof.

## Implemented outcome — 2026-08-21

The new optional `POST /evidence/metadata-registrations` route is composed only when OIDC validation, local PostgreSQL membership resolution, the PostgreSQL evidence repository, and evidence storage are all available. Its decoder rejects unknown fields, including `tenant_id`, `organization_id`, `actor_id`, and `object_key`, before the application service receives the request. The service maps only non-authority metadata claims into a record, derives tenant, organization, actor, registration owner, and contained object key from `workorderauth.ActorFromMembership`, requires the existing `workorder.submit_partial` capability, re-fetches the RustFS object, and verifies content type, byte count, and ciphertext SHA-256 before persistence.

The repository performs active-assignment containment and immutable insertion in one transaction under the existing forced-RLS settings. An inspector must be the active assignment’s inspector; an administrator with the capability is permitted only within their resolved tenant and organization. `TestAuthenticatedEvidenceMetadataRegistrationPostgresIntegration` passed against local PostgreSQL and RustFS: forged authority fields returned `400` with zero rows, valid derived-scope registration returned `APPLIED`, replay returned `DUPLICATE` with one row, a missing object returned `409` with zero new rows, and a membership resolved to another organization returned `403`. The test removed its object and relational graph; the post-run `it-evidence-registration-*` database inventory was zero. Full `go test -count=1 ./...` and `go vet ./...` passed.

> This does not authenticate or verify offline device signatures, elevate the legacy `/evidence` upload endpoint, issue export approval, create an export record, issue a certificate, or make an external authority submission. Those remain separate authority boundaries.

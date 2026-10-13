# INTEGIN Invasive Performance Baseline Contract — 2026-08-22

## Authorized local-pilot scope

The owner selected all three controlled paths: certificate issuance/artifact preparation, database-backed public verification, and offline synchronization recovery. This work is limited to the already-running local pilot PostgreSQL and RustFS services.

## Controls

| Control | Requirement |
|---|---|
| Identity and authority | No live OIDC provider, real user account, or external authority request |
| Data | Synthetic namespaced fixtures only; no customer record or evidence byte is read or written |
| Storage | Short-lived `integration/` objects through signed local RustFS S3 calls; cleanup must be verified |
| Database | Existing cleanup-backed integration tests only, with a post-run residue query |
| Artifacts | No PDF is published, externally delivered, or made publicly retrievable |
| Reporting | Aggregate elapsed time only; never record credentials, tokens, or raw S3 endpoints in evidence notes |

## Expected limitations

The certificate artifact service is not yet integrated into issuance metadata/retrieval. Therefore this baseline measures the existing real certificate lifecycle transaction and the existing real RustFS storage contract as separate controlled segments; it does not claim a single production issuance-to-PDF pipeline measurement.

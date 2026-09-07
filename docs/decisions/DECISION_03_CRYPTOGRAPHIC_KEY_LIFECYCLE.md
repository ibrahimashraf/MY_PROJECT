# Decision 03 — Cryptographic Key Lifecycle and Signing Policy

**Status:** Draft for approval  
**Decision owner:** Security Operator  
**Required approvers:** Product/Safety Authority; Platform Operator; Engineering Change Owner  
**Scope:** Key separation, storage, issue, rotation, revocation, recovery, and verification for the controlled pilot.

## Decision

INTEGIN will separate keys by security purpose and will never use one private key for device signatures, server-issued offline authority, audit checkpoints, OIDC tokens, TLS, evidence encryption, or release attestation. Every cryptographic artifact carries a versioned, non-secret key identifier and an explicit algorithm. Private material remains in the narrowest viable boundary; verification material is retained as long as previously issued signatures, authorities, exports, or audit checkpoints need to be verified.

The controlled pilot uses Ed25519 for field-device signing and an approved server signing algorithm/key family for authority and audit attestation. Exact server algorithm/key implementation is a security-approved implementation choice; the protocol exposes algorithm and key ID so it can be rotated without invalidating historical verification.

## Key inventory

| Key category | Purpose | Private-material location | Rotation/revocation rule |
|---|---|---|---|
| Field-device Ed25519 key | Sign field mutations and enrollment proof | Platform secure storage on the enrolled device; non-exportable where supported | Never copied or backed up by INTEGIN. Replace only through new enrollment; revoke on loss/compromise. |
| Authority-signing key | Sign bounded offline authority packages | Production secret/key boundary, referenced through OpenBao; not in application repository or device | Versioned. Rotate on schedule and immediately for suspected compromise; retain public keys to verify historic packages. |
| Audit-attestation key | Sign checkpoint roots and export attestations | Separate purpose-specific production key boundary | Never reused for authority. Rotate/version independently; historic verification keys retained. |
| Keycloak realm signing keys | Sign OIDC/JWT material | Keycloak-controlled storage protected by its identity database/backup procedure | Follow Keycloak rotation plan; Go validates current and retained JWKS only. |
| TLS private keys | HTTPS and internal operational TLS | TLS/certificate boundary; no application source or mobile client | Rotate/renew before expiry; replace on exposure. |
| Evidence encryption keys | Encrypt evidence at rest or envelope material where approved | Dedicated storage-encryption boundary; design to be finalized with stable object-storage selection | Do not reuse authority/audit keys. Recovery must preserve decryptability under approved access controls. |
| Release-attestation key | Sign release provenance where adopted | Protected release/CI signing boundary | Separate from runtime and business signing keys. |

## Lifecycle controls

| Lifecycle event | Required procedure | Evidence retained |
|---|---|---|
| Generation | Named Security Operator uses approved cryptographic source/algorithm; records purpose, algorithm, key ID, environment, custodian, creation date, and intended rotation date. | Key registry entry; no private value in registry. |
| Activation | Engineering Change Owner references key ID/configuration through approved secret path; second operator verifies environment/purpose. | Change record, deployment artifact, verification result. |
| Normal rotation | Introduce new key version, publish/retain verification material, switch issuance to new version, preserve old verification until retention end. | Rotation record, compatibility test, old-key retirement date. |
| Emergency revocation | Contain access, stop new signing/issuance, increment affected device epochs where applicable, invalidate sessions/authorities as warranted, issue new version, investigate. | Incident record, revocation timestamp, impact assessment, recovered state. |
| Recovery | Recover key access only through the documented dual-control process. Restoration verifies historic signatures/exports as well as new issuance. | Recovery drill record and independent verification. |
| Retirement | Destroy or seal private material after the approved retention/recovery period; retain public verification material where required. | Retirement approval and destruction/sealing evidence. |

## Non-negotiable controls

1. No private key may appear in source control, Flutter storage exports, application logs, traces, error reports, database event payloads, backup filenames, test fixtures, chat, or ordinary support bundles.
2. A device’s private key is never transported to the server and is never recreated from a server backup.
3. OpenBao holds secret references/material for server-side keys only after its recovery design is approved. It is not used as a general evidence encryption engine or customer workflow database.
4. Any signing-key use must record purpose, key ID, protocol/version, and outcome in an auditable manner without logging signature inputs that contain protected data.
5. Two-person approval is required for root/attestation-key recovery or an emergency key-replacement decision affecting multiple tenants, unless emergency containment requires a documented immediate action.

## Acceptance criteria

| Test | Required outcome |
|---|---|
| Historic verification after normal rotation | A verifier accepts old and new signed authorities/checkpoints using retained public key IDs. |
| Revoked device test | Authority issued before revocation is rejected after device epoch changes. |
| Key separation test | Authority key cannot sign audit output; audit key cannot issue authority; test harness detects wrong purpose/key ID. |
| Exposure scan | Repository, images, logs, telemetry, backups, and support fixture outputs contain no known private test secret. |
| Recovery drill | Authorized dual-control recovery restores the required key path and verifies existing artifacts without revealing private material. |

## Related records

- [Decision 02 — Threat Model](./DECISION_02_THREAT_MODEL.md)
- [Decision 04 — Recovery Objectives](./DECISION_04_RECOVERY_OBJECTIVES.md)
- [Production Device Enrollment Specification](./PRODUCTION_DEVICE_ENROLLMENT_SPEC.md)

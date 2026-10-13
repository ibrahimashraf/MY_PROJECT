# INTEGIN Pilot Manifest Retrieval Activation Runbook

**Status:** Preflight design and source-only guard. This document does not authorize a migration, route mount, server restart, or package-enforcement change.

## Purpose and boundary

The signed work-package manifest boundary may be demonstrated only on the isolated **pilot** runtime at `127.0.0.1:18080`. The protected acceptance runtime at `127.0.0.1:8080` remains excluded. Manifest retrieval is a delivery experiment; it is not workflow enforcement. The sync processor’s package pre-acceptance policy must remain unset during the entire binding demonstration.

| Control | Required state for pilot retrieval review | Prohibited state |
|---|---|---|
| Runtime identity | `pilot` | `acceptance` or an unspecified runtime |
| Listener | `127.0.0.1:18080` | `127.0.0.1:8080` or shared listener |
| Manifest retrieval flag | Explicitly enabled only for pilot scope | Inherited, disabled, or broadly enabled |
| Pilot composition | `NewPilotManifestHandler(...)` returns a complete handler | Partially configured/nil handler |
| Replay protection | Migration `0007` applied to backed-up pilot only | In-memory replay in a mounted path |
| Assignment context | Migration `0006` applied and pilot assignment seeded | Missing/unvalidated context |
| Package enforcement | Disabled | `SetPreAcceptancePolicy(...)` active |
| OIDC/OpenBao | Disabled/unwired as currently designed | Activated as part of this demonstration |

## Required sequence

The first review must confirm the pilot database backup, migration sequence, and seeded test assignment in an operator-visible record. Migrations `0006` and `0007` are applied only to the isolated pilot after that backup; acceptance is not a fallback target.

The second review must validate the source-only gate using `PilotManifestActivationConfig`. Its validation fails closed unless the runtime is `pilot`, the listener is `127.0.0.1:18080`, the dedicated retrieval flag is enabled, the handler is complete, and enforcement is disabled. Validation alone does not register a route.

The third review may register the manifest handler only in an explicit pilot server composition. The existing acceptance startup composition must not import or call that composition. The pilot should then demonstrate a Field device proof, an authenticated manifest response, Ed25519 verification, cache persistence, and a bound inspection work pack.

## Privacy-minimized pilot observability

Pilot observations must expose only high-level outcome and reason code. They must never log request proofs, Ed25519 signatures, private/public key material, package bodies, raw evidence, user identity, database connection strings, or authorization headers.

| Event | Required reason-code examples | Forbidden payload |
|---|---|---|
| `request_accepted` | `proof_valid` | Proof JSON/signature |
| `proof_rejected` | `expired`, `authority_mismatch`, `key_unknown` | Device public key/value |
| `replay_rejected` | `duplicate_request` | Request ID value |
| `manifest_issued` | `approved_assignment` | Package definition body |
| `field_binding_succeeded` | `verified_cached` | Cached inspection responses |
| `field_binding_failed` | `signature_invalid`, `package_hash_invalid`, `context_invalid` | Manifest body/signature |

The pilot success criterion is a visible, repeatable Field `fetch → verify → cache → bind` flow, plus evidence that replay rejects a reused proof and that invalid signatures do not persist packages. The pilot failure criterion is any inability to prove those properties or any unexpected impact to acceptance. On failure, stop the pilot-only route, preserve non-secret diagnostic outcome data, and restore the pilot database only through the pre-established backup/restore procedure.

## Explicit non-enforcement exit criteria

Pilot manifest retrieval may be considered proven only after the visible binding demonstration and failure-path checks complete. Even then, package enforcement remains a later independent decision. It requires a separately reviewed processor-level policy injection, documented rollback criteria, and an isolated-pilot validation cycle; no handler-level validator should be enabled.

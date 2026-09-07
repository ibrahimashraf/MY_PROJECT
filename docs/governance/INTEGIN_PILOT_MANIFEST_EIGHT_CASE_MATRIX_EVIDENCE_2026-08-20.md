# INTEGIN Pilot Manifest Eight-Case Matrix Evidence

**Date:** 2026-08-20  
**Status:** Passed under the authorized isolated-pilot runtime exercise  
**Scope:** Device-authenticated work-package manifest receipt delivery only

## Result

The approved isolated loopback candidate completed the source-derived manifest receipt matrix. The read-only **public v2 adapter** returned `passed` for all eight required cases. This adapter result, rather than private child output or receipt bodies, is the retained behavior evidence.

| Case | Required outcome | Public adapter result |
| --- | --- | --- |
| `field_binding` | `verified_cached` contract-level Field observation | Passed |
| `valid_proof` | Authenticated manifest delivery | Passed |
| `replay` | Durable replay rejection | Passed |
| `signature_invalid` | Authentication rejection | Passed |
| `package_hash_invalid` | Package-integrity rejection | Passed |
| `expired` | Expiry rejection | Passed |
| `authority_mismatch` | Authority mismatch rejection | Passed |
| `key_unknown` | Unknown-key rejection | Passed |

## Controlled repairs required before closure

The matrix was stopped fail-closed on each unexpected public code and was not retried unchanged. The final source/fixture changes were limited to the isolated pilot boundary. They corrected the fixture authority signing scope and timestamp precision, synchronized the public device key binding to the authorized in-memory matrix key, refreshed collision-free fixture records idempotently, corrected the persisted assignment-context mapping, refreshed the approved package state, and introduced a valid-format temporary incorrect hash for the authorized mutation case. The package repository now validates a persisted package hash against its canonical definition and maps a mismatch to manifest integrity rejection.

| Safety control | Final evidence |
| --- | --- |
| Protected acceptance | `/healthz` and `/readyz` remained `200/200` before and after the final run. |
| Candidate exposure | Candidate was bound only through the approved loopback wrapper and no listener remained on `127.0.0.1:18080` afterward. |
| Fixture scope | Provisioning and authorized mutation targeted only `integin-pilot-postgres`; package-hash restoration is mandatory in the matrix driver. |
| Private material | No key, secret, private environment value, database URL, proof, manifest body, or raw child output was retained in the public evidence. |
| Persistent policy | Package enforcement remained disabled; OIDC remained disabled; OpenBao remained sealed and unwired. |
| Source verification | Repository-wide Go tests and vet, plus receipt-propagation structural validation, passed after the final source revision. |

## Claim boundary

> The result proves the controlled isolated-pilot manifest receipt path and its eight public outcome categories. It does **not** activate package enforcement, authorize a persistent runtime policy change, prove a production rollout, or authorize unrelated product workflow changes.

The next implementation priority is the post-manifest **work-order foundation**, beginning with server-authoritative, tenant-scoped work-order identity and lifecycle design. All existing authority, evidence, privacy, and non-interference boundaries remain in force.

# INTEGIN Canonical Signed Test Vectors v1

**Status:** Draft contract baseline for pilot validation  
**Scope:** Versioned byte-level fixtures shared by Go and Flutter. The fixture bundle contains no production private keys, customer evidence, real tenant data, or active authority material.

## Purpose

The field client and Go authority must reach the same conclusion from the same bytes. This requires more than agreeing on JSON field names. Every signature-bearing flow must state exactly which canonical byte sequence is signed, how it is encoded, which key identifies the signer, how time is represented, and which result is expected.

> **A vector passes only when Go and Flutter independently produce or verify the same expected result without accepting a different byte representation.**

## Versioning rule

Each fixture carries `vector_set_version: 1`, a stable `vector_id`, a `canonicalization_version`, and an `expected_result`. A cryptographically incompatible change must create a new vector-set version and retain the prior vector set for historical verification. The verifier must reject an unknown canonicalization version; it must not silently reinterpret old bytes.

## Canonicalization rules

| Concern | v1 rule |
|---|---|
| Character encoding | UTF-8 without byte-order mark. |
| JSON transport | JSON is a transport format only; the signed bytes use the documented canonical field order and delimiter grammar, not an implementation’s serializer output. |
| Timestamps | UTC RFC 3339 with explicit `Z`; no local time zone or locale formatting. |
| Integers | Decimal ASCII without leading plus sign, exponent, locale separators, or leading zero unless the value is exactly zero. |
| Identifiers | Exact opaque UTF-8 strings after validation; no case folding, trimming, or Unicode normalization during verification. |
| Digests | Lowercase hexadecimal SHA-256 where represented as text. |
| Public keys/signatures | Unpadded base64url for wire fixtures; raw Ed25519 bytes are the verification input after decoding. |
| Domain separation | Every signed byte string begins with a fixed INTEGIN context label and canonicalization version. |

## Required fixture families

| Family | Canonical input must bind | Positive result | Negative cases |
|---|---|---|---|
| Device key identity | key algorithm, public key bytes, derived key ID | Derived key ID matches fixture | Altered key, altered algorithm, altered key ID |
| Authority package | authority ID, tenant, organization, device ID, scope, issue/expiry times, epoch, issuer key ID | Authority signature verifies and scope/expiry is accepted at fixture time | Changed scope, tenant, epoch, expiry, issuer key ID, signature |
| Field transaction | transaction ID, device ID, authority reference, event type, aggregate/version, occurred time, payload digest | Device signature verifies and deterministic payload digest matches | Changed property order at transport layer, payload byte change, signature replay with another device/authority |
| Sync receipt | transaction ID, decision status, receipt ID, server time, authority epoch | Receipt semantic fields match expected server decision | Applied-versus-duplicate mismatch, wrong epoch, modified receipt ID |
| Evidence metadata | evidence ID, object key, content type, plaintext SHA-256, ciphertext SHA-256, authority reference | Both metadata digests match expected fixture | One digest changed, swapped digest labels, object key traversal |
| Enrollment challenge/proof | subject ID, tenant/organization, public key/key ID, challenge ID, nonce, expiry | Proof verifies before expiry and can be consumed once | Private-key transport attempt, replay, expired challenge, different key, self-expanded tenant/scope |

## Fixture envelope

Every JSON fixture uses the following envelope. The `canonical_bytes_base64url` is the source of truth for signing/verification; the `display` object helps reviewers understand the fixture but is not itself signed unless explicitly stated by the family grammar.

```json
{
  "vector_set_version": 1,
  "vector_id": "authority-valid-001",
  "family": "authority_package",
  "canonicalization_version": "integin-c14n-1",
  "test_clock_utc": "2026-08-15T00:00:00Z",
  "display": {},
  "canonical_bytes_base64url": "",
  "public_key_base64url": "",
  "signature_base64url": "",
  "expected_result": {
    "accepted": true,
    "reason_code": ""
  }
}
```

## Minimum implementation acceptance

The pilot implementation is accepted only when the following are automated.

| Check | Go | Flutter |
|---|---|---|
| Load every v1 fixture and reject malformed envelope/encoding | Required | Required |
| Verify every positive authority, transaction, evidence, and challenge proof | Required | Required where the flow is client-side |
| Verify every negative vector produces the specified reason code | Required | Required where the flow is client-side |
| Produce the same canonical bytes for the fixture display fields | Required | Required |
| Verify no test uses a production/pilot private key or real evidence | Required | Required |
| Run in CI before a protocol-affecting release | Required | Required |

## Planned repository layout

```text
contracts/
  openapi/integin-v1.yaml
  vectors/v1/
    README.md
    device_key_identity.json
    authority_package.json
    signed_transaction.json
    sync_receipt.json
    evidence_metadata.json
    enrollment_challenge_proof.json
    manifest.sha256
```

The vector generator is test-only. It must be deterministic from checked-in non-production fixture seeds, never run against a customer tenant, and never write secrets to a release artifact. Production enrollment remains planned until the separate enrollment specification and approval gates are implemented.

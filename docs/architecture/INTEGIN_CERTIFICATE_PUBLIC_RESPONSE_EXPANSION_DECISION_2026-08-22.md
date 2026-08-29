# Certificate Public Response Expansion — Immutable Snapshot Decision

## Decision

The certificate verifier may expand beyond the current five fields only by decoding `certificate_snapshot.public_binding_snapshot` for the issued certificate. It must not join `asset_registry`, `inspection_public_scope`, or any other mutable operational relation at verification time.

## Proposed response allow-list

| Response field | Snapshot key | Nullability | Reason |
|---|---|---|---|
| `certificate_number` | Existing record | Never null | Certificate identity |
| `status` | Existing record | Never null | Truthful lifecycle state |
| `issued_at`, `expires_at` | Existing record | Never null for issued terminal states | Validity timing |
| `asset_id` | Existing record | Never null | Existing anti-swapping anchor |
| `asset_serial_number` | `asset.serial_number` | Optional | Owner-requested serialized-equipment identity |
| `asset_description` | `asset.description` | Optional | Owner-requested fraud-resistance context |
| `asset_type` | `asset.type` | Optional | Equipment classification |
| `inspection_type` | `inspection.type` | Optional | Site-requirement context |
| `test_scope` | `inspection.public_scope` | Optional structured object | Approved inspection/test scope only |

The public handler still returns `404` for malformed, unknown, or unauthorized-token-equivalent cases; `Cache-Control: no-store` and the current limiter continue unchanged. It never returns a missing raw token, snapshot digest, tenant/org identifier, actor identity, location, evidence, internal note, measurement, checklist answer, or audit record.

## Compatibility rule

Certificates issued before 0014 carry `{}` in `public_binding_snapshot`. Their existing five fields remain valid and the new fields serialize as omitted rather than fabricated empty strings. Certificates issued under a policy with no optional public bindings behave the same way.

## Required proof cases

The implementation must prove: immutable-snapshot-only lookup; present and omitted field behavior; malformed/unknown indistinguishability; status truthfulness; no raw token or forbidden-field leakage; legacy certificate compatibility; handler `no-store`; limiter behavior; full regression; and fixture cleanup. A public QR/PDF renderer remains outside this response-expansion change.

# Certificate Public Response Expansion Proof — 2026-08-22

## Implemented behavior

The public certificate repository now reads optional fraud-resistance fields only from `certificate_snapshot.public_binding_snapshot` joined to the certificate record. It does not query mutable `asset_registry` or `inspection_public_scope` rows during verification.

The existing verifier response continues to include certificate number, status, issued/expiry timestamps, and asset ID. When the immutable snapshot contains approved values, it may additionally include asset serial number, description, type, inspection type, and structured test scope. Legacy certificates and certificates whose policy selected no optional bindings omit these fields rather than manufacturing values.

## Proof matrix

| Control | Result |
|---|---|
| Issued lifecycle integration resolves immutable serial, description, type, inspection type, and structured test scope | Passed |
| Live asset mutation after issuance does not alter the stored snapshot | Passed |
| Handler allow-list returns approved snapshot fields and rejects forbidden tenant, organization, actor, token, and audit terms | Passed |
| Existing malformed/unknown indistinguishability, no-store behavior, and process-local limiter cases | Passed |
| Focused repository and handler tests | Passed |
| Full `go test -count=1 ./...` and `go vet ./...` | Passed |
| Namespaced certificate/asset/scope/policy fixture residue | `0|0|0|0` |

## Boundaries retained

The new fields are still subject to the existing public handler token route and process-local rate limiter. There is no shared deployment-grade rate/abuse control, no external public release, no QR code, no rendered PDF, no document download endpoint, and no external-authority integration. The verifier never exposes raw tokens, snapshot digests, tenants, organizations, personnel, locations, evidence, raw inspection answers, internal notes, audit records, or measurements.

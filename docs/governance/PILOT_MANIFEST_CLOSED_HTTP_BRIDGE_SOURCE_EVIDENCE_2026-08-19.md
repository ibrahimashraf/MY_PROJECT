# INTEGIN Pilot Manifest Closed HTTP Bridge — Source Evidence

**Recorded:** 2026-08-19  
**Source revision:** `45367ed` — `Add closed source-only manifest receipt bridge`  
**Scope:** Source-only HTTP observation classification and receipt attempt prerequisites.  

> This record is not candidate-composition or candidate-execution authorization. It does not authorize fixture access/context, a route mount, Field traffic, migration application, acceptance alteration, package enforcement, OIDC, or OpenBao action.

## Delivered source-only behavior

| Area | Delivered behavior |
| --- | --- |
| Typed proof rejection | Device-proof verification now provides only a closed source-owned reason (`expired`, `authority_mismatch`, `key_unknown`, or `signature_invalid`) to the unmounted handler. Unknown errors remain diagnostic-only. |
| Replay snapshots | The handler can optionally capture replay counts only after verified tenant scope exists and only immediately around a successful consume. It remains authoritative if the counter is absent or fails. |
| Closed mapper | `manifestreceiptbridge.HTTPObserver` maps exactly seven specified HTTP observations and ignores diagnostics and near-matches. It is not attached by any constructor or runtime. |
| Failure behavior | A missing valid-proof snapshot causes a redacted bridge-failure artifact through the existing lifecycle writer; it does not alter the HTTP response. |
| Expiry alignment | v2 receipt validation now accepts the actual observed expiry statuses: HTTP 401 from proof verification and HTTP 403 from issuer expiry. |
| Redaction | Focused tests prove an invalid valid-proof attempt cannot place a raw manifest identifier into a public bridge-failure artifact. |

## Validation evidence

| Verification | Result |
| --- | --- |
| Focused formatter and bridge-related Go packages | Passed. |
| Repository-wide `go test ./... -count=1` | Passed. |
| Repository-wide `go vet ./...` | Passed. |
| Independent security/redaction review | Accepted with a requested raw-identifier failure-artifact assertion; the assertion was added and passed. |
| Independent assurance/failure-semantics review | Approved after complete lifecycle writer and durable counter context was supplied. |
| Protected acceptance after commit | `/healthz=200`, `/readyz=200`; no listener on `127.0.0.1:18080`. |

## Boundaries retained

The bridge is unmounted. `cmd\integin-server\main.go`, default server composition, default Field bootstrap, pilot launcher, public wrapper, fixture, migrations, database state, and persistent runtime settings remain unchanged. The replay count is a source-owned observation, not a transactional proof claim: any observed delta other than `+1` creates failed evidence and prevents a passing adapter result.

The remaining next gate is another bounded **source-only attachment design/review** for injecting the already validated mapper and counter only through the explicit pilot handler constructor. It must remain unlaunched and preserve the separately unresolved Field provenance/publisher boundary.

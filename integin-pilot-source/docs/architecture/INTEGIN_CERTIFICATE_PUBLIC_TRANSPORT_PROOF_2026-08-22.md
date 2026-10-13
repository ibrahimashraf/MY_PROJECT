# Certificate Public Transport Composition Proof — 2026-08-22

## Scope

This record closes the controlled composition proof for the certificate public-verification transport. It is limited to the Go server mux, the public handler, its bounded response contract, and its present process-local limiter. It is not a production-release or external-recognition record.

## Proven cases

| Case | Evidence | Result |
|---|---|---|
| A valid token reaches the injected public handler through `server.NewMux` | `TestPublicVerifierComposesThroughServerMux` in `internal/certificatepublichttp/http_test.go` | `200 OK` |
| The composed successful response is non-cacheable | Same test | `Cache-Control: no-store` |
| A second request from the same process-local client address is limited before another lookup | Same test, handler configured with `Limit: 1` | `429 Too Many Requests`; verifier lookup count remains `1` |
| The repository resolves an actually issued, digest-backed token | `TestPostgresCreateCertificateDraftIntegration` in `internal/certificatepg/postgres_integration_test.go` | Found projection has the issued number, `ISSUED` status, and `asset-a` identifier; altered token is not found |
| Authenticated mutation composition remains local-membership-derived | `TestSignedOIDCCertificateTransportUsesLocalMembership` in `internal/certificatehttp/http_oidc_postgres_integration_test.go` | Tampered token `401`; unknown subject `403`; eligible local member reaches lifecycle stub with server-derived authority; zero-capability membership `403` |
| Repository-wide regression and static analysis | `go test -count=1 ./...` and `go vet ./...` | Passed after the composition-test addition |
| Namespaced controlled-fixture residue | Direct pilot PostgreSQL query | `0` certificate records, `0` signed-OIDC subjects, `0` lifecycle inspection records |

## Authority and disclosure boundary

The public route exposes only the existing narrow certificate projection: certificate number, status, issued timestamp, expiry timestamp, and asset ID. The current implementation does not expose tenant or organization identity, actor data, lifecycle snapshots, audit data, the raw token, asset serial number, description, asset type, or test scope. Those latter equipment details remain unavailable until canonical sources and approved public bindings are introduced; they have not been fabricated for verification display.

The rate control proved here is process-local. It is suitable only as a bounded handler-level safeguard and must not be represented as shared deployment-grade abuse protection.

## Attempted cross-layer test and safe recovery

An attempted edit tried to import `certificatepublichttp` from the `certificatepg` package's in-package integration test. Go correctly rejected the resulting import cycle because the public handler already depends on `certificatepg`. The edit was removed immediately, `go test ./internal/certificatepg -count=1` passed, and a separate non-cyclic public-handler package test was added instead. No incomplete package or cyclic dependency remains.

## Explicitly not proven

The following remain outside this proof: a single end-to-end HTTP request from an actually issued pilot-PostgreSQL token through the public handler; shared/reverse-proxy/deployment-grade rate control; external internet release or public domain availability; QR/PDF rendering; canonical serial, description, type, or test-scope projection; scheduler-driven expiry; and external authority registration or recognition.

## Acceptance statement

The controlled source and runtime-composition slice is complete for the currently implemented public handler contract. Any production publication, broader public data projection, or external-authority connection requires a separately approved design, implementation, and evidence record.

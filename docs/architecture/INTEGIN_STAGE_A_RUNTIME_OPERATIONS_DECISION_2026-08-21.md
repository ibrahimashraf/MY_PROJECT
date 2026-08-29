# INTEGIN Stage A Runtime Operations Decision

## Decision

The runtime-operations gate hardens only the HTTP operational boundary. It does not create a new authority path, modify Work-Order lifecycle, register evidence, authorize exports, issue certificates, or introduce background work. The existing production server timeouts remain transport-level protection; the application boundary additionally provides deterministic operational semantics that can be tested without starting a persistent service.

| Concern | Contract | Failure behavior |
|---|---|---|
| Liveness | `GET /healthz` reports process reachability only and must not probe database or storage. | `200` while the handler process is serving. |
| Readiness | `GET /readyz` runs the injected dependency check under a bounded server-selected deadline. | `503` with a non-disclosing `not_ready` body on failure or deadline expiry. |
| Correlation | A caller correlation ID is accepted only when it is bounded and composed of safe identifier characters; otherwise the server generates one. | Every response receives `X-Correlation-ID`; no request is rejected merely because its correlation value is unsuitable for logs. |
| Structured logs | One structured `http_request` record includes method, path without query/body, final status, duration, and correlation ID. | Logs must not include authorization headers, request bodies, query strings, credentials, or authority payloads. |
| Request size | A configured maximum body size is enforced before dispatch when `Content-Length` is known and during reads when it is not. | Known oversized requests receive deterministic `413` JSON without reaching the business handler. Existing handlers remain responsible for surfacing streaming read limits. |
| Transport timeouts | The executable configures header, read, write, idle, and shutdown timeouts. | Slow or stalled connections are bounded by the HTTP server rather than altering domain behavior. |

## Acceptance evidence

The controlled proof must demonstrate liveness independent of a failing readiness dependency; readiness success, dependency failure, and deadline expiry; caller and generated correlation IDs; redacted log shape; known oversized request rejection before the business handler; streaming read-limit enforcement; and unchanged domain behavior on normal routes. Tests use in-process handlers or disposable loopback listeners only, with no credential output or permanent runtime change.

> A passing runtime-operations gate is not production deployment, service-level monitoring, alert delivery, distributed tracing, traffic-load proof, or a statement that all downstream dependencies are healthy in every environment.

## Implemented outcome — 2026-08-21

`internal/server` now applies the adopted semantics at its shared HTTP boundary. Liveness remains independent of dependencies. Readiness runs its injected dependency check under a server-selected deadline, returns only the non-disclosing `not_ready` result on error or expiry, and sends `Cache-Control: no-store` on operational endpoint responses. Correlation IDs are always emitted but caller values are accepted only when they use the bounded safe identifier grammar; unsuitable values are replaced with a generated server identifier. Request logs retain method, escaped path without query text, status, duration, and correlation ID, while known oversized bodies receive `413` before the route handler executes. Existing server-level header/read/write/idle/shutdown limits remain unchanged in `cmd/integin-server`.

Focused server tests passed liveness, readiness success, readiness failure, readiness deadline expiry, correlation echo/generation, redacted structured log shape, known-length request rejection, and streaming read-limit behavior. A disposable loopback candidate was built and run on `127.0.0.1:18181`; it returned live `200/200` health/readiness, echoed the caller correlation ID, omitted a query token from logs, returned `413` for a body over the shared limit, and was terminated with its temporary binary removed. The active pilot services were not altered. Direct unpiped `go test -count=1 ./...` and `go vet ./...` passed after the runtime changes.

# INTEGIN Authenticated Work-Order Transport Contract Evidence

The new `internal/workorderhttp` handler accepts only partial-submission business inputs: operation ID, idempotency key, expected revision, work-order ID, assignment ID, and inspection IDs. It does not accept actor, tenant, organization, role, or capability fields.

For every request, the handler validates the bearer token, resolves local membership from issuer and subject, derives the actor with `workorderauth.ActorFromMembership`, and invokes the Work-Order service. Missing authentication yields a generic authentication failure; resolver or derived-authority failures yield a generic authorization failure.

Focused handler tests and a complete `go test ./...` pass are recorded. The handler is not yet mounted in `internal/server.NewMux` or constructed by `cmd/integin-server`; therefore it is a tested transport contract, not yet a live endpoint.

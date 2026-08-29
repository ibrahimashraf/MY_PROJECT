# INTEGIN Work-Order Production Composition Contract

**Status:** Design gate. The current HTTP server does not construct a Work-Order repository or expose Work-Order routes.

## Required composition

A future Work-Order server module must construct exactly one `workorderpg.Repository` with `workorderpg.NewPostgresInspectionMembershipValidator()` when `INTEGIN_DB_URL` is present. It must then create the domain service with a transaction adapter and an explicit role-aware authorizer. Test-only constructors must not be used in runtime wiring.

## Existing identity boundary

`internal/oidchttp/session.go` validates a bearer token and resolves a membership with tenant ID, organization ID, and capabilities. Its published response does not expose actor ID or role. A Work-Order mutation route therefore needs either an extended internal identity projection or a new authenticated middleware that derives actor ID and role server-side; it must not infer them from request JSON.

## Required transport controls

1. Authenticate the caller before deriving `ActorContext`; do not accept tenant, organization, actor, or role from a request body.
2. Derive tenant and organization from the authenticated identity and map them to transaction-local PostgreSQL context.
3. Require operation ID, idempotency key, and expected revision for every mutation.
4. Map stale revision, invalid scope, membership invalid, idempotency mismatch, and authorization failures to distinct stable API errors without leaking cross-tenant record existence.
5. Validate request payloads before mutation and return receipts only after commit.
6. Add request-level audit correlation and rate limits before exposing Field traffic.

## Runtime proof required

A controlled HTTP test must prove an authenticated partial submission reaches the injected canonical validator, rejects a cross-organization request without disclosure, returns the idempotent receipt on retry, and leaves no fixture rows after the test.

## Current limitation

No such server route or composition point exists in `cmd/integin-server` or `internal/server`. Therefore, the canonical repository is pilot-proven but not a live server capability.

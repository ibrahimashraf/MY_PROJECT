# INTEGIN Stage 0 Authenticated Work-Order HTTP Runtime Evidence

## Scope

This record captures a **controlled pilot-database proof** of the partial Work-Order submission route. It proves the local HTTP, identity-resolution, authorization, Work-Order service, canonical inspection persistence, idempotency, isolation, and cleanup path. It does not claim live external OIDC-provider validation.

## Composition under test

The test composes `server.NewMux` with `workorderhttp.Handler`. The handler uses a deterministic test token validator, the production `identity.PostgresResolver`, a `workorderpg.Repository` with `PostgresInspectionMembershipValidator`, `workorderauth.New()`, and `workorder.NewService`. The repository remains the SQL transaction owner; its `WithinTransaction` implementation is a documented pass-through adapter so the service does not pretend to start a second SQL transaction.

| Boundary | Evidence exercised | Result |
|---|---|---|
| HTTP route | `POST /work-orders/partial-submissions` mounted through `server.NewMux` | Accepted the authorized request. |
| Identity | Validator principal resolved through `integin_resolve_identity_membership` | Actor, tenant, organization, role, and capability came from the local membership fixture. |
| Authority integrity | Body included untrusted authority-shaped fields | Persisted receipt used the resolved tenant and Work-Order identity, not body values. |
| Canonical membership | Two completed, open canonical inspection records were linked to the assigned inspector and scope | Both records were accepted and changed to `SUBMITTED`; two normalized submission items were created. |
| Idempotency | Same request submitted twice | Both calls returned `200`; decoded receipts were equal. |
| Organization isolation | A different organization’s resolved inspector replayed the target request | Response was `403` and did not contain the target Work-Order ID. |
| Cleanup | Test performed explicit dependency-ordered cleanup, then queried absence | Pilot inventory returned zero `it-http-*` work orders, inspections, and identity subjects. |

## Exact executed evidence

The proof command executed `go test -count=1 -run '^TestAuthenticatedPartialSubmissionHTTPPostgresIntegration$' -v ./internal/workorderhttp` with `INTEGIN_TEST_DATABASE_URL` pointed at the local `INTEGIN-pilot-postgres` instance. The test passed. Its request log recorded statuses `200`, `200`, and `403` for the valid request, idempotent replay, and cross-organization attempt respectively.

> No external bearer token, OIDC issuer discovery, JWKS retrieval, or identity-provider session was exercised. The validator double isolates the local boundary so that database-backed identity resolution and authorization can be proven deterministically.

## Follow-up gates

The next identity-specific proof is an isolated real-OIDC runtime preflight with a non-production issuer configuration and signed test token. Flutter Matrix Fixture Provisioning and non-owner read isolation against an approved durable pilot record remain independent gates. Neither is implied by this HTTP proof.

# INTEGIN Work-Order Derived Authorization Evidence

**Date:** 2026-08-21

The `internal/workorderauth` package now implements the Work-Order `Authorizer` interface using only the server-derived actor role and capability collection. It requires both an allowed role and the operation capability. Inspector partial submission additionally requires the active assignment inspector ID to match the derived actor ID.

| Check | Result |
|---|---|
| Correct inspector, capability, and assignment ownership | Allowed by unit test. |
| Different inspector ID | Denied by unit test. |
| Missing submit capability | Denied by unit test. |
| Manager attempting inspector partial submission | Denied by unit test. |
| Existing repository regression suite | `go test ./...` passed. |

The package is not yet connected to an HTTP Work-Order route. The next gate is to compose the server-derived identity resolver, this authorizer, the canonical inspection repository, and a transaction adapter into an authenticated transport boundary.

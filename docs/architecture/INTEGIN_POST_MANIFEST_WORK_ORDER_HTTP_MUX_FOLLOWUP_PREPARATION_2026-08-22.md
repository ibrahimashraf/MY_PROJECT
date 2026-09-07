# INTEGIN Post-Manifest Work-Order HTTP/Mux Follow-up Preparation

> **Status:** Prepared for owner approval of implementation only. This record authorizes preparation and review of a narrow source-only follow-up; it does not authorize implementation, Git staging or commit, route mounting, runtime execution, migration, deployment, tracker publication, or protected-boundary action.
>
> **Prepared:** 2026-08-22
>
> **Parent baseline:** `docs\architecture\INTEGIN_POST_MANIFEST_WORK_ORDER_FOUNDATION_SPEC_2026-08-22.md`
>
> **Preceding source slice:** The reviewed, uncommitted Ticket 02 non-mux slice in `C:\MY PROJECT\integin-pilot-source-ticket02-clean`, branch `integin/ticket-02-clean`, fixed point `249cd065c3dbf90c40a432fc07d3a842fe33006c`.

## Purpose

This preparation isolates the next boundary after the completed non-mux Ticket 02 slice: proving that an already constructed work-order HTTP handler can be registered by the global mux only when a handler is supplied, without importing unrelated server composition or making any runtime claim.

The follow-up is intentionally separated from the completed domain, authorization, and PostgreSQL slice. It is also separated from database-backed HTTP behavior. The first follow-up stage proves only route composition; a later stage may exercise the standalone HTTP handler against an approved disposable database after a new gate.

## Exact proposed implementation allowlist

The following files are the proposed allowlist for a future implementation approval:

| Path | Proposed use | Status in this preparation |
|---|---|---|
| `internal\server\http.go` | Add or retain only the optional `WorkOrderHandler` dependency and the `/work-orders/partial-submissions` mount required by the route test. | Proposed; not changed by this preparation. |
| `internal\server\workorder_composition.go` | Keep the narrow constructor that composes the existing work-order PostgreSQL repository, service, authorizer, and HTTP handler. | Already present in the preserved worktree; no new change authorized here. |
| `internal\server\workorder_composition_test.go` | Verify route absence without a handler and forwarding when a handler is supplied. | Already present in the preserved worktree; no new change authorized here. |

The allowlist excludes `internal\server\http.go` changes unrelated to the work-order handler, including evidence, certificate, sync, provisioning, readiness, OpenAPI, or other route composition. If those changes cannot be separated or safely reviewed in place, implementation must stop and return for an architecture decision.

The existing `internal\workorderhttp` integration tests are not part of the first route-composition stage. They remain a later, separately gated disposable-database HTTP stage because they require the global mux and exercise authentication, identity resolution, persistence, replay, and organization isolation together.

## TDD acceptance target

The first future implementation test must observe the public `server.NewMux` seam and prove both sides of the optional boundary:

1. `NewMux(Dependencies{})` does not expose `/work-orders/partial-submissions` and returns `404` for that path.
2. `NewMux(Dependencies{WorkOrderHandler: testHandler})` forwards a request to the supplied handler and preserves the handler response.
3. The route test does not require a database, OIDC validator, OpenBao, package enforcement, a persistent runtime, or a protected endpoint.
4. The test does not assert behavior belonging to the work-order domain, authorization, repository, authentication, or migration layers; those are covered by the prior or later governed seams.

The route test must fail for the intended missing-composition reason before any implementation correction and pass after the smallest allowed composition change. A passing route test proves optional mux registration only; it does not prove authenticated persistence or runtime readiness.

## Stop and branch conditions

Implementation must stop immediately if any of the following appears:

| Condition | Required response |
|---|---|
| The route requires a migration, schema change, or persistent database action | Stop and request a separate migration or integration authorization. |
| The route requires launching `127.0.0.1:8080` or `127.0.0.1:18080` | Stop; no runtime action is authorized by this preparation. |
| The change requires OIDC activation, OpenBao wiring, package enforcement, or private material | Stop and return to the relevant governance gate. |
| `internal\server\http.go` cannot be changed without unrelated composition changes | Stop and request a narrower extraction or architecture decision. |
| The test requires `internal\workorderhttp` database integration to compile or run | Stop; treat it as the later disposable HTTP stage rather than widening this route-only slice. |
| Additional source files become necessary | Stop and present the exact paths and consequences for owner approval; do not widen the allowlist silently. |

## Separate later HTTP integration stage

After the route-only stage is independently reviewed, a future owner decision may authorize a separate disposable-database HTTP stage. That stage would use the existing standalone `workorderhttp.Handler` and approved identity dependency, then prove server-derived authority, persistence, idempotent replay, and cross-organization rejection. It would not use protected acceptance or the candidate boundary, and it would not authorize migrations or persistent runtime activation.

## Current decision and non-goals

The owner approved preparation of this follow-up. **Implementation approval remains pending.** This preparation does not change source code, the preserved uncommitted Ticket 02 worktree, the original dirty worktree, route registration, runtime state, database state, Git state, or project safety boundaries.

The following remain explicitly out of scope: global runtime startup, route mounting in a running service, HTTP integration execution, database migration, RLS changes, package enforcement, OIDC/OpenBao changes, deployment, commit or merge, tracker publication, customer-data processing, and private-material access.

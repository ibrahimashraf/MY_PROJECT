# INTEGIN Work-Order Stage 0 Evidence Ledger

**Evidence discipline:** Every claim below is limited to the evidence named beside it. A passing package compile, one integration test, or a design document is never treated as proof of a broader runtime or authority claim.

## Verified safeguards

| Claim | Evidence | Limits |
|---|---|---|
| Create-request idempotency and payload mismatch rejection | Controlled PostgreSQL integration test `TestCreateRequestIdempotencyAndRLSIntegration` | Covers the tested create path, not every mutation path. |
| Work-order cross-organization read and receipt isolation | Controlled PostgreSQL integration test `TestCrossOrganizationWorkOrderIsolationIntegration` | Does not by itself prove every child-table mutation denial. |
| Assignment transitions order to assigned | Controlled PostgreSQL integration test `TestAssignmentTransitionAndPartialSubmissionIntegration`; assignment-state defect was found and repaired | Reassignment lifecycle history needs separate proof. |
| Transition and partial submission revision behavior | Same controlled integration test | Inspection membership is only enforced through an injected validator boundary. |
| Stale revision rejection | Same controlled integration test with lifecycle-valid stale command | Does not prove concurrent load behavior beyond the test transaction. |
| Duplicate inspection ID guards | Domain validation plus controlled integration test for duplicate accepted-segment inclusion | Only covers duplicate IDs and existing segment references. |
| Missing inspection membership authority | Controlled integration test proves `ErrInspectionMembershipUnavailable` when no validator is configured | It is intentionally not a canonical membership implementation. |
| Explicit production composition and opt-in route mounting | `TestNewMuxMountsWorkOrderRouteOnlyWhenHandlerProvided`, `TestWithinTransactionPassesRepositoryAndPreservesCallbackError`, and `go test ./...` after `cmd/integin-server` composition | The repository owns SQL mutation transactions; the service adapter deliberately prevents a misleading nested service transaction. |
| Authenticated partial-submission HTTP path through canonical PostgreSQL persistence | Controlled test `TestAuthenticatedPartialSubmissionHTTPPostgresIntegration` against `integin-pilot-postgres` | Uses a test OIDC validator double to isolate the local server transport boundary; it does not validate an external OIDC issuer or live identity-provider configuration. |
| Controlled OIDC discovery, JWKS, RS256 signature, temporal-claim, and HTTP composition | Controlled test `TestSignedOIDCPartialSubmissionHTTPPostgresIntegration` uses a local loopback issuer, discovery document, JWKS, signed RS256 token, production `oidcauth.Validator`, local identity resolver, server mux, and canonical persistence | The issuer is an in-process controlled test issuer, not a deployed external identity provider. |
| Disposable Keycloak provider-to-Work-Order HTTP execution | `TestKeycloakPartialSubmissionHTTPPostgresIntegration` ran with a real Keycloak `26.7.1` server, generated realm/client/user, actual password-grant access token, production discovery/JWKS validator, local identity resolver, and canonical PostgreSQL persistence | The Keycloak server was disposable and loopback-only; this does not establish a hosted production Keycloak environment. |
| Keycloak authentication-negative boundary | The disposable Keycloak test sent a tampered genuine Keycloak access token and received `401` before the valid token completed the Work-Order submission | This does not enumerate every provider outage, revoked-token, or operational key-rotation scenario. |
| OIDC negative rejection before Work-Order mutation | Same signed-OIDC test returned `401` for a tampered signature and an expired token before the valid signed token returned `200` | It does not enumerate every invalid claim combination, issuer outage, or live provider misconfiguration. |
| Field Matrix Fixture Provisioning runtime | The Field acceptance test ran against the existing loopback-only acceptance server, provisioned a device, applied signed sync, accepted evidence once, and returned duplicate-safe evidence handling on replay | The proof is limited to the controlled local acceptance environment; it does not establish Field behavior against a production service or external identity provider. |
| Field runtime cleanup | The Field test wrote a receipt before provisioning; the cleanup harness removed the receipt-named device, authority, sync rows, and S3/RustFS evidence object, then verified absence | The cleanup harness is intentionally receipt-scoped and must not be repurposed for unscoped deletion. |
| Client-provided Work-Order authority rejection | `TestHandlerRejectsClientSuppliedAuthorityFields` and the controlled PostgreSQL HTTP test prove unknown `tenant_id`, `organization_id`, and `actor_id` body fields return `400` before a mutation | The route accepts only its typed operation, revision, Work-Order, assignment, and inspection fields; authority remains server derived. |
| Server-derived identity authority and request-body authority rejection in runtime flow | Same controlled HTTP test supplies attacker-controlled `tenant_id`, `organization_id`, and `actor_id` body fields while the persisted receipt derives its tenant and work-order identity from the resolved local membership | The body decoder ignores unknown fields rather than actively rejecting them; future API tightening may reject those fields explicitly. |
| HTTP idempotent replay and non-disclosing cross-organization denial | Same controlled HTTP test: valid request `200`, replay `200` with equivalent receipt, cross-organization request `403` without the work-order ID in the response | Does not prove every Work-Order route or a real identity-provider-issued access token. |
| Controlled runtime cleanup | Same test performs explicit in-test cleanup assertion, and a post-run pilot inventory reported zero `it-http-*` work orders, inspections, and identity subjects | Applies only to its namespaced fixtures; it is not a general pilot-data retention policy. |
| Applied-pilot non-owner canonical inspection read isolation | Controlled `SET LOCAL ROLE integin_runtime` proof with a same-tenant, two-organization `it-rls-read-*` graph; the role saw one record in Organization A, one in Organization B after context switch, and zero with context unset | Covers the tested `inspection_record` read query and transaction-local RLS context only; it does not prove every table, UI route, or application authorization policy. |

## Canonical inspection candidate evidence (disposable database only)

| Claim | Evidence | Limits |
|---|---|---|
| Candidate SQL syntax and dependency order | `0005_work_order_foundation.candidate.sql` then revised `0008_inspection_persistence.candidate.sql` applied successfully in a fresh disposable PostgreSQL 18 container | Not applied to the controlled pilot database or any production environment. |
| Valid same-order containment graph | Namespaced two-order fixture created valid work orders, scopes, assignments, inspections, one segment, and one normalized submission item | Does not exercise the application reader or Work-Order repository composition. |
| Same-organization cross-order item denial | Deliberate attempt to attach Order B inspection to Order A segment failed through the composite inspection containment foreign key | Proves this candidate database constraint only. |
| Forced RLS, non-owner read isolation, and write denial | Candidate tables reported `relforcerowsecurity=true`; a non-owner runtime role saw 0 rows under Organization B and 1 row under Organization A. A cross-organization inspection insert was rejected with `new row violates row-level security policy`. | These checks apply only to the fresh disposable candidate database, not the pilot database or a production runtime role. Superuser checks are not RLS evidence because superusers bypass RLS. |
| Candidate rollback | Revised down candidate removed inspection tables and all three added containment constraints; post-check returned `true|true|true` | Does not prove restore from a pilot backup or compatibility with existing pilot data. |

## Explicitly unproven or blocked claims

| Claim not yet made | Blocker | Required next proof |
|---|---|---|
| Hosted production Keycloak operation | The disposable Keycloak proof establishes protocol integration and cleanup, but does not establish a hosted provider's availability, administrator configuration, deployment, monitoring, or key-rotation operations | Production-specific Keycloak operations, deployment evidence, and operational runbooks when the production environment is introduced |
| Full service-level SQL transaction spanning a multi-step application operation | Each repository mutation intentionally owns its authoritative SQL transaction; the current service adapter is explicit pass-through to avoid false nested-transaction semantics | A repository API redesign that exposes a transaction-bound unit of work, plus atomic multi-step test coverage |
| Broad non-owner read isolation across all Work-Order tables and application routes | The completed proof covers the canonical `inspection_record` read query with ephemeral, cleaned fixtures | Table- and route-specific tests proportionate to each additional read surface |
| Flutter Matrix Fixture Provisioning runtime gate | Flutter tooling remains unavailable on the workstation | Toolchain availability and isolated Field-app fixture run |

## Fixture rule

Each controlled database test must use a unique `it-*` ID namespace, list every inserted table, clean up in dependency order under the correct tenant and organization context, and fail its test if cleanup cannot commit. No test may delete rows outside its own namespace.

## Next exact artifact

Prepare a separately governed real-OIDC runtime preflight only after an isolated identity-provider configuration and signed test token are available. The existing database-backed HTTP proof is complete; it must not be overstated as external identity-provider proof.

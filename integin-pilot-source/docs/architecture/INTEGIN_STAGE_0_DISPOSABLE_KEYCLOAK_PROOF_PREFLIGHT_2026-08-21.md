# INTEGIN Stage 0 Disposable Keycloak Proof Preflight

## Objective

Prove the Work-Order partial-submission path against a real **Keycloak** OIDC implementation, rather than an in-process issuer double, while keeping the identity provider and all pilot data disposable.

## Isolated runtime boundary

| Component | Boundary |
|---|---|
| Keycloak runtime | One disposable Docker container bound only to `127.0.0.1:8180`; no production hostname, realm, client, user, or credential is reused. |
| Keycloak storage | Development-mode ephemeral storage inside the disposable container; no existing Keycloak or PostgreSQL container is started or modified. |
| Admin and test credentials | Generated in process for this one proof, never written to source, logs, task records, or user-facing output. |
| Realm/client/user | A generated `it-keycloak-*` realm, one direct-access OIDC client, and one generated inspector user. |
| Pilot data | Generated `it-keycloak-*` identity, membership, Work-Order, scope, assignment, and canonical inspection fixtures only. |

## Required proof

The test must use Keycloak discovery and JWKS through the production `oidcauth.Validator`; obtain a genuine Keycloak access token for the generated user; resolve the Keycloak issuer/subject through PostgreSQL local identity membership; submit canonical inspection records through the authenticated server mux; and assert at least one authentication-negative request before mutation.

## Cleanup requirements

The test must remove pilot rows through its established `t.Cleanup` helpers. The runtime harness must delete the generated Keycloak realm, stop and remove the disposable container, and assert that no `it-keycloak-*` pilot rows remain. A Keycloak, Docker, or database cleanup failure fails the proof.

> This proves Keycloak protocol and INTEGIN integration in a disposable local environment. It does not claim an internet-hosted production Keycloak deployment, its availability, or its operational key-rotation process.

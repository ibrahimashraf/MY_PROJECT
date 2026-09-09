# ADR 0032: Adoption of Casdoor as Lightweight Go-Native IdP (Replacing Keycloak)

## Status
Accepted

## Date
2026-09-09

## Context & Problem Statement
INTEGIN relies on an external OpenID Connect (OIDC) Identity Provider (IdP) for authenticating users (`Issuer` + `Subject`).
Currently, `integin-pilot-keycloak` (Quarkus/Java) is deployed for local testing and pilot harnesses. However:
1. **Memory Bloat**: Keycloak consumes **~520 MB RAM** at idle.
2. **Slow Cold Boot**: Keycloak takes **~35 seconds** to become healthy.
3. **Complex Configuration**: Heavy administration overhead for straightforward OIDC token issuance.

The user required an identity solution that is:
- **100% Free & Permissive Open Source** (No freemium traps, no paywalled multi-tenancy, no commercial seat licenses, no AGPL copyleft viral risks).
- **Lightweight** (substantial memory reduction from ~520 MB).
- **Architecturally Compliant with INTEGIN**:
  - Emits standard OpenID Connect discovery and RFC 7517 JWKS.
  - Adheres to the **RS256 ceiling** enforced by [`internal/oidcauth/validator.go`](../../integin-pilot-source/internal/oidcauth/validator.go).
  - Preserves the single source of truth in PostgreSQL RLS (`identity.PrincipalKey -> Resolver -> Membership`).
  - Strict database isolation (cannot pollute or share the `integin_dev` database pool).

## Evaluated Alternatives & Rejection Reasons
| Alternative | Rejection Rationale |
|---|---|
| **ZITADEL** | Relicensed to **AGPL-3.0** with dual-licensing (v3). Cloud free tier capped at 100 DAU. Self-hosting carries viral copyleft obligations for distributed platforms. |
| **SuperTokens** | **Paywalled**. Multi-tenancy, user roles dashboard, SAML SSO, and SCIM require an Enterprise License Key. |
| **Ory (Kratos + Hydra)** | Multi-tier trap. Ory gates B2B multi-tenancy behind commercial Ory Enterprise License (OEL). Requires deploying and bridging multiple separate daemons. |
| **Authentik** | Does not solve size/footprint. Requires Python/Django, Celery workers, Redis, and Postgres (~400 MB RAM total). |
| **PocketBase** | Cannot act as an OIDC server (OAuth consumer only; does not provide OIDC Discovery/JWKS). |
| **Cerbos** | PDP, not an IdP. Evaluates YAML policies over gRPC. Breaks offline bunker tablet evaluation. INTEGIN already uses PostgreSQL RLS + Go CEL as its authoritative PDP. |
| **Hanko** | Passkey-first frontend library. Signs tokens using ES256/EdDSA, which breaks the INTEGIN RS256 validator ceiling. |
| **Authelia** | Evaluated positively (MIT, 25MB RAM), but lacks a built-in multi-tenant organization web console for visual pilot admin. |

## Decision
Adopt **Casdoor** (`casbin/casdoor:latest` digest-pinned, currently v4.3.0 = `sha256:1b479655…`; never the `casdoor/casdoor` name, which does not exist on Docker Hub) as the Go-native IdP:
1. **License**: 100% Apache 2.0 with zero paywalled features.
2. **Footprint**: Single Go binary + built-in React web UI. Consumes **~110 MB RAM** (75% savings over Keycloak) and boots in **~2 seconds**.
3. **Protocol**: Compliant OpenID Connect provider supporting RS256 token signing.
4. **Isolation Boundary**: Runs in a dedicated container (`integin-pilot-casdoor`) on its own isolated network and database (`integin-pilot-casdoor-postgres`), completely separated from `integin_dev` and PgCat.

## Consequences
- Keycloak can be deprecated and cleanly replaced in local pilot environments.
- Developer laptop memory overhead reduced by >400 MB.
- Zero changes to Go domain code: `internal/identity/contracts.go` continues to process `PrincipalKey{Issuer, Subject}` identically.

## Live Verification (2026-09-09, this repo)- `operations/pilot/start-casdoor-pilot.ps1` boots isolated PG + Casdoor on `127.0.0.1:18180`; discovery verified.
- `initialize-casdoor-pilot-realm.ps1` (idempotent) seeds org `integin-pilot`, RS256 app `integin-live-matrix` (`cert-built-in`), inspector user; grants `authorization_code,password,client_credentials`.
- `TestCasdoorPartialSubmissionHTTPPostgresIntegration` PASSES live: password-grant → RS256 validate → RLS membership → 200 + persistence + cleanup. Casdoor password-grant tokens carry no `azp`; test config leaves `AuthorizedParty` empty (validator skips the check for single-`aud` tokens — no core change).
- `cmd/integin-live-matrix` accepts `INTEGIN_OIDC_TOKEN_ENDPOINT` override (Casdoor: `http://127.0.0.1:18180/api/login/oauth/access_token`); `client_credentials` grant verified live.
- Keycloak containers removed; pilot default issuer is now Casdoor. Rollback: restore Keycloak on `:18180` and point `INTEGIN_OIDC_*` back at `…/realms/integin-pilot`.

## Promotion Gates (do not promote past pilot without these)
- **SAML permanently out of scope.** CERT VU#780781 cluster (CVE-2026-9090/93/95/96/98, CVE-2026-15630) is SAML-only; we stay OIDC-only. Never enable a SAML provider or `/api/acs` flow.
- **MFA gated on CVE-2026-9091 fix** (GO-2026-5896, MFA bypass, no known fix as of 2026-09-09). Password-only until the advisory lists a patched version; verify by re-querying the GHSA entry.
- **Short token lifetimes as revocation substitute.** CVE-2026-9094/9097 (no reliable revocation): keep `MaxTokenAge` ≤ 15 min and never issue long-lived tokens.
- **No upstream federation without claim review.** CVE-2026-9092 (unverified email takeover): any Google/Azure/GitHub provider wiring must require verified email claims.
- **Release tracking.** Pinned image must stay within ~30 days of upstream; check `casdoor/casdoor` releases monthly. v4.1.0 (2026-09-02) is the first release past the CVE-2026-84423 (`upload-resource` missing auth) window and adds missing permission checks — minimum promotion floor.
- **PKCE** mandatory for any future browser `authorization_code` client.

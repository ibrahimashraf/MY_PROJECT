# INTEGIN Self-Hosted Identity and Authorization Design

**Status:** Draft v0.1 — Keycloak selected provisionally; implementation requires security review  
**Date:** 2026-08-15  
**Owner:** Security Operator and Engineering Change Owner  
**Scope:** Human authentication, service identity, OIDC token handling, tenant/organization authorization, and operational recovery.

## 1. Architecture decision

INTEGIN will use **Keycloak** as the provisional self-hosted OIDC provider. The decision is based on its Apache-2.0 licensing, official self-managed/container path, broad OIDC/OAuth 2.0/SAML interoperability, MFA/passkey/recovery capabilities, claim mapping, session management, directory federation, and client service accounts. [1] [2]

This is deliberately an **identity** decision, not a transfer of INTEGIN authority. Keycloak will establish who authenticated and how. The Go modular monolith and PostgreSQL will continue to decide which tenant/organization the identity may act in, which role/capability applies in that context, whether a device is trusted, whether an offline authority is valid, and whether any requested workflow action is permissible.

| Decision | Adopted pattern | Rejected pattern |
|---|---|---|
| Identity provider | Keycloak, self-hosted and patched under the security release process | Cloud-only IAM dependency or application-managed passwords |
| Realm topology | One application realm per environment: `integin-dev`, `integin-test`, `integin-prod` | One Keycloak realm per customer tenant |
| Tenant authorization | Go/PostgreSQL membership projection and transaction-scoped RLS | Trusting a client-submitted tenant ID or Keycloak group alone |
| Field-client login | Authorization Code with PKCE | Resource-owner password flow, embedded client secret, or device-held admin credential |
| Offline authority | Server-signed INTEGIN package bound to device epoch | Treating an OIDC access token as offline field authority |
| Identity database | Separate PostgreSQL database and credentials for Keycloak | Keycloak owning or writing INTEGIN business tables |

## 2. Why Keycloak is the baseline

The initial alternatives remain documented: authentik is a viable MIT-licensed self-hosted option with meaningful service-account controls, while ZITADEL offers an explicit B2B organization/project/role model. [3] [4] Keycloak is selected for the first baseline because its mature, broad standards surface allows INTEGIN to keep multi-organization authorization inside the existing Go/PostgreSQL authority rather than adopting a second competing tenancy model.

This choice can be re-opened only through the engineering change process, a documented export/migration plan for identities and clients, token/claim compatibility tests, and a rollback decision. It is not a reason to change the Go authority, Flutter client, PostgreSQL tenant model, or RustFS evidence contract.

## 3. Identity model and boundaries

### 3.1 Identity subject mapping

For every accepted OIDC identity, the Go authority maintains a local `identity_subject` record keyed by the immutable pair `(issuer, sub)`. Email, username, display name, and external directory attributes are profile data—not keys for authorization. An email change must not produce a new INTEGIN identity or silently join an existing one.

| Data element | Origin | How INTEGIN uses it |
|---|---|---|
| `iss`, `sub` | Validated OIDC token | Immutable identity key and audit actor reference. |
| `aud`, `azp`, `client_id` | Validated OIDC token | Ensures the token was issued for the intended INTEGIN client. |
| `exp`, `nbf`, `iat`, `auth_time`, `sid`, `amr`, `acr` | Validated OIDC token | Session freshness, MFA/step-up policy, logout/incident response. |
| `email`, `name`, `preferred_username` | OIDC token or directory sync | Display/contact convenience only; never tenant authorization. |
| Realm/client roles and groups | OIDC claims | Inputs to local role projection only; never an independent bypass of INTEGIN authorization. |
| Tenant/organization selection | Go authority | Resolved by active local membership; not trusted from a client request. |

### 3.2 INTEGIN-owned authorization records

The Go/PostgreSQL authority owns `tenant`, `organization`, `identity_subject`, `organization_membership`, `membership_role_binding`, `environment_access`, and `capability_grant` records. It projects only the minimal data needed for fast server authorization and RLS context selection.

Keycloak may hold global platform identities such as `platform_operator` or `security_operator`. Such a claim can enable access to a guarded INTEGIN administrative entry point, but INTEGIN still requires an active local record and explicit capability before a request can access customer data or mutate device/workflow state.

## 4. Keycloak configuration baseline

### 4.1 Deployment topology

Keycloak runs in a separate container/service boundary, behind an HTTPS reverse proxy or ingress. Its PostgreSQL database is separate from `integin`; it has a unique non-superuser database owner and credential. Only the Keycloak service may access the identity database. The Go service consumes OIDC discovery/JWKS material over TLS and never receives the Keycloak database credential or administration password.

The `master` realm is retained only for recovery administration and is not used by field or operations clients. The INTEGIN application realm is environment-specific, exports are encrypted and access controlled, admin events are enabled, and backups are exercised separately from INTEGIN business-state backup drills. Keycloak export files are not treated as the sole backup mechanism; the database and configuration recovery path must be tested. [1]

### 4.2 OIDC clients

| Client | Type and flow | Purpose | Prohibited configuration |
|---|---|---|---|
| `integin-field` | Public native client; Authorization Code + PKCE | Flutter human sign-in before online operations or enrollment | Client secret, password grant, implicit flow, wildcard redirect URI |
| `integin-operations` | Public browser client; Authorization Code + PKCE | Future TypeScript operations workbench | Token in local storage, wildcard redirect URI |
| `integin-advisor-ui` | Public browser client; Authorization Code + PKCE | Advisory-only console login | Any primary workflow/write authority |
| `integin-api` | Resource server / audience | Go API token validation | Treating frontend roles as sufficient tenant authorization |
| `integin-automation-*` | Confidential service client or dedicated service account | Narrowly scoped scheduled/service integration | Shared credentials, tenant-wide authority, interactive human login |

Redirect URIs, CORS origins, post-logout URIs, and token audiences must be exact per environment. Development clients and signing keys must never be re-used in test or production.

### 4.3 Authentication assurance

All privileged roles—Tenant Administrator, Platform Operator, Security Operator, and Engineering Change Owner—require MFA. Preferred factors are passkeys/WebAuthn or TOTP with recovery codes. Recovery factors must be controlled by an audited Security Operator workflow, not an unverified email reset alone. Field users require MFA before device enrollment and whenever tenant policy requires step-up authentication; a valid low-assurance session must not authorize device approval, evidence export, or role changes.

The initial token posture is short-lived access tokens, refresh-token rotation, server-side logout/session revocation, and re-authentication for high-risk actions. Exact time-to-live values are a security-policy setting and must be documented per environment rather than hard-coded by a client.

## 5. Go API token validation and authorization sequence

The Go API validates OIDC tokens before entering application authorization. A failure returns an unauthenticated response and never reaches a tenant-scoped repository.

1. Require TLS and extract the bearer token from the authorization header.
2. Resolve OIDC discovery and cached JWKS keys from the configured issuer; reject unknown/expired key material according to the key-rotation policy.
3. Validate signature, issuer, audience, authorized party where applicable, `exp`, `nbf`, and token type.
4. Bind `(iss, sub)` to the local `identity_subject` record. Resolve active local memberships and server-side role/capability grants.
5. If a request requires organization context, validate the selected organization against that membership. Do not infer an organization from URL, body, or token claim alone.
6. Set the validated tenant and organization context inside the PostgreSQL transaction. RLS remains the database backstop.
7. Evaluate capability, resource scope, workflow/environment state, device/authority state, and separation-of-duties constraints through the existing Go authorization engine.
8. Write an audit record for privileged or state-changing actions, then process the authoritative use case.

This sequence makes OIDC a necessary authentication layer but never a sufficient tenant authorization decision.

## 6. Device enrollment integration

Production device enrollment begins only after the field user has an authenticated OIDC session and a locally resolved active membership. The enrollment request stores the issuer/subject pair and local membership reference, then follows `PRODUCTION_DEVICE_ENROLLMENT_SPEC.md` for key proof, approval, authority issue, revocation, and recovery.

OIDC access and refresh tokens are not copied into the offline authority package and must not be used to sign offline mutations. The device uses its locally generated Ed25519 key for those signatures; the server uses the OIDC session only for online human authentication and enrollment/refresh requests.

## 7. Roles, service identities, and separation of duties

| INTEGIN role | Keycloak concern | INTEGIN-enforced capability boundary |
|---|---|---|
| Field User | Standard user session and policy-driven MFA | May request device enrollment and perform only assigned, scoped work through a valid authority. |
| Tenant Administrator | Strong MFA and elevated session assurance | May approve eligible enrollment and manage local membership; may not override RLS, signatures, authority epoch, or retention. |
| Security Operator | Strong MFA, step-up, restricted admin-client access | Controls IdP configuration, key policy, containment, and audited recovery—not tenant business workflows. |
| Platform Operator | Strong MFA | Operates runtime, backup, and recovery; no direct tenant workflow mutation. |
| Advisory Reviewer | Standard user session, optionally MFA by policy | Sees non-blocking advisory data; cannot alter primary decisions. |
| Automation service | Dedicated Keycloak service account per integration | Receives only an explicit API scope; cannot represent a human approver or own a field device. |

No service account may approve a device, grant itself a role, or impersonate a human subject. No Keycloak administrator action by itself may create a INTEGIN tenant membership, device trust state, or offline authority; each must pass through a Go application service and its audit path.

## 8. Break-glass and recovery

Break-glass access is an exceptional audited procedure for identity-provider loss, misconfiguration, or security containment. It consists of at least two named Security Operators, time-bounded access, a recovery ticket/reference, and post-incident review. It must not bypass the Go authority to edit tenant business data.

The procedure must document: IdP database/configuration restore; signing-key/JWKS recovery and rotation; disabling compromised clients or sessions; re-establishing an administrator under MFA; reconciliation of identity subject mappings; and an operator assertion that no production enrollment or offline authority was fail-open during the incident.

## 9. Implementation and acceptance gate

The implementation sequence is: provision isolated Keycloak and identity PostgreSQL; configure the environment realm and clients; implement Go discovery/JWKS validation and local subject mapping; create tenant-membership/role migrations with RLS; add OIDC and cross-tenant authorization tests; implement production enrollment; add observability for authentication failures without logging tokens; and perform a backup/restore drill of identity configuration and database.

Promotion requires successful tests for invalid issuer/audience/signature/expiry, stale key rotation, cross-tenant membership, disabled user/session, missing MFA/step-up for privileged action, service-account least privilege, logout/revocation, RLS context, and IdP outage without an authorization fail-open. It also requires an approved privacy review for profile claims and a release record linking Keycloak image digest, configuration version, database migration, recovery evidence, and rollback plan.

## References

[1]: https://www.keycloak.org/docs/latest/server_admin/index.html "Keycloak Server Administration Guide"

[2]: https://github.com/keycloak/keycloak "Keycloak source repository and container startup"

[3]: https://github.com/goauthentik/authentik "authentik source repository and self-hosting overview"

[4]: https://zitadel.com/docs/guides/solution-scenarios/b2b "ZITADEL B2B Multi-Tenant Authentication"

[5]: ./PRODUCTION_DEVICE_ENROLLMENT_SPEC.md "INTEGIN production device enrollment protocol"

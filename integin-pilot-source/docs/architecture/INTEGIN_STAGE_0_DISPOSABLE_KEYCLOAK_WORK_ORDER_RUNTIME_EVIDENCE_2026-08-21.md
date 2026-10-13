# INTEGIN Stage 0 Disposable Keycloak Work-Order Runtime Evidence

## Result

The final local provider gate **passed**. A temporary Keycloak `26.7.1` container was bound to loopback only. The harness created a generated realm, an OpenID Connect client, a generated fully configured test user, and a client audience mapper. It then obtained a genuine Keycloak password-grant access token.

The production OIDC validator fetched Keycloak discovery metadata and JWKS, validated the access token's issuer, audience, authorized party, signature, and temporal claims, and supplied only the validated issuer/subject principal to local identity resolution. Local PostgreSQL membership supplied tenant, organization, actor, role, and capability authority.

| Request | Observed result |
|---|---:|
| Tampered genuine Keycloak token | `401` before mutation |
| Valid Keycloak token | `200` |
| Canonical inspection records | Submitted with normalized submission items |
| Pilot Work-Order, inspection, and subject cleanup | Zero `it-keycloak-*` rows |
| Disposable Keycloak container | Removed |

## Recovery evidence

The first generated Keycloak user was rejected by Keycloak's default account-completeness policy. The provider returned only the bounded OAuth error description, `Account is not fully set up`; no token or credential was disclosed. The harness was corrected to create a complete, email-verified generated user and the proof was then repeated successfully. Every failed attempt removed its temporary realm/container and left zero pilot fixtures.

## Limit

This validates Keycloak protocol interoperability with the Stage 0 INTEGIN path. It does not make a claim about any hosted production Keycloak cluster, real administrator setup, provider uptime, certificate management, monitoring, or key rotation.

# INTEGIN Stage 0 Controlled OIDC Work-Order HTTP Evidence

## Result

`TestSignedOIDCPartialSubmissionHTTPPostgresIntegration` **passed** against the local pilot PostgreSQL database. The test uses a loopback-only issuer that serves OpenID discovery and a JWK Set containing a generated RSA public key. The production `oidcauth.Validator` fetched discovery metadata and the JWKS, validated an RS256 bearer token, and yielded only an issuer/subject principal to the production local identity resolver.

| Request | Expected result | Observed result |
|---|---:|---:|
| Tampered signed token | `401` | `401` |
| Expired signed token | `401` | `401` |
| Valid RS256 token with required audience, authorized party, AMR, issuer, and temporal claims | `200` | `200` |

For the accepted request, local identity membership supplied the tenant, organization, actor ID, Work-Order role, and capability. The Work-Order path then validated canonical inspections, persisted normalized submission items, changed the two canonical records to `SUBMITTED`, and returned the authoritative receipt.

## Scope and cleanup

The test uses generated `it-oidc-http-*` work-order, inspection, and identity rows only. It performs explicit cleanup and final absence assertions. A post-run inventory reported zero rows in all three namespaces. Repository-wide `go test ./...` and `go vet ./...` passed after the test was added.

> This is a controlled local OIDC-provider simulation. It proves INTEGIN discovery/JWKS/signature/claim handling and its composition with local identity and Work-Order persistence. It does not prove the availability, administration, key rotation, or operational configuration of a real external provider.

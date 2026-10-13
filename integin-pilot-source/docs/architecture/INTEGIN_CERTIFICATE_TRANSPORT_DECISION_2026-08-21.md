# INTEGIN Certificate Transport Decision

**Status:** Approved transport design for implementation. The repository-level lifecycle is proven; this record adds no renderer, QR image, external submission, or production public release.

## Authenticated mutation boundary

Each lifecycle mutation route validates a bearer token through the existing OIDC validator, resolves one local membership from the authenticated principal, and maps only the membership actor ID, tenant ID, organization ID, and membership capability set into the certificate authority actor context. The body cannot provide tenant, organization, actor, capability, policy, validity, certificate number, signer, issuer, public token, status, inspection facts, template snapshot, or audit content. Unknown JSON fields are rejected.

| Route | Accepted body fields | Repository operation | Required server-derived capability |
| --- | --- | --- | --- |
| `POST /certificates/drafts` | Certificate ID, inspection ID, template code/version, profile hint, self-issue reason. | `CreateDraft` | `certificate.prepare`; profile is still policy-validated. |
| `POST /certificates/{id}/submit` | None. | `Submit` | `certificate.prepare`. |
| `POST /certificates/{id}/review` | None. | `Review` | `certificate.review`. |
| `POST /certificates/{id}/sign` | None. | `Sign` | `certificate.sign`. |
| `POST /certificates/{id}/issue` | None. | `Issue` | `certificate.issue`; raw public token is returned only once to the authenticated caller. |
| `POST /certificates/{id}/revoke` | Bounded reason only. | `Revoke` | `certificate.revoke`. |
| `POST /certificates/{id}/supersede` | Replacement certificate ID only. | `Supersede` | `certificate.supersede`. |

All authenticated replies are `Cache-Control: no-store`; errors are bounded and do not disclose cross-scope existence. The initial transport maps repository validation failures to `400`, absent/invalid authentication to `401`, no local membership or missing capability to `403`, and scope-hidden or transition-ineligible records to `409` or `404` only where the route has already established caller scope.

## Public verifier boundary

The initial public route is `GET /verify/certificates/{token}`. It accepts only URL-safe token grammar with a fixed bounded length, applies a process-local fixed-window limiter before any database lookup, sets `Cache-Control: no-store`, and returns only the repository `PublicProjection`: certificate number, status, issue time, expiry time, and asset ID. Invalid/missing/unknown token responses are intentionally indistinguishable at the public boundary. It exposes no raw token, internal identifier, tenant/organization, person, evidence, snapshot, audit, private reason, asset serial, description, or test scope.

Full serial number, full description, and inspection test scope remain owner-required public fields, but their server-authoritative data sources and approved binding keys do not exist in the current catalog. The verifier must not fabricate them.

## Acceptance evidence

The transport proof must cover valid OIDC-to-membership actor derivation, absent/malformed token rejection, ambiguous/absent membership denial, authority-shaped/unknown payload field rejection, independent and self-issue capability denial, no-store headers, one-time issued-token disclosure contract, correct public projection, invalid-token indistinguishability, limiter rejection, cross-scope denial, no fixture residue, and full regression/vet.

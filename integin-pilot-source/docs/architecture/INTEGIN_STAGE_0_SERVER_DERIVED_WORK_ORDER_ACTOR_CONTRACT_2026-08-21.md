# INTEGIN Server-Derived Work-Order Actor Contract

**Status:** Design prerequisite. No identity migration or HTTP mutation route is implemented by this document.

## Required additive identity result

The migration-owned `integin_resolve_identity_membership(issuer, subject)` function must return exactly one active local membership containing: immutable local actor ID, tenant ID, organization ID, approved Work-Order role, and capabilities. Actor ID and role are local INTEGIN data, never trusted OIDC claims or request fields.

## Role policy

| Role | Minimum mutation authority |
|---|---|
| inspector | Submit partial work only for an active assignment owned by that inspector. |
| manager | Create requests, assign/reassign scope, transition authorized order states, reconcile provisional onboarding records. |
| reviewer | Request certificate validation; no direct inspection mutation unless separately authorized. |
| administrator | Administrative override only through audited explicit policy; no implicit bypass. |

## Required denial behavior

Unknown, ambiguous, inactive, revoked, cross-organization, or capability-inadequate memberships must fail before the Work-Order repository is called. The HTTP layer must return a generic authorization failure and must not reveal order existence.

## Evidence before route implementation

1. Identity migration candidate with RLS and runtime-function grants reviewed in a disposable database.
2. Resolver unit tests for one active membership, unknown subject, ambiguous membership, disabled membership, and role/capability mismatch.
3. Controlled OIDC-to-actor integration proof showing actor fields come only from the resolver.
4. Work-Order HTTP mutation tests proving request-body actor impersonation is ignored or rejected.

## Current conclusion

The existing identity resolver cannot yet construct a Work-Order `ActorContext` because `Membership` contains only tenant, organization, and capabilities. This contract is the immediate prerequisite for production composition.

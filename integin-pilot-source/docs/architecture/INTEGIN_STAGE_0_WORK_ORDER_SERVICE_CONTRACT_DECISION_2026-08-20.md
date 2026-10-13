# INTEGIN Stage 0 Work-Order Service Contract — Review and Decision

**Task template:** Complex feature  
**Mental effort:** Deep  
**Owner:** Manus AI  
**Mutation owner:** Manus AI only  
**Boundary:** Source-only service contract; no database migration, runtime authority change, certificate issuance change, or external submission.

## Review panel configuration

The bounded panel used six independent lenses because the service contract affects architecture, security, tenant/data isolation, offline Field behavior, assurance, and testing. The panel was configured as a review plus implementation workflow, with the owner retaining the final decision and mutation authority.

| Lens | Question |
|---|---|
| Systems architecture | Does the contract preserve one authoritative work-order boundary and clean dependencies? |
| Security and safety | Can tenant, client, assignment, and authority boundaries be bypassed? |
| Tenant/data isolation | Are revisions, idempotency, and cross-tenant reads/mutations explicit? |
| Field/offline workflow | Can preassigned and emergency provisional work reconcile without duplicate assets or lost drafts? |
| Evidence/assurance | Does the contract separate field completion, certificate validation, and invoice release? |
| Quality/verification | Are negative cases and evidence gates explicit before persistence or UI integration? |

## Panel findings

The architecture lens recommends a small service contract above the domain types and below transport adapters. The service must accept commands with tenant context, actor context, expected revision, and idempotency metadata. Transport handlers must not implement lifecycle policy themselves.

The security and data lenses require server-side tenant and assignment checks on every command. A local cache, client-provided client ID, or assignment snapshot must never be treated as authorization. Idempotency must be scoped to tenant and operation identity, and an expected-revision mismatch must produce a conflict rather than an implicit merge.

The Field lens confirms that the service needs explicit provisional-candidate commands and reconciliation outcomes. It must preserve local evidence when a candidate is matched, created, rejected, or placed in conflict. Existing assets are referenced by canonical ID whenever available; provisional candidates cannot silently claim a canonical identity.

The assurance lens requires separate commands for partial submission, work-order completion, certificate validation request, and commercial release. Acknowledging a partial submission must not imply certificate issuance or invoice release.

The testing lens requires focused pure-domain tests for cross-tenant rejection, out-of-scope assets, stale revisions, duplicate idempotency keys, reassignment after offline work, provisional-record outcomes, and lifecycle separation.

## Owner decision

Implement the next slice as **typed service interfaces and command/result contracts only**, backed by pure validation helpers. Do not add an in-memory fake as a production authority and do not add database persistence until the schema and transaction strategy are separately reviewed. This keeps the contract testable without accidentally presenting a non-authoritative store as implementation.

The service contract will expose separate operations for creating a work-order request, assigning scope, transitioning execution, submitting a partial segment, reassigning scope, reconciling a provisional record, and requesting certificate validation. Every mutation carries tenant/actor context, idempotency metadata, and an expected revision where applicable.

## Evidence gate

The slice may be considered complete only if the package compiles, focused tests cover positive and negative commands, all internal domain tests pass, the diff contains no migration or runtime changes, and the decision record remains consistent with the implemented API.

## Deferred risks

The service contract does not yet prove PostgreSQL transaction behavior, row-level security, production idempotency storage, Field synchronization, certificate rendering, or external authority integration. Those require later implementation and independent evidence.

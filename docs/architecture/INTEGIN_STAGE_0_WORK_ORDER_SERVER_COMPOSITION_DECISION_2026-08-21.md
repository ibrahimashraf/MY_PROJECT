# INTEGIN Work-Order Server Composition Decision

## Transaction boundary

The current `workorderpg.Repository` opens and commits its own authoritative SQL transaction for each mutation, including tenant/organization context, revision checks, idempotency, canonical inspection validation, normalized submission writes, and state changes. The Work-Order application service still expects a `TransactionRunner` interface.

For the current Stage 0 adapter, composition must use an explicit **repository-owned transaction adapter** whose `WithinTransaction` invokes the callback with the same repository and does not begin another SQL transaction. This is not a weakening of mutation atomicity: repository mutations retain their existing authoritative transaction. It merely prevents an invalid nested-transaction illusion at the service layer.

## Server composition requirements

1. Construct `workorderpg.Repository` with `NewPostgresInspectionMembershipValidator`.
2. Construct `workorder.Service` with that repository, the repository-owned transaction adapter, and `workorderauth.New()`.
3. Construct `workorderhttp.Handler` with the OIDC validator, local identity resolver, and service.
4. Mount only the selected partial-submission route after dependency construction succeeds.
5. Do not enable this route when OIDC or database identity dependencies are unavailable.

## Runtime evidence status

The controlled pilot proof is recorded in `INTEGIN_STAGE_0_AUTHENTICATED_WORK_ORDER_HTTP_RUNTIME_EVIDENCE_2026-08-21.md`. It used a server-derived local identity fixture, an authorized active assignment, canonical inspections, idempotency retry, cross-organization denial, and verified fixture cleanup. A real external OIDC-provider proof remains separate and unproven.

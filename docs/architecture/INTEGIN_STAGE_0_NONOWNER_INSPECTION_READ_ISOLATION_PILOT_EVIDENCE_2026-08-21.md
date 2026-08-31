# INTEGIN Stage 0 Non-Owner Canonical Inspection Read-Isolation Pilot Evidence

## Result

The controlled pilot proof **passed**. The non-owner `integin_runtime` role is neither superuser nor `BYPASSRLS`, has the required `SELECT` privileges, and queried canonical inspection records under forced RLS.

| Runtime context | Expected visible test records | Observed count |
|---|---:|---:|
| Shared test tenant and Organization A | Organization A only | 1 |
| Shared test tenant and Organization B | Organization B only | 1 |
| Tenant and organization context unset | None | 0 |

The proof used two generated `it-rls-read-*` Work-Order graphs in one tenant, one per organization. Every graph contained one work order, scope item, active assignment, assignment-scope relation, and completed/open canonical inspection record. It performed no privilege, policy, migration, server, or external-authority change.

> The proof was run through `SET LOCAL ROLE integin_runtime`, not as a table owner or superuser. This is material because the role cannot bypass row-level security.

## Cleanup and regression

Owner-side cleanup ran in dependency order under the matching tenant and organization contexts. The final inventory reported zero namespaced work orders and zero namespaced inspection records. A subsequent `go test ./...` passed across the complete repository.

## Limits

This establishes isolation for the tested `inspection_record` query, role, and RLS-context pattern. It does not establish all application read routes, external OIDC validation, certificate authority, or Flutter/Field behavior.

# INTEGIN Stage 0 Non-Owner Inspection Read-Isolation Preflight

## Objective

Prove that the applied pilot database returns only canonical inspection records belonging to the active tenant and organization when queried as the non-owner `integin_runtime` role. The proof is read-isolation evidence only; it must not alter any privilege, RLS policy, migration, server startup, certificate, or external-authority path.

## Verified preconditions

| Precondition | Observation | Consequence |
|---|---|---|
| Runtime role | `integin_runtime` is neither superuser nor `BYPASSRLS` | A session assuming this role is suitable for an RLS proof. |
| Read capability | The role has `SELECT` on the Work-Order and `inspection_record` tables needed to traverse the test graph | No privilege grant is required. |
| RLS | `inspection_record` and the supporting Work-Order graph have row security and forced row security enabled | Visibility must be tested through transaction-local tenant and organization settings. |
| Role assumption | `integin_pilot_owner` is a member of `integin_runtime` | A transaction can use `SET LOCAL ROLE integin_runtime` without exposing a role credential. |

## Fixture boundary

The proof will create exactly two self-contained `it-rls-read-*` Work-Order graphs in one tenant and two different organizations. Each graph contains one work order, one scope item, one active assignment, one assignment-scope relation, and one completed/open canonical inspection record. All records are temporary and use only generated IDs.

The owner creates and removes fixtures under the correct context. A separate read-only transaction assumes `integin_runtime`; it sets the first organization context and must observe one target inspection and zero other-organization inspections. It then sets the second organization context and must observe the inverse result. An unset-context query must observe zero records. The proof fails on any unexpected count, privilege error, or cleanup residue.

> The proof establishes isolation for the tested canonical inspection read path. It does not establish UI authorization, real OIDC authentication, certificate authority, or general authorization coverage across every table and route.

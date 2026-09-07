# INTEGIN Local-Provisioning Boundary Freeze v1

> **Status:** Test-only governance guard. It introduces no runtime code path, deployment change, production image, device enrollment flow, database migration, secret, or environment-file mutation.

## Boundary

The local provisioning bridge is a deliberately constrained development and pilot utility. It may activate only with the explicit `INTEGIN_LOCAL_PROVISIONING_ENABLED=true` runtime flag and only when the INTEGIN HTTP listener is loopback-bound. An absent or invalid flag is disabled. The service already stops startup if an enabled local bridge is bound to a non-loopback address.

| Rule | Frozen v1 position |
|---|---|
| Default state | Disabled. Production-facing image configuration must set `INTEGIN_LOCAL_PROVISIONING_ENABLED=false`. |
| Reachability | Loopback only when explicitly enabled; wildcard, LAN, public, and hostname bindings are rejected. |
| Authority | Local provisioning is not a production device enrollment or identity authority. Go/PostgreSQL remains authoritative. |
| Production enrollment | Not implemented. Any future design requires explicit human approval and a separately reviewed authority/revocation model. |
| Mutual exclusion | A production enrollment authority path must never coexist with enabled local provisioning. |

## Guard artifacts

| Artifact | Assurance |
|---|---|
| `contracts/local_provisioning_boundary_v1.json` | Machine-readable default, loopback, approval, and mutual-exclusion position. |
| `contracts/local_provisioning_boundary_v1_test.go` | Fails if the contract permits production enrollment or enables the production default. |
| `cmd/integin-server/local_provisioning_boundary_test.go` | Verifies the executable's absent/invalid flag default and loopback-only address predicate. |

## Explicitly deferred work

Production device enrollment, approval, device ownership transfer, revocation, recovery, operator access, certificate/identity issuance, OIDC coupling, and any production configuration require their own authority design, threat review, migration/recovery plan, and accountable human approval. This boundary artifact must not be treated as approval to implement or enable those workflows.

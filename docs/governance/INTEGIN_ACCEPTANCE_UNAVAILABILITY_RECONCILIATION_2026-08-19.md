# INTEGIN Acceptance Unavailability Reconciliation

**Recorded:** 2026-08-19  
**Decision owner:** INTEGIN engineering owner  
**Workflow:** Delegate and Debate — critical, adversarial evidence burden, owner-only hold  
**Scope:** Protected acceptance availability only.

> No recovery action has been executed by this record. It does not authorize a restart, configuration change, migration, data action, private-material inspection, package-enforcement change, OIDC enablement, or OpenBao operation.

## Observed facts

| Evidence area | Read-only observation |
| --- | --- |
| Protected acceptance probes | `127.0.0.1:8080/healthz` and `/readyz` were unreachable. |
| Listener state | No loopback listener was present on `8080` or isolated-candidate `18080`. |
| Application process | No `integin-server-provision` process was present. |
| Dependencies | The retained acceptance PostgreSQL and RustFS containers were reported running. |
| Recovery prerequisites | The known-good acceptance binary, private environment file, and guarded restore script are present; no duplicate acceptance process exists. |
| Source-only change validation | Focused lifecycle tests, repository-wide Go tests, Go vet, and focused Field static analysis passed before the availability probe. |

## Independent failure-mode lenses

| Lens | Claim | Evidence | Confidence | Limitation |
| --- | --- | --- | --- |
| Operations | The immediate failure condition is an absent acceptance application process, not a listening-but-unready service. | No listener and no `integin-server-provision` process. | High | It does not establish why the process is absent. |
| Dependency safety | There is no read-only indication that PostgreSQL or RustFS must be recreated or reconfigured. | Both retained acceptance dependency containers were running. | Medium | Application startup performs the authoritative readiness check. |
| Recovery safety | The existing recovery script is the least-invasive documented path because it refuses duplicates, checks exact loopback binding internally, starts the known-good binary, waits, probes both endpoints, and stops its child if verification fails. | Script review. | High | The script necessarily reads private environment values in memory to start the known binary. |
| Assurance / red team | A green source test suite cannot prove acceptance availability. Direct health/readiness after recovery is mandatory; any failed recovery must remain stopped and be documented. | The outage occurred despite source-only validation. | High | A successful probe cannot prove broader workflow behavior. |

## Options considered

| Option | Decision |
| --- | --- |
| Start a new candidate or enable package enforcement | Rejected. It does not restore the protected control and violates active gates. |
| Manually reconstruct the acceptance command or edit configuration | Rejected. It bypasses the documented duplicate, binding, and failed-start cleanup controls. |
| Recreate containers, apply migrations, or inspect credentials | Rejected. Read-only evidence does not support any of those actions. |
| Leave acceptance unavailable | Safe but does not restore the protected control required before work can continue. |
| Execute `operations\acceptance\restore-acceptance-server.ps1` once | Recommended only after explicit recovery authorization. It is the documented guarded path. |

## Owner decision

The current critical boundary remains **held**. The only proposed next action is a single invocation of the guarded acceptance restore script. It must be followed immediately by `/healthz` and `/readyz` checks and a confirmation that no candidate listener exists. If it fails, the script’s own cleanup must leave no new acceptance child process running; no retry, manual launch, log disclosure, dependency change, or private-material inspection is authorized by this decision record.

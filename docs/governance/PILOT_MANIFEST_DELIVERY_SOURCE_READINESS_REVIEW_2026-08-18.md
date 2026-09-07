# INTEGIN Pilot Manifest Delivery Source-Readiness Review — 2026-08-18

**Status:** Source-only validation complete. This review does **not** authorize applying migrations, mounting a route, starting a candidate, restarting a runtime, or enabling package enforcement.

## Scope

This review reconciles the documented device-authenticated work-package manifest-delivery stream with its current committed source state after the data-backed policy matrix. It evaluates only source, static analysis, and focused unit/integration tests. No request was issued to a manifest endpoint, no database migration was applied, and no acceptance or pilot runtime was started or modified.

## Reviewed source boundary

| Layer | Reviewed state | Boundary preserved |
| --- | --- | --- |
| Field delivery flow | Commit `6c8dfd0` composes fetch, verify/bind, and cache behavior with a focused delivery test. | The transport is configurable and was not invoked against a live endpoint in this review. |
| Manifest issuer and replay handling | Prior reviewed commits provide signed manifest issuance, durable replay scoping, and contract vectors. | No replay persistence operation was executed. |
| Pilot composition | Commit `77c1277` provides a composition seam; `65b47cc` defines activation controls; `7dccd8d` gates route composition. | The source is fail-closed unless its dedicated pilot gates and dependencies are explicitly supplied. |
| Migration surface | `0006_work_package_assignment_context.sql` and `0007_manifest_proof_replay.sql` are additive schema definitions. | Neither was applied; the isolated pilot database was not altered. |

## Validation evidence

| Validation | Command or method | Result |
| --- | --- | --- |
| Field static analysis | `C:\flutter\bin\cache\dart-sdk\bin\dart.exe analyze` in `field_app` | Passed: `No issues found!` |
| Focused manifest Go tests | `go test ./internal/server ./internal/packagemanifest ./internal/workpackagepg ./internal/domain/sync -count=1` | Passed. |
| Focused manifest Go vet | `go vet ./internal/server ./internal/packagemanifest ./internal/workpackagepg ./internal/domain/sync` | Passed. |
| Runtime isolation | Review method | No candidate started, no manifest request issued, no migration applied, and no persistent enforcement setting changed. |

## Readiness conclusion

The source-level Field-to-server manifest-delivery chain is ready for the **next governance review**, not for automatic runtime activation. The principal untested boundary remains a separately governed isolated-pilot candidate that would require deliberate application of the additive pilot migrations, a dedicated pilot manifest signer, fail-closed route gating, and a rollback/evidence plan. The existing activation runbook remains authoritative: it explicitly withholds authorization for those steps.

> **Current decision:** keep the manifest retrieval route unmounted, package enforcement disabled, OIDC disabled, and OpenBao sealed/unwired. Preserve acceptance at `127.0.0.1:8080` as a protected control.

# External Codebase Review Triage

**Source reviewed:** User-supplied third-party analysis dated 2026-08-15  
**Status:** Triage only. No recommendation in this document authorizes a runtime change.  
**Validation rule:** The analysis is input, not authority. Recommendations are accepted only where they align with current INTEGIN architecture, runtime evidence, and pilot gates.

## Executive conclusion

The external review identified several useful engineering improvements, but it mixes **real developer-experience gaps**, **already-planned production controls**, and **items that should not be rushed into the current local pilot**. Its highest-value contribution is not a new framework or rewrite. It confirms that INTEGIN needs a clearer developer/operator entry point, repeatable commands, an explicit release pipeline, and formal protocol documentation.

The review must not cause a direct change to the healthy acceptance or pilot runtime. The correct order is to first finish the isolated OpenBao rehearsal, then implement source-only developer-experience improvements in the pilot source copy, validate them, and promote them deliberately.

## Recommendation classification

| External recommendation | Validation status | INTEGIN decision | Rationale and sequencing |
|---|---|---|---|
| Root `README.md` | **Verified absent** in the isolated pilot source | **Adopt now, source-only** | A root entry point is a high-value, low-risk improvement. It should describe the authority model, fast setup, test commands, two-runtime model, secret boundary, and documents—not expose credentials or oversimplify production. |
| Makefile / Taskfile / build scripts | **Verified absent** for `Makefile`; actual platform is Windows-first | **Adopt now as a PowerShell task runner, not blindly as Makefile** | A `scripts/integin.ps1` command surface better matches the connected environment. It can wrap Go test/vet/build, direct Flutter test, Python checks, pilot health, and contract vectors. A Makefile may be added later for POSIX CI, but it is not the primary local operator interface. |
| Server logging configuration | **Partially addressed**: observed runtime logs are structured request logs; logger initialization/configuration needs source check | **Adopt later after source verification** | `LOG_LEVEL`, redaction, correlation propagation, and production log policy are valid work. Do not change logging while OpenBao/recovery work is active. |
| `statusWriter` optional HTTP interfaces | **Needs source verification** because the desktop code-inspection connection disconnected | **Adopt if confirmed** | A wrapper should preserve optional `Flusher`, `Hijacker`, `Pusher`, and `ReaderFrom` behavior where middleware needs them. This is a contained Go hardening change with tests, not a current production blocker. |
| Checked database close / shutdown pattern | **Needs source verification** | **Adopt if confirmed** | Graceful shutdown and close-error logging matter, but the server already has readiness/health and must not be edited without a controlled source test. |
| Docker Compose for local development | **Deliberately not present** | **Defer/reframe** | The current two-runtime plan intentionally uses isolated named resources. A pilot Compose file can later reproduce the pilot stack, but it must not collapse acceptance and pilot or embed secrets in YAML. |
| Metrics, tracing, pprof | **Already planned** | **Defer to observability slice** | OpenTelemetry/Prometheus/Loki/Grafana is already the selected P1 platform. Pprof must be disabled by default and loopback/operator protected. |
| Centralized environment configuration | **Needs source verification** | **Adopt later, with strict secret policy** | A typed configuration boundary is good, but it should not read OpenBao/application configuration until the secrets rehearsal is complete. It must preserve existing fail-closed startup checks. |
| TLS configuration | **Intentional local deferral** | **Defer to shared/pilot host** | Loopback-only local services do not need self-signed TLS. TLS is mandatory at the shared/pilot HTTPS edge before non-loopback access. |
| Python advisory-service Dockerfile | **Deferred** | **Defer until advisory deployment slice** | The AI service remains advisory-only. Containerizing it now would add a service before its blocking=false, timeout, resource, observability, and deployment controls are complete. |
| CI/CD pipeline | **Not verified in source due disconnected inspection** | **Adopt soon, source-only** | A minimal pipeline should run Go formatting/vet/tests, Flutter analyze/tests using the direct runner, Python checks, OpenAPI/vector integrity, and secret scanning. Deploy/publish remains out of scope. |
| API documentation | **Partially addressed** | **Finish now in pilot source** | `OPENAPI_V1_DRAFT.yaml` exists in planning and must be placed/validated in the pilot source. It intentionally excludes local provisioning from production API surface. |
| Hardcoded Flutter demonstration fallback | **Partially addressed by controlled provisioning; source behavior needs verification** | **High priority before distribution** | Keep demo capability explicit and opt-in only. A non-debug build must not silently fall back to a demonstration identity after provisioning/session failure. |
| Benchmarks | **Not yet implemented** | **Defer to capacity slice** | Add focused sync/signature benchmarks after current protocol fixtures are stable; define a measured pilot target first. |
| Fuzz tests | **Not yet implemented** | **Adopt after canonical contract baseline** | Fuzz signature, envelope/parser, and evidence-metadata boundaries after the static vectors are committed. Do not fuzz against a live endpoint or real data. |

## Recommended source-only implementation order

The ordered improvements below do not require restarting PostgreSQL, RustFS, pilot Go, acceptance Go, or OpenBao.

| Order | Improvement | Gate before promotion |
|---:|---|---|
| 1 | Root README and Windows-first task runner | Contains no values/secrets; every documented command passes in the pilot source. |
| 2 | Place and validate OpenAPI/vector files in pilot source | Go fixture and direct-Dart Flutter parity test pass. |
| 3 | Minimal CI workflow | Runs the same local tests without deployment, credentials, or external services. |
| 4 | Explicit demo-session compile-time gate | Pilot Flutter tests prove normal builds fail closed rather than silently entering demo state. |
| 5 | HTTP-wrapper/shutdown/config hardening | Exact source claim verified; tests prove no loss of streaming/upgrade behavior and no startup-policy regression. |
| 6 | Fuzz tests and benchmarks | Stable fixture baseline exists; tests use no customer data or live authority keys. |

## Items not to do now

Do not add a generic Docker Compose stack, TLS to local loopback listeners, a service mesh, Kubernetes, external policy engine, blockchain, CI deployment credentials, Python-service container orchestration, dynamic database credentials, or broad configuration refactor during the OpenBao rehearsal. Each either duplicates authority or increases the recovery/change surface before the pilot security gate is complete.

## Next validation action

The desktop source sidecar disconnected during direct inspection of `internal/server/http.go` and `cmd/integin-server/main.go`. Once connected again, validate the following before code changes: `statusWriter` optional-interface behavior, logger initialization, database close/shutdown flow, environment helper duplication, current Flutter demonstration fallback, and presence/absence of CI files. This will turn the medium-confidence source recommendations into precise, testable changes.

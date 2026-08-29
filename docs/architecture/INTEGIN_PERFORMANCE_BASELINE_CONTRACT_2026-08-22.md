# INTEGIN Performance Baseline Contract — 2026-08-22

## Purpose

Measure reproducible baseline cost before considering any optimization. This work changes neither business semantics nor operational configuration and does not invoke external authorities, publish artifacts, or use production customer data.

## Benchmarked workloads

| Workload | Scope | Safety boundary |
|---|---|---|
| Public verifier handler | Valid token-shaped request through the no-store/rate-limit handler with an in-memory verifier stub | No database, raw token persisted, or outbound request |
| Certificate renderer | Fixed-cell PDF/QR render from a frozen synthetic issuance view | No storage write or real token |
| Artifact store adapter | In-memory `storage.Store` write of a prepared PDF result | No RustFS object, local filesystem artifact, or database metadata |
| RLS/certificate integration | Timed separately only against namespaced disposable pilot fixtures | Tenant/org setup, cleanup, and full regression required; not a high-iteration microbenchmark |
| Sync promotion | Existing deterministic unit/integration fixture path only | No Field device, live queue, or customer envelope |

## Measurement rules

Benchmarks use Go's standard testing harness with allocation reporting and a fixed run count. Results include Go/toolchain, OS, CPU, command, and timestamp; they are baseline evidence only, not an SLA.

No benchmark may weaken RLS predicates, bypass identity resolution, suppress lifecycle checks, log raw tokens, reuse production evidence bytes, or mutate persistent pilot fixtures without dependency-ordered cleanup. A candidate optimization must improve a named measured workload and rerun both its behavioral controls and the complete regression suite.

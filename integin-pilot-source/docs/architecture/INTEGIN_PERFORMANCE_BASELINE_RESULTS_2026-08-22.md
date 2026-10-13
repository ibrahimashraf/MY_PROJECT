# INTEGIN Performance Baseline Results — 2026-08-22

## Environment

The baseline was measured on Windows/amd64 with Go `1.26.5` and an Intel Core i5-10310U CPU at 1.70 GHz. It is a local pilot baseline, not a production throughput claim or service-level objective.

## Measurements

| Workload | Command shape | Observed baseline | Interpretation |
|---|---|---:|---|
| Public verifier handler | In-memory verifier stub, token-shaped GET, `-benchmem`, 1-second run | **10,643 ns/op** | Handler/mapping overhead is low in this isolated path; it excludes database/network/shared limiter infrastructure |
| Fixed-cell PDF/QR renderer | Synthetic frozen issuance view, `-benchmem`, 1-second run | **6,028,759 ns/op** | Dominant measured local workload; PDF/QR generation is expected to be materially heavier than JSON verification |
| In-memory artifact-store adapter | Prepared PDF result through `storage.Store`, `-benchmem`, 1-second run | **1,533 ns/op** | Isolates adapter/object-copy cost only; excludes RustFS network and disk latency |
| Certificate RLS lifecycle integration | One pilot PostgreSQL lifecycle proof with cleanup | **0.66 s** total test duration | End-to-end correctness exercise, not a microbenchmark; includes setup, policy, lifecycle, snapshot, and cleanup work |
| Sync durability integration | One pilot PostgreSQL recovery/atomicity proof with cleanup | **0.32 s** total test duration | End-to-end correctness exercise, not a microbenchmark; includes trigger atomicity and cleanup |

## Behavioral controls

The benchmark additions do not bypass authorization, alter tenant/org predicates, change lifecycle states, call an external authority, or use customer data. The certificate and synchronization integration tests passed. Full `go test -count=1 ./...` and `go vet ./...` passed after benchmark additions.

A namespaced trigger from an earlier synchronization fixture was discovered during residue verification. It was inspected, identified as a stale test-only `it_sync_held_rollback_*` trigger/function, removed narrowly, and its count was verified as zero. The current benchmark/integration runs left no namespaced certificate or synchronization row residue.

## Decision

This baseline does **not** justify a domain, RLS, certificate, or synchronization optimization. The renderer is the only measured local path materially more expensive than verifier/storage adapters, but it remains an internal, not-yet-issued artifact capability and has no production demand evidence. The next optimization may begin only if a user-visible or operational bottleneck is observed and then must target one measured path with a behavior-preserving proof.

# INTEGIN Public Manifest Fixture Operator Contract — 2026-08-18

**Status:** Public boundary interface. This contract provides a non-secret way to establish whether the protected disposable manifest fixture can execute under the approved candidate context. It does not expose, inspect, or retain fixture content, private keys, proof bodies, manifest bodies, sync secrets, database credentials, or environment values.

## Purpose

`run-public-manifest-fixture-boundary.ps1` is an operator-only wrapper for the protected fixture. It verifies acceptance control health, starts the approved loopback-only candidate, passes the one approved opaque fixture-path handle to the fixture with all fixture output suppressed, stops the candidate, rechecks acceptance, and emits one compact non-secret JSON record after fixture invocation. The wrapper never opens, interprets, or records the selected file's contents. It fails closed before emitting a success record if preflight, shutdown, or protected acceptance control fails.

| Public field | Meaning | Never contains |
|---|---|---|
| `contract_version` | Version of this public wrapper contract. | Secrets or fixture data. |
| `candidate_run_id` | Opaque per-invocation correlation identifier. Any future case receipt for this invocation must reuse this exact value. | Tenant, device, user, fixture path, or secret. |
| `candidate_started` | Whether the approved candidate bound to loopback successfully. | Process arguments or environment values. |
| `fixture_exit_code` | Integer fixture process exit code. It is emitted only after the fixture process was invoked; a pre-fixture failure emits no success JSON record. | Fixture output, a case payload, or a credential. |
| `output_suppressed` | Confirms fixture stdout/stderr were not read or retained. | Any transformed fixture output. |
| `protected_acceptance_before` | Measured acceptance control result before candidate launch. | Acceptance configuration. |
| `protected_acceptance_after` | Measured acceptance control result after fixture execution and candidate shutdown. | Acceptance configuration. |

## Invocation boundary

The default invocation is:

```powershell
& 'C:\MY PROJECT\operations\pilot\runtime\run-public-manifest-fixture-boundary.ps1' -FixturePath '<approved opaque fixture path>'
```

The wrapper requires acceptance health/readiness `200/200`, no pre-existing candidate listener, the trusted Windows `System32` HTTP client, the approved candidate launcher, the protected fixture directory, and exactly the approved existing fixture-path handle. The launcher must create its process-identity record atomically in the trusted runtime directory. The wrapper requires that record to be fresh for the current launch, rejects reparse points, and rejects broad `Everyone`, `BUILTIN\Users`, or authenticated-user write access before reading its numeric process identifier. It then fails closed unless that exact process owns exactly one listener: `127.0.0.1:18080`. It always stops that captured process in `finally`; a retained candidate is not supported by this public wrapper.

## Interpretation

An exit code is only a **fixture-runnability signal**. It is not independently sufficient evidence that every valid, replay, signature-invalid, package-hash-invalid, expired, authority-mismatch, or key-unknown case passed. A later public case adapter must map named cases to opaque receipt identifiers, expected reason codes, and count deltas before the complete proof matrix may be accepted.

> This contract avoids the unsafe alternatives: guessing protected arguments, opening protected fixture content, or copying secret-bearing output into a log.

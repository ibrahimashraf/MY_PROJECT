# INTEGIN Phase 2 — Core Engines

The Phase 2 domain packages contain deterministic business engines that consume the Phase 1 shared contracts.

| Package | Responsibility |
|---|---|
| `inspection` | Inspection aggregate, valid lifecycle transitions, findings, verdict computation, revision snapshots, separation-of-duties review checks, and tenant-scoped events. |
| `template` | JSON-compatible definitions, versioned registry, recursive asset-tree composition, frozen checklist snapshots, cycle detection, and deterministic expression evaluation. |
| `certificate` | Controlled certificate lifecycle, authenticated approval/signing/issuance, expiry, revocation, supersession, and documented separation exceptions. |
| `completeness` | Deterministic mandatory-field checks and non-blocking advisory suggestions. |
| `acceptance` | Cross-engine dependency-free workflow test from template resolution through inspection close and certificate issue. |

## Validation

From the project root:

```bash
gofmt -l $(find . -name '*.go' -type f)
go test ./...
go vet ./...
```

The Phase 2 suite passes against the mounted project source. The engines remain independent of PostgreSQL, MinIO, network services, and AI services at this stage, as required for deterministic unit and acceptance tests.

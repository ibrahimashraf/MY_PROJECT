# INTEGIN Shared Kernel — Phase 1

This directory contains the dependency-free shared contracts required by the INTEGIN engines.

| Package | Responsibility |
|---|---|
| `types` | Tenant context, environments, inspection states, verdicts, severities, device trust, response types, calibration states, and error codes. |
| `events` | Versioned event envelope, tenant and organization scope, UTC timestamps, event catalog, and representative event payloads. |
| `response` | Dynamic response definitions and specialized measurement payloads. |
| `calibration` | Calibration record schema, validation, expiry calculation, and lifecycle status. |
| `notifications` | Notification categories, mandatory-category rules, channels, preferences, and email delivery states. |
| `featureflags` | Feature keys, inherited/enabled/disabled/expired states, and organization/user/client/project/device scopes. |

The event envelope is constructed through `NewEnvelope`, which validates event identity and tenant scope, normalizes timestamps to UTC, serializes the payload, and returns a value that callers treat as immutable. Business state transitions are intentionally outside this package.

## Validation

Run:

```bash
go test ./...
```

The Phase 1 suite passes with Go 1.22.2 in the sandbox. The connected Windows workspace currently does not have `go` available on PATH; install Go 1.22+ locally before running the same command there.

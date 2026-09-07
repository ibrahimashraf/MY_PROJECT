# INTEGIN Multi-Environment Deployment Kit Evidence — 2026-08-18

**Scope:** Source-only deployment packaging. This record does not represent a Linux, cloud, or production deployment.

## Implemented artifact set

The deployment kit was committed to `integin-pilot-source` as **`a2af81e` — `Add multi-environment deployment kit`**. It adds a source-controlled `deploy/` directory and a root `.gitattributes` contract for LF checkout of POSIX scripts, Compose files, systemd templates, and environment schemas.

| Artifact | Purpose | Safety boundary |
| --- | --- | --- |
| `deploy/compose/integin-infrastructure.compose.yaml` | PostgreSQL 18 and RustFS infrastructure profile with named volumes and loopback-default port binding. | Declares services only; it was parsed but never started. |
| `deploy/env/*.example` and `*.schema` | Secret-free variable names, private-file path placeholders, and disabled-by-default acceptance/candidate guidance. | No credential, private key, database URL, or sync secret is stored. |
| `deploy/systemd/integin-server.service.template` | Linux native Go-server lifecycle template. | Template only; no Linux service was installed. |
| `deploy/scripts/build-targets.sh` | Windows and Linux `amd64` target builds. | Builds were written to a temporary directory and removed. |
| `deploy/scripts/verify-deployment-kit.sh` | Non-deploying static/Compose/build/test validation contract. | It contains no start, migration, or enforcement operation. |

## Validation evidence

| Check | Result |
| --- | --- |
| POSIX shell static syntax | Passed for the build and validation scripts. |
| Deployment contract markers | Passed: RustFS secret-file paths, OIDC disabled default, and systemd hardening markers present. |
| Docker Compose parse | Passed with the installed `docker-compose` v5.4 validator using `config --no-interpolate`; no service was started. |
| Target builds | Passed for Windows `amd64` and Linux `amd64` server/candidate binaries. Temporary outputs were removed. |
| Repository Go tests and vet | Passed: `go test ./... -count=1` and `go vet ./...`. |
| Field static analysis | Passed: direct-Dart `analyze` reported `No issues found!`. |
| Protected runtime isolation | Acceptance `/healthz` and `/readyz` both returned `200`; pilot port `18080` had no listener. |

## Deliberate non-actions

No Linux host was provisioned. No service template, Compose profile, database migration, or environment file was applied. No private configuration was read, copied, printed, or committed. Package enforcement remained disabled, OIDC remained disabled, OpenBao remained sealed/unwired, and the pilot candidate stayed stopped.

## Next governance gate

The next step is **not** deployment. It is preparation and review of an isolated Linux pilot runbook that selects a host, creates new pilot-only volumes and private files, proves backup/restore, verifies health and tenant isolation, and defines rollback to the protected Windows acceptance control.

## References

[1] [RustFS Docker installation and credential-file guidance](https://docs.rustfs.com/en/installation/container)

[2] [RustFS environment-variable reference](https://docs.rustfs.com/en/reference/environment-variables)

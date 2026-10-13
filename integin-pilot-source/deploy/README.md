# INTEGIN Multi-Environment Deployment Kit

This directory contains **source-only deployment artifacts** for a future Windows, Linux, or cloud-hosted INTEGIN environment. It does not launch a service, create a container, migrate a database, transfer a secret, or change the protected Windows acceptance runtime.

## Deployment model

The first portable profile is intentionally hybrid. The INTEGIN Go server is built as a target-native binary. PostgreSQL and RustFS are run through Docker Compose with named volumes. Linux uses `systemd` to supervise the Go binary; Windows keeps a separate lifecycle wrapper. The same environment-variable schema and health/readiness expectations apply to both.

| Artifact | Role |
| --- | --- |
| `compose/integin-infrastructure.compose.yaml` | PostgreSQL, RustFS, and isolated zero-egress renderer-worker services with loopback-only default port binding and named persistent volumes. |
| `renderer-worker/Dockerfile` | Minimal OCI container profile for WeasyPrint 69.0 + Pango + HarfBuzz isolated execution. |
| `renderer-worker/worker.py` | Piped stdin/stdout non-authoritative worker CLI enforcing zero egress and PDF/A-4b output. |
| `renderer-worker/fonts/manifest.sha256` | Pinned checksum manifest for licensed Google Noto Arabic & Latin font pack. |
| `env/compose.env.example` | Non-secret Compose variable names and safe defaults. |
| `env/acceptance.env.schema` | Secret-free application configuration contract. |
| `systemd/integin-server.service.template` | Linux native-server service template. |
| `systemd/integin-renderer-worker.service.template` | Linux systemd service template for isolated containerized renderer worker. |
| `scripts/build-targets.sh` | Reproducible Windows and Linux Go target builds. |
| `scripts/verify-deployment-kit.sh` | Static source, Compose, and target-build validation; never deploys. |

## Safe use boundary

1. Copy example configuration files outside source control and replace placeholders with privately managed values.
2. Validate the kit before use with `scripts/verify-deployment-kit.sh` from the repository root.
3. Start only a **new isolated pilot environment** after a separate review confirms backup/restore, tenant isolation, health, and rollback steps.
4. Do not point this kit at acceptance volumes or reuse acceptance secrets for a pilot. Do not use it to enable package enforcement, OIDC, or OpenBao wiring.

## Linux first-pilot sequence

Build `integin-server` for Linux, create root-owned environment files, bring up only the isolated Compose profile, and start the server through a pilot-specific `systemd` unit. Verify `/healthz`, `/readyz`, tenant boundaries, evidence storage, backup restoration, and a complete rollback before considering any acceptance cutover.

> The RustFS profile uses official `RUSTFS_ACCESS_KEY_FILE` and `RUSTFS_SECRET_KEY_FILE` secret-file support. Secret file paths are variables; values never belong in this repository. [1]

## References

[1] [RustFS Docker installation and credential configuration](https://docs.rustfs.com/en/installation/container)

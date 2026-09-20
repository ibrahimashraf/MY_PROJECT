# INTEGIN Acceptance Runtime Baseline — 2026-08-15

**Purpose:** Control record captured before creating any isolated local-pilot resource.  
**Scope:** Non-secret inventory only. This record does not disclose private environment values, database passwords, signing material, or RustFS access credentials.

## Active acceptance runtime

| Component | Current observed state |
|---|---|
| Go authority | `integin-server-provision.exe`, PID `15368`; `GET http://127.0.0.1:8080/healthz` returned HTTP 200 with `{"status":"ok","service":"integin"}`. |
| PostgreSQL | Container `integin-postgres`; image tag `postgres:18`; running for approximately eight hours; published on host port `5432`; data volume `integin-postgres-data`. |
| RustFS | Container `integin-rustfs`; image tag `rustfs/rustfs:1.0.0-rc.1`; running for approximately seven hours; loopback ports `9000` and `9001`; data/log volumes `integin-rustfs-data` and `integin-rustfs-logs`. |
| Docker network | Existing services use the default `bridge` network. No `integin-pilot-net` existed at capture time. |
| Additional recovery artifact | Existing isolated RustFS restore-drill volume: `integin-rustfs-restore-drill`. |

## Preservation controls

1. Do not stop, rename, recreate, reconfigure, or attach new volumes to `integin-postgres` or `integin-rustfs` during the pilot-clone work.
2. Do not alter the running Go process on `127.0.0.1:8080` or its private environment file.
3. Do not reuse `integin-postgres-data`, `integin-rustfs-data`, `integin-rustfs-logs`, or `integin-rustfs-restore-drill` in any pilot container.
4. Do not use the existing acceptance credentials, tenant, device, authority, signing material, or evidence objects in the pilot clone.
5. Before any later promotion from pilot to acceptance, create a new dated PostgreSQL backup/catalog and RustFS archive/manifest following the established binary-safe/fresh-volume procedures.

## Pilot isolation contract

The pilot uses `C:\INTEGIN-PILOT`, Docker network `integin-pilot-net`, pilot-only named volumes, host loopback ports `15432`, `19000`, and `19001`, and a separate private secrets file `C:\integin-secrets\integin-pilot.env`. This control record is the comparison point for all subsequent pilot changes.

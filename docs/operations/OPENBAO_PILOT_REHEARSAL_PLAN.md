# INTEGIN OpenBao Pilot Rehearsal Plan

**Status:** Completed isolated pilot rehearsal; not yet an application integration or production authorization  
**Scope:** Isolated local pilot only  
**Purpose:** Prove that a self-hosted secrets service can be initialized, sealed, unsealed, access-controlled, restarted, recovered, and kept outside the INTEGIN business-authority path.

## 1. Decision

The rehearsal will add one **pilot-only** OpenBao instance. It will be an operator-controlled secret store; it will not store workflow state, evidence, customer records, audit-event truth, device private keys, offline authority packages, or production credentials.

> **Go + PostgreSQL remains the authority for tenants, organizations, devices, scopes, authority packages, workflow transitions, receipts, and RLS. OpenBao only safeguards operational secret material.**

The pilot uses an explicit non-development configuration with persistent file storage and Shamir sealing. It deliberately does not use OpenBao development mode, in-memory storage, a fixed development root token, auto-unseal, public networking, or pilot/application credentials as root or recovery material.

## 2. Pilot topology

| Concern | Pilot decision | Boundary |
|---|---|---|
| Container | `integin-pilot-openbao` | Does not replace or join acceptance services. |
| Docker network | New `integin-pilot-secrets-net` | No PostgreSQL, RustFS, Go server, or acceptance container is connected during the rehearsal. |
| Host binding | `127.0.0.1:18200` → container `8200` | Not exposed on LAN, public network, or port `8200`. |
| Persistent data | `integin-pilot-openbao-data` → `/openbao/file` | A new named volume, never an acceptance/pilot application volume. |
| Audit output | `integin-pilot-openbao-logs` → `/openbao/logs` | New named volume, separate from INTEGIN application logs. |
| Configuration | `C:\INTEGIN-PILOT\openbao\config` → `/bao/config:ro` | No secret values appear in the configuration directory. |
| Initialization artifact | `C:\INTEGIN-SECRETS\openbao-pilot-recovery.json` | Private pilot-only file; excluded from source control and backups shared outside the operator recovery process. |
| Seal | Shamir, **3 shares / threshold 2** | Two deliberate recovery holders are required; no application process can unseal itself. |

## 3. Required configuration baseline

The configuration is intentionally limited to persistent file storage, one TCP listener inside the loopback-published container, a pilot cluster name, finite token leases, and a file audit device. TLS is disabled only because the entire rehearsal is bound to Windows loopback and contains **disposable** data. Any shared or production deployment requires TLS before an application client is connected.

```hcl
ui           = false
cluster_name = "integin-pilot-openbao"
api_addr     = "http://127.0.0.1:18200"

storage "file" {
  path = "/openbao/file"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = true
}

default_lease_ttl = "1h"
max_lease_ttl     = "4h"

audit "file" "integin-pilot-audit" {
  options {
    file_path = "/openbao/logs/audit.log"
  }
}
```

The container must start with the explicit directory configuration argument `server -config=/bao/config`, not with the default image command or a single-file configuration argument. The corresponding Docker command binds the two pilot-only volumes and read-only configuration directory mount, connects only to `integin-pilot-secrets-net`, and publishes only `127.0.0.1:18200:8200`.

## 4. Initialization and recovery procedure

The operator initializes OpenBao once with three Shamir key shares and a two-share threshold. Initialization output, including the initial root token, must be written directly to the private recovery file rather than copied into chat, source control, shell history, logs, or a generic desktop folder. The root token is bootstrap-only and must not be used by INTEGIN software.

| Stage | Required evidence | Failure behavior |
|---|---|---|
| Container start | Sealed/uninitialized status from local API | No application dependency exists; stop/delete the pilot container and volumes if configuration is incorrect. |
| Initialization | Private recovery file exists; no values displayed | Do not repeat initialization on an initialized volume. Restore from the isolated volume/backup only. |
| Unseal | Two distinct share submissions result in unsealed status | A single share must be insufficient. Failure to provide two shares leaves the service sealed. |
| Bootstrap restriction | Root token is used only to create policy, test token, and audit verification | Revoke bootstrap/root usage after the rehearsal; do not embed it in any environment file. |
| Restart recovery | Container restart returns sealed; two shares restore access to the persisted test secret | If data is missing, preserve the volume for diagnosis and record the failed recovery drill. |

The rehearsal shall use a private local recovery file only for a disposable test. Before any shared or production use, the three shares must be distributed to named recovery holders using an approved secure handoff; no individual, CI system, application process, or single local file may satisfy the operational separation-of-duties policy.

## 5. Least-privilege test

The only secret written during this rehearsal is a value such as `value = "pilot-disposable"` at `secret/data/integin-pilot/rehearsal`. It is not an INTEGIN credential and must be deleted at drill completion.

The root/bootstrap token creates a policy named `integin-pilot-read-rehearsal` that grants `read` capability only to `secret/data/integin-pilot/rehearsal`. A short-lived test token receives only that policy.

| Test | Expected result |
|---|---|
| Read `secret/data/integin-pilot/rehearsal` with test token | Allowed. |
| Write to that path with test token | Denied. |
| Read `secret/data/integin-pilot/other` with test token | Denied. |
| List a parent path with test token | Denied unless explicitly required and granted. |
| Use test token after its short TTL | Denied. |
| Query system/root administration endpoint with test token | Denied. |

## 6. Outage and sealing behavior

OpenBao will be sealed deliberately and then restarted. During this state, the local status endpoint must report sealed and secret reads must fail. No INTEGIN application is configured to depend on OpenBao during this rehearsal, so the Go server, Flutter client, PostgreSQL, RustFS, acceptance control, and pilot control must remain healthy.

The future Go integration rule is as follows: a request may use a secret already obtained and still valid in process memory according to an explicit bounded cache policy, but it must fail closed for **new secret issuance or retrieval** when OpenBao is unavailable. It must never substitute an unaudited hard-coded secret or silently fall back to the acceptance/pilot `.env` files in a shared/production deployment.

## 7. Acceptance gates

| Gate | Pass condition |
|---|---|
| Isolation | New container, network, port, volumes, config directory, recovery file, and test secret do not overlap acceptance or pilot application resources. |
| Persistent encrypted store | A post-restart, re-unsealed OpenBao instance can read the disposable test path from its own volume. |
| Shamir separation | One share cannot unseal; two shares can. |
| Least privilege | Test token can read only its one path; denied cases are demonstrated. |
| Audit | File audit record exists on the OpenBao logs volume and contains no root token, unseal share, secret plaintext, device private key, or INTEGIN evidence bytes. |
| Outage safety | Sealed/restarted OpenBao does not affect health/readiness of pilot `18080` or acceptance `8080`. |
| Cleanup | Test token revoked; disposable path deleted; recovery file retained only until the documented recovery evidence is accepted, then securely removed under the pilot data-handling decision. |

## 8. Execution result — 2026-08-15

The isolated rehearsal completed successfully using `openbao/openbao:2.6.0`. The implementation required two image-specific corrections established through disposable diagnostics: the documented labeled file-audit stanza shown above, and an explicit configuration-directory launch argument (`server -config=/bao/config`). No acceptance container, pilot application container, application volume, application secret file, or INTEGIN workflow authority was changed.

| Gate | Execution result |
|---|---|
| Isolation | Passed. The dedicated OpenBao container, loopback `18200` binding, network, data volume, audit volume, configuration directory, and private recovery file remained separate from acceptance/pilot application resources. |
| Shamir separation | Passed. One share remained sealed; two shares unsealed the service. The final state is initialized, sealed, threshold `2`, shares `3`, and unseal progress `0`. |
| Least privilege | Passed. The short-lived test token could read only its designated disposable path; attempted write, unrelated read, list, and system-mount operations were denied. |
| Audit safety | Passed. The declarative file audit wrote records and the rehearsal checked that the audit log did not contain the root token, submitted unseal shares, or disposable plaintext value. |
| Restart persistence | Passed. Restart returned the service to sealed state; two-share recovery restored access to the disposable test path. |
| Sealed outage | Passed. A deliberate seal blocked secret reads while acceptance `8080` and pilot `18080` health/readiness remained healthy. |
| Cleanup | Passed. The test token was revoked, disposable secret and policy were deleted, temporary policy material was removed, and OpenBao was returned to sealed state. |

The final non-destructive `verify-core` check also passed: Go tests/vet, deterministic vector regeneration, Flutter canonical-signature parity, and concurrent pilot/acceptance readiness all remained healthy. The protected private recovery file was not displayed or copied during the rehearsal. The isolated OpenBao instance remains **sealed** and is not wired into INTEGIN application configuration.

## 9. Explicit deferrals

This rehearsal does not add OpenBao to the Go server, Flutter app, Docker Compose production model, Keycloak, PostgreSQL, RustFS, or backup rotation. It does not test high availability, auto-unseal, HSM/KMS, TLS certificates, cloud KMS, dynamic database credentials, workload identity, or production recovery-share custody. Those decisions occur only after this small, recoverable pilot test passes.

## References

[1] [OpenBao Seal/Unseal](https://openbao.org/docs/concepts/seal/)

[2] [OpenBao Configuration](https://openbao.org/docs/configuration/)

[3] [OpenBao Official Docker Image](https://hub.docker.com/r/openbao/openbao)

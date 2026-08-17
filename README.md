# INTEGIN

**Lifting Equipment Inspection Management System** is evolving into a broader industrial-assurance platform. It records evidence-led field work while preserving a strict principle: **the server remains authoritative and AI is advisory only**.

## Authority model

| Component | Responsibility | Must not become authoritative for |
|---|---|---|
| Go server + PostgreSQL | Tenant/organization enforcement, workflow decisions, trusted devices, offline authorities, receipts, RLS, audit truth | Client-side state or AI recommendations |
| Flutter field app | Offline capture, durable outbox, Ed25519 device signing, evidence preparation, sync | Permanent authorization, certificate issuance, workflow override, tenant trust |
| RustFS | S3-compatible evidence-object storage | Evidence metadata truth, authorization, tenant access decisions |
| Python advisory service | Non-blocking advice only; `blocking=false` | Workflow mutation, risk approval, inspection/certificate decision |
| OpenBao (planned pilot rehearsal) | Operator secret storage, recovery, and access policy | Workflow/evidence/device authority or customer data |

> **Every tenant decision is server-authoritative. Every field mutation is attributable. Every AI result is advisory.**

## Current local environments

Two intentionally separate environments are running on the local machine.

| Environment | Purpose | Go endpoint | Data and credential boundary |
|---|---|---|---|
| Acceptance control | Retains the proven Flutter/Go/PostgreSQL/RustFS acceptance environment | `http://127.0.0.1:8080` | Original database, evidence store, and private acceptance environment |
| Pilot | Isolated hardening environment for contract, secrets, identity, and enrollment work | `http://127.0.0.1:18080` | Separate Docker network, PostgreSQL/RustFS volumes, evidence bucket, source copy, and private pilot environment |

Do not point the acceptance Flutter run at the pilot by accident. Do not copy a pilot secret into acceptance, or an acceptance secret into pilot.

## Prerequisites

The local workflow expects Go, Docker Desktop, PostgreSQL/RustFS pilot resources, Python, and Flutter at `C:\flutter`. The Windows `flutter.bat` wrapper can be silent in this environment, so the task runner invokes Flutter’s bundled Dart tooling through the reviewed pilot test script.

Private values stay in `C:\INTEGIN-SECRETS\`. They must never be placed in source control, Markdown, chat, test fixtures, Docker YAML, logs, screenshots, or generated reports.

## Safe developer commands

Run these from the repository root after saving `scripts\integin.ps1`.

| Command | What it does | Runtime impact |
|---|---|---|
| `powershell -ExecutionPolicy Bypass -File scripts\integin.ps1 help` | Lists safe tasks | None |
| `... -Task go-test` | Runs all Go tests | None |
| `... -Task go-vet` | Runs Go vet | None |
| `... -Task contract-generate` | Regenerates deterministic **test-only** signed vectors | Changes only pilot source fixture files |
| `... -Task flutter-contract` | Runs Flutter parity against the fixed fixture | None |
| `... -Task pilot-health` | Reads pilot health/readiness at `18080` | None |
| `... -Task acceptance-health` | Reads acceptance health/readiness at `8080` | None |
| `... -Task verify-core` | Runs Go tests/vet, contract vector generation, Flutter parity, and both health checks | Does not start, stop, migrate, or reconfigure services |

## Canonical signed-contract baseline

The pilot source contains a first v1 deterministic test-only Ed25519 vector at:

```text
contracts\vectors\v1\signed_transaction_ed25519_v1.json
```

Go generates the fixture; Flutter rebuilds the established v1 canonical string, verifies the valid signature, and rejects the byte-altered canonical bytes. The canonical v1 form uses a `v1|` prefix and normalized UTC timestamps such as `2026-08-15T00:00:00Z`. Any protocol-affecting change must add a new versioned fixture rather than reinterpret old bytes.

## Key documents

| Document | Purpose |
|---|---|
| `LOCAL_RUNTIME_STRENGTHENING_PLAN.md` | Two-runtime isolation, promotion, and rollback plan |
| `PRODUCTION_TRUST_BASELINE_ROADMAP.md` | Ordered path to production trust controls |
| `PRODUCTION_DEVICE_ENROLLMENT_SPEC.md` | Production device-enrollment lifecycle; local provisioning is not production enrollment |
| `IDENTITY_AUTHORIZATION_DESIGN.md` | Keycloak identity-only boundary; Go/PostgreSQL authorization authority |
| `openapi/integin-v1.json` | API contract baseline; local provisioning is deliberately excluded from production surface |
| `CANONICAL_SIGNED_VECTOR_SPEC_V1.md` | Cross-language signing/vector rules |
| `OPENBAO_PILOT_REHEARSAL_PLAN.md` | Future isolated secret-store rehearsal and recovery gates |
| `EXTERNAL_REVIEW_TRIAGE.md` | Independent classification of external codebase recommendations |

## Non-production limitations

The local provisioning bridge exists only for controlled loopback acceptance testing. It must stay disabled in any shared or production deployment. The local pilot OpenBao rehearsal, when resumed, must contain only disposable secrets and no application/customer/device/evidence material. TLS, shared hosting, production enrollment, Keycloak integration, observability, signed audit checkpoints, release policy, and recovery exercises are separately gated roadmap work.

## Contribution rule

Before promoting a pilot-source change to the acceptance source, run the relevant safe task(s), record the result, and preserve a rollback point. Never “fix” tenant isolation, signing, evidence storage, or recovery by disabling a guard or introducing a client-side bypass.

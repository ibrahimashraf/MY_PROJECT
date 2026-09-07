# 🌐 INTEGIN Master Workspace Specification & Single Source of Truth (SSOT)

**Workspace Root:** `C:\MY_PROJECT`  
**Document Version:** 3.0.0 (Unified Canonical Standard)  
**Last Reconciled:** 2026-09-07  
**Authority:** The single permanent reference document for all human engineers and AI agents (Claude, Gemini, Codex, Manus).

---

## 1. 🤖 AI Agent Universal Startup Prompt & Operating Standard

When starting a session with any AI agent, point it to this file (`WORKSPACE.md`) or copy and paste this exact prompt:

```text
You are an expert systems engineer working in the local workspace at `c:\MY_PROJECT`.
The product is INTEGIN (Integrated Inspection & Assurance), with the modular Go engine located in `integin-pilot-source/`.

### Operating Standard (Exactly 2 Files in Workspace Root):
1. `c:\MY_PROJECT\WORKSPACE.md` — The permanent master reference (read this for architecture, rules, live ports, and verification commands).
2. `c:\MY_PROJECT\TRACKER.md` — The active sprint tracker (read this for active tasks; update this with progress, todo checkboxes, and findings).

### Strict Operating Rules:
- Zero New Root Files: Never create new markdown or tracking files in the root directory.
- Single Active Source: Make all code changes in `c:\MY_PROJECT\integin-pilot-source/`. Never modify `integin-source/` (it is a read-only comparison snapshot).
- Mandatory Agent Memory Codex: You MUST obey `.agents/rules/agent-immunity-harness.md` and `integin-pilot-source/AGENTS.md` before writing code. Defend actively against all 40 hazards from line one.
- Token & Cost Optimization (8 Levers): Obey `docs/architecture/AI_TOKEN_AND_COST_OPTIMIZATION_GUIDE.md`. When delegating via Lever 8 (`opencode-delegate`), the orchestrator AI MUST perform symbol recon, isolate the 1–3 target files, identify the verification test command, and generate the structured XML relay brief automatically.
- Multi-Tenant RLS: Every query on domain tables must set session tenant GUCs (`SELECT set_config('integin.tenant_id', ..., true)` with `is_local = true`).
- Opaque Secrets: Never read, print, or log anything inside `c:\MY_PROJECT\private/`.
- Quality Gate: Every change must pass `go test -count=1 ./...`, `go vet ./...`, and `integin-live-matrix`.
- Handoff: When finished, update `c:\MY_PROJECT\TRACKER.md`.

Acknowledge that you have read WORKSPACE.md and report the active phase from TRACKER.md before taking action.
```

---

## 2. 🧭 Workspace Layout & Safety Invariants

```
c:\MY_PROJECT\
│
├── WORKSPACE.md             # 🌐 THE MASTER REFERENCE: Permanent architecture, rules, live ports, roadmap
├── TRACKER.md               # 🚀 THE ACTIVE TRACKER: Single file for sprint plan, todo, progress, & findings
│
├── integin-pilot-source/      # 🚀 ACTIVE SYSTEM ENGINE (The single active source tree)
│   ├── cmd/                 # Executables (integin-server, integin-live-matrix, river-benchmark)
│   ├── internal/            # Domain packages (sync, workorder, certificate, identity, chaos, etc.)
│   ├── pkg/                 # Universal shared packages (domain DIDs, licensing, onboarding, rulesengine)
│   ├── migrations/          # 75+ canonical PostgreSQL SQL migrations
│   ├── field_app/           # 📱 UNIVERSAL CLIENT: Flutter app (Web, Android, iOS, Desktop)
│   ├── ai_service/          # 🧠 ADVISORY LAYER: Python FastAPI service (strictly blocking=false)
│   ├── quiet-signal/        # 📊 OPERATIONS UI: React 19 + TypeScript + Vite web console
│   └── deploy/              # 🐳 DEPLOYMENT: Docker Compose stacks, worker profiles, and systemd templates
│
├── integin-source/            # 🔒 FROZEN BASELINE: Read-only reference snapshot (NEVER MODIFY)
├── integin-pilot-admin/       # 🛡️ PILOT GOVERNANCE: OpenBao security policies and cryptographic manifests
├── docs/                    # 📚 PERMANENT REGISTRY: Governed architecture, ADRs, and evidence
│   ├── architecture/        # ADRs, design specs, [Global Architecture Plan](./docs/architecture/GLOBAL_ARCHITECTURE_PLAN.md), [GitHub Repos](./docs/architecture/GITHUB_REPOSITORIES_REFERENCE.md)
│   └── governance/          # Controlled pilot evidence, audit matrices, and review records
│
├── operations/              # ⚙️ RUNTIME CONTROLS: Launchers, health checks, and Keycloak/RustFS configs
├── tools/                   # 🔧 DEV UTILITIES: Local CLI helpers, architecture graph analyzers, scripts
├── private/                 # 🔒 SECRETS BOUNDARY: Local credentials, test fixtures (OPAQUE — NEVER COMMIT)
├── archive/                 # 📦 ARCHIVES: Retained historical planning logs (>600KB) and superseded docs
├── backups/                 # 💾 DATABASE & RUNTIME BACKUPS: Protected test-run snapshots and rollbacks
└── preservation-archives/   # 📜 EVIDENCE VAULT: Locked, hash-verified pre-commit tarballs
```

### The 5 Inviolable Safety Invariants
1. **Go + PostgreSQL is Sole Authority**: Business logic, tenant isolation, certificate issuance, and cryptographic verification are strictly server-authoritative.
2. **PostgreSQL Multi-Tenant RLS**: Every query on domain tables must execute under `SET LOCAL integin.tenant_id` and `integin.organization_id` with `FORCE ROW LEVEL SECURITY`.
3. **Non-Blocking Advisory AI**: The Python advisory service (`ai_service`) is read-only (`blocking=false`). AI is strictly prohibited from setting verdicts, issuing certificates, or altering workflow state.
4. **Opaque Private Material**: Never inspect, print, log, upload, or commit anything inside `c:\MY_PROJECT\private\`.
5. **Verified Verification Gate**: Every Go change must pass `go test -count=1 ./...` and `go vet ./...` before being considered complete.

---

## 3. ⚡ Live Infrastructure & Runtime State

### Active Container & Port Topology

| Service | Container Name | Local Port | Role & Context | Status |
|---|---|---|---|---|
| **PostgreSQL 16** | `integin-dev-postgres` | `127.0.0.1:15432` | DB: `integin_dev`, User: `integin_runtime` | **UP (Active)** |
| **PgCat Pooler** | `integin-pgcat` | `127.0.0.1:6432` | Multiplexes `integin_dev` with zero GUC leaks | **UP (Active)** |
| **Keycloak IAM** | `integin-pilot-keycloak` | `127.0.0.1:18180` (HTTP)<br>`127.0.0.1:19090` (Health) | Realm: `integin-pilot`, Client: `integin-live-matrix` | **UP (Active)** |
| **RustFS S3** | `integin-pilot-rustfs` | `127.0.0.1:19000` (S3 API)<br>`127.0.0.1:19001` (Console) | Bucket: `integin-pilot-evidence`, Region: `us-east-1` | **UP (Active)** |

> [!NOTE]
> Keycloak port `18180` is the HTTP application and admin console (`/admin/`, token endpoint). Port `19090` is management/metrics only (`/health`, `/metrics`) and returns an empty page in browsers.

### Local Environment Variables Quick Reference

```powershell
$env:INTEGIN_DB_URL             = "postgres://integin_runtime:integin_live_run_2026@127.0.0.1:15432/integin_dev?sslmode=disable"
$env:INTEGIN_TENANT_ID          = "integin-integration-tenant"
$env:INTEGIN_SYNC_SECRET        = "299fe914a8994ce584940439d6fb6be4412c7cef8a904ebf8f158cce3208f622"
$env:INTEGIN_LIVE_FIXTURE_FILE  = "C:\MY_PROJECT\private\integin-secrets\integin-live-fixture.json"
$env:INTEGIN_SERVER_URL         = "http://127.0.0.1:18080"
$env:INTEGIN_HTTP_ADDR          = "127.0.0.1:18080"
$env:INTEGIN_OIDC_ENABLED       = "true"
$env:INTEGIN_OIDC_ALLOW_INSECURE_LOOPBACK = "true"
$env:INTEGIN_OIDC_ISSUER        = "http://127.0.0.1:18180/realms/integin-pilot"
$env:INTEGIN_OIDC_AUDIENCE      = "account"
$env:INTEGIN_OIDC_CLIENT_ID     = "integin-live-matrix"
$env:INTEGIN_OIDC_CLIENT_SECRET = "matrix-secret-2026"
$env:INTEGIN_EVIDENCE_STORE     = "rustfs"
$env:INTEGIN_S3_ENDPOINT        = "http://127.0.0.1:19000"
$env:INTEGIN_S3_BUCKET          = "integin-pilot-evidence"
$env:INTEGIN_S3_REGION          = "us-east-1"
$env:INTEGIN_S3_ACCESS_KEY      = "pilot-zXk5NviVcn8obpN1Mll1VyMJ"
$env:INTEGIN_S3_SECRET_KEY      = "ZFsOePCin0ceJfDK3Bpar3x4diAYJeyD8jR6Q6mftzofrZSwZNQOUzkK8cqCob6asrhr4qqmKP6hwKHKzzSUaYCSzeQ7URCc"
```

---

## 4. 🎯 Master Architecture & 12-Tier Roadmap

### Executive Vision
INTEGIN is an **Integrated Inspection & Assurance Platform** designed for worldwide high-concurrency industrial compliance (lifting equipment, NDT, marine, pressure vessels).

```
                       ┌────────────────────────────────────────────────────────┐
                       │          INTEGIN HYBRID INTELLIGENCE ENGINE            │
                       │     (Standards Federation + Legal Metadata Hub)        │
                       └───────────────────────────┬────────────────────────────┘
                                                   │
                   ┌───────────────────────────────┴────────────────────────────────┐
                   ▼                                                                ▼
    ┌─────────────────────────────────────────────┐                  ┌─────────────────────────────────────────────┐
    │  1. AI Synthesis & Natural Language Search  │                  │ 2. Official Copyright-Safe Metadata Index   │
    │  • Answers complex engineering questions    │                  │ • Official Catalog Abstracts (API/ASME/ISO) │
    │  • 100% Fair Use / Safe Harbor compliant    │                  │ • Exact Lifecycle State (Active/Superseded) │
    └─────────────────────────────────────────────┘                  └─────────────────────────────────────────────┘
                                                   │
                                                   ▼
                       ┌────────────────────────────────────────────────────────┐
                       │       3. Actionable 1-Click Field Execution Bridge     │
                       │ • Injects verified Standard DIDs into Work Orders      │
                       │ • Evaluates Google CEL formula limits on rugged tablets│
                       │ • Binds cryptographic Ed25519 signatures to certs      │
                       └────────────────────────────────────────────────────────┘
```

### The 12-Tier Architecture Topology

| Tier | Name | Key Components & Scope | Active Packages / Status |
|---|---|---|---|
| **L0** | **Root PKI & Licensing** | Asymmetric Ed25519 license keys, offline covenants, feature flags | `pkg/licensing`, `licensehttp`, `licensepg` ✅ |
| **L1** | **Hybrid Standards Engine** | Copyright-safe standards discovery, Google CEL formula AST evaluator | `pkg/standardsync`, `pkg/rulesengine` 🚀 *(In Progress)* |
| **L2** | **Dynamic Jurisdictions** | Multi-country tax, currency, and regulatory matrices (ZATCA, OSHA, CE) | `pkg/jurisdictions`, `identity`, `tenant` 🚀 *(Planned)* |
| **L3** | **Discipline Package Scoping**| Inspector competency gating, training validation, courses | `traininghttp`, `trainingpg`, `equipment` ✅ |
| **L4** | **Operational Hierarchy** | Work orders, assignments, hierarchical branch/area/zone registers | `workorderhttp`, `workorderpg`, `domain/workorder` ✅ |
| **L5** | **Certificate Governance** | Deterministic PDF rendering, 4-eyes review, authority lifecycle | `certificatehttp`, `certificatepg`, `certificaterender` ✅ |
| **L6** | **Competency Matrix** | Dynamic scheduling calendar, qualification tracking | `scheduling`, `identity` ✅ |
| **L7** | **Calibrated Tool Registry** | ISO 17020 Sec 6.2 calibration gating, encrypted evidence metadata | `evidenceapi`, `evidenceexport`, `evidencepg` ✅ |
| **L8** | **Hardware Tablet Attestation**| Apple Secure Enclave & Android StrongBox signing, signed outbox | `packagemanifest`, `workpackageenforcement`, `pkg/onboarding` ✅ |
| **L9** | **Decentralized Asset Passport**| W3C DIDs (`did:integin:...`), equipment quarantine lifecycles | `pkg/domain` ✅ |
| **L10**| **Bitemporal Audit Ledger** | Immutable append-only transaction log, full-text search | `auditloghttp`, `eventstore`, `searchpg` 🚀 *(Planned)* |
| **L11**| **Stateless Edge Trust** | Zero-backend-cost browser WebCrypto QR verification (`verify.integin.com`)| `pkg/verification` 🚀 *(Planned)* |

---

## 5. 🛠️ Engineering Protocol & Standard Verification Loop

### The 4-Step Operating Loop

```
 1. ONBOARD (Read)              2. SCOPE & BUILD (Execute)
 ┌────────────────────────┐     ┌────────────────────────┐
 │ • WORKSPACE.md         │────▶│ • Edit in integin-pilot  │
 │ • TRACKER.md           │     │ • Set tenant RLS GUCs  │
 └────────────────────────┘     └────────────────────────┘
             ▲                               │
             │                               ▼
 4. HANDOFF (Update)            3. VERIFY (Test)
 ┌────────────────────────┐     ┌────────────────────────┐
 │ • Update TRACKER.md    │◀────│ • go test -count=1     │
 │   (tasks, log, gotchas)│     │ • integin-live-matrix    │
 └────────────────────────┘     └────────────────────────┘
```

### Standard Verification Sequence

Execute this exact sequence before claiming any milestone complete:

```powershell
# 1. Navigate to active source
cd c:\MY_PROJECT\integin-pilot-source

# 2. Format & Lint
go vet ./...

# 3. Clean-Cache Unit & Package Tests
go test -count=1 ./...

# 4. Run live end-to-end matrix against local containers
go run ./cmd/integin-live-matrix seed
go run ./cmd/integin-live-matrix exercise
```

### Relay Dispatch Standard (Headless Implementers: OpenCode / Codex / Subagents)

When delegating implementation work via `opencode-delegate` / `relay.mjs` (Lever 8), orchestrator agents across all platforms (Claude, Antigravity, Codex, Qwen, DeepSeek, Kimi, Grok) must structure briefs with block-fenced XML and enforce the 5 mandatory boundaries:

1. **Target Blast Radius**: Pinpoint 1–3 explicit files to touch (`internal/...`, `migrations/...`); strictly forbid touching `integin-source/`, `private/`, or the root directory.
2. **Multi-Tenant RLS & SQL**: Every domain query must set tenant GUCs with `is_local = true` and use strictly `$1, $2` placeholders (no string concatenation).
3. **Portability & Determinism**: Pure Go (`CGO_ENABLED=0`), zero runtime LLM math (Google CEL ASTs only).
4. **Git Sandbox Boundary**: Worker strictly never runs `git commit`, `add`, or `push`. Modifications remain uncommitted in the working tree for orchestrator review.
5. **Expected Evidence (Hazard 9)**: Completion reports must include raw terminal stdout from `go test -race` and `go vet`. No live terminal output = unverified.

Reference the full dispatch skeleton in [`docs/architecture/AI_TOKEN_AND_COST_OPTIMIZATION_GUIDE.md`](./docs/architecture/AI_TOKEN_AND_COST_OPTIMIZATION_GUIDE.md).

### Core Architecture & Coding Rules
1. **Sub-Nano Struct Alignment**: Order mutation structs with descending alignment (8-byte, 4-byte, 2-byte, 1-byte) with explicit padding to prevent L1 cache-line splits. Verify zero heap escapes with `go build -gcflags="-m"`.
2. **Deterministic Outcomes**: Sync endpoints return structured JSON outcomes (`APPLIED`, `DUPLICATE`, `HELD`, `CONFLICT`, `SECURITY_FAILURE`). Expected conflicts and rejections must never crash as naked 500s.
3. **Database Migrations & HOT Optimization**: Sequentially number migrations (`0075_...sql`). Enforce `ENABLE` and `FORCE ROW LEVEL SECURITY`. Append-heavy tables specify `WITH (fillfactor = 85)` for Heap-Only Tuple updates.
4. **Governed Architectural Decisions**:
   - [Big Tech Silicon Valley Adoption Debate & ADR](./docs/architecture/INTEGIN_BIG_TECH_SILICON_VALLEY_ADOPTION_DEBATE_2026-09-07.md)
   - [Global Architecture Plan](./docs/architecture/GLOBAL_ARCHITECTURE_PLAN.md)
   - [User GitHub Repositories Catalog](./docs/architecture/GITHUB_REPOSITORIES_REFERENCE.md)
   - [Strategic Competitor Deep Dive](./docs/architecture/INTEGIN_STRATEGIC_DEEP_DIVE_2026-08-31.md)

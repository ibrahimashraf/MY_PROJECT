# 🌐 Master GitHub Repositories, Developer Ecosystems & Technology Catalog

**Document ID:** REF-2026-09-07-MASTER-ECOSYSTEM-CATALOG  
**Location:** `c:\MY_PROJECT\docs\architecture\GITHUB_REPOSITORIES_REFERENCE.md`  
**Purpose:** Canonical inventory of all 24 authoritative GitHub repositories, developer ecosystems, and technology references cataloged across engineering sessions.  
**Methodology Sequence:** `RE-VISION > REVIEW > REVISE > REVIEW > DEBATE > PLAN`

---

## 📑 Master Repository & Tooling Taxonomy (24 Resources)

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                        INTEGIN TECHNOLOGY & ECOSYSTEM TAXONOMY                         │
├──────────────────────────┬──────────────────────────┬──────────────────────────────────┤
│ 1. Core Language & Engine│ 2. Edge, Mobile & Tablet │ 3. AI Agent Harnesses & Memory   │
│    • go.dev (Golang)     │    • flutter.dev         │    • Graphify-Labs/graphify      │
│    • awesome-go (avelino)│    • pub.dev (cel, drift)│    • affaan-m/ECC                │
│    • awesome-go (uhub)   │    • flutterawesome.com  │    • supermemoryai/supermemory   │
│                          │    • flutterflow.io      │    • tt-a1i/hive & mco-org/mco   │
│                          │    • awesome-dart        │    • archify & simplify-codebase │
├──────────────────────────┼──────────────────────────┼──────────────────────────────────┤
│ 4. Database, RLS & ORM   │ 5. Secrets & Appliance   │ 6. Security & Infrastructure     │
│    • awesome-postgres    │    • Infisical/infisical │    • Awesome-Hacking             │
│    • PostgREST/postgrest │    • coollabsio/coolify  │    • awesome-scalability         │
│    • drizzle-team/drizzle│    • awesome-selfhosted  │                                  │
│    • prisma/orm          │                          │                                  │
└──────────────────────────┴──────────────────────────┴──────────────────────────────────┘
```

---

## 🛠️ Group 1: Core Systems Engineering & Language Platforms

### 1. [go.dev](https://go.dev/)
*   **Domain**: Official Go Programming Language Platform & Standard Library.
*   **Role in INTEGIN**: Master authority for `integin-pilot-source` (`integin-server`, `integin-live-matrix`). Provides zero-allocation compiler optimizations (`-gcflags="-m"`), runtime memory arena profiling, high-performance concurrency channels, native cryptographic primitives (`crypto/ed25519`, `crypto/sha256`), and fuzz testing.
*   **Active Value**: Sprint 2 nanocell runtime engine (`pkg/rulesengine`).

### 2. [awesome-go (avelino)](https://github.com/avelino/awesome-go) & [awesome-go (uhub)](https://github.com/uhub/awesome-go)
*   **Domain**: Curated Go libraries and high-throughput systems components.
*   **Role in INTEGIN**: Source for `jackc/pgx/v5` binary PostgreSQL drivers, `riverqueue/river` transactional outbox queue, and `google/cel-go` expression sandbox.

---

## 📱 Group 2: Edge Workforce, Rugged Tablets & Mobile Clients

### 3. [flutter.dev](https://flutter.dev/)
*   **Domain**: Official Flutter Cross-Platform Framework.
*   **Role in INTEGIN**: Powers `field_app/` across Android (rugged industrial tablets), iOS (iPads), Web, and Windows desktop. Native C/C++ FFI channels allow binding directly to hardware crypto chips.

### 4. [pub.dev](https://pub.dev/)
*   **Domain**: Official Package Repository for Dart and Flutter.
*   **Key Packages for INTEGIN**:
    *   `cel` & `libcel`: Native Dart implementation of Google Common Expression Language. **Critical Finding**: Allows offline mobile tablets in `field_app` to evaluate the exact same ASME/ISO mathematical AST formulas offline as the Go server evaluates in `pkg/rulesengine`.
    *   `drift`: Reactive, type-safe SQLite persistence for offline field package manifests.
    *   `pointycastle` / `cryptography`: Ed25519 asymmetric cryptographic signing on mobile outbox receipts.
    *   `qr_flutter` & `mobile_scanner`: High-density barcode/QR generation and scanning for W3C Asset Passports (`did:integin`).

### 5. [flutterawesome.com](https://flutterawesome.com/)
*   **Domain**: Curated Directory of Flutter UI components, templates, and libraries.
*   **Role in INTEGIN**: Reference library for rugged high-contrast UI widgets, offline inspection form builders, signature capture canvases, photo markup tools, and industrial checklist widgets.

### 6. [flutterflow.io](https://flutterflow.io/)
*   **Domain**: Low-code visual builder for Flutter applications.
*   **Role in INTEGIN**: Architectural reference for dynamic form generation and rapid prototyping of tenant-specific inspection screens without rebuilding the core Flutter binary.

### 7. [awesome-dart](https://github.com/yissachar/awesome-dart)
*   **Domain**: Curated Dart libraries and ecosystem resources.
*   **Role in INTEGIN**: Deep references for offline-first state synchronization, memory optimization, and background sync workers.

---

## 🧠 Group 3: AI Agent Orchestration, Knowledge Graphs & Memory

### 8. [Graphify-Labs/graphify](https://github.com/Graphify-Labs/graphify)
*   **Domain**: Deterministic Codebase & Document Knowledge Graph Visualizer.
*   **Role in INTEGIN**:
    *   Transforms ASTs, SQL schemas, 70 migrations, and documentation into a queryable knowledge graph for multi-agent navigation.
    *   Directly models the **Federated Standards Knowledge Graph** (`pkg/standardsync`), linking standard codes (e.g. ASME B30.5) to equipment DIDs and calculation formulas.

### 9. [affaan-m/ECC](https://github.com/affaan-m/ECC)
*   **Domain**: Everything Claude Code / Multi-Agent Operating Harness.
*   **Role in INTEGIN**: Agent harness providing coordinated sub-agents (planners, security auditors, TDD runners) and persistent session memory across agent invocations. Complements local skills in `.agents/skills/`.

### 10. [supermemoryai/supermemory](https://github.com/supermemoryai/supermemory)
*   **Domain**: Memory & Context Engine for AI, Vector Search & Knowledge Hub.
*   **Role in INTEGIN**: Reference architecture for the **Perplexity-style AI Synthesizer** in `pkg/standardsync/search.go`. Powers local vector abstract indexing and semantic search over engineering standards.

### 11. [tt-a1i/archify](https://github.com/tt-a1i/archify) & [tt-a1i/simplify-codebase](https://github.com/tt-a1i/simplify-codebase)
*   **Domain**: Local architecture visualizer and codebase entropy recovery.
*   **Role in INTEGIN**: Installed skills in `.agents/skills/` used for standalone interactive SVG diagrams and dead-code elimination audits.

### 12. [mco-org/mco](https://github.com/mco-org/mco) & [tt-a1i/hive](https://github.com/tt-a1i/hive)
*   **Domain**: Multi-provider AI coordination and local PTY agent supervisor.
*   **Role in INTEGIN**: Active multi-agent execution skills for structured debates and local CLI dispatch.

### 22. [nextlevelbuilder/ui-ux-pro-max-skill](https://github.com/nextlevelbuilder/ui-ux-pro-max-skill)
*   **Domain**: AI design-intelligence skill (MIT) — deterministic BM25-ranked UI styles (50 active), 192 industry palettes, 74 font pairings, 119 UX/accessibility guidelines, stack-specific rules for 22 frameworks.
*   **Role in INTEGIN**:
    *   Advisory design reference for `quiet-signal/` (React 19 + TS), `field_app/` (Flutter is a supported stack), and `tools/onboarding-wizard/` + lifting-simulator HTML/Canvas surfaces.
    *   Pre-delivery checklist (4.5:1 contrast, visible focus, `prefers-reduced-motion`, resilient text reflow) aligns with bilingual Arabic/Latin certificate surfaces (Sprint 3.9) and sovereign gov-console expectations.
    *   Install via `uipro init --ai universal` (`.agents/skills/`) if trialed; output is advisory only — never commits, never touches the Go engine or migrations.

### 23. [gemini-cli-extensions/conductor](https://github.com/gemini-cli-extensions/conductor)
*   **Domain**: Spec-driven development plugin (Apache-2.0) — Context → Spec & Plan → Implement lifecycle with per-track `spec.md`/`plan.md`, review and git-aware revert.
*   **Role in INTEGIN**: Methodology reference for Sprint 3+ tracks: plan-before-build with human plan approval matches our orchestrator/implementer sandwich (Lever 8). Caution: author notes higher token burn on large projects — keep tracks micro-scoped per Lever 1.

### 24. [addyosmani/agent-skills](https://github.com/addyosmani/agent-skills)
*   **Domain**: 25 production-grade lifecycle skills (MIT) — spec, plan, TDD, code review, security hardening, CI/CD — each with anti-rationalization tables and mandatory verification evidence.
*   **Role in INTEGIN**: Closest external match to our immunity harness: verification-non-negotiable, `code-simplification` mirrors Hazard 14, `constraint-driven-development` mirrors our quality gates. Candidate source for cherry-picking checklist content into `.agents/` (adopt selectively, never wholesale).

### 25. [sickn33/agentic-awesome-skills](https://github.com/sickn33/agentic-awesome-skills)
*   **Domain**: 2,100+ skill catalog with local agent-first control plane (AAS Core) — read-only MCP catalog search, agent-owned selection pinned in `aas-stack.json`, immutable plan preview before writes.
*   **Role in INTEGIN**: Distribution/curation reference only. Caution: full-catalog installs can exhaust agent context / crash-loop watchers — our standing rule (narrow installs into `.agents/skills/`, e.g. `--risk safe,none`) already matches their own guidance. Approval-before-writes model matches our orchestrator review gate.

### 26. [wshobson/agents](https://github.com/wshobson/agents)
*   **Domain**: Multi-harness plugin marketplace (MIT) — 94 plugins / 202 agents / 183 skills / 105 commands from one Markdown source, with OpenCode + Antigravity adapters and a 3-layer `plugin-eval` quality framework.
*   **Role in INTEGIN**: Single richest skill source for our two harnesses (OpenCode relay implementers, Antigravity orchestration). Of interest: orchestrator workflows (16) for debate patterns and `plugin-eval` static scoring as a model for our own skill quality bars. Install per-plugin only, never the marketplace whole.

### 27. [mksglu/context-mode](https://github.com/mksglu/context-mode)
*   **Domain**: Context-window optimizer — MCP sandbox tools (98% claimed raw-output reduction), SQLite FTS5 session memory across compactions, hook-enforced routing on 17 platforms incl. OpenCode.
*   **Role in INTEGIN**: Directly serves Token Guide Levers 1/6/7 (session resets, disk-backed memory, tool-output filters). Trial candidate for orchestrator sessions only — relay implementers stay lean. Caution: hook-enforced routing must never override Hazard 9 (raw `go test` stdout is the only proof) or mask RLS/security evidence.

### 28. [DietrichGebert/ponytail](https://github.com/DietrichGebert/ponytail)
*   **Domain**: Minimalism skill (MIT) — 7-rung reuse ladder (YAGNI → stdlib → native → one-liner → minimum), measured −54% LOC / −20% cost on agentic benchmarks with safety guards intact. OpenCode plugin supported.
*   **Role in INTEGIN**: Enforces Hazard 11 (Rule of Three) and Hazard 14 (net-lines-removed) inside headless implementers where hyper-generative boilerplate (Hazard 2/11) is the top failure mode. `/ponytail-review` over-engineering pass fits our review step before landing relay diffs.

---

## 🗄️ Group 4: Database Engineering, RLS & Object-Relational Models

### 13. [pg-tr/awesome-postgres](https://github.com/pg-tr/awesome-postgres)
*   **Domain**: Exhaustive Directory of PostgreSQL Extensions, Performance, and HA Tools.
*   **Role in INTEGIN**: Architectural reference for:
    *   `pg_stat_statements` & `pg_trgm` query optimization.
    *   PostgreSQL Heap-Only Tuple (HOT) in-place page tuning (`fillfactor = 85`).
    *   `pg_partman` automated 24-hr partition management for `sync_idempotency_cache` and `sync_receipt`.

### 14. [PostgREST/postgrest](https://github.com/PostgREST/postgrest)
*   **Domain**: Standalone Web Server Turning PostgreSQL Schemas into RESTful APIs.
*   **Role in INTEGIN**: The industry standard for **Row-Level Security (RLS) API enforcement**. Serves as our architectural reference model for how session GUCs (`integin.tenant_id`, `integin.organization_id`) are injected from JWT claims and enforced automatically by PostgreSQL policies.

### 15. [drizzle-team/drizzle-orm](https://github.com/drizzle-team/drizzle-orm)
*   **Domain**: Headless, Zero-Overhead TypeScript ORM for SQL Databases.
*   **Role in INTEGIN**: Reference model for strict schema declaration and type safety in the `quiet-signal/` operations console (React 19 + TypeScript).

### 16. [prisma/orm](https://github.com/prisma/orm)
*   **Domain**: Declarative Relational Schema Modeling & Migration Tool.
*   **Role in INTEGIN**: Visual schema modeling and relational data browser reference. (Note: Kept strictly separate from the core Go engine to preserve zero-overhead Go performance).

---

## 🛡️ Group 5: Secrets Management, Sovereign Appliance & Cloud Deployment

### 17. [Infisical/infisical](https://github.com/Infisical/infisical)
*   **Domain**: Open-Source Secret Management, Dynamic PKI & Key Management Service (KMS).
*   **Role in INTEGIN**:
    *   Architectural pattern for **Tier L0 Dynamic Licensing & PKI Key Rotation**.
    *   Provides Go SDK (`infisical-go`) for securely injecting database credentials, Keycloak secrets, and Ed25519 vendor signing keys into `integin-server` in containerized cloud environments.

### 18. [coollabsio/coolify](https://github.com/coollabsio/coolify)
*   **Domain**: Open-Source, Self-Hostable PaaS (Alternative to Heroku/Netlify).
*   **Role in INTEGIN**: **The Golden Model for Deliverable 4.3: Air-Gapped Sovereign Edge Appliance**. Coolify demonstrates how to orchestrate Docker Compose, Traefik ingress, automated health checks, and PostgreSQL on a single sovereign physical server with zero external cloud dependencies.

### 19. [awesome-selfhosted](https://github.com/awesome-selfhosted/awesome-selfhosted)
*   **Domain**: Curated List of Network Services and Web Apps for Local Hosting.
*   **Role in INTEGIN**: Blueprint for air-gapped backups (`pgBackRest`), offline reverse proxies (`Traefik`, `Caddy`), and local S3 object stores (`RustFS`, `MinIO`).

---

## ⚡ Group 6: Scalability & Adversarial Security

### 20. [awesome-scalability](https://github.com/binhnguyennus/awesome-scalability)
*   **Domain**: Large-Scale System Architecture Patterns.
*   **Role in INTEGIN**: Blueprints for cell-based blast radius isolation, connection pooling via PgCat, and distributed idempotency.

### 21. [Awesome-Hacking](https://github.com/Hack-with-Github/Awesome-Hacking)
*   **Domain**: Security Hardening & Penetration Testing Resources.
*   **Role in INTEGIN**: Adversarial test vectors used in `integin-live-matrix` to attack PostgreSQL RLS, test GUC leakage, and prevent timing attacks on cryptographic signatures.

---

*Catalog v2.0.0 — Reconciled and Adopted into INTEGIN Master Architecture.*

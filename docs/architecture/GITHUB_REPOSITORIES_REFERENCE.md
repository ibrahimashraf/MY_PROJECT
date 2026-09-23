# 🌐 Master GitHub Repositories, Developer Ecosystems & Technology Catalog

**Document ID:** REF-2026-09-07-MASTER-ECOSYSTEM-CATALOG  
**Location:** `c:\MY_PROJECT\docs\architecture\GITHUB_REPOSITORIES_REFERENCE.md`  
**Purpose:** Canonical inventory of all 126 authoritative GitHub repositories, developer ecosystems, and technology references cataloged across engineering sessions.  
**Methodology Sequence:** `RE-VISION > REVIEW > REVISE > REVIEW > DEBATE > PLAN`
**Library Rule:** Collect everything; tier on read. `Tier: ADOPT` = trial in our harness. `Tier: WATCH` = track releases/issues only. `Tier: REFERENCE` = ideas shelf. `Take vs Build` = extract the good, create ours — never wholesale-adopt a harness into the authority path.

---

## 📑 Master Repository & Tooling Taxonomy (126 Entries)

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
├──────────────────────────┼──────────────────────────┼──────────────────────────────────┤
│ 7. OpenCode Plugins & Ops│ 8. Memory & Context      │ 9. Agent CLIs & Harnesses        │
│    • oh-my-openagent     │    • codegraph           │    • agent-browser (vercel)      │
│    • cc-switch           │    • rtk-ai/rtk          │    • aidlc-workflows (aws)       │
│    • openwork            │    • okf-agent-memory    │    • ruflo / hermes-agent        │
│    • quota/mem/pruning/  │    • headroom (compress) │    • teamai-cli / deepseek-      │
│      telegram/gemini-auth│                          │      harness / openclaw /        │
│    • learn/awesome/slim  │                          │      career-ops                  │
├──────────────────────────┼──────────────────────────┼──────────────────────────────────┤
│ 10. Skills & Methods     │ 11. Flutter/Media Shelf  │                                  │
│    • superpowers (obra)  │    • codebeg.org         │                                  │
│    • beads (steveyegge)  │    • pro_image_editor /  │                                  │
│    • hacker-laws/spec-kit│      image_compress/tilt │                                  │
│    • taste-skill/design  │    • fluttergems/candies │                                  │
│    • go-skills/conc/     │    • scrapling docs      │                                  │
│      hatchet/langchaingo │    • 2× Medium guides    │                                  │
│    • InsForge + 5×resolved│                          │                                  │
├──────────────────────────┼──────────────────────────┼──────────────────────────────────┤
│ 12. Orchestration & Run  │ 13. Skills/Knowledge/Mem │ 14. API/Codegen/Token Tools      │
│    • OmniRoute / axonhub │    • planning-with-files │    • hey-api / humanlayer        │
│    • AionUi / gentle-ai /│    • plannotator / skill │    • superset / tokscale /       │
│      iPolloWork / ourob. │      store / court /     │      jcodemunch / magic-context  │
│    • atlas / openwolf /  │      scholar / ctf /     │    • html-anything / copilot-    │
│      agentic-stack / arc │      research / wiki /   │      chat (Group 15)             │
│    • agent-teams / deleg.│      mem-mcp / memsearch │                                  │
│    • awesome-orch / OAC /│    • distilly / stop-shit│                                  │
│      council / ralph ×3 /│    • free-models / MTPLX │                                  │
│      J-Space / kungfu /  │                          │                                  │
│      MonkeyCode / hapi / │                          │                                  │
│      costrict / nvim /tap│                          │                                  │
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

---

## 🧩 Group 7: OpenCode Plugins, Quota & Harness Operations

*(affaan-m/ECC already cataloged as #9 — cross-ref, no duplicate entry.)*

### 29. [code-yeongyu/oh-my-openagent](https://github.com/code-yeongyu/oh-my-openagent) — ★69k · TS · pushed 2026-09-22
*   **Domain**: OpenCode multi-agent orchestration suite ("mass ulw" graph engineering).
*   **Role in INTEGIN**: **Tier: WATCH.** Largest OpenCode agent suite; mine its delegation patterns for our relay implementers. Never install whole — per-plugin only.
*   **Take vs Build**: Take task-delegation UX; build our own relay gating (Hazard 9/11 hold).

### 30. [farion1231/cc-switch](https://github.com/farion1231/cc-switch) — ★134k · Rust/Tauri · MIT · pushed 2026-09-22
*   **Domain**: Cross-platform desktop All-in-One for Claude Code / Codex / OpenCode / Grok / Hermes — provider + skills management.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Operator-console UX reference for `quiet-signal/`; provider-switching model vs our Casdoor IdP work (no credential handling borrowed).
*   **Take vs Build**: Take multi-provider switching UX; build auth ourselves (Casdoor, ADR in this folder).

### 31. [jenslys/opencode-gemini-auth](https://github.com/jenslys/opencode-gemini-auth) — ★1.8k · TS · MIT
*   **Domain**: Gemini auth plugin for opencode.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Pattern for provider-auth plugins if a Gemini relay is ever needed. No secrets leave our boundary.
*   **Take vs Build**: Take plugin shape; build against our own IdP.

### 32. [slkiser/opencode-quota](https://github.com/slkiser/opencode-quota) — ★977 · TS · MIT · pushed 2026-09-22
*   **Domain**: Quota & token-usage monitor with zero context-window pollution; multi-provider (OpenCode Go, Cursor, Copilot, Kimi, Chutes, Antigravity…).
*   **Role in INTEGIN**: **Tier: ADOPT (trial).** Directly serves AI Token Guide levers — pair with our shunt-routing rule (AGENTS.md) to measure before/after.
*   **Take vs Build**: Take measurement first; build custom budgets only if its model doesn't fit ours.

### 33. [supermemoryai/opencode-supermemory](https://github.com/supermemoryai/opencode-supermemory) — ★1.6k · TS · pushed 2026-09-17
*   **Domain**: Supermemory plugin for OpenCode (companion to #10 supermemory).
*   **Role in INTEGIN**: **Tier: WATCH.** Evaluates with #10 as one unit; memory must stay disk-backed and tenant-scoped.
*   **Take vs Build**: Take recall UX; build retention policy ourselves.

### 34. [grinev/opencode-telegram-bot](https://github.com/grinev/opencode-telegram-bot) — ★1.2k · TS · MIT · pushed 2026-09-19
*   **Domain**: OpenCode mobile client via Telegram — run/monitor tasks from a phone, execution stays local.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Field-ops notification pattern for rugged-tablet workflows (approve-on-device), not remote code execution.
*   **Take vs Build**: Take approve-by-phone UX; build on our own signed channel.

### 35. [tickernelz/opencode-mem](https://github.com/tickernelz/opencode-mem) — ★1.7k · TS · MIT · pushed 2026-09-21
*   **Domain**: Persistent memory for coding agents on a local vector DB.
*   **Role in INTEGIN**: **Tier: WATCH.** Compare against #10/#33 and Group 8 on the same rubric (recall quality, tenant isolation, compaction behavior).
*   **Take vs Build**: Take eval rubric; build the winner's adapter only.

### 36. [Tarquinen/opencode-dynamic-context-pruning](https://github.com/Tarquinen/opencode-dynamic-context-pruning) — ★4.2k · TS · AGPL-3.0 · pushed 2026-09-21
*   **Domain**: Dynamic conversation-context pruning for OpenCode (token optimization).
*   **Role in INTEGIN**: **Tier: WATCH.** ⚠️ AGPL-3.0 — ideas only, no code in our tree. Compare pruning strategy vs our shunt-routing + lean-ctx modes.
*   **Take vs Build**: Take pruning heuristics as study material; build our own (license firewall).

### 37. [different-ai/openwork](https://github.com/different-ai/openwork) — ★23.7k · TS · pushed 2026-09-22
*   **Domain**: Open-source Claude Cowork alternative, powered by opencode.
*   **Role in INTEGIN**: **Tier: WATCH.** Cowork-pattern reference for orchestrator UX; never a runtime dependency.
*   **Take vs Build**: Take cowork UX patterns; build orchestration on our relay model.

### 38. [vbgate/learn-opencode](https://github.com/vbgate/learn-opencode) — ★1.7k · Shell · pushed 2026-08-25
*   **Domain**: OpenCode Chinese hands-on course (lesson-per-page, intro → real workflows).
*   **Role in INTEGIN**: **Tier: REFERENCE.** Onboarding material for relay-implementer operators; bilingual-ops precedent (cf. our Arabic/Latin surfaces).
*   **Take vs Build**: Take lesson structure; build our own operator runbooks.

### 39. [alvinunreal/oh-my-opencode-slim](https://github.com/alvinunreal/oh-my-opencode-slim) — ★9k · TS · MIT · pushed 2026-09-22
*   **Domain**: Lean multi-agent suite for opencode — mix any models, auto-delegate.
*   **Role in INTEGIN**: **Tier: WATCH.** Lean counterpart to #29; auto-delegate rules vs our orchestrator review gate (advisory-by-default, hard gate only for irreversible risk).
*   **Take vs Build**: Take leanness checklist; build delegation policy ourselves.

### 40. [awesome-opencode/awesome-opencode](https://github.com/awesome-opencode/awesome-opencode) — ★10.3k · CC0-1.0 · pushed 2026-07-03
*   **Domain**: Curated awesome-list of opencode plugins, themes, agents, resources.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Discovery feed for future Group 7 candidates; re-sweep quarterly.
*   **Take vs Build**: Take discovery; build our own vetting (this file).

---

## 🧬 Group 8: Agent Memory, Token Compression & Context

### 41. [colbymchenry/codegraph](https://github.com/colbymchenry/codegraph) — ★71.8k · C · MIT · pushed 2026-09-22
*   **Domain**: Pre-indexed code knowledge graph, auto-sync on change, 100% local, multi-harness (Claude/Codex/Gemini/Cursor/OpenCode/Antigravity/Kiro/Copilot/Hermes).
*   **Role in INTEGIN**: **Tier: WATCH.** Head-to-head vs code-review-graph on our monorepo: index quality, incremental correctness (cf. CRG #1058/#1060/#1062), token cost per review. One graph wins; no dual indexing.
*   **Take vs Build**: Take benchmark harness; build nothing until the bake-off decides.

### 42. [rtk-ai/rtk](https://github.com/rtk-ai/rtk) — ★81.4k · Rust · Apache-2.0 · pushed 2026-09-21
*   **Domain**: CLI proxy cutting LLM token burn 60–90% on common dev commands. Single binary, zero deps.
*   **Role in INTEGIN**: **Tier: ADOPT (trial).** Cheapest token lever on the shelf — proxy relay-implementer shells, measure vs our shunt-routing baseline. Single binary fits air-gap story.
*   **Take vs Build**: Take measurement first; build custom filters only for gaps it leaves.

### 43. [okf-memory/okf-agent-memory](https://github.com/okf-memory/okf-agent-memory) — ★713 · Go · MIT · pushed 2026-09-22
*   **Domain**: Git-native persistent memory for coding agents (Google OKF v0.2, sub-300µs in-memory BM25).
*   **Role in INTEGIN**: **Tier: WATCH.** Git-native memory fits our provenance model (memory as diffable commits); Go matches engine stack. Small-star risk — evaluate code, not hype.
*   **Take vs Build**: Take OKF schema + BM25 approach; build tenant-scoped store ourselves.

### 44. [headroomlabs-ai/headroom](https://github.com/headroomlabs-ai/headroom) — ★73.5k · Python · Apache-2.0 · pushed 2026-09-22
*   **Domain**: Compress tool outputs, logs, files, RAG chunks pre-LLM (~20% fewer coding-agent tokens).
*   **Role in INTEGIN**: **Tier: ADOPT (trial).** Output-side counterpart to rtk (#42, command-side) — stack both, measure compound savings on relay sessions. Never let compression mask RLS/security evidence (Hazard 9).
*   **Take vs Build**: Take compressor; build policy gates around what must never be compressed.

---

## 🤖 Group 9: Agent CLIs, Harnesses & Career Ops

### 45. [vercel-labs/agent-browser](https://github.com/vercel-labs/agent-browser) — ★43k · Rust · Apache-2.0 · pushed 2026-09-22
*   **Domain**: Browser-automation CLI built for AI agents.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Pattern for agent-driven verification of web surfaces (`quiet-signal/`, onboarding wizard) — deterministic Playwright assertions remain the proof; this is the exploration layer only.
*   **Take vs Build**: Take CLI shape; build verification on our own Playwright gates.

### 46. [awslabs/aidlc-workflows](https://github.com/awslabs/aidlc-workflows) — ★4.8k · TS · MIT-0 · pushed 2026-09-22
*   **Domain**: AI-Driven Life Cycle adaptive workflow steering rules for coding agents.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Steering-rules format vs our conductor-style plan/implement tracks (#23) — adopt the format, keep our gates.
*   **Take vs Build**: Take rule format; build track content ourselves.

### 47. [ruvnet/ruflo](https://github.com/ruvnet/ruflo) — ★73k · TS · MIT · pushed 2026-09-22
*   **Domain**: Multi-player agent swarms, autonomous workflow coordination ("original agent harness").
*   **Role in INTEGIN**: **Tier: WATCH.** Swarm-coordination patterns for debate orchestration; ledger/ordering guarantees vs our dispatch model (cf. hive #42 lessons).
*   **Take vs Build**: Take swarm patterns as study; build on hive/mco coordination we already run.

### 48. [NousResearch/hermes-agent](https://github.com/NousResearch/hermes-agent) — ★248k · Python · MIT · pushed 2026-09-22
*   **Domain**: Self-growing agent framework ("the agent that grows with you").
*   **Role in INTEGIN**: **Tier: WATCH.** Largest-star harness on the shelf; growth/learning-loop design vs our immunity-harness determinism. Learning loops must never touch the authority path unreviewed.
*   **Take vs Build**: Take learning-loop design; build with our review gates mandatory.

### 49. [Tencent/teamai-cli](https://github.com/Tencent/teamai-cli) — ★4.9k · TS · license NOASSERTION · pushed 2026-09-22
*   **Domain**: "Make Every Team AI Native" — team-oriented AI CLI.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Team-UX reference only. ⚠️ No license asserted — no code reuse until licensed.
*   **Take vs Build**: Take team-UX ideas; build everything (license firewall).

### 50. [deepseek-ai/deepseek-harness](https://github.com/deepseek-ai/deepseek-harness) — ★233k · TS · MIT · pushed 2026-09-22
*   **Domain**: "Everything is a Plugin" harness from DeepSeek.
*   **Role in INTEGIN**: **Tier: WATCH.** Plugin-architecture reference at the highest star count in class; provider-diversity hedge (DeepSeek-backed models for relay cost tiers).
*   **Take vs Build**: Take plugin boundaries; build provider policy ourselves.

### 51. [openclaw/openclaw](https://github.com/openclaw/openclaw) — ★390k · TS · pushed 2026-09-22
*   **Domain**: "The AI that really does things. Any OS." (formerly OpenClaude — user's "openclaude" term resolves here).
*   **Role in INTEGIN**: **Tier: WATCH.** Any-OS agent runtime relevant to our Windows→Linux portability baseline; sandboxing model vs our BatchMode/non-interactive rules.
*   **Take vs Build**: Take portability notes; build runtime policy ourselves.

### 52. [career-ops-hq/career-ops](https://github.com/career-ops-hq/career-ops) — ★72.4k · JS · MIT · pushed 2026-09-22
*   **Domain**: Open-source AI job search inside coding CLIs — portal scan, A–H report + 1–5 score, CV tailoring, application tracking.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Non-engineering shelf item: structured-report pattern (A–H + global score) is reusable for our own assessment outputs (e.g. debate scorecards).
*   **Take vs Build**: Take report-card pattern; build domain content ourselves.

---

## 🧭 Group 10: Skills, Methods & Engineering Discipline

*(Rule: get the good, create ours. Extract checklists/patterns; build INTEGIN-native versions. Never wholesale-adopt.)*

### 53. [obra/superpowers](https://github.com/obra/superpowers) — ★290k · Shell · MIT · pushed 2026-09-20
*   **Domain**: Agentic skills framework + software-development methodology that works (brainstorm→plan→execute→review spine).
*   **Role in INTEGIN**: **Tier: ADOPT (patterns).** Closest methodology match to our orchestrator sandwich + immunity harness. Mine its brainstorm/spec discipline for Sprint 3+ tracks.
*   **Take vs Build**: Take the spine; build our Hazard/gate content on it.

### 54. [steveyegge/beads](https://github.com/steveyegge/beads) → now [gastownhall/beads](https://github.com/gastownhall/beads) — ★27.4k · Go · MIT · pushed 2026-09-21
*   **Domain**: Agent-native issue tracker ("memory upgrade for your coding agent"); viewers: mgalpert/beads-viewer, zjrosen/perles (BQL kanban TUI).
*   **Role in INTEGIN**: **Tier: WATCH.** Git-backed agent task tracking vs our TRACKER.md + work-order system. Migration only if it proves quieter than files.
*   **Take vs Build**: Take dependency-graph task model; build only if TRACKER.md strains.

### 55. [dwmkerr/hacker-laws](https://github.com/dwmkerr/hacker-laws) — ★27.3k · CC-BY-SA-4.0 · pushed 2026-09-10
*   **Domain**: Laws, theories, principles & patterns for developers (Conway, Hofstadter, Hyrum…).
*   **Role in INTEGIN**: **Tier: REFERENCE.** Vocabulary shelf for debate records and ADRs — cite, don't enforce. (CC-BY-SA: attribute on quote.)
*   **Take vs Build**: Take vocabulary; build our own Hazards/Levers as the enforceable set.

### 56. [github/spec-kit](https://github.com/github/spec-kit) — ★138k · Python · MIT · pushed 2026-09-22
*   **Domain**: Spec-Driven Development toolkit (spec → plan → implement).
*   **Role in INTEGIN**: **Tier: ADOPT (format).** Canonicalize our Sprint-track `spec.md`/`plan.md` shape on it; human plan approval stays mandatory (Lever 8).
*   **Take vs Build**: Take the format; build our approval gates on top.

### 57. [Leonxlnx/taste-skill](https://github.com/Leonxlnx/taste-skill) — ★89k · JS · MIT · pushed 2026-09-20
*   **Domain**: Anti-slop frontend taste skill; synthesized downstream in h3nryprod01/design-taste (+ pbakaus/impeccable) and Hayatelin/taste-skill-zh-CN.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Pairs with our vendored impeccable 4.3.1 for `quiet-signal/` + certificate surfaces. Skill, not authority.
*   **Take vs Build**: Take anti-slop checks; build project-specific taste into DESIGN.md.

### 58. [samber/cc-skills-golang](https://github.com/samber/cc-skills-golang) — ★3.3k · Go · MIT · pushed 2026-09-07
*   **Domain**: Golang agentic skills (language, testing, security, observability) by a top Go author.
*   **Role in INTEGIN**: **Tier: ADOPT (selective).** Direct feed for `integin-pilot-source` relay skills — concurrency vetting, `go vet` discipline, race-detector gates.
*   **Take vs Build**: Take Go-specific checklists; build engine rules ourselves.

### 59. [gamontal/awesome-katas](https://github.com/gamontal/awesome-katas) — ★3.2k · pushed 2026-07-16
*   **Domain**: Curated code-katas list.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Operator-training drills (TDD katas for relay-implementer onboarding). No production bearing.
*   **Take vs Build**: Take drills; build INTEGIN-flavored fixtures (RLS, idempotency) ourselves.

### 60. [sourcegraph/conc](https://github.com/sourcegraph/conc) — ★10.4k · Go · MIT · pushed 2026-09-01
*   **Domain**: Better structured concurrency for Go (goroutine lifecycle, panics, pools).
*   **Role in INTEGIN**: **Tier: ADOPT (evaluate).** Candidate for `pkg/rulesengine` + sync workers — but stdlib-first per ponytail ladder; new dep only if it deletes real code.
*   **Take vs Build**: Take patterns first (stdlib `errgroup`/`semaphore`); build dep case only on measured pain.

### 61. [hatchet-dev/hatchet](https://github.com/hatchet-dev/hatchet) — ★8k · Go · MIT · pushed 2026-09-22
*   **Domain**: Durable workflow engine — background tasks, AI agents, durable execution.
*   **Role in INTEGIN**: **Tier: WATCH.** Durable-execution reference for outbox/sync workers (`riverqueue/river` already in our awesome-go shortlist #2 — compare before any trial).
*   **Take vs Build**: Take durability patterns; build on river unless hatchet wins a bake-off.

### 62. [tmc/langchaingo](https://github.com/tmc/langchaingo) — ★9.7k · Go · MIT · pushed 2026-01-11
*   **Domain**: LangChain-for-Go LLM application framework. ⚠️ Push age 8 months — check maintenance before trial.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Advisory-LLM plumbing patterns only; never in the deterministic authority path (`blocking=false` rule holds).
*   **Take vs Build**: Take provider-abstraction ideas; build minimal clients ourselves.

### 63. [InsForge/InsForge](https://github.com/InsForge/InsForge) — ★13k · TS · Apache-2.0 · pushed 2026-09-19
*   **Domain**: All-in-one open-source backend for agentic coding (DB, auth, storage, compute, hosting, AI gateway).
*   **Role in INTEGIN**: **Tier: REFERENCE.** Sovereign-appliance contrast vs our Coolify model (#18): agent-provisioned backends are the anti-pattern for our authority path — study the shape, keep our control plane.
*   **Take vs Build**: Take agent-skill packaging ideas; build backend ourselves, always.

### 64. [danielmiessler/LifeOS](https://github.com/danielmiessler/LifeOS) — ★19.1k · TS · MIT · pushed 2026-09-04
*   **Domain**: Personal-OS / agent-memory system — everything about you as files (goals, routines, knowledge), loadable by any agent. Runner-up variant: `luneth90/lifeos` (Obsidian).
*   **Role in INTEGIN**: **Tier: REFERENCE.** Personal-knowledge-as-files model for operator machines: files → diffable memory (pairs with git-native #43). Never in the authority path.
*   **Take vs Build**: Take files-as-memory convention; build our tenant-scoped version.

### 65. [JuliusBrussee/caveman](https://github.com/JuliusBrussee/caveman) — ★107.5k · Go · NOASSERTION · pushed 2026-09-23
*   **Domain**: Terse, no-fluff communication/prose skill for agents (~65% output-token cut). Also a Go HTTP proxy compressing model responses.
*   **Role in INTEGIN**: **Tier: REFERENCE.** ⚠️ NOASSERTION — ideas only, no code reuse. Validates our lean-ctx compression rules; the token-cut claim is worth a bake-off vs our shunt-routing baseline.
*   **Take vs Build**: Already built (compression block in AGENTS.md); take benchmark data only.

### 66. [ayghri/i-have-adhd](https://github.com/ayghri/i-have-adhd) — ★50.6k · Python · MIT · pushed 2026-09-19
*   **Domain**: ADHD-friendly productivity skill — 10 output rules (single next action, no wall-of-text, chunked tasks), SessionStart hook.
*   **Role in INTEGIN**: **Tier: ADOPT (patterns).** Operator-UX shelf: low-friction task capture + single-next-action surfacing directly serve relay-operator ergonomics. Hook shape is License-MIT-safe.
*   **Take vs Build**: Take the hook + rules; build our operator runbook around them.

### 67. [timeplus-io/proton](https://github.com/timeplus-io/proton) — ★2.3k · C++ · Apache-2.0 · pushed 2026-09-20
*   **Domain**: Streaming SQL engine (ClickHouse-derived, lightweight) — (ProtonMail privacy suite is the alternative reading; intent resolves to streaming SQL.)
*   **Role in INTEGIN**: **Tier: REFERENCE.** Streaming-SQL pattern vs our sync/idempotency + standards-sync model; read the pipeline model, not the C++ engine.
*   **Take vs Build**: Take streaming-pipeline ideas; build on our existing DB/outbox stack.

---

## 🎨 Group 11: Flutter, Media & Docs Reference Shelf

*(One-liners. Supports `field_app/` image flows and operator learning. No tiers — shelf only.)*

### 68. [codebeg.org](https://codebeg.org/) — awesome livecoding collection.
*   **Role in INTEGIN**: Live-demo technique shelf for operator training and showcase builds.

### 69. [Medium — GenUI sales-analytics in Flutter (BYO model provider)](https://medium.com/@fluttergems/building-a-genui-sales-analytics-app-in-flutter-with-bring-your-own-byo-ai-model-provider-f875098d17b0)
*   **Role in INTEGIN**: BYO-provider GenUI pattern — advisory-AI dashboards over deterministic metrics (mirrors our standardsync synthesizer rule: server computes, model narrates).

### 70. [Medium — −80% Flutter image-upload sizes](https://medium.com/@ankii8946/how-we-reduced-image-upload-sizes-by-80-in-flutter-without-sacrificing-quality-879ca94e82bd)
*   **Role in INTEGIN**: Field-photo upload pipeline: compress on-device before sync (pairs with #71/#72) — critical for rugged-tablet bandwidth.

### 71. [hm21/pro_image_editor](https://github.com/hm21/pro_image_editor)
*   **Role in INTEGIN**: Photo-markup tool for field inspection evidence (cf. flutterawesome #5 widget shelf).

### 72. [fluttercandies/flutter_image_compress](https://github.com/fluttercandies/flutter_image_compress)
*   **Role in INTEGIN**: On-device compression primitive behind #70's pipeline.

### 73. [fluttergems.dev](https://fluttergems.dev/) + [fluttercandies publisher](https://pub.dev/publishers/fluttercandies.com/packages) + [flutter_tilt](https://pub.dev/packages/flutter_tilt)
*   **Role in INTEGIN**: Package discovery feeds (extends pub.dev #4): image/crop/tilt widgets for certificate and inspection surfaces.

### 74. [scrapling — spiders advanced (concurrency control)](https://scrapling.readthedocs.io/en/latest/spiders/advanced.html#concurrency-control) + [scrapling docs](https://scrapling.readthedocs.io/en/latest/)
*   **Role in INTEGIN**: Standards-portal scraping patterns (concurrency control, politeness) for `pkg/standardsync` ingestion — deterministic parsers downstream, never raw HTML into authority.

---

---

## 🐝 Group 12: Agent Orchestration, Harnesses & Runtimes

### 75. [diegosouzapw/OmniRoute](https://github.com/diegosouzapw/OmniRoute) — ★69k · TS · MIT · pushed 2026-09-22
*   **Domain**: Free MIT AI gateway — one endpoint, 359 providers (150+ free), 1200+ models.
*   **Role in INTEGIN**: **Tier: WATCH.** Provider-diversity hedge for relay cost tiers, beside axonhub (#95). Free-tier reliability must be measured, never assumed for authority-adjacent work.
*   **Take vs Build**: Take provider catalog; build routing policy ourselves.

### 76. [iOfficeAI/AionUi](https://github.com/iOfficeAI/AionUi) — ★33k · TS · Apache-2.0 · pushed 2026-09-09
*   **Domain**: Open-source 24/7 Cowork app for OpenClaw, Hermes, Claude Code, Codex, OpenCode + 20 CLIs.
*   **Role in INTEGIN**: **Tier: WATCH.** Cowork-runtime UX vs openwork (#37); 24/7 operator console ideas for field ops.
*   **Take vs Build**: Take always-on console patterns; build on our relay model.

### 77. [Gentleman-Programming/gentle-ai](https://github.com/Gentleman-Programming/gentle-ai) — ★7.1k · Go · MIT · pushed 2026-09-22
*   **Domain**: Configures the agents you already use (Claude Code, Cursor, OpenCode, Codex, Pi…) — setup, not another harness.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Go codebase + config-first philosophy matches our stack; multi-harness dotfile patterns for operator machines.
*   **Take vs Build**: Take config conventions; build our own operator bootstrap.

### 78. [Devin-AXIS/iPolloWork](https://github.com/Devin-AXIS/iPolloWork) — ★6.6k · TS · license NOASSERTION · pushed 2026-09-22
*   **Domain**: Enterprise, local-first Agent Workbench for people + agent teams, multi-engine.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Workbench UX study only. ⚠️ No license asserted — no code reuse.
*   **Take vs Build**: Take workbench layout ideas; build everything (license firewall).

### 79. [Q00/ouroboros](https://github.com/Q00/ouroboros) — ★6.1k · Python · MIT · pushed 2026-09-21
*   **Domain**: Agent OS — interview-gated, staged self-improvement evaluations.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Interview-gated improvement vs our debate-gated changes (same instinct: no ungated self-modification). Evaluation staging worth stealing.
*   **Take vs Build**: Take eval-gate design; build our immunity-harness version.

### 80. [pacifio/atlas](https://github.com/pacifio/atlas) — ★6k · TS · Apache-2.0 · pushed 2026-09-22
*   **Domain**: Source control for agents — many coding agents, one place to track + query changes.
*   **Role in INTEGIN**: **Tier: WATCH.** Multi-agent change provenance vs our git-native discipline; query-across-agents vs our per-relay diffs.
*   **Take vs Build**: Take provenance UX; build on git (already ours).

### 81. [cytostack/openwolf](https://github.com/cytostack/openwolf) — ★2.4k · TS · AGPL-3.0 · pushed 2026-09-15
*   **Domain**: Portable project memory across Claude Code/Codex/OpenCode + token accounting from measured usage.
*   **Role in INTEGIN**: **Tier: WATCH.** ⚠️ AGPL-3.0 — ideas only. Portable `.agent/` memory + measured accounting vs our quota (#32) + memory bake-off (Group 8).
*   **Take vs Build**: Take accounting method; build our own (license firewall).

### 82. [codejunkie99/agentic-stack](https://github.com/codejunkie99/agentic-stack) — ★2.3k · Python · Apache-2.0 · pushed 2026-09-16
*   **Domain**: "One brain, many harnesses" — portable `.agent/` folder (memory + skills + protocols) for Claude/Codex/Cursor.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Portable-agent-folder convention vs our `.agents/` layout — compare, possibly converge naming.
*   **Take vs Build**: Take folder convention; build content ourselves.

### 83. [tractorjuice/arc-kit](https://github.com/tractorjuice/arc-kit) — ★2.2k · JS · license NOASSERTION · pushed 2026-09-03
*   **Domain**: Enterprise Architecture Governance Harness (strategy → architecture → delivery → assurance).
*   **Role in INTEGIN**: **Tier: REFERENCE.** Governance-harness shape vs our debate-record system; assurance-stage ideas for Sprint governance. ⚠️ No license — ideas only.
*   **Take vs Build**: Take stage model; build our debate/ADR flow (already ours).

### 84. [777genius/agent-teams-ai](https://github.com/777genius/agent-teams-ai) — ★2.2k · TS · AGPL-3.0 · pushed 2026-09-22
*   **Domain**: Boss + agent-team chatops (agents message each other, review).
*   **Role in INTEGIN**: **Tier: REFERENCE.** ⚠️ AGPL-3.0 — ideas only. Team-chat metaphor vs our dispatch ledger (cf. hive #42 ordering lessons).
*   **Take vs Build**: Take chatops UX; build on ordered dispatch (already ours).

### 85. [amElnagdy/delegate-skills](https://github.com/amElnagdy/delegate-skills) — ★2.2k · JS · MIT · pushed 2026-09-20
*   **Domain**: Delegate a task to a separate agent CLI, review the diff, land it yourself (mirrors our opencode-delegate skill).
*   **Role in INTEGIN**: **Tier: REFERENCE.** Validates our `.agents/skills/opencode-delegate` pattern; compare review prompts, adopt the better lines.
*   **Take vs Build**: Take review-prompt improvements; build already built — merge deltas only.

### 86. [andyrewlee/awesome-agent-orchestrators](https://github.com/andyrewlee/awesome-agent-orchestrators) — ★2k · CC0-1.0 · pushed 2026-09-21
*   **Domain**: Awesome-list of agent orchestrators.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Discovery feed for future Group 12 candidates; re-sweep quarterly (with #40).
*   **Take vs Build**: Take discovery; build our own vetting (this file).

### 87. [darrenhinde/OpenAgentsControl](https://github.com/darrenhinde/OpenAgentsControl) — ★4.9k · TS · MIT · pushed 2026-09-14
*   **Domain**: Plan-first agent framework with approval-based execution, multi-language.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Plan-first + approvals ≅ our orchestrator sandwich; approval-UX details worth one read.
*   **Take vs Build**: Take approval UX; build policy ourselves.

### 88. [0xNyk/council-of-high-intelligence](https://github.com/0xNyk/council-of-high-intelligence) — ★4.4k · Shell · MIT · pushed 2026-09-22
*   **Domain**: Structured multi-perspective deliberation (full councils, triads) for hard decisions.
*   **Role in INTEGIN**: **Tier: ADOPT (format).** Council/triad deliberation maps directly onto our debate system — adopt triad shape for small decisions, full council for architecture.
*   **Take vs Build**: Take deliberation formats; build verdict/record conventions ourselves.

### 89. [Th0rgal/open-ralph-wiggum](https://github.com/Th0rgal/open-ralph-wiggum) — ★1.9k · TS · MIT · pushed 2026-06-02
*   **Domain**: `ralph "prompt"` loop runner for OpenCode (+ prompt file, status check). ⚠️ Push age ~4 months.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Ralph-loop primitive; staleness check before any use — prefer #90 (actively maintained).
*   **Take vs Build**: Take loop semantics; build on maintained runner.

### 90. [mikeyobrien/ralph-orchestrator](https://github.com/mikeyobrien/ralph-orchestrator) — ★3.2k · Rust · MIT · pushed 2026-09-10
*   **Domain**: Improved Ralph Wiggum autonomous-orchestration implementation (Rust).
*   **Role in INTEGIN**: **Tier: WATCH.** Maintained Ralph runner; loop + status-check design vs our relay supervision.
*   **Take vs Build**: Take loop supervision; build INTEGIN-gated version.

### 91. [michaelshimeles/ralphy](https://github.com/michaelshimeles/ralphy) — ★3k · TS · pushed 2026-02-05
*   **Domain**: Personal Ralph setup (bash loop over Claude/Codex/OpenCode/Cursor). ⚠️ Stale (Feb 2026).
*   **Role in INTEGIN**: **Tier: REFERENCE.** Historical pattern only; do not adopt stale glue.
*   **Take vs Build**: Take nothing executable; build fresh if needed.

### 92. [Tiger3807861189/J-Space-Cognition-Suite](https://github.com/Tiger3807861189/J-Space-Cognition-Suite) — ★3k · Python · Apache-2.0 · pushed 2026-09-14
*   **Domain**: Model-agnostic inference-time control suite (deep reasoning, long-horizon tasks).
*   **Role in INTEGIN**: **Tier: REFERENCE.** Inference-time control vs our prompt/hook-level control; study before any long-horizon relay design.
*   **Take vs Build**: Take control-suite taxonomy; build our own gates.

### 93. [kungfu-systems/kungfu](https://github.com/kungfu-systems/kungfu) — ★4.5k · C++ · Apache-2.0 · pushed 2026-09-19
*   **Domain**: Same work keeps moving across Codex/Claude/OpenCode — handoff continuity, not re-dispatch.
*   **Role in INTEGIN**: **Tier: WATCH.** Handoff-continuity vs our session-resume discipline (cf. hive #41); C++ core irrelevant, protocol is the interest.
*   **Take vs Build**: Take handoff protocol; build on our resume rules.

### 94. [chaitin/MonkeyCode](https://github.com/chaitin/MonkeyCode) — ★4.7k · TS · AGPL-3.0 · pushed 2026-09-22
*   **Domain**: Team AI-coding platform (Chaitin).
*   **Role in INTEGIN**: **Tier: REFERENCE.** ⚠️ AGPL-3.0 — ideas only. Team-platform UX vs our operator console plans.
*   **Take vs Build**: Take UX notes; build ourselves (license firewall).

### 95. [looplj/axonhub](https://github.com/looplj/axonhub) — ★5.3k · Go · Apache-2.0 · pushed 2026-09-22
*   **Domain**: Open-source AI gateway — any SDK, 100+ LLMs, failover, load balancing, cost controls. Go stack.
*   **Role in INTEGIN**: **Tier: ADOPT (evaluate).** Gateway bake-off vs OmniRoute (#75): failover + cost controls + Go (our stack) are strong signals. Measure on relay traffic.
*   **Take vs Build**: Take gateway; build routing/cost policy ourselves.

### 96. [tiann/hapi](https://github.com/tiann/hapi) — ★5.1k · TS · AGPL-3.0 · pushed 2026-09-22
*   **Domain**: Mobile vibe-coding app (Codex/Claude/Pi/OpenCode/Kimi/Grok) — code anywhere.
*   **Role in INTEGIN**: **Tier: REFERENCE.** ⚠️ AGPL-3.0 — ideas only. Mobile-approval UX vs telegram-bot (#34) pattern.
*   **Take vs Build**: Take mobile UX; build on our own channel.

### 97. [zgsm-ai/costrict](https://github.com/zgsm-ai/costrict) — ★4.4k · TS · Apache-2.0 · pushed 2026-09-17
*   **Domain**: Strict enterprise AI coder — quality-first (Agent, CodeReview, Completions).
*   **Role in INTEGIN**: **Tier: REFERENCE.** Strictness-as-feature vs our Hazard gates; review strictness levels worth comparing with integin-review-governance skill.
*   **Take vs Build**: Take strictness rubric; build our governance (already ours).

### 98. [nickjvandyke/opencode.nvim](https://github.com/nickjvandyke/opencode.nvim) — ★3.8k · Lua · MIT · pushed 2026-09-22
*   **Domain**: Neovim ↔ OpenCode in-flow integration.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Editor-integration pattern if Neovim enters operator tooling; no bearing otherwise.
*   **Take vs Build**: Take integration shape; build only on demand.

### 99. [liaohch3/claude-tap](https://github.com/liaohch3/claude-tap) — ★3.2k · Python · MIT · pushed 2026-09-22
*   **Domain**: Intercept + inspect agent API traffic (Claude/Codex/Gemini/Cursor CLIs).
*   **Role in INTEGIN**: **Tier: WATCH.** Traffic inspection for token auditing (pairs with quota #32, tokscale #117) and debugging provider issues. Privacy: inspect own traffic only.
*   **Take vs Build**: Take audit method; build our measurement pipeline around it.

---

## 📚 Group 13: Skills, Knowledge, Memory & Research

### 100. [OthmanAdi/planning-with-files](https://github.com/OthmanAdi/planning-with-files) — ★27k · Shell · MIT · pushed 2026-09-21
*   **Domain**: Persistent file-based planning for agents and long tasks — crash-proof markdown plans (already cited in our continuity docs).
*   **Role in INTEGIN**: **Tier: ADOPT (format).** File-plan durability vs our TRACKER.md + beads watch (#54) — converge on one crash-proof planning file convention.
*   **Take vs Build**: Take durability conventions; build our work-order format on them.

### 101. [backnotprop/plannotator](https://github.com/backnotprop/plannotator) — ★8.9k · TS · Apache-2.0 · pushed 2026-09-21
*   **Domain**: Visually annotate + review agent plans and diffs, share with team, send feedback to the agent.
*   **Role in INTEGIN**: **Tier: WATCH.** Visual plan-review UX for orchestrator approval gates (human plan approval, Lever 8) — reviewer experience matters at scale.
*   **Take vs Build**: Take annotation UX; build approval authority ourselves.

### 102. [anbeime/skill](https://github.com/anbeime/skill) — ★7.1k · Python · pushed 2026-09-22
*   **Domain**: Chinese skill store — curated skill packs (docs, content, dev, ML, automation) + auto-crawler over GitHub skills by category.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Discovery feed for skill candidates (with #40/#86); crawler taxonomy vs our vetting in this file.
*   **Take vs Build**: Take discovery; build our own vetting (this file).

### 103. [internet-court/internet-court-skill](https://github.com/internet-court/internet-court-skill) — ★5.9k · TS · license NOASSERTION · pushed 2026-08-19
*   **Domain**: Trust layer for agent-to-agent commerce — NL mandates, ERC-7710 delegated permissions.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Delegated-permission model worth studying for multi-agent authority boundaries. ⚠️ No license — ideas only.
*   **Take vs Build**: Take mandate/delegation model; build our authority boundaries (already ours).

### 104. [Galaxy-Dawn/claude-scholar](https://github.com/Galaxy-Dawn/claude-scholar) — ★5.6k · Python · MIT · pushed 2026-08-27
*   **Domain**: Semi-automated research assistant (academic + software), multi-CLI support.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Research-workflow pattern vs our integin-primary-research skill — compare source-citation discipline, adopt better lines.
*   **Take vs Build**: Take citation discipline; build our research skill (already ours).

### 105. [ljagiello/ctf-skills](https://github.com/ljagiello/ctf-skills) — ★3.3k · Python · MIT · pushed 2026-09-14
*   **Domain**: Agent skills for CTF (web exploit, pwn, crypto, reversing).
*   **Role in INTEGIN**: **Tier: REFERENCE.** Adversarial skill shapes for `integin-live-matrix` attack vectors (pairs with Awesome-Hacking #21) — lab use only, never against third parties.
*   **Take vs Build**: Take drill shapes; build our matrix vectors (already ours).

### 106. [Weizhena/Deep-Research-skills](https://github.com/Weizhena/Deep-Research-skills) — ★2.2k · Python · MIT · pushed 2026-08-23
*   **Domain**: Structured deep-research skill (Claude/Codex/OpenCode) with human-in-the-loop control.
*   **Role in INTEGIN**: **Tier: REFERENCE.** HITL checkpoints vs our review gates — research-track control points for Sprint methodology.
*   **Take vs Build**: Take checkpoint design; build our track gates (already ours).

### 107. [SamurAIGPT/llm-wiki-agent](https://github.com/SamurAIGPT/llm-wiki-agent) — ★3.6k · Python · MIT · pushed 2026-09-21
*   **Domain**: Self-building, self-maintaining personal knowledge base from dropped-in sources.
*   **Role in INTEGIN**: **Tier: WATCH.** Self-maintaining wiki vs our docs discipline — maintenance-loop design (stale detection, refresh triggers) is the takeaway.
*   **Take vs Build**: Take maintenance loops; build doc-freshness checks ourselves.

### 108. [DeusData/codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp) — ★44k · C · MIT · pushed 2026-09-22
*   **Domain**: High-performance code-intelligence MCP — persistent knowledge graph over codebases.
*   **Role in INTEGIN**: **Tier: WATCH (priority).** Strongest CRG challenger by stars + C performance claim — include in the Group 8 bake-off (index quality, incremental correctness, token cost). One graph wins.
*   **Take vs Build**: Take benchmark duel; build nothing until it decides.

### 109. [zilliztech/memsearch](https://github.com/zilliztech/memsearch) — ★2.6k · Python · MIT · pushed 2026-09-22
*   **Domain**: Persistent unified memory layer for all agents (Milvus/Zilliz-backed).
*   **Role in INTEGIN**: **Tier: WATCH.** Vendor-backed memory vs local-vector options (#35) and Git-native (#43) — same eval rubric, tenant-isolation first.
*   **Take vs Build**: Take eval data point; build the winner's adapter only.

### 110. [titanwings/distilly](https://github.com/titanwings/distilly) — ★25k · Python · MIT · pushed 2026-09-22
*   **Domain**: Distill how experts think into reusable agent/bot Skills (ex-Colleague Skills).
*   **Role in INTEGIN**: **Tier: ADOPT (method).** Distillation method for turning our debate verdicts and operator know-how into `.agents/skills/` content — process over product.
*   **Take vs Build**: Take distillation method; build our skill content (already ours).

### 111. [lennney/stop-that-shit](https://github.com/lennney/stop-that-shit) — ★2.2k · JS · MIT · pushed 2026-09-21
*   **Domain**: Multi-platform Hook + Skill guard vs unprompted hashes/checksums/scope-creep in Codex/GPT agents.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Hook-guard pattern vs our scope discipline (Hazard: scope-creep) — guard-shape study for orchestrator hooks.
*   **Take vs Build**: Take guard shapes; build our hook policy (already ours).

### 112. [vava-nessa/free-coding-models](https://github.com/vava-nessa/free-coding-models) — ★2.8k · HTML · license NOASSERTION · pushed 2026-09-21
*   **Domain**: Find, benchmark, install 170+ free coding LLMs across 15+ providers in-CLI, real time.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Free-tier model discovery for relay cost tiers (with #75/#95 gateways). ⚠️ No license — data only, no code.
*   **Take vs Build**: Take model shortlists; build benchmark/choice ourselves.

### 113. [youssofal/MTPLX](https://github.com/youssofal/MTPLX) — ★2.4k · Python · Apache-2.0 · pushed 2026-09-19
*   **Domain**: Fast local Qwen runner on Mac (125 tok/s, OpenCode-ready).
*   **Role in INTEGIN**: **Tier: REFERENCE.** Local-runner precedent for air-gap evaluation; Mac-only today — note for Linux-runtime portability work, not a dependency.
*   **Take vs Build**: Take local-eval pattern; build our own runner choice per platform.

---

## 🔧 Group 14: APIs, Codegen, Token Tooling & Mobile

### 114. [hey-api/hey-api](https://github.com/hey-api/hey-api) — ★5.4k · TS · MIT · pushed 2026-08-24
*   **Domain**: OpenAPI specs → production SDKs, validators, mocks (20+ plugins).
*   **Role in INTEGIN**: **Tier: ADOPT (evaluate).** SDK generation for our API surfaces (console + field sync) — validators/mocks double as contract tests. Compare vs hand-rolled clients on churn cost.
*   **Take vs Build**: Take generation; build API design ourselves (spec first).

### 115. [humanlayer/humanlayer](https://github.com/humanlayer/humanlayer) — ★11.6k · TS · license NOASSERTION · pushed 2026-06-19
*   **Domain**: Human-in-the-loop for agents on hard problems in complex codebases. ⚠️ Push age 3 months + no license asserted.
*   **Role in INTEGIN**: **Tier: REFERENCE.** HITL-contact patterns vs our approval gates; verify maintenance + license before any deeper look.
*   **Take vs Build**: Take contact-pattern ideas; build our gates (already ours).

### 116. [superset-sh/superset](https://github.com/superset-sh/superset) — ★14.5k · TS · license NOASSERTION · pushed 2026-09-22
*   **Domain**: Agentic IDE orchestrating 100+ coding agents in parallel, bring-your-own agents. (Not Apache Superset.)
*   **Role in INTEGIN**: **Tier: WATCH.** Parallel-fleet orchestration at 100+ scale vs our relay model — scheduling/observation design worth studying. ⚠️ No license — ideas only.
*   **Take vs Build**: Take fleet-observation design; build our relay supervision.

### 117. [junhoyeo/tokscale](https://github.com/junhoyeo/tokscale) — ★5.5k · Rust · MIT · pushed 2026-09-22
*   **Domain**: Terminal token-usage tracker across agents + global leaderboard.
*   **Role in INTEGIN**: **Tier: ADOPT (trial).** Measurement trio with quota (#32) and claude-tap (#99): per-relay token accounting → feeds Token Guide budgets. Leaderboard is fun; accounting is the point.
*   **Take vs Build**: Take accounting; build budgets ourselves.

### 118. [jgravelle/jcodemunch-mcp](https://github.com/jgravelle/jcodemunch-mcp) — ★2.7k · Python · license NOASSERTION · pushed 2026-09-22
*   **Domain**: Symbol-level GitHub code-exploration MCP (claims 95%+ token cuts on exploration).
*   **Role in INTEGIN**: **Tier: WATCH.** Symbol-level exploration vs CRG graph queries — bake-off candidate on exploration-token cost. ⚠️ No license + big claim — verify before trust.
*   **Take vs Build**: Take measurement duel; build nothing until verified.

### 119. [cortexkit/magic-context](https://github.com/cortexkit/magic-context) — ★2.2k · TS · MIT · pushed 2026-09-22
*   **Domain**: "Unbounded context, self-managing memory — one session for life" (hippocampus for coding agents).
*   **Role in INTEGIN**: **Tier: REFERENCE.** Lifetime-session vision vs our session-reset discipline (Token Guide Lever 1) — opposing thesis worth reading before dismissing.
*   **Take vs Build**: Take the counter-thesis; build our reset discipline (already ours) unless evidence flips it.

### 120. [nexu-io/html-anything](https://github.com/nexu-io/html-anything) — ★8.9k · HTML · Apache-2.0 · pushed 2026-09-15
*   **Domain**: Agentic HTML editor — local agent writes the HTML, you ship it (75 skills × 9 surfaces).
*   **Role in INTEGIN**: **Tier: REFERENCE.** Agent-built HTML surfaces vs our impeccable-governed workflow — skill-packaging model (75 skills) vs our curated `.agents/skills/` vetting.
*   **Take vs Build**: Take skill-packaging scale notes; build our vetted set (this file).

---

## 📎 Group 15: Late Additions (by theme: OpenCode plugins)

### 121. [ltmoerdani/opencode-copilot-chat](https://github.com/ltmoerdani/opencode-copilot-chat) — ★189 · TS · MIT · pushed 2026-09-11
*   **Domain**: 30+ AI models (DeepSeek, Kimi, GLM, Claude, GPT, Gemini, Grok) inside GitHub Copilot Chat, free via BYOK.
*   **Role in INTEGIN**: **Tier: REFERENCE.** BYOK model-multiplexing inside Copilot Chat vs our gateway bake-off (#75 vs #95) — small-star, single-maintainer risk; ideas only until proven.
*   **Take vs Build**: Take provider-list breadth; build routing/policy ourselves.

---

## 🧠 Group 16: Awesome-OpenCode Deep Picks & Resolved Terms

*(From the awesome-opencode sweep + resolved `~unverified` terms. All real repos, metadata verified via GitHub API on 2026-09-23.)*

### 122. [smc2315/harness-memory](https://github.com/smc2315/harness-memory) — ★19 · TS · MIT · pushed 2026-04-09
*   **Domain**: Evidence-gated memory — auto-captures interaction evidence, materializes via multi-gate pipeline with human review; 4-layer activation; replaces CLAUDE.md (−73% tokens claimed); local-first sql.js WASM, zero cloud.
*   **Role in INTEGIN**: **Tier: WATCH.** Evidence-gating at write time matches our immunity harness (nothing unverified enters memory). Tiny star count + older push = verify before trial.
*   **Take vs Build**: Take evidence-gate concept; build on our git-native memory (#43) + verdict records.

### 123. [joshuadavidthomas/opencode-handoff](https://github.com/joshuadavidthomas/opencode-handoff) — ★172 · TS · MIT · pushed 2026-08-26
*   **Domain**: Generates a focused handoff prompt (goal/state/next, decisions, open threads) to continue in a fresh session.
*   **Role in INTEGIN**: **Tier: ADOPT (pattern).** Fits Token Guide Lever 1 (fresh sessions) + our session-resume discipline; minimal, no daemon/memory DB. Small-star but MIT + tiny surface = low risk.
*   **Take vs Build**: Take the handoff schema; build our work-order handoff (already ours) on it.

### 124. [agostinilabsrl/opencode-telemetry](https://github.com/agostinilabsrl/opencode-telemetry) — ★6 · TS · MIT · pushed 2026-06-04
*   **Domain**: Passive per-session token audit — local SQLite (tokens, tool calls, skills, per-turn cost), `octm` CLI, no network. (Alternatives: eserete/opencode-token-tracker, Howardzhangdqs/opencode-throughput, IgorWarzocha/Opencode-Context-Analysis-Plugin, slkiser/opencode-quota #32.)
*   **Role in INTEGIN**: **Tier: WATCH.** Measurement trio with quota (#32) + tokscale (#117); tiny star count — evaluate the SQLite schema, not the project.
*   **Take vs Build**: Take audit schema; build around our quota/tokscale pipeline.

### 125. [h3nryprod01/design-taste](https://github.com/h3nryprod01/design-taste) — ★58 · JS · MIT · pushed 2026-09-22
*   **Domain**: Frontend taste skill (anti-slop) — synthesized downstream of #57 taste-skill; pairs with h3nryprod01/design-taste + pbakaus/impeccable.
*   **Role in INTEGIN**: **Tier: REFERENCE.** Design-slop checklists for `quiet-signal/` surfaces; same niche as our vendored impeccable 4.3.1 — read the deltas, don't stack three overlapping skills.
*   **Take vs Build**: Take delta checks; build into our impeccable workflow (already ours).

### 126. [pbakaus/impeccable](https://github.com/pbakaus/impeccable) — ★70.2k · JS · Apache-2.0 · pushed 2026-09-22
*   **Domain**: UI/UX review skill (61-check detector / generator) — upstream repo of our vendored impeccable 4.3.1 skill.
*   **Role in INTEGIN**: **Tier: ADOPT (in use).** Canonical source for the impeccable skill already vendored in `.agents/skills/impeccable`; track upstream releases here (current = v4.3.1).
*   **Take vs Build**: Take as-is; it is the build. Upstream diff-tracking point.

---

## 🧭 Group 17: Life-Domain Map — Every Catalog Entry by Domain

*(One entry can appear in many domains. This is the full life spectrum — not a curated subset. Gaps marked `— no entry yet; add when swept.` Entry numbers = catalog numbers.)*

### 💻 Coding & Engineering
*   go.dev #1 · awesome-go ×2 #2 · flutter #3 · pub.dev #4 · awesome-dart #7 · samber/cc-skills-golang #58 · conc #60 · hatchet #61 · langchaingo #62 · awesome-postgres #13 · postgrest #14 · drizzle #15 · prisma #16 · codebase-memory-mcp #108

### 🧠 Brainstorming & Ideation
*   superpowers #53 (brainstorm→plan spine) · council-of-high-intelligence #88 (triad/full deliberation) · spec-kit #56 · planning-with-files #100 · MTPLX #113 (local eval → rapid iteration) · h3nryprod01/design-taste #125

### 🎯 Problem Solving & Debugging
*   hacker-laws #55 · awesome-scalability #20 · Awesome-Hacking #21 · ctf-skills #105 · codebase-memory-mcp #108 · deepseek-harness #50

### 🏷️ Branding & Design
*   ui-ux-pro-max-skill #22 · taste-skill #57 · design-taste #125 · impeccable #126 · flutterawesome.com #5 · pro_image_editor #71 · html-anything #120

### 📚 Studying & Learning
*   learn-opencode #38 · awesome-katas #59 · claude-scholar #104 · Deep-Research-skills #106 · llm-wiki-agent #107 · LifeOS #64 · I-have-Adhd #66

### 👩‍🏫 Teaching & Training
*   awesome-katas #59 · plannotator #101 · claude-scholar #104 · learn-opencode #38 · superpowers #53 · distilly #110

### 🔍 Analyzing & Decision-Making
*   hacker-laws #55 · council-of-high-intelligence #88 · arc-kit #83 · OpenAgentsControl #87 · MTPLX #113 (eval-driven choices)

### 💰 Financing & Token Budgets
*   quota #32 · claude-tap #99 · tokscale #117 · opencode-telemetry #124 · free-coding-models #112 · rtk #42 · headroom #44

### 💸 Budgeting & Cost Control
*   quota #32 (per-provider budgets) · tokscale #117 (spend ledger) · OmniRoute #75 (cost controls) · axonhub #95 (cost controls) · rtk #42 (60–90% token cut) · headroom #44 (~20% cut) · free-coding-models #112 (free-tier selection)
*   — personal/family/business budgeting: no entry yet; add when swept.

### 👔 Leadership & Management
*   arc-kit #83 (governance harness) · council #88 (decision deliberation) · teamai-cli #49 (team-native ops) · agent-teams-ai #84 (boss+team chatops) · OpenAgentsControl #87 (approval-based delegation) · hacker-laws #55 (management laws: Brooks/Conway)
*   — human-leadership training (1-on-1s, org design): no entry yet; add when swept.

### ⏱️ Productivity & Attention
*   i-have-adhd #66 (single next action) · LifeOS #64 (routines as files) · planning-with-files #100 (crash-proof plans) · beads #54 (task tracker) · opencode-handoff #123 (fresh-session resume)

### 🧬 Researching & Evidence
*   claude-scholar #104 · Deep-Research-skills #106 · codebase-memory-mcp #108 · memsearch #109 · harness-memory #122 · integin-primary-research (our skill)

### 📋 Organization & Planning
*   beads #54 · atlas #80 · planning-with-files #100 · LifeOS #64 · I-have-Adhd #66 · career-ops #52 (application structure)

### 📣 Marketing & Content
*   html-anything #120 · distilly #110 (content skills) · scrapling docs #74 (content portals) · anbeime/skill #102 · agentic-awesome-skills #25

### 🔄 Communication & Writing
*   caveman #65 (terse output) · i-have-adhd #66 (chunked messaging) · stop-that-shit #111 (scope guard) · context-mode #27 (focused reads) · opencode-handoff #123 (state handoff)

### 🔐 Security & Privacy
*   Awesome-Hacking #21 · Infisical #17 · ctf-skills #105 · supermemory #10 (tenant isolation)

### ☁️ Orchestration & Automation
*   ECC #9 · mco #12 · hive #12 · learn-opencode #38 · delegate-skills #85 · OpenAgentsControl #87 · council #88 · kungfu #93 · openwolf #81 · oh-my-openagent #29 · openwork #37 · AionUi #76

### 🧑‍💼 Entrepreneurship & Business
*   InsForge #63 (all-in-one backend) · coolify #18 (sovereign PaaS) · career-ops #52 (ops discipline) · distilly #110 (skill → product) · OmniRoute #75 + axonhub #95 (gateway infra)
*   — sales/negotiation/pricing: no entry yet; add when swept.

### 🚀 Career & Growth
*   career-ops #52 (A–H reports, scores) · claude-scholar #104 (skill compounding) · llm-wiki-agent #107 (knowledge compounding) · awesome-katas #59 (skill drills)

### 🎨 Creativity & Media
*   html-anything #120 · pro_image_editor #71 · flutter_image_compress #72 · fluttergems #73 · codebeg.org #68 · Medium GenUI #69 · impeccable #126 · taste-skill #57

### 🤝 Networking & Relationships
*   career-ops #52 (application + portfolio ops) · opencode-telegram-bot #34 (mobile reach)
*   — relationship/CRM/social capital: no entry yet; add when swept.

### 🏠 Home & Life Admin
*   LifeOS #64 (goals/routines/knowledge as files) · i-have-adhd #66 (low-friction capture) · llm-wiki-agent #107 (personal knowledge base)
*   — budgeting/family ops/health: no entry yet; add when swept.

### 🧘 Health & Wellness
*   — no entry yet; add when swept (fitness, sleep, focus, mental health).

*Catalog v5.1.0 — 2026-09-23. Resolved 4× ~unverified (#64–67); added Group 16 (awesome-opencode deep picks #122–126) and full-spectrum Group 17 (Life-Domain Map, 23 domains incl. brainstorming/branding/problem-solving/budgeting/leadership). Gaps flagged in-G17 for future sweeps. Reconciled into INTEGIN Master Architecture.*

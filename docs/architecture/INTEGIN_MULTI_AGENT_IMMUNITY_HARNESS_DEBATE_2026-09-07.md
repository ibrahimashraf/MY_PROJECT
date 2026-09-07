# ⚖️ INTEGIN Multi-Agent Immunity & Governance Harness Debate

**Document ID:** `DEBATE-AGENT-HARNESS-2026-09-07`  
**Date:** 2026-09-07  
**Status:** CLOSED & RATIFIED  
**Governing Standard:** [`integin-review-governance`](../../.agents/skills/integin-review-governance/SKILL.md) & [`integin-evidence-guard`](../../.agents/skills/integin-evidence-guard/SKILL.md)  
**Topic:** Formal Adversarial Examination of the Agent Immunity Framework: Data Quality, Complexity ("Code Not Written"), Scalability, Hallucinations, Bias, Trust, Transparency, Stale RAG, Inaccuracy, Security, Reasoning Limits, High Token Cost, Context Limits, and Non-Determinism.

---

## 1. Decision Frame

### 1.1 Objective
To establish an unbreachable, mechanical engineering harness that insulates INTEGIN from the failure modes of autonomous AI agents (Claude, Gemini, Codex, Manus, subagents, and human contributors) across life-critical industrial inspection, load calculations, and cryptographic governance.

### 1.2 Comparison Point
*   **Status Quo (Loose Agent Guidance):** System prompts asking models to "be careful," "write clean code," "don't hallucinate," and "test your work."
*   **The Proposed Standard (The Mechanical Immunity Harness):** Mechanical sandboxes, strict compiler and RLS firewalls, deterministic AST calculation nanocells (CEL), 2-file root SSOT, zero-allocation memory budgets, and the Rule of Three anti-bloat protocol.

### 1.3 The 4 Adversarial Review Lenses
1.  **Lens A: The Industrial Cynic (Zero-Trust Reliability & Anti-Hallucination)**
2.  **Lens B: The Pragmatic Minimalist ("The Code You Don't Write" & Anti-Bloat)**
3.  **Lens C: The Context & Economic Architect (Cost, RAG, Context Decay & Token Bounds)**
4.  **Lens D: The Legal & Forensic Auditor (Courtroom Traceability & Non-Reliance)**

---

## 2. Evidence Ledger

| ID | Source / Evidence Item | What It Proves | Limitation / Blind Spot |
|---|---|---|---|
| **EV-01** | `c:\MY_PROJECT\WORKSPACE.md` § 2 | 2-file root rule & 5 safety invariants preserve workspace hygiene across multi-agent sessions. | Invariants must be mechanically checked, not merely documented. |
| **EV-02** | `c:\MY_PROJECT\TRACKER.md` | Single-file tracking prevents divergent agent branches and stale work. | Requires immediate commit checkpoints after each step. |
| **EV-03** | `pkg/domain/models.go` & `pkg/licensing` | Ed25519 asymmetric verification and W3C DIDs run purely deterministically without LLM intervention. | Does not govern newly generated agent business logic. |
| **EV-04** | PostgreSQL RLS Migrations `0001–0070` | Transaction-scoped GUCs (`is_local = true`) mechanically prevent cross-tenant data leaks. | If an agent writes a migration missing RLS, the DB can leak if unindexed. |
| **EV-05** | `evaluator_bench_test.go` pattern | Automated benchmark tests can enforce $<15\mu\text{s}$ latency and `0 allocs/op`. | Only protects tested paths; untested paths can accumulate heap bloat. |

---

## 3. Adversarial Multi-Lens Debate

### 3.1 Clash 1: Can Agent Hallucinations and Inaccuracies Be Controlled via Prompts?
*   **Lens A (Industrial Cynic):** "Prompt engineering is a placebo in safety-critical systems. When an agent is calculating a 250-ton mobile crane outrigger ground bearing pressure, a prompt saying 'calculate accurately' will still produce floating-point rounding errors and invented equations. An agent will confidently hallucinate that an ASME B30.5 formula exists in a package that has never been written."
*   **Lens D (Legal Auditor):** "If an agent hallucinates a safety calculation and a crane collapses, the prompt is completely inadmissible as due diligence. The calculation must be executed by a verifiable, sandboxed AST with cryptographic provenance."
*   **Consensus / Resolution:** **Mechanical Separation of Execution.** Agents are strictly prohibited from calculating physical values in runtime. Agents may only declare rules using Google CEL (Common Expression Language). The pure Go kernel (`pkg/rulesengine`) executes the AST. Hallucination is rendered physically impossible at runtime because the LLM is nowhere in the execution loop.

---

### 3.2 Clash 2: "The Good Code Is the Code You Don't Write" vs. Agent Generation Proliferation
*   **Lens B (Pragmatic Minimalist):** "The deadliest failure mode of AI coding assistants is **Hyper-Generative Bloat**. Because generating code is free for the LLM, an agent asked to implement a simple status enum will generate 3 interfaces, 2 factory classes, an abstract strategy pattern, and 400 lines of boilerplate. Every line of unneeded code is a permanent liability, an attack surface, and an anchor on cognitive comprehension."
*   **Lens C (Economic Architect):** "Every extra line of code bloats future context windows. In 3 weeks, that agent's boilerplate will consume 20,000 tokens of input context for every subsequent agent turn, driving up costs and degrading attention."
*   **Lens A (Industrial Cynic):** "Furthermore, complex code hides subtle race conditions that automated testing misses."
*   **Consensus / Resolution:** **The "Rule of Three" & Code Deletion Metric.**
    1. Agents are forbidden from introducing an abstraction or interface unless the exact logic is duplicated across 3 distinct domain packages.
    2. Before writing new code, the agent must check if Go standard library or existing models solve it.
    3. PRs and commits that *reduce* net line count while retaining test coverage are explicitly prioritized.

---

### 3.3 Clash 3: Context Window Limits, Stale RAG, and Lost-in-the-Middle Decay
*   **Lens C (Economic Architect):** "Why do agents fail on long tasks? Context exhaustion and naive RAG. Standard RAG takes code, chops it into 500-token chunks, and embeds it. But code is not prose! Chunks slice functions in half, sever lexical scopes, and pull out-of-date snippets from 2024. The agent gets confused, reads the wrong files, loops 15 times, burns $10 in API tokens, and crashes."
*   **Lens B (Pragmatic Minimalist):** "And the worse the context pollution, the more the agent suffers from 'lost-in-the-middle' attention decay. It forgets the primary instruction from 10 steps ago."
*   **Consensus / Resolution:** **Deterministic Symbol Maps & The 2-File SSOT.**
    1. Standard vector RAG is banned for codebase navigation. Agents must use deterministic AST discovery: `grep_search`, `view_file`, Go symbol definitions (`go doc`), and curated catalog manifests ([`GITHUB_REPOSITORIES_REFERENCE.md`](./GITHUB_REPOSITORIES_REFERENCE.md)).
    2. Exactly 2 canonical root files are maintained: [`WORKSPACE.md`](../../WORKSPACE.md) (static truth) and [`TRACKER.md`](../../TRACKER.md) (dynamic truth). Zero documentation sprawl in root.
    3. Micro-sprints: One deliverable per agent turn. State is committed immediately to Git so context compaction can resume cleanly in 1 prompt.

---

### 3.4 Clash 4: Agent Autonomy vs. System Authority & Security
*   **Lens D (Legal Auditor):** "What prevents a rogue or confused agent from altering a database migration to remove RLS, or bypassing Keycloak authentication to make a quick test pass?"
*   **Lens A (Industrial Cynic):** "Nothing, unless the harness makes security violations fatal."
*   **Consensus / Resolution:** **The Inviolable Server & DB Authority.**
    1. Database RLS is enforced at the PostgreSQL engine level (`FORCE ROW LEVEL SECURITY`). Even if an agent writes sloppy SQL, the engine refuses cross-tenant queries.
    2. The AI advisory service (`ai_service`) is strictly read-only (`blocking=false`). It has zero database write access and cannot sign or alter certificates.
    3. Secret boundaries: `private/` is opaque. Any attempt to inspect or leak secrets aborts the task.

---

## 4. Findings & Dispositions Ledger

| ID | Lens | Finding / Dilemma | Evidence | Severity | Disposition | Mandatory Enforcement Protocol |
|---|---|---|---|:---:|:---:|---|
| **F-01** | Lens A | Agents claim "tests pass" without executing commands. | `integin-evidence-guard` | **CRITICAL** | **CONFIRMED** | Raw terminal stdout required for every completion claim. No output = task rejected. |
| **F-02** | Lens B | Agents write premature, over-engineered abstractions. | `simplify-codebase` | **HIGH** | **CONFIRMED** | Enforce "Rule of Three": zero interfaces without 3 distinct production callers. |
| **F-03** | Lens C | Standard vector RAG destroys code scope and context. | Architecture search | **HIGH** | **CONFIRMED** | Banish vector chunking for code. Use ripgrep, AST tools, and curated catalog references. |
| **F-04** | Lens D | Non-deterministic AI output cannot be audited in court. | ISO 17020 / Legal ADRs | **CRITICAL** | **CONFIRMED** | Math & rules compiled to Google CEL ASTs. AI advisory layer is strictly `blocking=false`. |
| **F-05** | Lens C | Runaway token loops and context degradation cause amnesia. | Agent trajectory logs | **MEDIUM** | **CONFIRMED** | Enforce 2-file root discipline (`WORKSPACE.md`, `TRACKER.md`) and 1-deliverable micro-sprints. |
| **F-06** | Lens A | Memory leaks and hidden heap escapes on hot paths. | Go runtime benchmarks | **HIGH** | **CONFIRMED** | Hot paths must maintain `evaluator_bench_test.go` asserting $<15\mu\text{s}$ and `0 allocs/op`. |

---

## 5. Final Ratified Verdict & Safe Operating Protocol

The debate is formally closed with unanimous convergence across all four lenses:

> **The INTEGIN Agent Immunity Standard:**  
> AI agents are treated as high-velocity stochastic compilers operating within rigid, mechanical constraints. The platform's integrity is guaranteed by **Deterministic Compilers, CEL ASTs, PostgreSQL RLS, Bounded Contexts, and Live Terminal Evidence**—never by trusting model compliance.

### Approved Immediate Next Actions:
1. Proceed with **Sprint 2 Deliverable 2.1: Computational Nanocell (`pkg/rulesengine`)**.
2. Implement Google CEL sandboxing, ASME/ISO proof-load formulas, and rigging/geotechnical math vectors with strict `0 allocs/op` benchmarks.
3. Keep root workspace strictly limited to `WORKSPACE.md` and `TRACKER.md`.

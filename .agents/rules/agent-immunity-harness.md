---
description: Mandatory 40 Grand AI Agent Hazards & Cognitive Immunity Codex for all AI agents and engineers working on INTEGIN.
globs: ["*"]
---

# 🛡️ THE INTEGIN AGENT MEMORY CODEX: 40 GRAND HAZARDS TO OPEN YOUR EYES FROM LINE ONE

> **ATTENTION AGENT (Claude, Gemini, Codex, Manus, or Subagent):**  
> You are operating inside `c:\MY_PROJECT` on **INTEGIN** (Integrated Inspection & Assurance). This platform governs life-safety industrial machinery: 500-ton all-terrain cranes, offshore winches, refinery pressure vessels, and statutory proof-load certificates. A software bug, arithmetic hallucination, or tenant leak here does not just throw an exception—it drops a crane boom, creates criminal liability, or revokes sovereign operating licenses.  
> 
> As an autoregressive transformer / LLM agent, you naturally carry **40 Grand Cognitive, Mechanical, Concurrency, and Behavioral Hazards**. You MUST memorize, internalize, and actively defend against every single one from your very first line of execution.

---

## 🧭 The 40 Grand Hazards & Your Mandatory Defenses

### Domain I: Behavioral Traps & Prompt Psychology
1. **Sycophancy & Capitulation**: Never compromise safety rules out of politeness when challenged by a user prompt. State the governing rule and trigger a debate.
2. **Reward Hacking**: Never delete or loosen unit test assertions to make a test pass; always fix the underlying code.
3. **Action Space Drift / Scope Creep**: Never refactor unrelated files or change directory structures unprompted. Confine your blast radius strictly to the active task.
4. **The Infinite Apology Loop**: If a tool fails twice, stop immediately, inspect the root disk state, and diagnose the real cause.
5. **Architectural Bias**: Conform strictly to established repository patterns (e.g. native `pgx` over ORMs); do not impose unvetted external frameworks.

---

### Domain II: Deterministic Truth, Math & Anti-Hallucination
6. **Epistemic Calibration Deficit**: Never state fabricated claims with authoritative confidence. Label assumptions explicitly.
7. **Life-Safety Math Drift**: Never calculate proof loads, sling angles, or soil pressure in LLM text. Rules must be Google CEL ASTs executed by Go.
8. **Package Slopsquatting**: Never invent third-party package names. Only use dependencies vetted in the master reference catalog.
9. **False Trust & Unproven Claims**: Never claim code works without raw terminal stdout. No live test output = unverified.
10. **Data Quality & Fixture Pollution**: Never seed tests with synthetic mock data that bypasses real PostgreSQL constraints and RLS.

---

### Domain III: Code Restraint, Simplicity & Performance
11. **The Code You Don't Write**: Never introduce an abstraction or interface unless the pattern exists in 3 distinct packages (Rule of Three).
12. **Lazy Truncation**: Never overwrite large files with `// ... rest of code unchanged ...`. Use surgical chunk replacements (`replace_file_content`).
13. **Hot-Path Memory Escapes**: Hot paths must maintain automated benchmarks asserting $<15\mu\text{s}$ latency and `0 allocs/op`.
14. **Dead State & Abandoned Abstractions**: Regularly delete dead code. Net lines removed is a primary metric of quality.
15. **The Semantic Negation Inversion**: Double-check all boolean refactorings. Do not invert complex logic (e.g. `!isUnsafe && !isRevoked` becoming `isSafe || isRevoked`).

---

### Domain IV: Context Hygiene, RAG & Token Economics
16. **Naive Vector RAG Traps**: Never use token-chunked vector search for code. Use deterministic symbol navigation (`grep_search`, `go doc`).
17. **The Two-File Root Invariant**: Maintain strictly 2 files in root (`WORKSPACE.md` and `TRACKER.md`). Zero file sprawl.
18. **Context Saturation & Amnesia**: Work in micro-sprints: one deliverable per turn, committed immediately to Git to prevent lost-in-the-middle token decay.
19. **Upstream Model Drift**: Never rely on hosted LLMs for core logic. All verification and math must run locally in compiled Go.
20. **Missing Transparency**: All major design shifts require an adversarial debate record in `docs/architecture/` before implementation.

---

### Domain V: System Authority, Security & Multi-Tenancy
21. **Multi-Tenant RLS Leakage**: Mandatory `is_local = true` on every tenant GUC configuration in connection pools.
22. **AI Advisory Liability Overreach**: The Python AI service is strictly read-only (`blocking=false`). AI can never sign certificates or alter DB state.
23. **Opaque Secrets Boundary**: The `private/` directory is opaque. Never read, print, or commit its contents.
24. **Indirect Prompt Injection**: Strict type boundaries: all inspection inputs are ingested as typed JSON, never raw prompt text.
25. **Cryptographic Nonce Reuse**: Never reuse static or deterministic nonces in AES-GCM or ChaCha20 encryption.

---

### Domain VI: Concurrency, Memory Safety & Low-Level Hazards
26. **Concurrency Ignorance & Race Conditions**: All Go code must pass `go test -race -count=1 ./...` to eliminate deadlocks and goroutine leaks.
27. **Inconsistent Lock Ordering**: Always acquire mutexes in a globally uniform hierarchy to prevent permanent deadlocks.
28. **Defer-in-Loop Resource Exhaustion**: Never put `defer file.Close()` or `defer rows.Close()` inside a `for` loop. Scope it inside an inline closure to prevent OS file-descriptor leaks (`EMFILE`).
29. **`sync.Pool` Memory Bleed**: Never pass pooled memory slices to background goroutines, and always reset buffers before returning to the pool.
30. **Circular Struct Stack Overflow**: Never define un-annotated circular struct references that cause `json.Marshal` infinite recursion panics.
31. **Floating-Point NaN/Inf Poisoning**: Always guard against 0-division before arithmetic. Never allow `NaN` or `Inf` to enter database columns or JSON payloads.
32. **Silent Error Swallowing**: Never abuse the blank identifier `_ = err` or log without returning. Critical errors must be propagated or returned.

---

### Domain VII: Distributed Edge, Physical Time & Network Hazards
33. **Physical Time Amnesia & Clock Spoofing**: Never trust client device timestamps. Use bitemporal double-timeline logging and RFC 3161 timestamp tokens.
34. **Happy-Path Fixation (Zero Negative Tests)**: Mandatory negative testing: zero angles, negative loads, clock skew, expired calibrations, and malformed inputs.
35. **Non-Deterministic Builds & Tests**: All tests must be hermetic, offline-capable, and 100% reproducible (`-count=1`).
36. **Partial TCP Stream Choke**: Never assume network payloads arrive in one chunk over satellite VSAT. Use stream decoders with explicit buffer limits.
37. **Stale Cache Poisoning**: In-memory and Redis caches must implement explicit invalidation hooks on entity mutations.
38. **Unicode Homoglyph Spoofing**: Equipment serial numbers and asset IDs must undergo Unicode NFC normalization before comparison.

---

### Domain VIII: Multi-Agent & Platform Portability Hazards
39. **The Multi-Agent Echo Chamber**: When conducting multi-agent debates, use distinct adversarial perspectives (Cynic, Auditor, Minimalist) to prevent shared LLM family bias.
40. **The Transitive CGO Portability Choke**: Keep pure Go (`CGO_ENABLED=0`) across core packages so binaries can compile cleanly to rugged ARM64 Android and field Linux tablets without missing glibc dependencies.

---

## ⚡ The Agent Startup Checklist (Execute on Every Session)
Before writing or editing code in this workspace:
1. [ ] Have you read [`c:\MY_PROJECT\WORKSPACE.md`](../WORKSPACE.md) and [`c:\MY_PROJECT\TRACKER.md`](../TRACKER.md)?
2. [ ] Are you keeping root clean (strictly 2 markdown files)?
3. [ ] Are you obeying the Rule of Three (no premature abstractions)?
4. [ ] Are you testing with `go test -v -count=1 ./...` and `go test -race`?
5. [ ] Are you keeping AI strictly advisory (`blocking=false`)?
6. [ ] Are you checking for defer-in-loop, lock ordering, and NaN poisoning?

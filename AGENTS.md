# 🤖 INTEGIN Pilot Source: Autonomous Agent Operating Standard & Immunity Harness

**Authority:** Single Source of Truth for all autonomous implementers (OpenCode, Claude Code, Codex, Gemini CLI) operating within `integin-pilot-source/`.  
**Master Workspace Context:** `C:\MY_PROJECT` (Parent Workspace Reference: `../WORKSPACE.md`)

---

## 1. 🛡️ The 8 Strict Operating Invariants

Every agent executing commands or modifying files within this repository MUST follow these invariants without exception:

1. **Single Active Source**: All implementation work happens strictly inside `integin-pilot-source/`. NEVER modify or copy files into `../INTEGIN-source/` (it is a frozen read-only reference snapshot).
2. **Zero Root File Sprawl**: NEVER create files in the parent workspace root `C:\MY_PROJECT\`. The parent workspace strictly maintains the **Two-File Root Invariant** (`WORKSPACE.md` and `TRACKER.md` only).
3. **Opaque Secrets Boundary**: NEVER inspect, read, print, or log anything inside `../private/` or subdirectories.
4. **Mandatory Multi-Tenant RLS**: Every PostgreSQL query on domain tables must execute under explicit session tenant configuration:
   ```sql
   SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true);
   ```
   Always use `is_local = true` (or `SET LOCAL`) to prevent GUC pollution in connection pools.
5. **Deterministic Physics & Math (Zero LLM Calculations)**: NEVER calculate load ratings, proof loads, sling angles, or soil bearing pressures in runtime text or prompt logic. All calculations must be evaluated via Google CEL (Common Expression Language) ASTs in `pkg/rulesengine/`.
6. **Code Simplicity (The Rule of Three)**: Never introduce generic abstractions, wrappers, or factory patterns unless identical logic is duplicated across 3 distinct domain packages. Prefer standard library and concrete implementations.
7. **The Commit Boundary**: Autonomous implementers NEVER commit or stage code (`git add` / `git commit`). Leave changes uncommitted in the working tree for orchestrator inspection and verification.
8. **Statutory Quality Gates**: Every code modification must pass:
   - `go vet ./...` (0 warnings)
   - `go test -count=1 ./...` (0 failures, fully offline reproducible)
   - `gofmt -l .` (0 unformatted files)

---

## 2. ⚡ Concrete Mechanical Guardrails (Low-Model Immunity Shield)

Autonomous coders—especially compact and free-tier models—must strictly adhere to these syntax-level negative constraints:

- **NO Silent Error Swallowing (Hazard 32)**: NEVER write `_ = err` or ignore errors. Always return or handle: `if err != nil { return ..., err }`.
- **NO Defer-in-Loop Resource Leaks (Hazard 28)**: NEVER put `defer file.Close()` or `defer rows.Close()` inside a `for` loop. Close immediately or wrap loop bodies in an inline closure (`func() { ... }()`).
- **NO Floating-Point NaN / Inf Poisoning (Hazard 31)**: ALWAYS guard divisors before dividing (`if denom == 0 { return ..., err }`). Never allow `NaN` or `Inf` to enter database columns, struct fields, or JSON payloads.
- **NO Reward Hacking (Hazard 2)**: NEVER delete, comment out, or weaken test assertions (e.g. relaxing exact equality to `!= ""` or ignoring errors) to make tests pass. If a test fails, fix the code under test.
- **NO Hyper-Generative Boilerplate (Hazard 11)**: NEVER create unprompted interfaces, mock factories, or adapter hierarchies. Write concrete structs and direct methods.
- **NO Concurrency Data Races (Hazard 26)**: ALWAYS synchronize shared map or slice access across goroutines with `sync.RWMutex` or `sync.Mutex`. Tests must pass `go test -race` cleanly.
- **NO Transitive CGO Dependencies (Hazard 40)**: Pure Go only (`CGO_ENABLED=0`). Do not import packages that require external C libraries or native host headers, ensuring builds remain portable to rugged ARM64 tablets.
- **NO Parameterized SQL String Concat (Hazard 21)**: Always use parameterized `$1, $2, ...` placeholders. Never use `fmt.Sprintf` or string concatenation to inject user-supplied values into SQL statements.

---

## 2b. 🦎 Reuse Ladder (Ponytail adaptation — mandatory for big-pickle, mimo, all relay implementers)

Before writing code, stop at the first rung that holds (lazy about the solution, never about reading — trace the real flow first):

1. Does this need to exist? → no: skip it (YAGNI).
2. Already in this codebase? → reuse it, don't rewrite.
3. Stdlib does it? → use it.
4. Smallest diff that passes the gates? → write only that.

Trust-boundary validation, error handling, security, RLS guards, and tests are never on the chopping block. Delete-list over add-list: prefer removing lines to adding them.

---

## 3. 🚦 Model Catalog & Usage Guidelines

OpenCode requires an explicit model on fresh runs (`--model`). Only the human owns billing approvals:

| Model Category | Approved Models | Usage Guidance |
|---|---|---|
| **Free / Flat-Rate (Approved)** | `opencode/big-pickle`<br>`opencode/muse-spark-1.3-contributor-free`<br>`opencode/mimo-v2.5-free` | **Default choice** for routine implementation, refactoring sweeps, mechanical tests, and unit features. |
| **Metered / Paid API** | Google Gemini (`google/gemini-2.5-pro`)<br>OpenRouter providers | **Requires explicit human confirmation** before dispatch to prevent surprise API charges. |

---

## 4. 🛠️ Standard Verification & Gate Commands

Run these from the `integin-pilot-source` directory before declaring any implementation step complete:

```powershell
# 1. Format Check
gofmt -l .

# 2. Static Analysis / Vet
go vet ./...

# 3. Clean-Cache Test Suite
go test -count=1 ./...

# 4. Concurrency Race Detection (Clean)
go test -race -count=1 ./...
```

---

## 5. 📋 Structured Output Contract

When stopping after an implementation run, always end with a closing summary in this format:

```text
## Implementation Report
1. What Changed and Why: Concise summary of code changes.
2. Files Touched: Exact paths modified or created.
3. Verification Outcomes: Raw test pass/fail counts and go vet results.
4. Deviations or Open Questions: Any edge cases left open or requiring human decision.
```



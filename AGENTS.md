# 🤖 INTEGIN Pilot Source: Autonomous Agent Operating Standard & Immunity Harness

**Authority:** Single Source of Truth for all autonomous implementers (OpenCode, Claude Code, Codex, Gemini CLI) operating within `integin-pilot-source/`.  
**Master Workspace Context:** `C:\MY_PROJECT` (Parent Workspace Reference: `../WORKSPACE.md`)

---

## 1. 🛡️ The 8 Strict Operating Invariants

Every agent executing commands or modifying files within this repository MUST follow these invariants without exception:

1. **Single Active Source**: All implementation work happens strictly inside `integin-pilot-source/`. NEVER modify or copy files into `../integin-source/` (it is a frozen read-only reference snapshot).
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

## 2. 🚦 Model Catalog & Usage Guidelines

OpenCode requires an explicit model on fresh runs (`--model`). Only the human owns billing approvals:

| Model Category | Approved Models | Usage Guidance |
|---|---|---|
| **Free / Flat-Rate (Approved)** | `opencode/big-pickle`<br>`opencode/muse-spark-1.3-contributor-free`<br>`opencode/mimo-v2.5-free` | **Default choice** for routine implementation, refactoring sweeps, mechanical tests, and unit features. |
| **Metered / Paid API** | Google Gemini (`google/gemini-2.5-pro`)<br>OpenRouter providers | **Requires explicit human confirmation** before dispatch to prevent surprise API charges. |

---

## 3. 🛠️ Standard Verification & Gate Commands

Run these from the `integin-pilot-source` directory before declaring any implementation step complete:

```powershell
# 1. Format Check
gofmt -l .

# 2. Static Analysis / Vet
go vet ./...

# 3. Clean-Cache Test Suite
go test -count=1 ./...

# 4. Memory Escapes on Hot Paths (Latency < 50µs, 0 allocs/op)
go test -benchmem -run=^$ ./pkg/rulesengine/...
```

---

## 4. 📋 Structured Output Contract

When stopping after an implementation run, always end with a closing summary in this format:

```text
## Implementation Report
1. What Changed and Why: Concise summary of code changes.
2. Files Touched: Exact paths modified or created.
3. Verification Outcomes: Raw test pass/fail counts and go vet results.
4. Deviations or Open Questions: Any edge cases left open or requiring human decision.
```

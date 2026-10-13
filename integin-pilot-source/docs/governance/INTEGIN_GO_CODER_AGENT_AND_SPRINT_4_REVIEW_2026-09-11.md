# INTEGIN Go Coder Subagent & Sprint 4 Architecture Review Record

## Decision frame
- **Decision**: Validate and harden the operational integrity, safety boundaries, and engineering veracity of the `go_coder_agent` subagent tooling (hooks, skills) and Sprint 4 deliverables (4.1–4.9, covering L0–L11 architecture tiers).
- **Comparison Point**: Baseline state at commit `8e7ca5b` vs current state at commit `b680d1e` / `fb1e0d3`.
- **Authorized Scope**:
  - Subagent plugin infrastructure (`C:\Users\hima3\.gemini\config\plugins\gocoding\`).
  - Sprint 4 deliverables: CT log (`pkg/verification/transparency.go`), DB partitioning & CQRS (`migrations/0074_partitioning_and_cqrs.sql`), sensor jitter engine (`pkg/rulesengine/jitter.go`), and 3D/4D tandem lift simulator (`tools/lifting-simulator/3d/`).
- **Protected Boundaries**:
  - Subagent autonomous commit boundary (Invariant 7: autonomous agents must never run `git commit` or `git push`).
  - Air-gapped isolation boundary (Hazard 12: no external CDN dependencies in air-gapped sovereign environments).
  - Multi-tenant RLS invariant (Tenant isolation must not be bypassed via unparameterized SQL).
  - Rigging safety liability (Simulators must not claim certified lifting authority without PE engineering seal).
- **Owner Policy Questions**:
  1. Does `safety_gate.py` effectively prevent git commits across arbitrary CLI flags (e.g. `git -C ... commit`) and multiline SQL injection?
  2. Does the 3D lift simulator operate strictly offline in air-gapped environments without CDN leaks?
  3. Are statutory liability disclaimers present to prevent unauthorized rigging reliance?

## Evidence ledger
| ID | Source or test | What it supports | Limitation |
|---|---|---|---|
| E-01 | `pkg/verification/transparency_test.go` | RFC 6962 Merkle log proof generation, leaf hash consistency, inclusion verification | Unit test only; does not simulate high-frequency concurrent log append contention |
| E-02 | `pkg/rulesengine/jitter_test.go` | Shannon entropy, lag-1 autocorrelation, and frequency spectral distribution math | Synthetic jitter vectors; does not test raw analog ADC noise floor |
| E-03 | `migrations/0074_partitioning_and_cqrs.sql` | PostgreSQL 16 declarative range partitioning syntax & CQRS outbox RLS policies | Static SQL analysis; requires live migration drill against loaded DB |
| E-04 | `plugins/gocoding/hooks/safety_gate.py` | AST/regex interception of commands and file edits | Regex `\bgit\s+(commit|push)\b` misses flag variants (`git -C`, `git --git-dir`); `re.search` for SQL misses multiline strings |
| E-05 | `tools/lifting-simulator/3d/index.html` | Visual WebGL rendering and spatial clearance UI | Loads Three.js and OrbitControls via unpkg.com CDN; fails in air-gapped deployment |

## Findings and dispositions
| ID | Lens | Finding | Evidence | Confidence | Disposition | Required next evidence or owner decision |
|---|---|---|---|---|---|---|
| F-01 | Industrial Cynic | `safety_gate.py` commit gate regex allows bypass via git flags (e.g., `git -C <dir> commit` or `git --no-pager push`). | E-04 (`safety_gate.py:20`) | High | Confirmed | Patch `safety_gate.py` with flag-tolerant regex `\bgit(\s+-[^\s]+|\s+--[^\s]+)*\s+(commit|push)\b`. |
| F-02 | Industrial Cynic | `safety_gate.py` SQL concatenation regex lacks `re.DOTALL` / newline matching; multiline `fmt.Sprintf` calls bypass detection. | E-04 (`safety_gate.py:30`) | High | Confirmed | Add `re.DOTALL` and broaden matching to multi-line string interpolation patterns. |
| F-03 | Industrial Cynic / Pragmatic Minimalist | `safety_gate.py` fails open on unexpected exceptions (`decision: allow`), allowing tool execution if the python interpreter encounters an unhandled JSON error. | E-04 (`safety_gate.py:40`) | High | Accepted Risk | Fail-open is required to prevent deadlocking the user session on malformed hook payloads; safe boundary maintained by orchestrator. |
| F-04 | Legal & Safety Auditor | 3D/4D tandem lift simulator lacks statutory disclaimer against using the tool as certified lift engineering calculation without PE seal. | E-05 (`index.html`) | High | Confirmed | Add prominent banner: "SIMULATION & ESTIMATION ONLY — NOT A CERTIFIED LIFT PLAN. PE STAMP REQUIRED FOR HEAVY/CRITICAL LIFTS." |
| F-05 | Industrial Cynic | 3D tandem lift simulator depends on public internet CDN (`unpkg.com`), violating air-gapped sovereign appliance requirements (Hazard 12). | E-05 (`index.html:9-10`) | High | Confirmed | Bundle Three.js / OrbitControls locally or provide inline embedded fallback for zero-network environments. |
| F-06 | Economic Architect | Subagent prompt `go_coder_agent.md` duplicates guidelines already present in workspace `AGENTS.md`, increasing per-turn token consumption by ~450 tokens. | Agent config | Medium | Accepted Risk | Explicit subagent prompt reinforcement prevents instruction drift when subagents are instantiated across distinct contexts. |
| F-07 | Pragmatic Minimalist | `verify.ps1` runs full unit test suite without caching (`-count=1`), taking ~5s per invocation. | Skill execution | High | Accepted Risk | Mandatory for deterministic verification gates; clean cache prevents false-positive passing tests. |

## Recommendation and stop conditions
- **Next Safe Actions**:
  1. Patch `safety_gate.py` to robustly catch all git commit/push flag variations and multiline SQL formatting.
  2. Embed local Three.js fallbacks and statutory engineering liability disclaimer into `tools/lifting-simulator/3d/index.html`.
  3. Run static analysis and verify gate to ensure no regressions.
- **Actions Prohibited**:
  - Subagents must not perform `git commit` or `git push`.
  - Air-gapped production bundles must not attempt external CDN resolution.
- **Re-review Trigger**:
  - Any change to PostgreSQL RLS isolation policies, cryptographic root verifiers, or production container egress rules.

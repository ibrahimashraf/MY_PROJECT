# ⚖️ INTEGIN AI Token & Cost Optimization Governance Debate

**Document ID:** `DEBATE-TOKEN-COST-2026-09-07`  
**Date:** 2026-09-07  
**Status:** CLOSED & RATIFIED  
**Governing Standard:** [`integin-review-governance`](../../.agents/skills/integin-review-governance/SKILL.md) & [`integin-evidence-guard`](../../.agents/skills/integin-evidence-guard/SKILL.md)  
**Topic:** Formal Adversarial Examination of AI Token, Quota, and Cost Optimization vs. Engineering Rigor, Safety Invariants, and Context Continuity.

---

## 1. Decision Frame

### 1.1 Objective
To evaluate whether aggressive AI token and cost-reduction mechanisms (session resets, tiered model routing, context truncation, and output token restraint) introduce safety regressions, context amnesia, or compromised verification in INTEGIN.

### 1.2 Comparison Point
*   **Unconstrained Agent Context (Status Quo):** Indefinite multi-turn conversations (30+ turns), full-file dumps, maximum model reasoning across all tasks, unbudgeted token burn.
*   **The Proposed Optimization Framework:** The 7-Lever Cost Optimization Standard codified in [`AI_TOKEN_AND_COST_OPTIMIZATION_GUIDE.md`](./AI_TOKEN_AND_COST_OPTIMIZATION_GUIDE.md).

### 1.3 The 4 Adversarial Review Lenses
1.  **Lens A: The Cost & Economic Pragmatist (Frugal Engineering)**
2.  **Lens B: The Zero-Trust Safety & Forensic Auditor (Anti-Amnesia)**
3.  **Lens C: The Systems Minimalist (Disk-Backed Truth)**
4.  **Lens D: The Developer Velocity & Ergonomics Champion (Human Friction)**

---

## 2. Evidence Ledger

| ID | Source / Evidence Item | What It Proves | Limitation / Risk |
|---|---|---|---|
| **EV-01** | `TRACKER.md` v3.3.0 | Comprehensive sprint checklist acts as a complete external memory bus. | Requires strict discipline to update before ending a session. |
| **EV-02** | `WORKSPACE.md` § 1 & § 2 | Core architecture, ports, and invariants can be re-ingested in $<3,000$ tokens. | Startup prompt must remain stable to maximize prompt cache hits. |
| **EV-03** | Web Research (Anthropic, Google, OpenAI) | Prompt caching provides an automatic 90% discount on static prefix tokens. | Any 1-character change to system prompt invalidates the cache. |
| **EV-04** | Go Benchmark Gate (`0 allocs/op`) | Automated tests objectively catch performance/code regressions regardless of model tier. | Small models may struggle to write complex benchmark setups. |

---

## 3. Adversarial Multi-Lens Debate

### 3.1 Clash 1: Does Resetting Sessions Cause Context Amnesia and Lost Safety Constraints?
*   **Lens B (Safety Auditor):** "If you close a session after 5 turns to save money, the incoming agent forgets the subtle edge cases we just spent 2 hours debugging. It might forget that outrigger pressure requires transaction-scoped `is_local = true` or that sling angles $<30^\circ$ are fatal."
*   **Lens C (Systems Minimalist):** "That is a misconception. Keeping 150,000 tokens of messy conversation in memory is *worse* than a clean reset. In long sessions, models suffer from **Attention Degradation (Lost-in-the-Middle)**. The model hallucinates *more* in turn 30 than in turn 2 because the prompt is polluted with obsolete tool logs and discarded drafts. A fresh session reading `WORKSPACE.md` and `TRACKER.md` has 100% sharp attention on the exact task at hand."
*   **Consensus / Resolution:** **Disk-Backed Memory Invariant.** A session reset is ONLY permitted after state is safely committed to Git and checklist items are marked in `TRACKER.md`. Context is not lost; it is promoted from ephemeral chat memory to immutable repository truth.

---

### 3.2 Clash 2: The Danger of Tiered Model Routing (Cheap Models on Critical Code)
*   **Lens A (Cost Pragmatist):** "Why pay $15/M for Claude 3.5 Sonnet to run `git status`, search for a file, or format a Markdown table? Route utility tasks to Gemini Flash ($0.075/M) and save 95%."
*   **Lens B (Safety Auditor):** "If a cheaper model is allowed to edit Go code, it might introduce subtle race conditions, silent error swallowing (`_ = err`), or floating-point rounding errors that look plausible but fail in production."
*   **Consensus / Resolution:** **Strict Tier Fencing.**
    *   **Tier 1 (Frontier Models Only):** Go kernel code (`pkg/rulesengine`, `pkg/ledger`), cryptographic licensing, database migrations, and architectural debates.
    *   **Tier 2 (Fast/Cheap Models Permitted):** File searches, documentation formatting, test log grepping, and git operations.
    *   **The Invariant:** Regardless of model tier, NO code enters the codebase without passing `go test -race -count=1` and `go vet`.

---

### 3.3 Clash 3: Output Token Restraint vs. User Transparency
*   **Lens D (Developer Ergonomics):** "If an agent becomes ultra-concise to save output tokens, the user doesn't know what happened. Engineers need to see the reasoning and the code diffs."
*   **Lens A (Cost Pragmatist):** "Output tokens cost 5x more than input tokens. Printing 1,500 words of conversational restatement of a file that was just written to disk burns $0.25 per turn for zero added value."
*   **Consensus / Resolution:** **Pointer-Based Transparency.** The agent must write the complete artifact to disk (`docs/architecture/` or source code) and provide a concise, 10-line executive summary in chat with clickable file links. The user gets full transparency by viewing the file, without burning expensive conversational output tokens.

---

## 4. Findings & Dispositions Ledger

| ID | Lens | Finding / Concern | Evidence | Severity | Disposition | Mandatory Enforcement Policy |
|---|---|---|---|:---:|:---:|---|
| **F-01** | Lens B | Resetting session before recording state causes amnesia. | `TRACKER.md` | **HIGH** | **CONFIRMED** | Never reset a session without Git commit + `TRACKER.md` update. |
| **F-02** | Lens B | Cheap models writing core safety logic causes bugs. | LLM Benchmark data | **HIGH** | **CONFIRMED** | Tier fencing: Frontier models strictly required for Go kernel and crypto. |
| **F-03** | Lens A | Output token verbosity burns 5x credits unnecessarily. | API Pricing matrices | **MEDIUM** | **CONFIRMED** | Enforce concise chat output; write detailed records to disk artifacts. |
| **F-04** | Lens C | System prompt churn destroys prompt caching. | Anthropic/Google docs | **MEDIUM** | **CONFIRMED** | Keep `WORKSPACE.md` and `.agents/rules/` static to lock in 90% KV cache discounts. |

---

## 5. Final Ratified Verdict

> **The INTEGIN Cost-Safety Treaty:**  
> Token and cost optimization is an architectural necessity, not an aesthetic preference. Cost reduction must be achieved through **Session Resets, Prompt Caching, and Disk-Backed State**, NEVER by compromising compiler gates, `-race` tests, or frontier reasoning on life-safety Go logic.

### Next Safe Action:
Update `TRACKER.md` with Decision Reference #14, and proceed directly to **Sprint 2, Deliverable 2.1 (`pkg/rulesengine`)**.

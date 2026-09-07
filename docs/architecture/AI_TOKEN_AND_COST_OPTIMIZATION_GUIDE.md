# 💰 The INTEGIN Guide to AI Token, Quota & Cost Optimization
### How to Slash AI Coding Spend by 80%–95% Without Sacrificing Engineering Rigor

**Document Version:** 1.0.0  
**Date:** 2026-09-07  
**Authority:** Single Source of Truth for Human Engineers & AI Agents  
**Target Environments:** Google Antigravity, Claude Code, Cursor, Gemini CLI, Aider, OpenAI, DeepSeek  

---

## 1. The Root Cause: Why AI Coding Assistants Burn So Much Money

If you are experiencing rapid credit depletion, high API bills, or quota exhaustion while working with AI agents, you are caught in **The Compounding Context Trap (The Snowball Effect)**.

```
                  THE COMPOUNDING CONTEXT TRAP (THE SNOWBALL)
Turn 1:   [Prompt: 5k tokens]                                      ──▶ Cost: $0.015
Turn 5:   [History + Files + Tools + Prompt: 40k tokens]           ──▶ Cost: $0.12
Turn 15:  [History + 10 File Reads + Diffs + Prompt: 110k tokens]  ──▶ Cost: $0.33
Turn 25:  [Massive Context Re-sent on EVERY message: 160k tokens]  ──▶ Cost: $0.48 / message!
──────────────────────────────────────────────────────────────────────────────────────
A single 25-turn chat re-sends over 2.5 MILLION input tokens! Total cost: $10 – $35+
```

### The 4 Invisible Token Burners:
1. **Compounding History Re-transmission**: Every time you type *"debate"*, *"check this"*, or *"continue"*, your client re-transmits the **entire past conversation**—including every large tool output, terminal log, and file read—from Turn 1 to Turn 25.
2. **Output Token Premium**: Output tokens (what the AI writes) cost **3x to 5x more** than input tokens ($15/M vs $3/M on frontier models). An agent writing 2,000 words of conversational boilerplate burns massive credits.
3. **Redundant Whole-File Reads**: An agent reading an 800-line file 5 times in a session re-ingests 30,000 tokens repeatedly.
4. **Model Mismatch**: Using top-tier frontier models ($15/M–$60/M) to run simple regex checks, git status, or formatting that a $0.15/M Flash model could do instantly.

---

## 2. The 7 Golden Levers to Cut Costs by 80%–95%

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                                 THE 7 COST-SLICING LEVERS                              │
├─────────────────────────┬──────────────────────────────┬───────────────────────────────┤
│ Lever 1: Session Reset  │ "1 Step = 1 Fresh Chat"      │ 85%–95% Total Cost Reduction  │
├─────────────────────────┼──────────────────────────────┼───────────────────────────────┤
│ Lever 2: Prompt Caching │ Static Prefix Alignment      │ 90% Input Token Discount      │
├─────────────────────────┼──────────────────────────────┼───────────────────────────────┤
│ Lever 3: Tiered Models  │ Flash for Tools, Sonnet Math │ 60%–80% Blended Cost Savings  │
├─────────────────────────┼──────────────────────────────┼───────────────────────────────┤
│ Lever 4: Surgical Slices│ `StartLine/EndLine` Reading  │ 75% Context Volume Reduction  │
├─────────────────────────┼──────────────────────────────┼───────────────────────────────┤
│ Lever 5: Concise Output │ No Conversational Fluff      │ 70% Output Token Savings      │
├─────────────────────────┼──────────────────────────────┼───────────────────────────────┤
│ Lever 6: Disk Memory    │ Files over Chat History      │ Prevents Context Bloat        │
├─────────────────────────┼──────────────────────────────┼───────────────────────────────┤
│ Lever 7: Tool Filters   │ Pipe commands to grep/head   │ Prevents Terminal Log Floods  │
└─────────────────────────┴──────────────────────────────┴───────────────────────────────┘
```

---

### Lever 1: The "1 Micro-Sprint = 1 Fresh Session" Protocol (The #1 Money Saver)

This is the single most powerful action you can take: **Stop keeping long-running 30-turn chats alive.**

* **The Problem:** In Turn 20, you pay for 150,000 tokens on every single message.
* **The Solution:**
  1. Complete one clear milestone (e.g., Deliverable 2.1 Step 1).
  2. Commit the changes to Git.
  3. Ensure [`TRACKER.md`](../../TRACKER.md) is updated.
  4. **Close the chat and open a FRESH NEW SESSION.**
* **Why this works:** When the new session starts, the agent reads only `WORKSPACE.md` and `TRACKER.md` (~3,000 tokens total). Your next turn costs **$0.009 instead of $0.45**—an instant **98% cost cut** with zero loss of progress!

---

### Lever 2: Maximizing Prompt Caching (90% Discount on Static Context)

Major AI providers (Anthropic, Google Gemini, OpenAI) offer **Prompt Caching** (KV Cache reuse):
* **Normal Input Cost:** ~$3.00 per million tokens.
* **Cached Input Cost:** ~$0.30 per million tokens (**90% discount!**).

#### How to Guarantee Cache Hits in INTEGIN:
* **Rule: Static First, Dynamic Last.** Keep unchanging files at the top of instructions (`WORKSPACE.md`, `.agents/rules/agent-immunity-harness.md`).
* **Do Not Edit System Prompts Constantly:** Changing even a single character in your system prompt or rules file invalidates the entire cache for all subsequent requests. Keep rules stable.

---

### Lever 3: Tiered Model Routing (Right Model for the Right Job)

Never use an expensive reasoning model for mechanical chores.

| Task Category | Optimal Model Tier | Cost per Million Tokens | Example Tasks |
|---|---|:---:|---|
| **Frontier Reasoning** | Claude 3.5 Sonnet / Gemini 1.5 Pro / o3-mini | ~$3.00 in / $15.00 out | Architectural debates, CEL formula math, security audits, complex Go algorithms. |
| **Workhorse / Standard** | DeepSeek-V3 / GPT-4o | ~$0.27 in / $1.10 out | Writing standard unit tests, implementing boilerplate structs, documentation. |
| **Fast / Utility** | Gemini 1.5 Flash / Claude 3.5 Haiku | ~$0.075 in / $0.30 out | Searching directories, checking git status, running linters, simple regex edits. |

---

### Lever 4: Surgical File Slices (Stop Ingesting Giant Files)

* **The Waste:** An agent calls `view_file` on a 1,200-line file to check 15 lines of code. This wastes ~10,000 tokens.
* **The Practice:**
  * Always use `StartLine` and `EndLine`:
    ```json
    { "AbsolutePath": "pkg/rulesengine/evaluator.go", "StartLine": 45, "EndLine": 85 }
    ```
  * Ingest only the 40 lines needed (~300 tokens instead of 10,000).

---

### Lever 5: Output Token Restraint (Concise Responses & Artifact Pointers)

Because output tokens are up to **5x more expensive** than input tokens:
* **The Waste:** When an agent finishes writing a markdown document or code file, it repeats the entire document in the chat response.
* **The Practice:**
  * The agent writes the artifact to disk (`docs/architecture/...`).
  * In chat, the agent outputs a **concise 10-line summary with a link**.
  * Never re-summarize what is already visible in the artifact.

---

### Lever 6: Disk-Backed Memory over Conversational Memory

* Do not explain complex history in conversation.
* Keep the history in **Git** and in **`docs/architecture/`**.
* The agent reads the specific document with `view_file` on-demand only when needed, keeping the default context clean.

---

### Lever 7: Filter Command Outputs at the Source

* **The Waste:** Running `go test -v ./...` on the whole repository can spew 500 lines of output into the agent context (8,000 tokens).
* **The Practice:**
  * Target specific packages: `go test -v ./pkg/rulesengine/...`
  * Pipe noisy commands: `go test ./... | Select-String "FAIL"` or `git status -s`.
  * If a command outputs massive logs, write them to a scratch file (`scratch/test.log`) and grep the error rather than dumping raw stdout.

---

## 3. Human Workflow Playbook: What You Can Do Right Now

To stop token drain immediately in your daily workflow:

1. **Keep Prompts Focused**: Instead of saying *"analyze everything and debate and plan and fix"*, give one atomic instruction: *"Run the benchmarks in pkg/rulesengine"*.
2. **Use the "Fresh Session" Reset**: After every major milestone, type:
   > *"Commit this, update TRACKER.md, and summarize where we pick up. I will start a fresh session."*
   Then open a new chat, paste the startup prompt from `WORKSPACE.md`, and continue. You will immediately see your credit consumption plummet.
3. **Leverage `/compact`**: In tools that support it (Claude Code, Aider, Cursor), run `/compact` or `/clear` periodically to prune dead tool traces from the buffer.

---

## 4. Summary Cost Matrix: Before vs. After Optimization

| Dimension | Default Unoptimized Agent Workflow | Optimized INTEGIN Standard | Savings |
|---|---|---|:---:|
| **Avg. Session Length** | 30–40 turns (150k+ tokens compounding) | 5–8 turns per micro-sprint | **85%** |
| **Input Token Pricing** | Full standard rate ($3.00/M) | Prompt Cached ($0.30/M) | **90%** |
| **File Inspections** | Full-file dumps (800+ lines) | Targeted line slices (`StartLine/EndLine`) | **75%** |
| **Output Volume** | 1,500 words of conversational restatement | Concise pointers to disk artifacts | **70%** |
| **Blended Spend** | **$15.00 – $40.00 per feature** | **$0.75 – $2.50 per feature** | **~90% NET SAVINGS** |

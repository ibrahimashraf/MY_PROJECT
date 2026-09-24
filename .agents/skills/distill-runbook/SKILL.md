---
name: distill-runbook
description: 'Turn a debate verdict, session learning, or expert walkthrough into a reusable .agents/skills/ checklist. Invoke with /distill. Method only — no integrations, no external calls. Replaces the over-specific upstream colleague-distiller (titanwings/distilly, Feishu/DingTalk-bound) with a generic distilled-here workflow.'
license: MIT
metadata:
  tags: "Distillation, Skills, Knowledge-capture"
  category: "productivity"
---

# distill-runbook

Distill how someone (you, a debate, an expert session) thinks into a skill others can run. Process over product: the output is always a small `.agents/skills/<name>/SKILL.md`, never a framework.

## When to distill

- A council/triad verdict produced reusable criteria (not just a one-off answer).
- You solved something twice and the second time was still slow.
- An operator runbook lives only in someone's head.
- Do NOT distill: one-off facts (use beads `remember` or TRACKER), secrets, personal data.

## Protocol (`/distill <source>`)

1. **Extract** — from the source (verdict transcript, session log, interview), list: triggers (when this applies), checks (ordered, each falsifiable), anti-patterns (what failure looks like), one worked example.
2. **Compress** — cut to one screen. Delete anything that restates another skill. If it overlaps an existing skill, append a delta section there instead of a new skill.
3. **Shape** — write `SKILL.md` with frontmatter (`name`, `description` starting with when-to-use, `license: MIT`, `metadata.tags`) + Commands + Rules + Upgrade-path (`// lean-ctx:` ceiling note when relevant).
4. **Verify** — re-run the source scenario against the new checklist. If the checklist would not have changed the outcome, delete it.
5. **Record** — log the new skill in TRACKER with source link (upstream repo or verdict date).

## Rules

1. Smallest skill that works: one trigger, one checklist, one example.
2. Never copy upstream skill files wholesale — extract the method, write our words (license firewall).
3. Every skill names its upgrade path; speculative generality is a deletion candidate (see `ponytail`, `simplify-codebase`).

## Upgrade path

// lean-ctx: manual protocol; automate extraction (transcript → draft skill) only after 5+ manual distillations prove the pattern.

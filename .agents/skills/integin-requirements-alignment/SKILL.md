---
name: integin-requirements-alignment
description: Clarify an ambiguous feature or design request and record shared terminology and decisions. Use before substantial INTEGIN changes when requirements, domain language, or scope are unclear.
---

# Requirements Alignment

Use this skill to reduce misunderstanding before implementation. Keep the owner responsible for decisions and do not publish, commit, or mutate external systems during clarification.

## Process

1. Read the nearest project instructions, `CURRENT_STATE.md`, `CONTEXT.md`, and relevant decision records when they exist. Identify the project’s protected boundaries and the requested outcome.
2. State the decision to be made, the known constraints, and the unresolved questions. Ask focused questions in small batches, prioritizing questions that change the design.
3. Build a concise glossary as terms become stable. Prefer the project’s existing vocabulary and flag conflicts instead of silently renaming concepts.
4. Record durable decisions in the project’s established documentation location. Use an ADR only for a consequential choice; do not create documentation merely for ceremony.
5. Summarize the agreed problem, desired behavior, acceptance boundaries, non-goals, and remaining uncertainty. Ask the owner to confirm the summary before implementation.

## Guardrails

Do not invent requirements to fill gaps. Do not treat a plausible implementation as an approved decision. Do not expose secrets or protected fixtures while gathering context. If the user cannot answer a decision, turn it into a bounded questionnaire or research task rather than guessing.

## Completion evidence

The skill is complete when the owner has a concise, confirmed statement of the problem, terminology, scope, acceptance expectations, and unresolved decisions. The record must distinguish confirmed facts from assumptions.

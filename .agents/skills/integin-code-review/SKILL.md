---
name: integin-code-review
description: Review a change against its fixed comparison point and originating requirements. Use before merging or committing meaningful INTEGIN code changes, pull requests, or work in progress.
---

# Code Review

Review evidence, not confidence. Keep review read-only and leave mutation and final ownership with the project owner.

## Process

1. Pin the fixed point supplied by the owner and verify that it resolves. Capture the diff and commit range once. Stop if the comparison is empty or ambiguous.
2. Identify the originating requirement, issue, or spec. If none exists, report that the specification axis cannot be fully assessed.
3. Run two independent passes: **Standards**, covering project rules, maintainability, security, and known smells; and **Spec**, covering requested behavior, acceptance criteria, edge cases, and scope.
4. Report each finding with severity, exact evidence, affected behavior, confidence, and a concrete remediation. Separate findings from questions and non-blocking observations.
5. Summarize what was checked, what was not checked, and whether unresolved high-severity blockers remain. Do not commit, merge, publish, or close an issue from this skill.

## Review rules

Prefer behavioral evidence over style preference. Do not invent a missing requirement. Treat tests, type checks, linters, and runtime checks as distinct evidence, and state their exact commands and limitations. Use an independent reviewer or the project’s delegation workflow when the change is consequential.

## Completion evidence

The review contains the fixed point, spec source or its absence, standards sources, both review results, reproducible file/line evidence, unresolved risks, and an owner-facing recommendation.

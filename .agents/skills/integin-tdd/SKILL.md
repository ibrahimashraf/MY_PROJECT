---
name: integin-tdd
description: Build or repair behavior through a disciplined test-first loop at agreed public seams. Use when an INTEGIN change should be developed test-first or requires integration coverage.
---

# Test-First Development

Use a small red → green → refactor loop that tests behavior through stable public boundaries.

## Process

1. Read the project instructions, domain context, and relevant decisions. Identify the user-visible behavior and the public seam that can observe it.
2. Agree on the seam and the acceptance example before writing the test. Prefer one high-level seam over many implementation-facing tests.
3. Write one failing test using an independent expected result. Confirm that it fails for the intended reason.
4. Implement only the smallest change that makes that test pass. Avoid speculative features and broad refactors.
5. Repeat one vertical slice at a time. Keep tests focused on behavior, not private methods, internal collaborators, or incidental data access.
6. Refactor only after the slice is green, then rerun the focused checks and the relevant broader checks.

## Test quality rules

Reject tests that mock internal details, recompute the expected value using the same algorithm as the implementation, or predeclare a large horizontal test suite before the behavior is understood. If no correct seam exists, record the architectural limitation instead of creating a misleading test.

## Guardrails

Do not modify protected acceptance or runtime boundaries without the project’s required authority gate. Do not commit or publish merely because a test passes. State exactly which tests ran and what they prove.

## Completion evidence

Each slice has a behavior-focused test that was observed failing before the implementation and passing afterward. The final note identifies the tested seam, commands run, remaining untested risks, and any architecture limitation.

---
name: integin-debugging-loop
description: Diagnose difficult, intermittent, slow, or regression-prone defects through a red-capable feedback loop. Use when an INTEGIN behavior is broken, failing, throwing, or unexpectedly slow.
---

# Debugging Loop

Do not begin with a theory. Begin by making the reported symptom observable and repeatable.

## Process

1. Read the relevant project instructions and domain context. Redact secrets from every command, log, trace, screenshot, and captured artifact before showing or saving it.
2. Build one tight feedback loop that can detect the user’s exact symptom. Prefer a failing test, then a focused HTTP or CLI check, browser check, trace replay, harness, fuzz loop, or bisection as appropriate.
3. Run the loop and confirm that it reproduces the described failure. Minimize inputs, callers, configuration, and steps one change at a time until every remaining element is load-bearing.
4. Write three to five ranked, falsifiable hypotheses. For each, state the observation that would support or reject it.
5. Instrument only the boundaries that distinguish the hypotheses. Change one variable at a time, tag temporary logs, and remove them afterward.
6. Fix at the correct seam, write the regression test before the fix when a reliable seam exists, rerun the minimized and original scenarios, and clean up all temporary artifacts.

## Guardrails

Do not hypothesize indefinitely without a red-capable loop. Do not print credentials, protected fixtures, authentication headers, or private payloads. Do not add production instrumentation or alter a protected runtime without explicit approval and a documented boundary.

## Completion evidence

Record the reproduction command, observed symptom, minimization result, tested hypothesis, fix, regression evidence, cleanup check, and any missing seam that prevented a trustworthy test.

---
name: integin-prototype
description: Build a disposable prototype to answer one design, state-model, or interface question. Use before production implementation when the behavior or UI direction is uncertain.
---

# Prototype

A prototype is an experiment, not an early production implementation.

## Process

1. State one question the prototype must answer. Choose either a logic/state experiment or a UI exploration; do not mix unrelated questions.
2. Place the experiment near the relevant code or in the project’s designated scratch area, mark it visibly as disposable, and make it trivial to run.
3. Keep state in memory by default. Avoid persistence, production credentials, external side effects, and integration work unless those are the question being tested and the project owner approves the boundary.
4. Surface the relevant state after each meaningful action or variant switch so the owner can inspect what was learned.
5. Compare the result against explicit criteria and record the verdict, trade-offs, and unresolved questions.
6. Transfer only validated decisions into production planning. Delete or isolate the prototype so it cannot be mistaken for production code.

## Guardrails

Do not polish a prototype into production by accident. Do not add hidden persistence, telemetry, secrets, or irreversible external operations. Do not claim production readiness from a successful experiment.

## Completion evidence

The prototype has a named question, a reproducible run instruction, visible state or variants, a recorded verdict, and a clear disposition: delete, archive as an experiment, or replace with production implementation.

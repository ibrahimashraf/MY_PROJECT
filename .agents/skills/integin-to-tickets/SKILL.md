---
name: integin-to-tickets
description: Break an approved INTEGIN plan or specification into demonstrable vertical-slice tickets with explicit dependencies. Use when work needs multiple independently verifiable implementation slices.
---

# Dependency-Aware Tickets

Turn an approved plan into work items that can be implemented and verified independently. Draft first; publish only after owner approval.

## Process

1. Read the approved spec or plan, current project state, domain glossary, and relevant decisions. Do not create tickets from an unconfirmed idea.
2. Identify any enabling refactor that must precede the feature. Separate broad mechanical migrations from ordinary vertical slices.
3. Draft narrow, complete slices that cross the relevant layers and produce demonstrable behavior. Keep each ticket small enough for one fresh context and large enough to be meaningful on its own.
4. For every ticket, write its title, end-to-end outcome, acceptance criteria, and blockers. Keep the dependency graph minimal: a ticket should list only true gates.
5. For wide refactors, use an expand → migrate in bounded batches → contract sequence so the system can remain verifiable between steps.
6. Present the numbered draft to the owner and ask whether the granularity and blocking edges are correct. Revise until approved.
7. After approval, publish to the configured tracker or write one local Markdown file per ticket in dependency order. Do not modify a parent issue or close existing work without separate authorization.

## Ticket structure

Each ticket must include **What to build**, **Acceptance criteria**, **Blocked by**, and **Status**. Describe user-visible or externally verifiable behavior rather than a layer-by-layer file checklist.

## Guardrails

Do not publish tickets during the drafting phase. Do not assign labels, create native blocking links, alter tracker state, commit, or begin implementation without owner approval. Do not place secrets or protected fixture details in tickets. If the spec is incomplete, stop and return the missing decisions instead of manufacturing scope.

## Completion evidence

The owner has approved the ticket count, slice boundaries, acceptance criteria, and blocker graph. The published or local ticket set is numbered in dependency order, references the confirmed spec, and states what remains out of scope.

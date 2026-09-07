---
name: integin-to-spec
description: Turn an already-discussed INTEGIN feature request into a confirmed implementation specification. Use when the problem and solution are sufficiently understood and need a durable spec before ticketing.
---

# Specification Synthesis

Synthesize what is known; do not restart the requirements interview. If major decisions remain ambiguous, use requirements alignment first.

## Process

1. Read the current project state, relevant domain terminology, ADRs, research notes, and the conversation or issue that defines the request.
2. State the user’s problem and the intended solution without adding unconfirmed behavior.
3. Describe user stories, implementation decisions, interfaces, data or API contracts, testing decisions, out-of-scope items, and further notes. Prefer concepts and module responsibilities over brittle file paths or code snippets.
4. Identify the highest useful public testing seam. Prefer an existing seam; propose a new one only when necessary. Confirm the seam with the owner before implementation.
5. Mark every item as confirmed, assumed, or unresolved. Resolve or explicitly defer unresolved items before the spec is considered ready.
6. Save the spec in the project’s established documentation location. Only publish to a tracker after owner review and explicit authorization.

## Spec structure

Use these headings: **Problem Statement**, **Solution**, **User Stories**, **Implementation Decisions**, **Testing Decisions**, **Out of Scope**, and **Further Notes**.

## Guardrails

Do not invent user stories to make the document look complete. Do not publish issues, labels, comments, branches, or code from this skill. Do not include secrets, private fixture content, or unverified claims. A spec is not approval to implement; it is the artifact that makes approval reviewable.

## Completion evidence

The spec has a confirmed problem and solution, explicit scope boundaries, implementation and testing decisions, a named seam, a list of assumptions and unresolved items, and owner approval before any tracker publication.

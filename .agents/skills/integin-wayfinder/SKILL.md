---
name: integin-wayfinder
description: Map and resolve decisions for an initiative too large for one agent session. Use before major cross-cutting INTEGIN work with unresolved architecture, research, or scope decisions.
---

# Large-Initiative Decision Map

Use this workflow to find a route to a defined destination before implementation. The map resolves decisions; it does not silently become an execution plan.

## Decide whether to use it

Use a map only when the initiative spans multiple sessions or contains several decisions whose order is not yet clear. If the work fits one session or the route is already obvious, use the ordinary project plan instead.

## Chart the map

1. Define the destination in one or two sentences. The destination determines scope.
2. Record standing constraints, relevant project skills, and decision owners.
3. List decisions that can be stated precisely now. Keep unresolved areas that cannot yet be phrased as precise questions in a separate **not yet specified** section.
4. Create decision records with a question, type, dependencies, owner, and evidence required. Use types such as research, prototype, discussion, or prerequisite task.
5. Build the dependency graph only after the decision records have stable identifiers. The frontier is the set of open, unclaimed records with all blockers resolved.
6. Stop charting after the map and initial frontier are clear. Do not start implementation from this workflow.

## Resolve the map

Resolve at most one ordinary decision per session. Claim it before working, inspect only the context needed, record the answer with evidence, and update the map’s decisions-so-far index. Promote newly precise questions from **not yet specified** into decision records. Move work beyond the destination into **out of scope** rather than allowing it to expand the map.

## Guardrails

Do not create tracker issues, branches, comments, or assignments without owner authorization and a configured target. Prefer project-local Markdown records when no tracker is approved. Never place secrets, protected fixtures, or authority-bearing actions in the map. A decision record is not proof of implementation readiness; it must name remaining validation.

## Completion evidence

The map is complete when the destination is explicit, no in-scope decision remains unresolved before implementation, dependencies are visible, out-of-scope work is recorded, and the next handoff can begin without reopening the entire discovery conversation.

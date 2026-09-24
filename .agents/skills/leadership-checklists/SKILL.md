---
name: leadership-checklists
description: 'Human-leadership checklists: 1-on-1s, approval-gated delegation, team feedback, and lightweight org notes. Invoke with /lead. Content checklists only — no tracking backend, no integrations. Fills the Leadership/Management human side (arc-kit/teamai-cli are shapes only, license-blocked).'
license: MIT
metadata:
  tags: "Leadership, Management, 1-on-1, Delegation"
  category: "productivity"
---

# leadership-checklists

Checklists for leading humans. No software to install, no data stored beyond what the user pastes in-session. Pairs with `council-triad --triad decision` for hard calls and `opencode-delegate` review discipline for task handoffs.

## Commands

- `/lead 1on1 — <context>` — emit a 1-on-1 prep sheet: (1) 3 openers that are not status questions, (2) recognition slot (one specific recent contribution to name), (3) growth thread (one skill + one stretch opportunity), (4) blockers the lead can actually remove, (5) agreed next action with owner + date. Ends with: "What NOT to do: cancel twice in a row, turn it into a status meeting, give feedback with no example."
- `/lead delegate — <task + person>` — approval-gated delegation plan: outcome (observable done-state), constraints (budget/time/quality bounds), authority level (recommend / decide-inform / decide-autonomous), checkpoints (dates, not vibes), revert criteria (when to pull it back), RACI in one table. Rule: never delegate accountability for irreversible risk — flag it if detected.
- `/lead feedback — <situation>` — SBI sheet (Situation-Behavior-Impact) + one curious question + one concrete request. Rules: one topic per conversation, behavior not character, within 48h of the event, private first.
- `/lead org — <change>` — change checklist: who is affected, what changes for them Monday morning, what stays the same (say it explicitly), rollback story, comms order (affected first, observers second, public last).

## Rules

1. Human judgment stays with the human — checklists prepare, never decide personnel outcomes.
2. Everything is drafts and sheets for the user to run themselves; nothing is sent or filed anywhere.
3. Keep each sheet to one screen. Cut anything that does not change what the user says or does in the room.
4. Close with exactly one next action (owner + date).

## Sources (ideas only, license firewall)

- Delegation/approval shape: `darrenhinde/OpenAgentsControl`, `amElnagdy/delegate-skills` (review-then-land).
- Management vocabulary: `dwmkerr/hacker-laws` (Brooks, Conway) — cite, don't enforce.
- No code reused from NOASSERTION/AGPL harnesses (`arc-kit`, `teamai-cli`, `agent-teams-ai`).

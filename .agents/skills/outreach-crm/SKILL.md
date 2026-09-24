---
name: outreach-crm
description: 'Lightweight relationship log + approval-gated outreach drafting. Invoke with /outreach. Contacts live in a local JSON file; drafts are shown for human approval and never sent automatically. Fills the Networking/Relationships gap (no upstream CRM adopted).'
license: MIT
metadata:
  tags: "Networking, CRM, Outreach, Local-only"
  category: "life-admin"
---

# outreach-crm

A minimal CRM for people, not pipelines. Local JSON log + drafted (never sent) outreach. The human always presses send elsewhere.

## Storage

- File: `contacts.json` in the working directory (create if missing).
- Shape: `{"contacts": [{"name": "...", "context": "met at X / JD link", "last_contact": "YYYY-MM-DD", "next": "follow up re Y", "notes": "..."}]}`.
- Never transmit contacts anywhere. Never enrich from external sources without being asked.

## Commands

- `/outreach add <name> — <context>` — append a contact. One line confirm.
- `/outreach list` — table of contacts sorted by stalest `last_contact` first. Flag any untouched >30 days.
- `/outreach draft <name> — <goal>` — draft a short message (≤100 words) grounded ONLY in that contact's `context`/`notes` plus the stated goal. Print the draft under a `--- AWAITING APPROVAL ---` banner with: what it references, what it asks, what it deliberately omits.
- `/outreach log <name> — <what happened>` — update `last_contact` to today, append to `notes`.

## Rules

1. NEVER send, schedule, or post anything. Drafts only. If asked to send, refuse in one line and restate the draft for copy-paste.
2. Never invent facts about a contact — only file contents + the current prompt.
3. Keep drafts short, specific, and easy to decline (low-pressure ask, clear opt-out).
4. One next action at the end (pairs with `single-next-action` style).

## Upgrade path

// lean-ctx: flat JSON file, fine under ~5k contacts; move to SQLite FTS when search slows or the user asks for tagging/filters.

---
name: envelope-budget
description: 'Envelope budgeting from a local bank/Actual CSV export. Invoke with /budget. Reads a CSV the user points at, assigns spending to envelopes, reports over/under per envelope. Local-only: finance data never leaves the device. Fills the personal-budgeting gap (Actual/Firefly are apps, not skills).'
license: MIT
metadata:
  tags: "Budgeting, Finance, CSV, Local-only"
  category: "life-admin"
---

# envelope-budget

Thin CSV → envelope wrapper. The user exports from their bank or Actual; this skill categorizes and reports. No accounts connected, no data transmitted.

## Inputs

- `/budget init` — print the default envelope set and write `envelopes.json` in the working directory: `{"envelopes": {"Housing": 0, "Food": 0, "Transport": 0, "Health": 0, "Fun": 0, "Savings": 0}, "currency": "YER"}`. User edits amounts to set monthly targets (0 = track only).
- `/budget review <csv-path>` — read the CSV (expects a header row with date/description/amount columns; tolerate bank variants by matching column names case-insensitively: `date|transaction date`, `description|memo|payee`, `amount|sum`). Assign each row to an envelope by keyword rules below, then report per-envelope: spent vs target, over/under, top 3 transactions. Unmatched rows go to `Unsorted` — listed, never force-fit.

## Keyword rules (edit in-session when the user corrects)

- Housing: rent, landlord, landlord name, mortgage, utilities, electricity, water.
- Food: restaurant, grocery, market, bakery, coffee, delivery.
- Transport: fuel, petrol, taxi, bus, fare, mechanic, spare parts.
- Health: pharmacy, clinic, hospital, gym.
- Fun: cinema, games, subscription, travel, gifts.
- Savings: transfer to savings, deposit (positive amounts to savings accounts).

When the user reassigns a transaction, remember the keyword → envelope mapping for the rest of the session and offer to persist it to `envelope-rules.json`.

## Rules

1. Read-only on the CSV. Never modify, move, or delete the user's export.
2. All amounts stay on-device; never paste finance data into external calls or examples.
3. Report ends with: total in, total out, per-envelope over/under, `Unsorted` count, exactly one next action.
4. No financial advice (no investing/debt recommendations) — categorization and arithmetic only.

## Upgrade path

// lean-ctx: keyword rules + one CSV at a time; graduate to Actual Budget API/self-host sync only if the user asks and accepts the setup cost.

---
name: health-log
description: 'Log meals, workouts, sleep, and focus sessions to a local JSON file and produce weekly summaries. Invoke with /health-log. Local-only: nothing leaves the device, no cloud, no external calls. Fills the Health & Wellness gap (no upstream skill adopted).'
license: MIT
metadata:
  tags: "Health, Wellness, Logging, Local-only"
  category: "life-admin"
---

# health-log

Local-only health logging. No network calls, no cloud sync, no third-party services. All data stays in one JSON file the user owns.

## Storage

- File: `health-log.json` in the current working directory (create if missing).
- Shape: `{"entries": [{"date": "YYYY-MM-DD", "type": "meal|workout|sleep|focus|weight|note", "detail": "...", "value": null}]}`.
- Never write outside the working directory. Never transmit entries anywhere.

## Commands

- `/health-log <type> <detail>` — append one entry for today. Types: `meal`, `workout`, `sleep`, `focus`, `weight`, `note`.
  - Ex: `/health-log workout 30min run, 5km` → appends `{date: today, type: workout, detail: "30min run, 5km"}`.
  - Ex: `/health-log sleep 7.5h, late screen time` → appends sleep entry.
- `/health-log week` — read the file, summarize the last 7 days: counts per type, totals (workout minutes, avg sleep), one-line trend, single next action. No diagnosis, no medical advice — logs and patterns only.
- `/health-log export` — print the raw JSON so the user can back it up themselves.

## Rules

1. One entry per command; confirm in one line (`Logged workout 2026-09-24.`).
2. Never infer or fabricate entries — only what the user stated.
3. Never give medical, nutritional, or training prescriptions. Summarize what is logged; suggest seeing a professional for anything beyond patterns.
4. If `health-log.json` is missing or corrupt, create/fix it and say so in one line.
5. Weekly summary ends with exactly one next action (pairs with `single-next-action` style).

## Upgrade path

// lean-ctx: flat JSON file, fine under ~10k entries; move to SQLite when weekly summary slows or the user asks for charts.

# Quiet Signal — Pre-delivery UI Checkpoint — 2026-08-29

**Location:** `integin-pilot-source/quiet-signal` (Vite + React 19 + TypeScript 6)
**Build:** `tsc -b && vite build` — `TSC_OK`, `LINT_OK` (oxlint 0 warnings), `dist/index-CRHA4AHC.css 4.41kB`, `index-D1jTeCFm.js 195.30kB gzip 61.74kB` — **PASS** (no blocking chunk warnings, bundle < 500k advisory threshold)
**Go:** `go vet ./...` `GOVET_OK` (dual-service additions preserved `blocking=false` boundary)

## What was delivered

* **Advisory-only boundary** — banner `role=note` states: cannot set verdicts, approve inspections, issue certificates, validate calibration, authorize requests, accept sync transactions, or expand public QR projections. `blocking=false` always.
* **Evidence-linked signals** — 2 mock signals (MONITORING/REGULATION) with `confidence`, `rationale`, `evidenceRefs` (kind/id/label), `limitations`, `model` provider/version, `blocking=false`
* **Reasoning traces** — per-signal expandable `ol.trace` (lens, versioned contract v1, `internal/advisory` normalization)
* **Health/Canary** — `baseline v1.26.5` / `candidate v1.27.0-rc1` / `healthy` / `traffic 12%` / `rollback ready` — injectable transport, explicit env `VITE_ADVISORY_API_URL`, no embedded credentials
* **Responsive:** `grid 320px+1fr` → `@media (max-width:900px) 1fr` stack, `topbar` wraps, tests: desktop and mobile render the same signals/traces without mutation hooks

## Contract

* **Versioned HTTP:** `POST /v1/advisory` with `zone: MONITORING|REGULATION` (AI-free zones rejected)
* **Transport:** injectable `fetch` wrapper, `VITE_ADVISORY_API_URL` (empty = mock), timeouts/normalization in `internal/advisory` (Go) and mirrored in UI mock
* **View model:** secondary advisor data separate from primary aggregates — no secret/prompt/private-credential exposure, only approved fields (`confidence`, `rationale`, `evidenceRefs`, `model`, `limitations`)

## How to run

```powershell
cd integin-pilot-source/quiet-signal
npm install
# mock (no live service):
npm run dev
# live (explicit env):
$env:VITE_ADVISORY_API_URL="http://127.0.0.1:8081"; npm run dev
npm run build; npx tsc --noEmit; npx oxlint
```

## Checkpoint evidence

* `src/App.tsx:1` — `AdvisoryBoundary`, `HealthPanel`, `SignalCard` with `zone`/`confidence`/`evidence`/`blocking=false`/`trace`
* `src/App.css:1` — `grid` + `@media` responsive, `banner` advisory-only
* `dist/` — production build 17 modules, `195kB` (61k gzip)
* `ai_service` `main.py` `test_main.py` `4 passed` in sandbox (Python service already `complete` per `task_plan.md:9`)

**Next:** present this checkpoint + `ai_service` handoff; do not claim live integration without explicit `VITE_ADVISORY_API_URL` and opt-in `go test` mocks.

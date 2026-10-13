# Phase 10 — Cross-Service Handoff Checkpoint — 2026-08-29

**Base:** `integin-pilot-source` `2c7d1c1` (AI observations generic + advanced NDT + trust+quality) on top of `d400859` (Phase 8 Quiet Signal) `8159b6c` (whole A)

## Validation

* **Go:** `go vet ./...` `GOVET_OK`, `go test -count=1 ./...` `GOTEST_OK` (env stripped, all `ok`, `workorderpg`/`workorderhttp`/`storage`/`sync`/`syncstate` PASS)
* **UI:** `quiet-signal` `npx tsc --noEmit` `TSC_OK`, `oxlint` `0 warnings`, `vite build` `17 modules` `201.56kB gzip 63.54kB` (<500k) — `CHECKPOINT_2026-08-29.md`
* **Python:** `ai_service` `pytest 5 passed` (`HEALTH` `AI_FREE_ZONE 403` `NDT_DEFECT`/`LIFTING_DEFECT` generic `blocking=false`)

## Persistent records

* `task_plan.md:17` Phase 8 `complete` (Quiet Signal), Phase 9 `complete` (Python advisory), Phase 10 `in_progress` → this checkpoint closes it
* Commits: `705fb6a` Ticket 02 → `2ac15c4` A1 Keycloak → `223ab5c` A2 storage → `12e5937` A3 replay → `7f21053` A4 sync → `72a26ac` A5 docs → `6157b5c` A6 field-app → `8159b6c` A7 candidates → `d400859` Phase 8 → `9dc8299` AI design → `2c7d1c1` AI generic + advanced NDT + trust+quality
* Backups: `backups/a1-keycloak-*`, `a2-storage-*`, `a3-replay-*`, `a-final-*`, `clean-cloth-pre-reset-*`

## Delivery — working console + service instructions

**Console (Quiet Signal):**
```powershell
cd integin-pilot-source/quiet-signal
npm install
npm run dev          # mock, no live API
$env:VITE_ADVISORY_API_URL="http://127.0.0.1:8081"; npm run dev  # live
npm run build; npx tsc --noEmit; npx oxlint
```
Advisory-only banner, evidence-linked signals, reasoning traces, health/canary, inspection photo opt-in (Upload directly / Get AI observations — generic for any INTEGIN type, PAUT/EDDY_CURRENT/PECT/TOFD/RT etc., bbox/heatmap placeholder, confidence slider, multi-photo, audit chip `suggested/attached/dismissed`, model card `provider/model/version/blocking=false`)

**Service (Python):**
```powershell
cd integin-pilot-source
pip install -r ai_service/requirements.txt; pip install httpx
python -m pytest ai_service/test_main.py -v  # 5 passed
uvicorn ai_service.main:app --host 127.0.0.1 --port 8081  # /healthz 200, POST /v1/advisory version:v1 zone:any advisory (NDT_DEFECT/LIFTING_DEFECT/MONITORING — AI-free 403) blocking=false
```
Contract `v1` — `AI_FREE_ZONES` deny-list only, all other zones advisory via generic fallback, no `GRANT`/RLS, `VITE_ADVISORY_API_URL` explicit, integration tests opt-in/mocked.

**Next:** owner presents this checkpoint + `quiet-signal/CHECKPOINT_2026-08-29.md` + `ai_service` handoff; no new `GRANT`/RLS/deployment is claimed.

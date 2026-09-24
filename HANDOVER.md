# HANDOVER — INTEGIN Pilot Acceptance (2026-09-22, matrix closed 2026-09-23)

## Mission Status
- **Durability proof: COMPLETE** — app re-provisioned, NEW sync_receipt created post-relaunch (`tx-1790093307801923-1819` seq 1 APPLIED 16:12:43Z), both devices `last_accepted_sequence=1`.
- **Matrix: COMPLETE (2026-09-23)** — `seed` OK then `exercise` EXIT 0 against appliance stack: sync APPLIED/DUPLICATE/HELD/CONFLICT/SECURITY_FAILURE + evidence APPLIED/DUPLICATE/CONFLICT/SECURITY_FAILURE. Restart recovery verified (receipts APPLIED=3/HELD=3, 2 evidence objects survive full data-plane restart).
- **Commit: DONE** — `c997e51` (4 files: 3 modified + new test). Uncommitted: `deployments/docker-compose.appliance.yml` OIDC audience default (see 2026-09-23 note).
- **go.mod corruption: FIXED** — stray `go 1.2227.1` / `module testintegin` / `test.txt` reverted; tree verified clean.

## 2026-09-23 Matrix Notes (appliance stack)
- DB via pgcat `127.0.0.1:6432/integin_appliance` + `default_query_exec_mode=simple_protocol` (fixes SQLSTATE 26000).
- OIDC via appliance-casdoor `127.0.0.1:18181`, app-built-in client creds, `client_credentials` enabled in `grant_types` (casdoor-postgres volume state). Server `INTEGIN_OIDC_AUDIENCE` must equal token aud (app-built-in client_id); compose default updated. Token iss `http://appliance-casdoor:8000`.
- Server `127.0.0.1:8080` (`/healthz`); matrix guard requires `INTEGIN_SERVER_URL=http://127.0.0.1:18080` → TCP forwarder `port18080.py` (background) bridges 18080→8080.
- Tenant/secret must match server: `INTEGIN-integration-tenant` / server sync secret. Order: seed → restart server (registers authority) → exercise. Runner: `C:\Users\hima3\AppData\Local\Temp\opencode\run_matrix.ps1`.

## 2026-09-23 Phase 4 Gates (Android only; Apple/desktop deferred)
- Rig: host `integin-server.exe` (built from source) on `127.0.0.1:8081` with `/local/provision` (loopback guard forbids it on the `0.0.0.0` appliance container); device reverse `tcp:8080→tcp:8081`. OIDC boot via `stub_oidc.py` (RSA JWKS) on 18180 — device flow uses HMAC only.
- Gate 1 PASS: `TrickyStoreService: TEE verification successful`, `/local/provision` 200, UI "Trusted and valid" (30-min authority).
- Gate 2 PASS (airplane ON): response recorded (required→0), queue → "1 queued" → Sync → `POST /sync` 200 → "applied or duplicate recorded". Receipt `field-528c…` seq 1 APPLIED `tx-1790181857180729-8125`. Transport note: USB adb reverse bypasses airplane radios; offline-sign+queue behavior genuine.
- Gate 4 PASS (`adb reboot`): rebooted, `Trusted and valid` with fresh authority, 0 queued, receipt history intact, no corruption. Note: provision mints a fresh device id per enrollment (`field-9700` → `field-528c`).
- Gate 5 PASS: "Re-submit last receipt" → `/sync` 200 idempotent replay, single receipt row, no dup side effects.
- Biometric/PIN fallback (Gate 3) BLOCKED by code gap, not hardware: live submit path uses software Ed25519 (`SecureDeviceKeyStore.loadOrCreate`), never an auth-bound hardware key — `generateAttestedKey` has zero callers and `inspection_submission_service` (the only `AUTH_REQUIRED` retry) is off the live path. So `AUTH_REQUIRED` can never fire on this build. Additionally `biometricOnly: true` contradicts the plan's PIN-fallback claim. Device HAS fingerprint+face HALs. Needs Flutter rebuild to wire auth-bound keys + device-credential fallback (no Flutter SDK on this workstation).
- Photo cycle: SERVER LEG PROVEN LIVE — TUS enabled (`INTEGIN_TUS_SCRATCH_DIR`), full `create→append→complete→key` against `/uploads` with real OIDC bearer passes (unit tests in `internal/storage` also green). DEVICE LEG BLOCKED on two items needing Flutter rebuild: (a) no OIDC token wired into `TusClient.authToken` (`main.dart:200` TODO) so the app gets 401; (b) this ROM's `OneShotCamera` (sole `IMAGE_CAPTURE` handler) shows no confirm affordance after shutter, so `image_picker` returns null. Camera activity exits cleanly on Back (verified, no stuck state).

## 2026-09-23 MI 9 Stuck-Outbox Root Cause + Fix (UNCOMMITTED)
- Symptom: old records never sync while new ones do; only cache-clear/uninstall helps (which DESTROYS the work).
- Root cause (traced, deterministic): every fresh provision mints a NEW authority id (`newAuthorityID()`, 30-min lifetime). `SyncGuard.check` rejects any entry whose device/authority differs from current (`sync_guard.dart`) → flush marks it `securityFailure`, and `pending()` never returns failure states — terminal, invisible, unretryable. Drift's pending query additionally drops `HELD`. No re-sign/re-queue path existed.
- Fix (in working tree, VERIFIED with Flutter 3.47.5 at `D:\LEIMS-TOOLS\flutter-3.47.0`): `FieldAppController.reconcileOutboxIdentities()` runs after `restoreOutbox()` in `main.dart`: re-binds all unacknowledged entries (queued/held/failed) to the current identity, re-sequences contiguously, re-signs, re-seals the chain, returns them to `queued`. Tx ids stable → server replays idempotent. Files: `outbox.dart` (+`replace`), `persistent_outbox.dart`, `drift_outbox_store.dart`, `app_database.dart` (+`replaceOutboxMutation`, HELD in pending), `field_app_controller.dart`, `main.dart`, new test `outbox_identity_reconcile_test.dart`, updated `drift_outbox_store_test.dart` (HELD now pending by contract).
- Verification: `flutter analyze` clean on all touched files; new migration tests 2/2 pass (incl. double-check fix: guard assertions must walk sequences in order); surrounding suites (sync_client/guard, persistent/drift outbox, controller, transport, full-flow, domain, provision roundtrip) all pass; FULL suite 206 pass with 3 failures in `00_enrollment_screen_test.dart` that fail identically on the clean tree (pre-existing timeouts, unrelated).
- Bug-fix round 2 (all verified): 3 pre-existing analyzer lints fixed (renamed `00_…` test file, removed dead import + dead `service` assignment); flush-ordering hardened — deterministic `ORDER BY createdAt, transactionId` on drift reads + `_lastAcceptedSequence` re-derived from stored states instead of index-pairing two separate `pending()` reads. FULL suite now **209/209 green** (the 3 enrollment timeouts pass with the file rename + cold cache; nothing else changed).
- Deep dive: server replay semantics verified airtight in `internal/domain/sync/sync.go` — durable DB receipt check returns DUPLICATE *before* device/authority/sequence checks, durable `GetLastAcceptedSequence` survives restarts, server-side HELD explicitly resumable, terminal states never persisted so migration replays always re-validate fresh. Migration hardened: same-user-only (never re-attributes another inspector's work) + signer-keyId fallback (server rejects empty keyId), covered by a 3rd test. `internal/sync/sync.go` is legacy (no importers; HTTP uses `domain/sync`). Enrollment timeouts root-caused to C:-disk-exhaustion pressure (flake; green in isolation and post-cleanup). `go test ./...` clean, `go vet` clean on all sync packages. Sync/outbox suites re-run green 27/27 after final edits.
- Toolchain (2026-09-23, all current): Flutter 3.47.5 / Dart 3.13.0 (`D:\LEIMS-TOOLS\flutter-3.47.0`), AGP 9.4.1, Kotlin 2.4.20, Gradle 9.7.1, compileSdk 37 (platforms 35–37 + BT 36.0.0 installed). `flutter build apk --debug` GREEN (106s) incl. the migration change. Runs on JDK 26.0.1 with benign warnings only (Gradle "newer than known valid version" + native-access) — compatible in practice, not yet certified; only JDK 26 present.
- CRITICAL BUILD GOTCHA: endpoints are `--dart-define` baked (`INTEGIN_SYNC_ENDPOINT`, `INTEGIN_LOCAL_PROVISIONING_ENDPOINT`); a plain `flutter build apk` yields a DEMO-session app that never touches the network (silent, healthy-looking, zero logs). Device builds MUST pass both defines with `http://127.0.0.1:8080/...` (device loopback → adb reverse → host server). Rebuilt+installed with defines, `install -r` preserved data, provision 200, trusted.
- WEB LANE (2026-09-23, Chrome Beta headless CDP): `flutter build web` + `--release` both compile; RELEASE renders (canvas nodes, canvaskit/wasm fetched, 33KB first paint). Debug build stays blank in headless (DDC loads, no JS errors — DWDS/headless artifact; headed instance unaffected). WEB BLOCKER: server sends zero CORS headers (`OPTIONS /sync` → bare 400), so no browser sync/evidence/TUS call can pass preflight — needs a scoped CORS middleware decision before any web E2E. Drift/IndexedDB outbox path (incl. migration) covered by unit tests only so far.
- Phase-4 device is the MI 9 (`59af14c0`) itself. New build (with migration fix) installed on it 2026-09-23 ~22:05, trusted, receipt history intact. Debug note: a timed-out `flutter run` leaves the app waiting for a dead debugger (blank, silent) — force-stop + cold launch fixes it; VM-service stack via forwarded port shows the truth.
- Disk: C: was 0 bytes free → moved `~/.gradle` (8.9 GB) to `D:\gradle-home`; build with `GRADLE_USER_HOME=D:\gradle-home` (run `setx GRADLE_USER_HOME D:\gradle-home` to persist). Do NOT delete `D:\` temp backups (`gitbak-*`, `winlibs`) — other sessions' artifacts.

---

## Critical Infrastructure

| Component | Port/Path | Status |
|-----------|-----------|--------|
| Working adb | `D:\LEIMS-TOOLS\Android\platform-tools\adb.exe` | Device `59af14c0` |
| Acceptance server | `integin-server-provision.exe` PID 7484, `127.0.0.1:8080` | **UP** (health/ready 200) |
| Sync proxy | `sync_proxy.py` PID 14448, `127.0.0.1:18080` → 8080 | **UP** (captures REQ/RES) |
| adb reverse | `tcp:18080 tcp:18080` + `tcp:8080 tcp:18080` | **ACTIVE** |
| Pilot Casdoor | `integin-pilot-casdoor` `127.0.0.1:18180→8000` | **UP** |
| Pilot RustFS | `integin-pilot-rustfs` `127.0.0.1:19000→9000` | **UP** |
| Appliance pgcat | `appliance-pgcat` `127.0.0.1:6432` | **UP** (host-reachable) |
| Appliance postgres | `appliance-postgres` `5432/tcp` (internal only) | **Internal** |

---

## Key Findings

### Blank App Root Cause (RESOLVED)
- App's `app.dill` defines: `http://127.0.0.1:8080/local/provision` + `http://127.0.0.1:8080/sync`
- Missing `adb reverse tcp:8080 tcp:18080` orphaned provisioning → blank screen
- Fixed: both `tcp:8080` and `tcp:18080` reverse → host `18080` (proxy) → server `8080`

### Identity Correction
- Provisioning uses **original** SecureDeviceKeyStore key (`field-9700b9aeac391b79adce8c681eba4b58`), NOT enroll-screen keys
- `deviceId = 'field-' + keyId.substring(0, 32)` (main.dart:87)

### Durability Proof
- Receipts: only 2 total (both APPLIED, seq 1)
- Post-relaunch flush created NEW receipt `tx-1790093307801923-1819` at 16:12:43Z
- `sync_device_state`: both devices `last_accepted_sequence=1`
- `sync_held_transaction`: empty

---

## Matrix Blocker

```
integin-live-matrix seed
→ failed to connect to `user=integin_runtime database=leims`: 127.0.0.1:5432
```

**Root cause**: `INTEGIN_DB_URL` from private env resolves to pilot DB (`integin_runtime/leims` on 5432) which is **not exposed to host**. Acceptance server works — its actual DB connection must differ (likely pgcat 6432).

### Next Session Action Required
1. **Verify acceptance server's actual DB**: Check what INTEGIN_DB_URL the running server (PID 7484) uses. The restore script loads private env, but server's stderr shows old pilot DB error at 17:09:33 — current server may use different connection.
2. **Fix matrix DB URL**: Matrix needs host-reachable DB. Options:
   - Use pgcat 6432 with correct credentials (appliance DB: `integin_appliance`)
   - Extract working DB URL from acceptance server's process env (if accessible)
   - Check private env for a host-reachable DB variable (different key?)
3. **OIDC for evidence**: Matrix requires OIDC token (Casdoor 18180, client `integin-live-matrix` / `matrix-secret-2026` from WORKSPACE.md). Token endpoint: `http://127.0.0.1:18180/api/login/oauth/access_token`

---

## Env Loading Pattern (Opaque Boundary Respected)
```powershell
# Load private env (never print values)
$envFile = 'C:\MY_PROJECT\private\integin-secrets\integin-server.env'
Get-Content -LiteralPath $envFile | ForEach-Object {
    if ($_ -match '^([^#=][^=]*)=(.*)$') {
        [Environment]::SetEnvironmentVariable($matches[1], $matches[2], 'Process')
    }
}
# Override with documented pilot values from WORKSPACE.md (public)
$env:INTEGIN_SERVER_URL = "http://127.0.0.1:18080"
$env:INTEGIN_OIDC_TOKEN_ENDPOINT = "http://127.0.0.1:18180/api/login/oauth/access_token"
# ... etc
```

---

## Files Touched This Session
- `field_app/lib/main.dart` — added `adb reverse tcp:8080 tcp:18080` instruction comment
- `field_app/lib/provisioning/local_provisioning_client.dart` — provisioning client uses 8080 defines
- `field_app/lib/storage/app_database.dart` — DB schema updates
- `field_app/test/provisioned_session_roundtrip_test.dart` — NEW test

---

## Quick Commands for Next Session
```bash
# Check acceptance server's actual env (PID 7484)
# Check proxy log
tail -f C:\Users\hima3\AppData\Local\Temp\opencode\sync_proxy.log

# Test pgcat connectivity
"SELECT current_database();" | docker exec -i appliance-pgcat psql "postgres://integin_owner:integin_sovereign_db_secret@127.0.0.1:6432/integin_appliance?sslmode=disable" -P pager=off

# Run matrix with corrected DB URL (once known)
cd C:\MY_PROJECT\integin-pilot-source
$env:INTEGIN_DB_URL = "postgres://integin_owner:integin_sovereign_db_secret@127.0.0.1:6432/integin_appliance?sslmode=disable"
# ... load rest of env ...
& C:\Users\hima3\AppData\Local\Temp\opencode\integin-live-matrix.exe seed
& C:\Users\hima3\AppData\Local\Temp\opencode\integin-live-matrix.exe exercise
```
## 2026-09-24 Ordered Batch (CORS, upload tokens, credential fallback — all verified)
1. **Scoped CORS**: `INTEGIN_CORS_ALLOWED_ORIGINS` (exact-match, fail-closed default, no credentials, `Vary: Origin`); preflight 204 live-verified for listed origin, evil origin gets no headers. `internal/server/cors.go` + tests.
2. **Provision-bound upload tokens**: `/local/provision` mints HMAC-bound `upload_token` (expires with authority, in `localprovision` package to avoid import cycles); `/uploads` accepts OIDC JWT *or* token; app threads it `session → TusClient(authToken:)` with legacy-cache compat. Live E2E: provision → token → TUS create/append/complete → key, zero OIDC involved.
3. **Device-credential fallback**: `_retryWithBiometric` now `biometricOnly: false` (soiled-glove PIN/pattern path) + full-path unit test. Full hardware-key migration (auth-bound TEE keys in the live path) remains Phase-5 scope: needs key-migration design + on-device verification (MI 9 currently detached).
- Verification: `flutter analyze` 0 issues, app suite **212/212**, `go vet` + `go test` green on server/localprovision/serverboot. Host 8081 server rebuilt with all three and live-verified.

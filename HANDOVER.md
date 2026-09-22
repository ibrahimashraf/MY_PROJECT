# HANDOVER — INTEGIN Pilot Acceptance (2026-09-22)

## Mission Status
- **Durability proof: COMPLETE** — app re-provisioned, NEW sync_receipt created post-relaunch (`tx-1790093307801923-1819` seq 1 APPLIED 16:12:43Z), both devices `last_accepted_sequence=1`.
- **Matrix: BLOCKED** — `integin-live-matrix seed` fails on DB connection. Private env `INTEGIN_DB_URL` appears to point at pilot DB (`integin_runtime/leims` on 5432, not exposed). Acceptance server (PID 7484, 8080) runs health/ready 200 — its actual DB connection needs verification.
- **Commit: DONE** — `c997e51` (4 files: 3 modified + new test).

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
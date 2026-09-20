# Casdoor IdP Migration: Structured Implementation Brief for OpenCode

## 1. Objective
Replace the heavy Java Keycloak container (`integin-pilot-keycloak`, 520 MB RAM, 35s boot) with a Go-native Casdoor deployment (`casdoor/casdoor`, ~110 MB RAM, 2s boot, 100% Apache 2.0).

---

## 2. Architectural Invariants & Boundaries
1. **Zero Domain DB Contamination**: Casdoor MUST run on its own isolated PostgreSQL container/volume (`integin-pilot-casdoor-postgres`) and its own Docker network (`integin-casdoor-pilot-net`). Never point it to `integin_dev` or through PgCat.
2. **Strict RS256 Ceiling**: Casdoor application certificate must be configured with an RSA key pair (`RS256`) so that tokens validate cleanly against `internal/oidcauth/validator.go`.
3. **Contracts Untouched**: `internal/identity/contracts.go` and `internal/identity/postgres.go` remain unmodified. Token payload `iss` and `sub` are mapped to local INTEGIN `identity_subject` and `identity_membership`.
4. **Port Consistency**: Map Casdoor to `127.0.0.1:18180` for drop-in port compatibility with existing pilot harnesses.

---

## 3. Work Breakdown & Target Files

### Slice 1: Configuration & Launch Scripts
Create in `operations/pilot/`:
- `casdoor/app.conf`:
  - Casdoor application configuration pointing to the isolated Postgres container (`integin_casdoor_pilot` database).
  - Port `8000` internally, telemetry disabled (`isDemoMode = false`).
- `start-casdoor-pilot.ps1`:
  - Creates network `integin-casdoor-pilot-net`.
  - Runs `postgres:16-alpine` (pinned digest, record in script header) container as `integin-pilot-casdoor-postgres`.
  - Runs `casdoor/casdoor:v1.600.0` (pinned, NOT :latest; record digest) container as `integin-pilot-casdoor`, binding `127.0.0.1:18180:8000`.
  - Env: `INTEGIN_OIDC_ALLOW_INSECURE_LOOPBACK=true` required for http issuer; fail fast if Keycloak still holds :18180.
  - Discovery: probe `/.well-known/openid-configuration` AND Casdoor per-app endpoints; use whichever returns valid `issuer`+`jwks_uri`, adapt validator config (do NOT assume Keycloak URL shape).
- `stop-casdoor-pilot.ps1`:
  - Gracefully stops and removes Casdoor and Postgres containers and network.

### Slice 2: Realm & Seed Script
Create in `operations/pilot/`:
- `initialize-casdoor-pilot-realm.ps1`:
  - Calls Casdoor REST API (`/api/add-organization`, `/api/add-application`, `/api/add-user`).
  - Creates organization `integin-pilot`.
  - Creates application `integin-live-matrix` with client ID / secret, redirect URI `http://127.0.0.1:18080/callback`.
  - Configures token signing certificate to built-in RSA cert (`cert-builtin`).
  - Seeds pilot inspector user.

### Slice 3: Integration Test Harness
Create in `integin-pilot-source/internal/workorderhttp/`:
- `http_casdoor_postgres_integration_test.go`:
  - Patterned after `http_keycloak_postgres_integration_test.go`; MUST skip (t.Skip) when Casdoor/test DB unreachable — never fail offline `go test ./...`.
  - Secrets (client secret, DB URL) via env only, never hardcoded; `go vet + gofmt + go test -race` gates.
  - Acquires an access token via Casdoor OAuth2 `/api/login/oauth/access_token` endpoint.
  - Passes token into `oidcauth.NewValidator(...)`.
  - Verifies `Principal{Issuer, Subject}`.
  - Resolves local `identity.Membership`.
  - Executes authenticated partial work order submission.

---

## 4. Verification Gate
```powershell
# 1. Start Casdoor pilot
.\operations\pilot\start-casdoor-pilot.ps1

# 2. Seed pilot organization & app
.\operations\pilot\initialize-casdoor-pilot-realm.ps1

# 3. Verify discovery endpoint
Invoke-RestMethod http://127.0.0.1:18180/.well-known/openid-configuration

# 4. Run Go integration test
cd integin-pilot-source
$env:INTEGIN_TEST_DATABASE_URL = "postgres://integin_runtime:integin_live_run_2026@127.0.0.1:15432/integin_dev?sslmode=disable"
$env:INTEGIN_CASDOOR_TEST_ISSUER = "http://127.0.0.1:18180"
$env:INTEGIN_CASDOOR_TEST_CLIENT_ID = "integin-live-matrix"
go test -v -count=1 -run TestCasdoorPartialSubmission ./internal/workorderhttp/...

# 5. Check memory reduction
docker stats --no-stream
# Verify Casdoor is ~110 MB vs Keycloak's 520 MB
```

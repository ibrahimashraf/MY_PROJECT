# Anti-Koshary Fix Plan — 2026-10-11

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all Critical and High findings from the anti-koshary audit, then Medium and Low.

**Architecture:** Fix in priority order: Critical → High → Medium → Low. Stop after each group to verify build + tests pass.

**Tech Stack:** Go 1.27.1, chi, pgx, river, vault, Dart/Flutter, Python

**Spec:** Audit findings from 4 parallel explore agents + 1 general agent

## Global Constraints

- CGO_ENABLED=0 (pure Go only)
- All PostgreSQL queries use `SET LOCAL integin.tenant_id` and `integin.organization_id`
- No `fmt.Sprintf` for SQL with user-derived strings
- No `defer` inside loops
- No silent error swallowing (`_ = err`)
- Tests must pass: `go test -count=1 ./...`
- `go vet ./...` must be clean
- `gofmt -l .` must be empty

---

## Group 1: Critical (5 fixes)

### C1. Remove auth bypass in shortlinkhttp/handler.go
- **File:** `internal/shortlinkhttp/handler.go:661-676`
- **Fix:** Remove the fallback that trusts `X-Tenant-ID`/`X-Organization-ID` headers. Return `http.StatusServiceUnavailable` when validator/resolver are nil.
- **Test:** Verify admin endpoints return 503 when auth not configured

### C2. Fix SQL injection in shortlinkpg/repository.go
- **File:** `internal/shortlinkpg/repository.go:950,1132,663,682`
- **Fix:** Use strict whitelist for column names in `UPDATE ... SET %s`. Replace `fmt.Sprintf` for LIMIT/OFFSET with parameterized queries.
- **Test:** Verify no user-derived strings reach SQL

### C3. Remove hardcoded secrets from .env
- **File:** `integin-pilot-source/.env`
- **Fix:** Delete the file. Rotate the password. Use OpenBao for secret injection.
- **Test:** Verify no plaintext credentials in repo

### C4. Remove hardcoded secrets path in Flutter
- **File:** `field_app/lib/pilot/pilot_fixture_bootstrap.dart:14-15`
- **Fix:** Use environment variable or build-time define for fixture root path
- **Test:** Verify no hardcoded paths in source

### C5. Remove HTTP route paths from domain layer
- **File:** `pkg/domain/fieldtrust.go:533-537`
- **Fix:** Move `NonStatutoryRouteAllowlist` to HTTP handler layer
- **Test:** Verify domain package has no HTTP route strings

---

## Group 2: High (12 fixes)

### H1. Decompose run.go (872 lines)
- **File:** `internal/serverboot/run.go`
- **Fix:** Extract into `buildConfig()`, `openDatabase()`, `loadDeviceTrust()`, `buildHandlers()`, `startQueue()` — each returning sub-dependencies

### H2. Fix defer-in-loop in shortlinkpg/repository.go
- **File:** `internal/shortlinkpg/repository.go:202,225,248,271,294,314,334`
- **Fix:** Wrap each loop body in `func() { ... }()` or call `rows.Close()` explicitly

### H3. Remove mutable global state in onboarding
- **Files:** `pkg/onboarding/attestation_chain.go:45,51-52`, `pkg/onboarding/apple_attest.go:52,58-59,73,78-79`
- **Fix:** Pass roots/appID as explicit parameters

### H4. Remove single-impl interfaces
- **Files:** `pkg/domain/did.go:115-117`, `pkg/licensing/license_validator.go:27-29`, `pkg/conformity/connectors.go:37-44`
- **Fix:** Delete interfaces; pass `time.Time` directly; keep concrete structs

### H5. Extract shared writeError/writeJSON
- **Files:** 12+ handler files
- **Fix:** Create `internal/httputil` package with canonical `WriteError` and `WriteJSON`

### H6. Deduplicate defaultWebhookRetryIntervals
- **Files:** `internal/shortlinksvc/service.go:484-493`, `internal/shortlinkpg/repository.go:33-42`
- **Fix:** Move to shared package

### H7. Extract shared auth middleware
- **Files:** `internal/server/http_routes.go:242-312`, `internal/shortlinkhttp/handler.go:620-677`
- **Fix:** Extract `oidc.Middleware(validator, resolver, next)` in `internal/middleware`

### H8. Split god files
- **Files:** `internal/shortlinkpg/repository.go` (1727 lines), `internal/shortlinkhttp/handler.go` (1345 lines), `internal/shortlinksvc/service.go` (1093 lines)
- **Fix:** Split by responsibility

### H9. Fix Flutter god file
- **File:** `field_app/lib/presentation/field_app.dart` (948 lines)
- **Fix:** Break into focused files

### H10. Fix duplicated PIN zeroing
- **Files:** `field_app/lib/auth/pin.dart:252-256`, `field_app/lib/auth/duress.dart:79-83`
- **Fix:** Extract to shared security utility

### H11. Add HTTP timeout in Flutter
- **File:** `field_app/lib/security/inspection_submission_service.dart:108-112`
- **Fix:** Add `.timeout(Duration(seconds: 30))`

### H12. Fix insecure Random in Flutter
- **File:** `field_app/lib/application/field_app_controller.dart:785`
- **Fix:** Use `Random.secure()`

---

## Group 3: Medium (18 fixes)

### M1-M18: See audit report for full list

---

## Group 4: Low (10 fixes)

### L1-L10: See audit report for full list

---

## Execution Order

1. Group 1 (Critical) — verify build + tests
2. Group 2 (High) — verify build + tests
3. Group 3 (Medium) — verify build + tests
4. Group 4 (Low) — verify build + tests

Each group: fix → build → test → commit → next group

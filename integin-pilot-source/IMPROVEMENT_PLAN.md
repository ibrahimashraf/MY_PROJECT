# INTEGIN Improvement Plan

**Date:** 2026-08-15
**Scope:** Code quality, operations, observability, and developer experience
**Priority:** Pragmatic, highest-impact-first

---

## Phase 1 — Developer Experience (Day 1)

### 1.1 Root README.md
Create a comprehensive project README with:
- Architecture overview with the existing Mermaid diagram
- Tech stack summary (Go 1.22, Flutter, Python/FastAPI, PostgreSQL, RustFS/MinIO)
- Directory structure explanation
- Quick start: `make test`, `make run`
- Environment variable quick reference table
- Links to `ARCHITECTURE_EVOLUTION_ROADMAP.md`, `LOCAL_INFRASTRUCTURE.md`, `AI_DEPLOYMENT.md`

### 1.2 Build Automation (Makefile)
Create a `Makefile` with these targets:
```
test          — go test ./... + flutter test + pytest
fmt           — gofmt -l + dart format
cover         — go test -coverprofile + HTML report
build-server  — go build ./cmd/integin-server
build-matrix  — go build ./cmd/integin-live-matrix
run-server    — build + run with .env file
migrate-up    — run SQL migrations in order
migrate-down  — rollback migrations
lint          — go vet + staticcheck (if installed)
docker-up     — docker compose up for local stack
clean         — remove build artifacts
```

### 1.3 .gitignore
Ensure comprehensive `.gitignore` covers:
- Go: `*.exe`, `/bin`, `/dist`, `coverage.out`
- Flutter: `/build`, `.dart_tool`, `.flutter-plugins`
- Python: `__pycache__`, `.venv`, `*.pyc`
- IDE: `.vscode/`, `.idea/`, `*.swp`
- Secrets: `.env`, `*.pem`, `fixture.json`

---

## Phase 2 — Code Quality Fixes (Day 1–2)

### 2.1 `statusWriter` HTTP Interface Compliance
In `internal/server/http.go`, implement `http.Flusher`, `http.Hijacker`, and `http.Pusher` on `statusWriter` to prevent middleware chain breaks:

```go
func (w *statusWriter) Flush() {
    if f, ok := w.ResponseWriter.(http.Flusher); ok {
        f.Flush()
    }
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
    if h, ok := w.ResponseWriter.(http.Hijacker); ok {
        return h.Hijack()
    }
    return nil, nil, fmt.Errorf("hijacking not supported")
}
```

### 2.2 Structured Logging Setup
In `cmd/integin-server/main.go`, replace `slog.Default()` with an initialized logger:

```go
func initLogger(level string) *slog.Logger {
    var lv slog.Level
    _ = lv.UnmarshalText([]byte(level))
    return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv}))
}
```

Pass this logger through `Dependencies` to `NewMux` and replace `slog.Default()` calls.

### 2.3 Database Cleanup Reliability
In `cmd/integin-server/main.go`, wrap database close:

```go
defer func() {
    if database != nil {
        if err := database.Close(); err != nil {
            logger.Error("database_close_failed", "error", err)
        }
    }
}()
```

### 2.4 Formatting Audit
Run `gofmt -w` across the entire tree and fix any inconsistent indentation (noted in `internal/server/http.go`).

---

## Phase 3 — Configuration & Operations (Day 2–3)

### 3.1 Centralized Config Package (`internal/config`)
Create a typed configuration struct with struct tags and validation:

```go
type Config struct {
    HTTPAddr        string        `env:"INTEGIN_HTTP_ADDR" default:":8080"`
    DBURL           string        `env:"INTEGIN_DB_URL"`
    SyncSecret      string        `env:"INTEGIN_SYNC_SECRET" required:"true"`
    EvidenceStore   string        `env:"INTEGIN_EVIDENCE_STORE" default:"disabled"`
    LogLevel        string        `env:"LOG_LEVEL" default:"info"`
    TLS             TLSConfig     `envPrefix:"INTEGIN_TLS_"`
    DB              DBConfig      `envPrefix:"INTEGIN_DB_"`
    // ... etc
}
```

Migrate all `envInt`, `envBool`, `firstEnv`, `envList` helpers out of `main.go` into this package.

### 3.2 Docker Compose for Local Development
Create `docker-compose.yml` with:
- `postgres:15-alpine` with healthcheck and persistent volume
- `minio/minio` (or rustfs if available) with `INTEGIN-evidence` bucket auto-created
- `integin-server` Go service with `.env` file binding
- `ai_service` Python service (optional, behind profile)
- Shared network, dependency ordering (`depends_on` with condition)

Create `.env.example` from `INTEGIN_LOCAL_ENV.example` with safe defaults.

### 3.3 Python Service Containerization
Add `ai_service/Dockerfile`:
```dockerfile
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY main.py test_main.py ./
EXPOSE 8000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
```

---

## Phase 4 — Observability (Day 3–4)

### 4.1 Metrics Endpoint (`/metrics`)
Add a lightweight Prometheus-compatible metrics endpoint to `internal/server`:
- `http_requests_total` (counter, labeled by method, path, status)
- `http_request_duration_seconds` (histogram)
- `sync_transactions_total` (counter, labeled by outcome)
- `evidence_uploads_total` (counter, labeled by outcome)

Use the standard library only (no external deps). Parse `/metrics` manually or accept a simple text format.

### 4.2 Trace/Correlation ID Propagation
Upgrade the existing `X-Correlation-ID` header to propagate through `context.Context`:

```go
func WithCorrelationID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, correlationIDKey, id)
}

func CorrelationID(ctx context.Context) string {
    if id, ok := ctx.Value(correlationIDKey).(string); ok {
        return id
    }
    return ""
}
```

Pass context through to the sync processor, event store, and PostgreSQL calls. Include `correlation_id` in all structured logs.

### 4.3 Health Endpoint Enhancement
Enhance `/readyz` to return structured JSON with subsystem status:
```json
{
  "status": "ready",
  "service": "INTEGIN",
  "subsystems": {
    "database": "ok",
    "evidence_store": "ok",
    "sync_processor": "ok"
  }
}
```

---

## Phase 5 — Testing & Reliability (Day 4–5)

### 5.1 HTTP Integration Tests
Create `internal/server/integration_test.go` with a full stack test:
1. Start an in-memory server with all dependencies
2. POST `/sync` with valid/invalid transactions
3. Assert outcomes, HTTP status codes, correlation IDs
4. POST `/evidence` with valid/conflicting/tampered payloads
5. Assert store state

Use `httptest.Server` and in-memory implementations only.

### 5.2 Benchmarks
Add benchmark tests for hot paths:
- `BenchmarkSyncSubmit` — concurrent transaction submission
- `BenchmarkSignTransaction` — Ed25519 signing
- `BenchmarkVerifyTransaction` — Ed25519 verification
- `BenchmarkEventStoreAppend` — append-only write throughput

### 5.3 Fuzz Tests
Add fuzz tests for:
- `security.VerifyDeviceMutation` — random signatures, keys, messages
- `canonicalPayload` — random JSON bytes
- `syncapi` request decoding — random HTTP bodies

### 5.4 Test Coverage Reporting
Add `make cover` target that generates HTML coverage reports and prints package-level summaries. Target: >70% coverage for `internal/domain/*`, `internal/security/*`, `internal/syncapi/*`.

---

## Phase 6 — Documentation & Contracts (Day 5)

### 6.1 API Documentation (`API.md`)
Document all HTTP endpoints:
- `/sync` — request/response schema, example payloads, outcome codes
- `/evidence` — upload, duplicate, conflict behavior
- `/local/provision` — provisioning flow
- `/healthz`, `/readyz`, `/metrics` — operational endpoints

### 6.2 Environment Variable Reference
Create `docs/ENVIRONMENT.md` with a table:
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `INTEGIN_HTTP_ADDR` | No | `:8080` | HTTP listen address |
| `INTEGIN_DB_URL` | No | — | PostgreSQL DSN |
| `INTEGIN_SYNC_SECRET` | Yes | — | HMAC signing secret |
| ... | ... | ... | ... |

### 6.3 ADR Template
Add `docs/adr/0000-adr-template.md` and seed with one ADR recording the decision to use a modular monolith with Go as the sole authority.

---

## Phase 7 — CI/CD Pipeline (Day 5–6)

### 7.1 GitHub Actions Workflow
Create `.github/workflows/ci.yml`:
```yaml
name: CI
on: [push, pull_request]
jobs:
  go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go test ./...
      - run: go vet ./...
      - run: test -z "$(gofmt -l .)"
  python:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with: { python-version: '3.11' }
      - run: pip install -r ai_service/requirements.txt
      - run: pytest ai_service/
```

### 7.2 Pre-commit Hooks (Optional)
Suggest `.pre-commit-config.yaml` for:
- `gofmt`, `go vet`
- `trailing-whitespace`, `end-of-file-fixer`

---

## Phase 8 — Security Hardening (Day 6+)

### 8.1 TLS Support
Add TLS configuration to `cmd/integin-server/main.go`:
```go
if certFile != "" && keyFile != "" {
    err = httpServer.ListenAndServeTLS(certFile, keyFile)
} else {
    err = httpServer.ListenAndServe()
}
```

### 8.2 Request Rate Limiting
Add IP-based rate limiting to `productionMiddleware` using a simple token bucket per IP (in-memory, standard library only).

### 8.3 CORS Configuration
If the TypeScript advisor UI is served from a different origin, add configurable CORS to the mux.

---

## Execution Order

| Phase | Deliverables | Est. Time |
|-------|-------------|-----------|
| 1 | README, Makefile, .gitignore | 2–3 hours |
| 2 | statusWriter fix, logging, cleanup, formatting | 3–4 hours |
| 3 | Config package, docker-compose, Python Dockerfile | 4–6 hours |
| 4 | Metrics, trace propagation, health enhancements | 4–5 hours |
| 5 | Integration tests, benchmarks, fuzz tests, coverage | 6–8 hours |
| 6 | API.md, ENVIRONMENT.md, ADR | 3–4 hours |
| 7 | GitHub Actions CI | 2–3 hours |
| 8 | TLS, rate limiting, CORS | 3–4 hours |

**Total estimated effort:** ~27–37 hours of focused implementation

---

## Acceptance Criteria

1. A new developer can clone the repo, run `make test`, and have all tests pass within 5 minutes.
2. `make docker-up` brings up PostgreSQL + MinIO + INTEGIN server locally.
3. `curl http://localhost:8080/metrics` returns request counters.
4. `go test ./...` runs in under 30 seconds and reports coverage.
5. CI passes on every push without manual intervention.
6. No `gofmt -l` violations exist in the tree.

---

## Deferred (Future)

- **Kubernetes manifests** — per Architecture Roadmap Stage A, Docker Compose is sufficient now
- **Redis caching** — no measured cache requirement yet
- **Kafka/RabbitMQ** — database outbox pattern is sufficient
- **Full OpenTelemetry tracing** — correlation IDs are the pragmatic first step
- **Automated security scanning** (SAST/dependency scanning) — add after CI is stable

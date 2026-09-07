# INTEGIN Phase 4 Infrastructure

Phase 4 supplies infrastructure seams without moving domain business rules into adapters.

| Component | Implementation |
|---|---|
| Event store | `internal/eventstore`: append-only in-memory store for unit tests and `database/sql` PostgreSQL adapter with optimistic concurrency and replay ordering. |
| Event bus | `internal/eventbus`: in-process subscriptions and append-before-publish committer. |
| Repository | `internal/repository`: generic aggregate replay using a caller-provided reducer. |
| Object storage | `internal/storage`: defensive in-memory store and MinIO-compatible HTTP adapter with injectable signing. |
| Email | `internal/email`: validated SMTP provider with injectable sender and health checker plus explicit delivery states. |
| Database migration | `migrations/0001_event_log.sql`: partitioned append-only event log with tenant and replay indexes. |

## Runtime Services

The PostgreSQL adapter expects the event-log migration to be applied before use. The MinIO adapter expects an S3-compatible endpoint, bucket, and request signer. The SMTP provider expects host, port, sender address, and optional username/password credentials. These services are intentionally not required for default unit tests.

## Validation

From the project root:

```bash
gofmt -l $(find . -name '*.go' -type f)
go test ./...
go vet ./...
```

Service-backed integration tests should be added only behind explicit configuration such as `INTEGIN_POSTGRES_DSN`, `INTEGIN_MINIO_ENDPOINT`, and `INTEGIN_SMTP_HOST`; they must not run by default when those variables are absent.

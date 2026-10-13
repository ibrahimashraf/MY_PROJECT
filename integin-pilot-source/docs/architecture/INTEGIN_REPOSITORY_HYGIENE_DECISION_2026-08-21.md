# INTEGIN Repository Hygiene Decision — 2026-08-21

## Reviewed removal set

The workspace was reviewed before deletion. The selected removals are ignored, reproducible outputs that contain no source, migration, receipt, or evidence record.

| Path class | Decision | Reason |
|---|---|---|
| `field_app/build/` | Remove | Reproducible Flutter test/build output; approximately 102 MB. |
| `ai_service/.pytest_cache/` and `ai_service/__pycache__/` | Remove | Reproducible Python test/interpreter cache. |
| `field_app/.dart_tool/` | Retain | Reproducible but retained to avoid disrupting the known snapshot-based Flutter test workaround during active development. |
| `ai_service/.verification-venv/` | Retain | Reproducible but retained as the active local Python verification environment. |
| `field_app/.idea/`, `.iml`, and editor settings | Retain | User-local IDE preferences; small and not necessary to remove. |
| `docs/evidence/`, `docs/architecture/`, source, migrations, task records, and pilot backups | Retain | Protected implementation and evidence records. |
| `field_app/lib/evidence/` | Retain | Ignored path may contain project-specific material; no deletion without explicit content review. |

## Safety rule

Deletion is restricted to the listed ignored generated paths. No `git clean`, wildcard removal, database change, object-store deletion, or migration operation is permitted under this hygiene decision.

## Outcome

The reviewed removal completed for `field_app/build/`, `ai_service/.pytest_cache/`, and `ai_service/__pycache__/`. The initial guard correctly stopped before deleting `.pytest_cache/` because its ignore rule is contained within the cache itself rather than the root ignore file. Its standard pytest cache structure was then verified before removal. All protected paths remained present.

`git diff --check`, `go test ./...`, and `go vet ./...` passed after cleanup. The warnings about future LF-to-CRLF conversion are repository line-ending notices, not a source or test failure.

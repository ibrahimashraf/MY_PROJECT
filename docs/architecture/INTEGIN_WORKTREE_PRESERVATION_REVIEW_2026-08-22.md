# INTEGIN Worktree Preservation Review — 2026-08-22

> Non-destructive review only. No files were staged, committed, reset, migrated, or deployed.

## Observed Baseline

Repository HEAD: `249cd06`. A read-only inventory found 64 changed or untracked status entries. `go test -count=1 ./...` and `go vet ./...` passed against this working tree.

## Proposed Logical Preservation Units

| Unit | Principal areas | Required review |
|---|---|---|
| A. Sync durability and identity | Sync, syncstate, identity, storage, module files, tests | Keep signature, receipt, and atomic-persistence contracts together. |
| B. Work-order foundation | Work-order packages, 0005/0008/0009, composition, Field test, operations | Keep candidate migrations and acceptance evidence together. |
| C. Evidence metadata/export | Evidence packages, 0010/0011, storage/server, export records | Keep registration, object verification, and export evidence together. |
| D. Certificate lifecycle/public transport | Certificate packages, 0012-0015, server/OpenAPI, renderer docs | Keep lifecycle, snapshots, and routes coherent; exclude future renderer selection. |
| E. Documentation and planning | Docs and planning records | Capture after reviewed technical units only. |

## Recommendation

Use archive-first, review-then-commit. Before any commit, create an owner-approved complete local archive of the worktree and compare its manifest to the 64-entry inventory. Then review shared server and OpenAPI files for cross-slice coupling. This approval does not authorize commits, migrations, runtime changes, or public release.

## Completed Archive

Owner-approved archive: $archive. Manifest: $archive.manifest.txt. SHA-256: $hash. Archive readability passed; .git internal metadata was excluded; post-archive working-tree status remained 64 entries. No repository mutation occurred.

## Shared-File Coupling Map

| Shared area | Coupled units | Future handling |
|---|---|---|
| `cmd/integin-server/main.go` | Work-order, evidence registration, certificate lifecycle/public verifier | Review and stage by hunk only after each handler dependency is captured. |
| `internal/server/http.go` and tests | Operational controls, work-order route, evidence route, certificate routes, token-path redaction | Preserve route additions with their unit; isolate shared middleware/operational changes if their test boundary is independent. |
| `openapi/integin-v1.json` | Health, sync, evidence, work-order, provisioning, identity, certificate/public routes | Treat as a contract-reconciliation companion; use interactive hunk review, not an all-or-nothing file commit. |
| `go.mod` and `go.sum` | Go 1.26.5 upgrade and frozen Go renderer dependencies | Review separately from future renderer selection; do not treat dependency presence as stack adoption. |
| Planning and architecture records | Every technical unit | Capture last, after the related code and evidence boundaries are frozen. |

## Future Staging Order

1. Review the Go directive separately from frozen renderer dependencies, then sync/identity/storage with focused and full Go checks.
2. Preserve work-order migrations 0005/0008/0009, packages, Field acceptance, and matching shared hunks together.
3. Preserve evidence metadata/export migrations 0010/0011, registration, storage verification, composition, and evidence records together.
4. Preserve certificate migrations 0012-0015 and lifecycle/public transport in dependency order; hunk-review server and OpenAPI changes.
5. Capture documentation and planning records last, after they match the technical captures.

## Coupling Risks

Classify untracked tmp and operations material before version control. Shared server and OpenAPI files need hunk-level staging. Do not bundle legacy evidence policy, final renderer selection, public controls, or production activity into preservation commits.

## Proposed Unit 1 Commit Boundary

Proposed message: `fix(sync): harden signed transaction handling and durable held-receipt promotion`.
Include only: `internal/domain/sync/sync.go`, `internal/domain/sync/sync_test.go`, `internal/syncstate/postgres.go`, and `internal/syncstate/postgres_integration_test.go`.
Exclude: identity actor fields (work-order unit); storage retrieval (evidence unit); Go directive (toolchain review); Go PDF/QR dependencies and sums (frozen renderer prototype); server/OpenAPI/docs/planning files.
Required gates before any future commit: focused `go test -count=1 ./internal/domain/sync ./internal/syncstate`; full `go test -count=1 ./...`; `go vet ./...`; changed-file whitespace review; staged-path verification; and a review that no unrelated file is staged.

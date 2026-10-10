# AGENTS.md

Canonical instruction file for this repository, shared by all coding agents
(Claude Code, Codex, Cursor, and others). Claude Code reaches it through the
`@AGENTS.md` import in `CLAUDE.md`. Repository guidance belongs here, while
tool-specific pointer files contain no duplicate guidance.

## Project knowledge

<!--
Fill this in as the project develops. Keep it under a page so stale context does
not burden every session. When an agent makes the same mistake twice, record the
correction here.
-->

### Commands

- Build: `cd integin-pilot-source && CGO_ENABLED=0 go build -v -o bin/integin-server ./cmd/integin-server`
- Test: `cd integin-pilot-source && CGO_ENABLED=0 go test -count=1 ./...`
- Lint: `cd integin-pilot-source && go vet ./...`

### Conventions

- **Pure Go Only (`CGO_ENABLED=0`)**: Zero in-process `import "C"` or shared libraries (`.so`, `.dll`).
- **PostgreSQL RLS Authority**: All domain operations must execute with `SET LOCAL integin.tenant_id` and `integin.organization_id`.
- **Rust Boundary**: Rust is strictly confined to out-of-process CLI verifiers (`tools/integin-export-verifier`) and storage appliances (`RustFS`); never linked via FFI.

### Architecture

- Go modular monolith (`integin-pilot-source/`) is sole authority for domain transactions, tenant isolation, and certificate issuance.
- Statically linked, single-binary distribution target.

### Things agents get wrong

- Never attempt to import C or use `cgo` / Rust FFI inside the Go runtime.
- Never bypass `CGO_ENABLED=0` in builds or tests.
- Never create competing microservice backends that bypass PostgreSQL RLS.

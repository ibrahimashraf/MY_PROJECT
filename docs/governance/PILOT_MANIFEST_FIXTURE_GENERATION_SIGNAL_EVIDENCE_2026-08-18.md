# INTEGIN Pilot Manifest Fixture Generation Signal Evidence

**Recorded:** 2026-08-18  
**Scope:** User-authorized source-owned fixture participation.  
**Status:** Completed bounded signal. It is not a live eight-case matrix result.

## Decision and implementation

The user authorized the pilot fixture to participate in evidence production. Structural inspection showed that the fixture is a local fixture-input generator: it produces the data required for the candidate scenario but does not itself call the candidate, Field application, database, or a network service. Its bounded evidence can therefore truthfully assert only that its own reviewed generation step completed.

Commit `c319776` adds `internal/fixturesignals` and tracks the formerly untracked `cmd\pilot-manifest-fixture\main.go`. After local generation succeeds, the fixture conditionally emits one atomic, redacted `fixture_generated` signal only when an explicit child-only signal context is present. With no context, legacy fixture behavior is unchanged. Partial, unsupported, unsafe, or duplicate signal context fails the fixture hard gate without printing data.

## Independent review decision

Independent security, assurance-provenance, and privacy/operational reviews accepted fixture participation only under a strict split of authority. The fixture has no candidate run ID, candidate receipt path, private configuration, final receipt ability, status authority, case verdict, replay state, identity field, payload field, proof field, manifest field, key, credential, or output forwarding capability.

The wrapper now creates a fresh ACL-restricted fixture-signal directory, invokes the fixture through a minimal child-only environment, blackholes child stdout/stderr, validates the exact closed signal shape, removes the directory after validation, and records only the public boolean `fixture_generated`. The fixture signal does not become a case receipt.

## Validation

| Verification | Result |
| --- | --- |
| `go test .\internal\fixturesignals .\cmd\pilot-manifest-fixture -count=1` | Passed. |
| Public wrapper syntax parse | Passed. |
| Strengthened wrapper smoke check | Passed: candidate started loopback-only; `fixture_exit_code=0`; `fixture_generated=true`; output suppression true; acceptance before/after `200/200`; candidate stopped. |
| Wrapper cleanup recheck | Passed after a public cleanup-condition correction: the fixture signal root and per-run artifacts were absent; acceptance stayed `200/200`; no candidate listener remained. |
| `go test ./... -count=1` | Passed. |
| `go vet ./...` | Passed. |
| Field static analysis | Passed: `No issues found!` |

The smoke check ran only through `run-public-manifest-fixture-boundary.ps1`; it did not expose fixture output or private values. Package enforcement remained disabled, OIDC remained disabled, OpenBao remained sealed/unwired, and protected acceptance remained unchanged.

## Matrix-readiness decision

| Required public proof property | Current result |
| --- | --- |
| Fixture local generation provenance | Present, redacted, and independently bounded. |
| Source-derived seven HTTP case receipts | Not yet composed. |
| Source-derived Field `field_binding` receipt | Not yet composed into an independent pilot seam. |
| Candidate lifecycle/finalization proof for duplicate/publish failure | Not yet designed or implemented. |
| Final v2 public receipt inventory and live adapter validation | Not yet available. |
| Live isolated-pilot eight-case matrix | **Not ready and not claimed.** |

The next required design work is the candidate lifecycle/finalization protocol, then the independent Field pilot seam, then source-owned HTTP observation composition. The fixture signal can be used only as a bounded wrapper lifecycle fact during those later steps.

# INTEGIN Offline-Sync Durability Correction Decision

## Decision

Correct the offline-sync processor as a bounded durability/security slice before certificate-template work. A transaction durably recorded as `HELD` because a prior sequence is missing is **not applied** and must never be represented as `DUPLICATE`. Once its missing predecessors are accepted, replaying the same authenticated envelope may advance it to `APPLIED` exactly once.

The processor must authenticate and validate the envelope before any durable receipt lookup can disclose whether a transaction ID exists. It must accept only explicitly supported signature algorithms. For the current contract, those values are exact `Ed25519` and the explicit legacy `HMAC-SHA256` constant; all other values fail closed as `SECURITY_FAILURE`. No algorithm label may silently select a verification branch.

## Persistence semantics

The durable repository gains a single atomic held-state write: it records the `HELD` receipt and held envelope in one transaction. A later transition from `HELD` to `APPLIED` atomically updates the receipt outcome, advances the device sequence through the existing guarded compare-and-set operation, and removes the held envelope. An already `APPLIED` matching transaction is idempotently `DUPLICATE`; a matching `HELD` transaction is resumable; a mismatched hash is `CONFLICT`.

> A held state is an authorized recovery condition, not a terminal acceptance, duplicate, or business mutation.

The correction does not invent automatic background replay. The Field client remains responsible for resubmitting queued/held envelopes after the missing sequence is resolved. The server provides correct durable semantics for that retry. A future server-side drain mechanism, if needed, requires its own authorization and operational design.

## Required acceptance matrix

| Case | Expected result |
|---|---|
| Valid sequence 2 arrives before sequence 1 | `HELD`; exactly one receipt and one held envelope are atomically present. |
| Valid sequence 1 arrives | `APPLIED`; durable last accepted sequence becomes 1. |
| Original valid sequence 2 is retried | `APPLIED`; held envelope is removed and durable sequence becomes 2. |
| Sequence 2 is retried again | `DUPLICATE`; durable sequence remains 2. |
| Same transaction ID with a different payload hash | `CONFLICT`; no state is altered. |
| Invalid/unknown signature algorithm with a known transaction ID | `SECURITY_FAILURE`; no receipt-existence or conflict detail is disclosed and no state changes. |
| Invalid signature with a known transaction ID | `SECURITY_FAILURE`; no state changes. |
| Atomic held write fails before commit | Neither held receipt nor held envelope remains. |
| Tenant/organization mismatch or unauthenticated envelope | `SECURITY_FAILURE`; no durable receipt detail is disclosed. |

## Boundaries

This correction does not change Work-Order lifecycle, evidence metadata, certificate issuance, object storage, OIDC membership, or external integrations. It retains tenant-scoped durable storage and transaction-local RLS settings. It must be validated with focused unit tests, PostgreSQL integration evidence where the existing test configuration permits, and full `go test ./...` plus `go vet ./...` before closure.

## Implemented outcome — 2026-08-21

`SubmitContext` now performs device/authority/envelope-signature verification before any durable receipt lookup and accepts only exact `Ed25519` or `HMAC-SHA256` algorithm labels. A durable `HELD` receipt with the matching payload hash is explicitly resumable rather than an immediate duplicate; after the missing predecessor advances the sequence, replay can become `APPLIED`, and a subsequent replay becomes `DUPLICATE`. In-memory held state is also updated only after a successful durable held write and removed only after application succeeds.

`syncstate.PostgresRepository.SaveHeld` now creates or updates the held receipt and envelope in one PostgreSQL transaction. `SaveReceipt` supports only the allowed `HELD` to `APPLIED` transition, atomically advances the guarded device sequence, and deletes the held envelope in that same transaction. The controlled `TestPostgresHeldRecoveryAndAtomicPersistenceIntegration` passed against the local pilot PostgreSQL service: it demonstrated durable hold, predecessor acceptance, held-to-applied promotion, sequence advancement to `2`, held-envelope removal, an induced held-envelope insert failure that left neither receipt nor envelope, and zero namespaced row residue after test completion. The first proof uncovered fixture-cleanup ordering and parameterized multi-command cleanup defects in the test harness; those were corrected and the proof was rerun successfully.

Focused `internal/domain/sync` and `internal/syncstate` tests cover held replay, invalid replay authentication before duplicate handling, unknown algorithm rejection, and direct no-durable-receipt-probe behavior. The full unpiped `go test -count=1 ./...` and `go vet ./...` gates passed.

> This proves the bounded offline recovery semantics. It does not implement automatic server-side replay, a background queue drain, client UI for held transactions, or a production Field rollout policy.

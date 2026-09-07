# Planning-With-Files: Existing-Name Clean-Room Replacement Contract

## Owner decision

The owner selected **Path B**: update the current `planning-with-files` skill itself rather than rely on the separate `integin-persistent-planning` successor.

This is a semantic replacement of the active skill package under its existing name. It is **not** a direct v3.11.2 source update, a fork, or an external-content import.

## Replacement rules

| Area | Required treatment |
|---|---|
| Active package name | Retain `planning-with-files`. |
| Active workflow | Replace with the independently written, hook-free host-compatible planning workflow. |
| External source text, scripts, templates, hooks | Remove from the active package; retain only in the immutable rollback archive. |
| Existing local version/source identity | Do not present the replacement as external v3.11.2 or reuse external version history. |
| Supported frontmatter | Use only the local validator’s supported `name` and `description` metadata. |
| Separate `integin-persistent-planning` successor | Retire from active use by removing it after the replacement package validates; retain it in the rollback archive and ledger. |
| Change tracking | Record baseline, applied changes, exclusions, validation, and retirement in the reconciliation ledger. |

## Required internal behavior

The active package must provide durable planning records, explicit authorized-root selection, safe recovery from conflicting or corrupt records, separated evidence/decisions/progress, re-read-before-write coordination, a continuity checkpoint, and honest closure.

The active package must not provide hooks, automatic context injection, completion blocking, background loops, scheduling, external script execution, package installation, or direct external-source copying.

## Rollback

The immutable archive `rollback/planning-skill-packages-precleanroom-replacement-20260824.tar.gz` contains the pre-replacement `planning-with-files` and `integin-persistent-planning` package states. Its SHA-256 is `67775aced7bd21aaad8d72d527516e9270575293583cbc898318c5dd8e84b315`.

Restoring from this archive is a separate, owner-directed operation. No automatic rollback action is authorized by this replacement.

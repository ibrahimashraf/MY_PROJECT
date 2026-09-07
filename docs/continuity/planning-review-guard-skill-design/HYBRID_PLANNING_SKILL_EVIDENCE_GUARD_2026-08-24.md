# Hybrid Planning-With-Files Evidence Guard Record

## Authorized scope and constraints

This record covers the active local `planning-with-files` package, its three bundled templates, and its explicit local validator. Checks were limited to static inspection and disposable-directory execution. No host hooks, external source execution, network access, protected project records, or runtime services were involved.

## Claim-evidence matrix

| ID | Claim | Requirement/source | Test or observation | Negative-path evidence | Status | Limitation or required next action |
|---|---|---|---|---|---|---|
| HG-01 | The active package provides reusable planning templates. | Hybrid design, F-01 | Three separate templates exist for plan, findings, and progress. | Templates are not auto-copied or auto-created. | Verified | Actual project adoption remains user-controlled. |
| HG-02 | The validator is explicitly invoked and read-only. | Hybrid design, F-02/F-03 | A disposable template set passed both profiles without changing the pre-existing record hashes. | A missing or noncanonical record set returns failure rather than attempting repair. | Verified | It does not assess semantic correctness of project decisions. |
| HG-03 | The validator distinguishes invalid invocation from record nonconformance. | Validator contract | Structured template set passed; noncanonical findings returned exit `1`; invalid root returned exit `2`. | Both failure modes were checked. | Verified | It does not choose a root on behalf of the caller. |
| HG-04 | The active package remains free of prohibited host and network automation. | Hybrid design exclusions | Static scan found no hooks, host paths, external source identifiers, network APIs, shell execution, or file-write APIs. | Scan targeted the named forbidden classes. | Verified | Static scanning does not prove safety in an arbitrary host. |
| HG-05 | The active package remains structurally valid. | Skill-creator validation requirement | Package validator passed; final active-collection sweep reported 54 of 54 valid packages. | N/A | Verified | Structural validation is not behavioral proof. |
| HG-06 | The pre-replacement state is still recoverable. | Existing-name replacement contract | Rollback archive SHA-256 check passed. | N/A | Verified | Restoration is a separate owner-directed operation. |

## Drift and contradiction register

| ID | Affected artifact | Drift or contradiction | Risk | Required correction or escalation |
|---|---|---|---|---|
| HD-01 | Retired `integin-persistent-planning` invocation | The user invoked a retired skill name during the revision request. | Duplicate trigger or ambiguous ownership if reintroduced. | Keep the durable-planning behavior within the active `planning-with-files` package; do not restore the duplicate unless separately authorized. |

## Result

The hybrid package may advance. The evidence verifies explicit template availability, validator behavior, selected negative paths, static boundary adherence, collection validation, and rollback archive integrity. It does not prove semantic adequacy for every project, host-level safety beyond the static checks, or automatic planning-state recovery; those claims remain out of scope.

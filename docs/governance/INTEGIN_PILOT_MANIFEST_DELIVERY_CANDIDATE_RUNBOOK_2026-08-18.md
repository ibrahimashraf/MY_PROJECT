# INTEGIN Pilot Manifest-Delivery Candidate Runbook — 2026-08-18

**Status:** Preparation complete; **not authorized for execution**. This runbook defines a separately governed, isolated-pilot candidate only. It does not authorize a database migration, route mount, candidate start, runtime restart, package-enforcement change, OIDC change, OpenBao wiring, acceptance change, or backup restore.

## Purpose and governing boundary

The next mandatory INTEGIN product gate is a **device-authenticated work-package manifest-delivery candidate**. Its limited purpose is to demonstrate that a Field device can retrieve a signed work-package manifest, verify and cache it locally, and submit one valid manifest-read proof to the isolated pilot. It must also demonstrate deterministic rejection of invalid, expired, unknown-authority, and replayed proof paths. The candidate is a delivery experiment, not a workflow-enforcement experiment. [1] [2]

> **Non-negotiable authority rule:** the candidate may create only disposable pilot evidence. It must never change an authoritative inspection, certificate, approval, package-enforcement, or acceptance workflow state.

| Runtime or capability | Required candidate state | Prohibited state |
|---|---|---|
| Product identity | **INTEGIN** in runbook, receipt, and user-facing wording | New user-facing INTEGIN branding |
| Acceptance control | `127.0.0.1:8080` health and readiness remain `200` before, during, and after the candidate | Restart, configuration edit, migration, or route change |
| Candidate listener | Isolated pilot only at `127.0.0.1:18080` | Shared listener, public bind, or acceptance port |
| Manifest route | Mounted only in the dedicated candidate process | Mounted in the acceptance process or standard persistent pilot process |
| Package enforcement | Explicitly disabled; sync pre-acceptance policy unset | Any blocking or workflow-state mutation |
| OIDC and OpenBao | OIDC disabled; OpenBao sealed and unwired | Enabling, unsealing, or adding an authority path |
| Data | Dedicated disposable pilot records and a traceable non-secret evidence ledger | Reuse of production, acceptance, or real customer data |

## Preconditions and stop conditions

Preparation confirms that the source contains the work-package persistence, assignment-context, and manifest-proof replay migrations; that the manifest route remains unmounted in the standard source/runtime path; and that Field static analysis and focused Go validation previously passed. These are readiness facts, not permission to execute. [1]

| Preflight check | Required evidence before launch | Immediate stop condition |
|---|---|---|
| Protected control | `GET /healthz` and `GET /readyz` at `127.0.0.1:8080` both return `200` | Either response is non-`200`, slow beyond the agreed operator limit, or changes unexpectedly |
| Pilot isolation | No listener exists on `127.0.0.1:18080` before the launch command; the pilot PostgreSQL and RustFS dependency containers are identified without exposing credentials | A port conflict, an unrecognized process, or an endpoint bound beyond loopback |
| Recovery asset | The pre-manifest pilot backup is present at `operations\pilot\backups\integin_pilot-pre-manifest-20260817-143038.dump`; file existence and metadata only may be checked | Backup missing, inaccessible, or a request to read or disclose protected material |
| Migration plan | The operator records the pilot database's current migration state and reviews the ordered application of `0005`, `0006`, and `0007` against a copied/recoverable pilot database | Migration target is acceptance, a shared database, or its state cannot be positively identified |
| Candidate source | The dedicated candidate composition is rebuilt from the reviewed source without editing `cmd/integin-server/main.go` | A direct change to the protected standard server entry point is proposed |
| Fixture boundary | The disposable manifest fixture is named, versioned, and invoked only through its approved candidate interface; its contents, private keys, and credentials are never printed, logged, or committed | Fixture content or a private secret would need to be exposed to proceed |
| Signing boundary | A dedicated, non-production signing key and matching public authority record are provisioned for this candidate only; the private key remains in `private\integin-secrets` and is not read into records | Acceptance signing material, an unknown signer, or a secret-bearing command/history/log |
| Feature boundary | Candidate configuration proves runtime identity `pilot`, loopback address `127.0.0.1:18080`, manifest retrieval enabled only in that process, and package enforcement disabled | Unclear runtime identity, inherited feature state, or any enforcement-enabled signal |

The candidate operator must capture the preflight result as a signed-off checklist with a timestamp, source commit, binary digest, migration-state summary, public-key fingerprint, listener address, and the two acceptance status codes. The checklist must contain no database URL, password, private key, token, fixture body, or manifest payload.

## Candidate execution sequence

The following sequence is deliberately **conditional**. It is not an execution instruction until a separate candidate activation approval explicitly names the pilot database, source commit, candidate binary, operator, time window, and recovery authority.

| Sequence | Candidate operation | Required non-secret evidence | Forbidden consequence |
|---:|---|---|---|
| 1 | Record preflight and take a metadata-only backup-presence check. | Acceptance health/readiness `200`; pilot listener absent; pilot dependency status; backup metadata. | Any backup restore or database mutation. |
| 2 | Create or select an isolated, recoverable pilot database target and record its identity without credentials. | Target label, migration-state summary, and verified recovery path. | Using acceptance or an unknown/shared database. |
| 3 | Apply only the approved work-package, assignment-context, and replay-protection migrations to that isolated target. | Migration identifiers `0005`, `0006`, and `0007`; before/after schema ledger. | Applying any migration to acceptance. |
| 4 | Seed disposable tenant, organization, device, authority, work-package, and assignment-context records through the reviewed fixture/interface. | Seed receipt with opaque IDs or digests; row-count ledger; no source payload or secret. | Seeding real inspection, certificate, or user data. |
| 5 | Assemble the dedicated candidate with the manifest route and retrieval state enabled only for `pilot`, with enforcement disabled. | Candidate binary digest; loopback listener check; configuration attestation. | Editing `cmd/integin-server/main.go`, starting a persistent runtime, or enabling enforcement. |
| 6 | Retrieve one manifest from Field and perform local verify/bind/cache. | One `manifest_issued` receipt and one `field_binding_succeeded` receipt, each carrying an opaque manifest identifier and reason code only. | Treating a Field cache result as an inspection approval. |
| 7 | Submit one valid manifest-read proof, then repeat it unchanged. | One `request_accepted` / `proof_valid` receipt and one `replay_rejected` receipt. | Accepting duplicate proof or mutating primary workflow state. |
| 8 | Exercise invalid-path checks using purpose-built disposable variants. | Rejection receipts for `signature_invalid`, `package_hash_invalid`, `expired`, `authority_mismatch`, and `key_unknown`; any supported context-mismatch check is recorded separately. | Suppressing a rejection, retrying with real data, or loosening validator behavior. |
| 9 | Stop the candidate, verify no listener remains at `127.0.0.1:18080`, and re-check acceptance. | Candidate exit evidence; pilot listener absent; acceptance health/readiness `200`. | Leaving a route-mounted or enforcement-capable process running. |

## Evidence and state-count matrix

Each candidate run must produce one non-secret evidence ledger. The ledger must use **expected**, **observed**, and **result** columns. It must state exact seed and table counts from the reviewed fixture before launch; this runbook does not invent counts that source and fixture outputs have not yet proven. At minimum, the observable candidate receipts and state deltas below must be accounted for.

| Evidence item | Expected minimum count | Expected state effect | Acceptance criterion |
|---|---:|---|---|
| Candidate preflight ledger | 1 | No database or runtime change | All checks pass; acceptance is `200/200` |
| Disposable seed receipt | 1 | Only the named isolated pilot target receives the declared seed rows | Opaque IDs/digests reconcile to the row-count ledger |
| Manifest issuance receipt | 1 | No primary workflow state mutation | Outcome `manifest_issued` |
| Field verified-cache receipt | 1 | Device-local cache changes only | Outcome `field_binding_succeeded`; no approval state |
| Valid proof receipt | 1 | Exactly one valid proof is accepted and the replay ledger reflects it | Outcome `request_accepted` with `proof_valid` |
| Duplicate-proof receipt | 1 | No second acceptance; replay protection blocks reuse | Outcome `replay_rejected` |
| Invalid field-binding receipts | 2 | No valid cache/binding from invalid signature or package hash | `signature_invalid` and `package_hash_invalid` each observed once |
| Invalid-proof receipts | 3 | No accepted proof from expired, wrong-authority, or unknown-key variants | `expired`, `authority_mismatch`, and `key_unknown` each observed once |
| Candidate shutdown receipt | 1 | Candidate listener removed; no persistent enforcement service remains | `127.0.0.1:18080` no longer listens |
| Protected-control verification | 2 | Acceptance remains untouched before and after the candidate | Both checks record health/readiness `200/200` |

The evidence ledger must additionally record: the exact source commit; candidate binary digest; migration IDs; public authority-key fingerprint; test-device opaque identifier; Field build/version; request correlation IDs; expected/observed isolated-table count deltas; reason codes; candidate start/stop times; and the operator's rollback conclusion. The ledger must not include a manifest body, proof body, signature, private key, database credential, personally identifying content, or customer data.

## Rollback and restoration boundary

Normal candidate rollback is non-destructive: stop the candidate process using its recorded process identifier, verify that `127.0.0.1:18080` has no listener, and re-run the acceptance health and readiness checks. A failed candidate is evidence, not a reason to bypass a validator or enable enforcement.

If the isolated pilot database must be restored, restoration is a **destructive-data action**. It requires explicit confirmation that the target is the disposable pilot database, confirmation that the documented pre-manifest backup is the intended source, and a fresh acceptance `200/200` check immediately before and after the operation. The restoration procedure must not read, print, or preserve private credentials. It must conclude with an isolated-pilot data-identity check and a no-listener verification.

| Trigger | Safe response | Evidence required |
|---|---|---|
| Acceptance health/readiness is not `200/200` | Stop before any candidate action; investigate acceptance separately. | Timestamp, two status codes, and no-action record. |
| Candidate does not bind exclusively to loopback `:18080` | Stop and terminate the candidate. | Binding observation and termination result. |
| A valid proof is accepted twice | Stop; preserve non-secret evidence; do not retry with altered controls. | Correlation IDs, receipt ledger, and replay-store count. |
| Any invalid proof or binding path succeeds | Stop; preserve evidence; do not relax a validator. | Reason-code mismatch and affected candidate build digest. |
| Any enforcement state becomes enabled | Stop candidate; restore disabled state only through the approved non-enforcement control; do not continue. | Configuration attestation and post-stop verification. |
| Pilot data recovery is required | Pause for the destructive restore authorization gate. | Explicit target confirmation and recovery decision. |

## Authorization gate and next decision

This preparation is complete only when the preflight checklist, source commit, candidate binary build method, disposable seed interface, signing-material custodian, evidence-ledger template, and restoration owner are all named. **It does not authorize execution.**

The next action is a separate, explicit decision to run the candidate in a defined time window. That decision must affirm every control in this document, especially the isolated pilot target, loopback-only listener, disabled package enforcement, disabled OIDC, sealed/unwired OpenBao, private-material boundary, and protected acceptance control. If any prerequisite is absent, the correct result is to stop and update the runbook—not to improvise an operational bypass.

## References

[1]: ../../integin-pilot-source/docs/PILOT_MANIFEST_ACTIVATION_RUNBOOK.md "INTEGIN Pilot Manifest Retrieval Activation Runbook"
[2]: PILOT_MANIFEST_DELIVERY_SOURCE_READINESS_REVIEW_2026-08-18.md "INTEGIN Pilot Manifest Delivery Source-Readiness Review — 2026-08-18"

# INTEGIN Data-Backed Pilot Policy Matrix Evidence — 2026-08-18

**Status:** Completed in the isolated pilot boundary. This record is not an authorization to enable package enforcement in acceptance or any persistent runtime.

## Purpose and Boundary

This exercise validated the pre-acceptance work-package policy against durable pilot PostgreSQL data while preserving the protected acceptance runtime. The candidate was the dedicated `integin-server-pilot-policy-candidate.exe` process on `127.0.0.1:18080`; it was launched only through the pilot wrapper with preparation, registration, and enforcement gates scoped to that candidate process. The protected acceptance service remained on `127.0.0.1:8080` and was not reconfigured.

The matrix used fresh, disposable fixtures generated into a private path outside source control. No fixture values, database URLs, credentials, identifiers, keys, signed payloads, or package contents are recorded here. The pre-existing pilot restoration backup remained available throughout; no restoration was required.

## Preconditions and Controls

Immediately before the final evidence run, acceptance readiness was `200` and no listener existed on the candidate port. The candidate then returned `200` from both health and readiness endpoints while acceptance readiness remained `200`. The matrix runner completed with its success marker, and the candidate-only rollback endpoint returned HTTP `200` with `{"status":"rolled_back"}`.

| Control | Observed result | Interpretation |
| --- | --- | --- |
| Acceptance readiness before candidate launch | `200` | The protected control was healthy before pilot activity. |
| Candidate health and readiness | `200` / `200` | The isolated, data-backed candidate started with its policy registration attached. |
| Acceptance readiness during candidate exercise | `200` | Candidate activity did not disrupt the protected control. |
| Candidate rollback control | `200`, `{"status":"rolled_back"}` | Candidate policy detachment and shutdown were explicitly invoked. |
| Post-rollback candidate listener | Absent on `127.0.0.1:18080` | No pilot candidate remained listening after exercise. |
| Post-rollback acceptance health and readiness | `200` / `200` | The protected acceptance service remained healthy and ready. |

## Final Disposable Matrix Results

The final run used a newly seeded approved work package and current assignment. The matrix driver asserted each protocol outcome itself and exited successfully. The supplementary aggregate reporter was intentionally restricted to the disposable device and prints counts only; it does not expose identifiers or private material.

| Scenario | Driver-asserted protocol category | Aggregate durable state immediately after scenario | Result |
| --- | --- | --- | --- |
| Valid signed binding | `APPLIED` | `receipts_total=1`, `applied=1`, `duplicate=0`, `security_failure=0`, `device_state_rows=1`, `max_accepted_sequence=1`, `held_transactions=0` | Passed. One accepted receipt and a single accepted sequence were durably recorded. |
| Exact duplicate replay | `DUPLICATE` | Unchanged: `receipts_total=1`, `applied=1`, `device_state_rows=1`, `max_accepted_sequence=1`, `held_transactions=0` | Passed. Replay produced the expected duplicate category without an additional durable receipt or sequence advance. |
| Invalid signature | `SECURITY_FAILURE` | Unchanged: `receipts_total=1`, `applied=1`, `device_state_rows=1`, `max_accepted_sequence=1`, `held_transactions=0` | Passed. Signature failure produced the expected security category without a receipt, state advance, or held transaction. |

> The duplicate and signature-failure category checks are performed by the matrix driver against the candidate response. The unchanged aggregate counts are the corroborating durable-state observation, not a substitution for those response assertions.

## Evidence Collection Note

An earlier aggregate observer initially queried outside the tenant row-level-security context and therefore reported zero rows despite a successful matrix. It was corrected to establish the tenant context in one read-only transaction, rebuilt, repository-tested, and rerun with a new disposable fixture. The table above is the authoritative final evidence. The correction affected only the aggregate observation helper; it did not change candidate policy composition, the candidate launcher, or `cmd/integin-server/main.go`.

Repository-wide validation completed after the evidence additions: `go test ./... -count=1` and `go vet ./...` both completed successfully.

## Governance Conclusion

The data-backed loopback candidate demonstrated the expected accepted, duplicate, and signature-failure categories against an approved package and current assignment, with durable state confined to the valid transaction. Rollback was explicit and acceptance stayed healthy. Package enforcement remains disabled in all persistent runtimes pending review of this evidence and a separate authorization decision.

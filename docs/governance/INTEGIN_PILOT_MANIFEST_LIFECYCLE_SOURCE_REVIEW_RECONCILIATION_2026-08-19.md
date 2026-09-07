# INTEGIN Pilot Manifest Lifecycle Source Review Reconciliation

**Recorded:** 2026-08-19  
**Template:** Delegate and Debate — complex review, deep effort, high evidence burden.  
**Decision owner:** INTEGIN engineering owner.  
**Decision:** Commit the source-only prerequisite; keep candidate composition closed.

## Review scope

The review covered only the non-sensitive lifecycle artifact writer, its focused tests, the advisory Field bridge and test, the read-only public v2 adapter, and the lifecycle/Field seam design record. No reviewer received protected fixture content, credentials, database data, logs, runtime configuration, or authority-bearing material.

## Independent perspectives and reconciliation

| Lens | Result | Owner reconciliation |
| --- | --- | --- |
| Security and safety | Approved with required lifecycle durability alignment and explicit unsafe-directory coverage. It also questioned cross-platform hard-link assumptions and source-to-adapter size alignment. | Accepted the durability and unsafe-directory repair. The existing approved v1/v2 writer already uses the same `os.Link` publication primitive, focused Windows Go tests passed, and the lifecycle artifacts directly use the shared `maxReceiptBytes` constant. No incompatible fallback was introduced. |
| Quality and verification | Approved with limitations. It confirmed the bridge is non-interfering, the adapter rejects failure/incomplete lifecycle states, and no default composition occurs. | Accepted. The focused Flutter runner stall remains documented as a limitation; complete Dart static analysis passed. |
| Evidence and assurance | The structured response could not be decoded from the selected reviewer. | Not counted as approval. The owner substituted deterministic evidence: focused Go test, full Go suite, Go vet, complete Dart analysis, adapter parser validation, and synthetic public-only positive plus two negative adapter checks. |

## Accepted repair and evidence

The lifecycle artifact publisher now performs the same non-Windows directory sync pattern as the existing receipt writer after atomic link publication. A focused test now rejects a path that is not a receipt directory. Formatting, focused lifecycle tests, repository-wide Go tests, Go vet, and complete Dart analysis passed after this repair.

The synthetic adapter checks proved the public adapter accepts exactly eight valid redacted receipt artifacts only when accompanied by a complete finalization artifact, rejects a bridge-failure artifact, and rejects a missing finalization artifact. These are synthetic contract checks, not a candidate or acceptance proof.

## Rejected shortcuts

The owner rejected candidate composition, fixture access, default server or Field bootstrap injection, manual runtime changes, rerunning the stalled Flutter test indefinitely, and treating a successful compile as live matrix evidence.

## Residual risk and boundary

The Windows Flutter test runner still has a bounded no-output stall limitation. It does not invalidate complete Dart analysis or the direct Field code review, but it prevents claiming that focused Flutter execution passed. No live manifest case is proven. The isolated candidate emitter, any run-context propagation, and Field pilot process composition remain closed pending the next explicit hard-gate plan.

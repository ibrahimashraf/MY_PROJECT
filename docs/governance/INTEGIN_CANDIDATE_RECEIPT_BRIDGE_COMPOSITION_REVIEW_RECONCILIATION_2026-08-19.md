# INTEGIN Candidate Receipt-Bridge Composition Review Reconciliation

**Recorded:** 2026-08-19  
**Template:** Delegate and Debate — complex architecture review, deep effort, high evidence burden.  
**Decision owner:** INTEGIN engineering owner.  
**Decision:** Accept the design record only. Do not implement, compose, launch, or transport anything under this decision.

## Independent review outcome

| Lens | Verdict | Material conclusion |
| --- | --- | --- |
| Security and isolation | Approved with limitations. | The design keeps run context outside the fixture, default server startup, and default Field bootstrap. It correctly treats context as correlation rather than a secret or authority token. |
| Field provenance and non-interference | Approved with limitations. | Deferring the concrete Field publisher is correct. A nonsecret run ID alone is insufficient provenance for a trustworthy Field case, and the existing observer must remain advisory. |
| Assurance and verification | Approved with limitations. | The design identifies all material prerequisites: typed/closed failure mapping, verified-scope replay arithmetic, expiry-status alignment, deterministic mapper tests, finalization ordering, redaction, and Field transport provenance. |

No reviewer approved candidate composition, fixture context propagation, route mounting, candidate execution, Field traffic, migration, runtime change, or a live proof-matrix claim.

## Owner reconciliation

The design is accepted because it identifies, rather than hides, the four material source gaps that make immediate composition unsafe:

1. The handler has no source-owned verified-scope counter snapshot for `valid_proof` replay arithmetic.
2. The current string-derived proof-failure categorization is unsuitable as a closed public-evidence mapper.
3. The observed issuer-expiry HTTP path and the current v2 allowed-status contract are inconsistent.
4. The Field publisher has no reviewed provenance/transport design, so the Field case cannot yet support a complete matrix claim.

The plan’s mandatory negative cases and stop conditions are accepted. The earlier correlation-contract wording is corrected for future work: the protected fixture receives no receipt context or receipt capability. The candidate and a future separately provisioned Field pilot process are the only possible correlation targets, subject to their own later reviews.

## Rejected shortcuts

The owner rejected silently mapping expiry status, retaining string parsing for public case classification, querying aggregate replay state after failed proof verification, passing context to the fixture, treating the Field abstraction as evidence provenance, allowing candidate self-finalization, and invoking a candidate before source-only gaps are proven.

## Next authorization gate

The next permitted work is one bounded source-only implementation plan for the **closed HTTP bridge prerequisites**: typed/explicit proof-failure mapping, verified-scope counter interface, expiry-status contract resolution, and mapper tests. It must still leave Field publication abstract, the candidate unlaunched, the fixture isolated, default runtime untouched, and all persistent authority controls disabled. Any request to compose the candidate or send Field traffic requires another explicit review and authorization.

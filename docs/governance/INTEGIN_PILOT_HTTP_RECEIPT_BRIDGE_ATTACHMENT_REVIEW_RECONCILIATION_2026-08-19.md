# INTEGIN Pilot HTTP Receipt-Bridge Attachment Review Reconciliation

**Recorded:** 2026-08-19  
**Template:** Delegate and Debate — complex source-attachment design review.  
**Decision owner:** INTEGIN engineering owner.  
**Decision:** Accept the revised plan as a design baseline only; require implementation-time review before any commit or composition decision.

## Evidence and review handling

The usable independent runtime-isolation review found no design blocker. It required explicit safeguards for all-empty context filesystem neutrality, loading context only after existing explicit pilot checks, exact replay-store/counter pointer identity, fail-closed counter absence, complete negative testing, and clear statement that no current runtime provides the candidate-only context values. These requirements were added to the plan.

A second independent assurance review did not return within its bounded window and was stopped. It is **not** counted as approval. Because this is a source-only design record rather than an implementation or runtime decision, the owner accepts the repaired plan with the restriction that any attachment implementation must obtain fresh implementation-focused review and validation before commit.

## Preserved decision boundaries

The planned attachment is not approved for coding or composition under this record. It does not permit candidate start, wrapper/launcher change, fixture context, Field transport, route mounting, migration, database change, acceptance activity, package enforcement, OIDC, or OpenBao work. No runtime behavior is claimed.

## Next explicit gate

The next permitted action is a bounded source-only attachment implementation proposal with tests. It must use the exact existing `*workpackagepg.ManifestProofReplayStore` instance for both consume and count, use `Load` only after pilot gates pass, keep all-empty context inert without filesystem writes, fail closed on counter/context faults, and leave all operational propagation and Field work outside scope. That implementation requires a fresh review before commit.

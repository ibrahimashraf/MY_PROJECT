# INTEGIN Field Control Model Roadmap 
 
Status: Proposed product and security control model. 
 
## Baseline Controls 
1. Data and version integrity: signed manifests, immutable template/package/hash, stable field IDs, append-only answer ledger, and version-specific server validation. 
2. Completion policy: compatible, review-required, and mandatory-safety-correction release classes with effective dates, grace periods, changelogs, and Needs form update blocking. 
3. Device and software trust: minimum app release, signed build, local-schema migration, authority epoch, accepted key status, revocation, and server refusal. 
 
## Required Rule 
Preserve original offline work. The server classifies each release and independently decides whether the draft may sync, needs review, or needs a mandatory safety update. 
 
## Higher-Assurance Extensions 
- Managed-device update enforcement for enterprise fleets. 
- Device and app attestation for regulated or high-risk work. 
- Immediate safety revocation for critical procedure defects. 
- Dual review after a mandatory safety update. 
- Signed update transparency log for release, package, authority, and key policy changes. 

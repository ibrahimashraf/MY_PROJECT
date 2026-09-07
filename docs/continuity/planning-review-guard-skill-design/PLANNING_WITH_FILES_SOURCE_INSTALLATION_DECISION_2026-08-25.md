# Planning-With-Files Online Source Installation Decision

## Requested action

Install the online `planning-with-files` source together with the current local package.

## Evidence boundary

The previously audited online comparison point is `OthmanAdi/planning-with-files` tag `v3.11.2`, commit `9e94390e5912b1ff296556505cd999ff84838160`. Its canonical subtree contains 29 artifacts, including an instruction document, hooks/automation metadata, scripts, templates, and references. The prior static review observed process-execution, install/dependency, network, credential, and host-automation terms as risk signals only; it did not establish safety, compatibility, or behavior.

## Installation decision

| Option | Description | Decision |
|---|---|---|
| Direct overwrite | Replace the active package with upstream files. | Rejected: would copy external code/prompt text and introduce unverified host hooks/scripts. |
| Direct merge | Combine selected upstream files with the local package. | Rejected: same clean-room and host-compatibility failure; merge provenance would be ambiguous. |
| Install beside local package | Add the upstream package as a second active skill. | Rejected: duplicate trigger and ambiguous authority; still imports unverified artifacts. |
| Clean-room parity reconciliation | Maintain an artifact ledger, audit later upstream revisions read-only, and independently write/validate local equivalents for every compatible capability. | Allowed and already used for the current local package. |
| Official host integration | Add lifecycle automation only after the host publishes a verified hook contract and the owner approves project binding, privacy, failure, and bypass policy. | Deferred; no compatible host interface is currently evidenced. |

## Current local capability position

The active package already contains independently written local equivalents for all 27 non-host-dependent audited roles, plus the separately deployed disabled PC-local bridge. Automatic context injection and a forced completion gate remain unavailable as automatic behavior because the current session platform does not provide a verified lifecycle hook interface to invoke the local PC bridge.

## Next safe update path

Keep the current local package active. For a future upstream revision, run a new bounded read-only source audit, update the artifact register, classify every changed role, and implement only independently written local changes that pass deterministic tests. This preserves the user-visible capabilities while retaining rollback and clean provenance.

# INTEGIN Field Compatibility and Completion Gate v1 
 
Status: Proposed contract. 
 
## Compatibility Manifest 
The server publishes signed minimum app release, local schema version, assigned package version/hash, authority epoch/expiry, and accepted signing-key identifiers. 
 
## Final Completion Rule 
Local drafts remain preserved. Final completion and authoritative sync are blocked when app, schema, package, authority, or key state is incompatible. The server independently enforces the same decision. 
 
## Update and Form Change Rule 
The app shows an update notice and signed changelog. Existing unsynced work is preserved with its original template version, package hash, and captured sequence. 
If a new required field or mandatory safety change applies, the draft becomes Needs form update and cannot sync until the new required additions are completed. If no applicable required change exists, the original draft may sync under version-specific server policy. 
 
## Release Classes 
- Compatible: original work may sync. 
- Review required: inspector or operator must review the approved change. 
- Mandatory safety correction: final sync blocks until the approved update is completed. 
